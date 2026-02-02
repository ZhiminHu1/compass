package agent

import (
	"context"
	"fmt"
	"log"
	"os"

	"cowork-agent/llm/parser"
	"cowork-agent/llm/providers"
	"cowork-agent/llm/tools"
	"cowork-agent/llm/vector"
	"cowork-agent/pubsub"

	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/cozeloop-go"
)

// Runtime Agent 运行时
type Runtime struct {
	agent       adk.Agent
	runner      *adk.Runner
	store       ConversationStore
	broker      *pubsub.Broker[adk.Message]
	ctx         context.Context
	cancelFunc  context.CancelFunc
	cozeClient  cozeloop.Client
	vectorStore vector.VectorStore // Vector store for knowledge base
}

// NewRuntime 创建新的 Agent 运行时
func NewRuntime(ctx context.Context, chatModel model.ToolCallingChatModel, toolsList []tool.BaseTool) (*Runtime, error) {
	// 创建 TechTutor Agent
	agt, err := NewTechTutorAgent(ctx, &TechTutorConfig{
		ChatModel: chatModel,
		Tools:     toolsList,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Agent 失败: %w", err)
	}

	// 创建 Runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agt,
		EnableStreaming: false, // 非流式
	})

	// 创建消息 Broker
	broker := pubsub.NewBroker[adk.Message]()

	// 创建上下文
	childCtx, cancel := context.WithCancel(ctx)

	return &Runtime{
		agent:      agt,
		runner:     runner,
		store:      NewMemoryStore(),
		broker:     broker,
		ctx:        childCtx,
		cancelFunc: cancel,
	}, nil
}

// Run 运行 Agent 处理用户输入
func (r *Runtime) Run(userPrompt string) error {
	// 创建用户消息
	userMsg := &schema.Message{
		Role:    schema.User,
		Content: userPrompt,
	}

	// 添加到存储
	if err := r.store.Add(r.ctx, userMsg); err != nil {
		return fmt.Errorf("存储用户消息失败: %w", err)
	}
	// 发布消息
	r.broker.Publish(pubsub.CreatedEvent, userMsg)

	// 获取历史消息
	history, err := r.store.List(r.ctx)
	if err != nil {
		return fmt.Errorf("获取历史消息失败: %w", err)
	}

	// 运行 Agent
	iter := r.runner.Run(r.ctx, history)

	// 处理事件并发布消息
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		r.handleAgentEvent(event)
	}
	r.broker.Publish(pubsub.FinishedEvent, nil)

	return nil
}

// handleAgentEvent 处理 ADK Agent 事件
func (r *Runtime) handleAgentEvent(event *adk.AgentEvent) {
	if event.Output == nil {
		return
	}

	output := event.Output.MessageOutput
	if output == nil {
		return
	}

	// 获取消息
	msg, err := output.GetMessage()
	if err != nil {
		log.Printf("获取消息失败: %v", err)
		// 发布错误消息
		r.broker.Publish(pubsub.UpdatedEvent, &schema.Message{
			Role:    schema.System,
			Content: fmt.Sprintf("错误: %v", err),
		})
		return
	}

	// 添加到存储
	if err := r.store.Add(r.ctx, msg); err != nil {
		log.Printf("存储消息失败: %v", err)
	}

	// 发布消息到 Broker（处理中的更新事件）
	r.broker.Publish(pubsub.UpdatedEvent, msg)
}

// Broker 获取消息 Broker
func (r *Runtime) Broker() *pubsub.Broker[adk.Message] {
	return r.broker
}

// Store 获取对话存储
func (r *Runtime) Store() ConversationStore {
	return r.store
}

// Close 关闭运行时
func (r *Runtime) Close() {
	r.cancelFunc()
	r.broker.Shutdown()
	// 关闭向量存储
	if r.vectorStore != nil {
		if err := r.vectorStore.Close(); err != nil {
			log.Printf("关闭向量存储失败: %v", err)
		}
	}
	// 关闭 Coze Loop 客户端
	if r.cozeClient != nil {
		r.cozeClient.Close(r.ctx)
	}
}

// SetupRuntime 设置 Runtime（从 main.go 调用）
func SetupRuntime(ctx context.Context) (*Runtime, error) {
	// 初始化 Coze Loop 观测
	cozeClient := initCozeLoop(ctx)

	// 创建 ChatModel
	chatModel, err := providers.CreateChatModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建 ChatModel 失败: %w", err)
	}

	// 初始化向量存储
	vectorStore, embedder, err := initVectorStore(ctx)
	if err != nil {
		log.Printf("初始化向量存储失败: %v (知识库功能将被禁用)", err)
		// Continue without vector store - knowledge tools will handle nil case
	} else {
		log.Println("向量存储已启用")
	}

	// 创建工具列表
	toolsList, err := createTools(ctx, vectorStore, embedder)
	if err != nil {
		return nil, fmt.Errorf("创建工具失败: %w", err)
	}

	runtime, err := NewRuntime(ctx, chatModel, toolsList)
	if err != nil {
		// Cleanup vector store if runtime creation fails
		if vectorStore != nil {
			vectorStore.Close()
		}
		return nil, err
	}
	runtime.cozeClient = cozeClient
	runtime.vectorStore = vectorStore

	return runtime, nil
}

// initVectorStore 初始化向量存储
func initVectorStore(ctx context.Context) (vector.VectorStore, embedding.Embedder, error) {
	// 检查是否启用 Redis 向量存储
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		return nil, nil, fmt.Errorf("REDIS_ADDR not set")
	}

	// 创建 embedding 模型
	embedder, err := providers.CreateEmbeddingModel(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("创建 embedding 模型失败: %w", err)
	}

	// 创建 Redis 向量存储
	redisConfig := vector.DefaultRedisConfig()
	vectorStore, err := vector.NewRedisStore(ctx, embedder, redisConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("创建 Redis 向量存储失败: %w", err)
	}

	// 初始化解析器注册表
	parserRegistry := parser.DefaultRegistry()

	// 初始化知识工具
	tools.InitKnowledgeVectorStore(vectorStore, parserRegistry, embedder)

	return vectorStore, embedder, nil
}

// initCozeLoop 初始化 Coze Loop 观测
func initCozeLoop(ctx context.Context) cozeloop.Client {
	cozeloopApiToken := os.Getenv("COZE_LOOP_API_TOKEN")
	cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID")

	if cozeloopApiToken == "" || cozeloopWorkspaceID == "" {
		log.Println("Coze Loop 未配置（需要 COZE_LOOP_API_TOKEN 和 COZELOOP_WORKSPACE_ID 环境变量）")
		return nil
	}

	client, err := cozeloop.NewClient(
		cozeloop.WithAPIToken(cozeloopApiToken),
		cozeloop.WithWorkspaceID(cozeloopWorkspaceID),
	)
	if err != nil {
		log.Printf("创建 Coze Loop 客户端失败: %v", err)
		return nil
	}

	log.Println("Coze Loop 观测已启用")

	// 注册全局回调处理器
	handler := clc.NewLoopHandler(client)
	callbacks.AppendGlobalHandlers(handler)

	return client
}

// createTools 创建所有工具
func createTools(ctx context.Context, vs vector.VectorStore, emb embedding.Embedder) ([]tool.BaseTool, error) {
	var toolsList []tool.BaseTool

	// 文件操作工具
	toolsList = append(toolsList, tools.GetReadFileTool())
	toolsList = append(toolsList, tools.GetWriteFileTool())
	toolsList = append(toolsList, tools.GetEditFileTool())
	toolsList = append(toolsList, tools.GetDeleteFileTool())
	toolsList = append(toolsList, tools.GetListDirTool())

	// 搜索工具
	toolsList = append(toolsList, tools.GetGrepTool())
	toolsList = append(toolsList, tools.GetGlobTool())

	// Bash 工具
	toolsList = append(toolsList, tools.GetBashTool())

	// 网络工具
	toolsList = append(toolsList, tools.GetSearchTool())
	toolsList = append(toolsList, tools.GetContentSummaryTool(ctx))

	// 知识库工具 (只在向量存储可用时添加)
	if vs != nil {
		toolsList = append(toolsList, tools.GetKnowledgeTool())
		toolsList = append(toolsList, tools.GetIngestDocumentTool())
		toolsList = append(toolsList, tools.GetListDocumentsTool())
		toolsList = append(toolsList, tools.GetDeleteDocumentTool())
		log.Println("知识库工具已启用")
	}

	return toolsList, nil
}
