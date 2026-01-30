package agent

import (
	"context"
	"fmt"
	"log"

	"cowork-agent/llm/providers"
	"cowork-agent/llm/tools"
	"cowork-agent/pubsub"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// Runtime Agent 运行时
type Runtime struct {
	agent      adk.Agent
	runner     *adk.Runner
	store      ConversationStore
	broker     *pubsub.Broker[adk.Message]
	ctx        context.Context
	cancelFunc context.CancelFunc
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

	// 创建事件 Broker
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

	// 发布用户消息创建事件
	r.broker.Publish(pubsub.CreatedEvent, userMsg)

	// 获取历史消息
	history, err := r.store.List(r.ctx)
	if err != nil {
		return fmt.Errorf("获取历史消息失败: %w", err)
	}

	// 运行 Agent
	iter := r.runner.Run(r.ctx, history)

	// 处理事件
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		r.handleEvent(event)
	}

	return nil
}

// handleEvent 处理 Agent 事件
func (r *Runtime) handleEvent(event *adk.AgentEvent) {
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
		r.broker.Publish(pubsub.CreatedEvent, &schema.Message{
			Role:    schema.System,
			Content: fmt.Sprintf("错误: %v", err),
		})
		return
	}

	// 添加到存储
	if err := r.store.Add(r.ctx, msg); err != nil {
		log.Printf("存储消息失败: %v", err)
	}

	// 发布事件
	r.broker.Publish(pubsub.CreatedEvent, msg)

	// 如果有工具调用，发布工具事件
	if len(msg.ToolCalls) > 0 {
		for _, tc := range msg.ToolCalls {
			toolMsg := &schema.Message{
				Role:    schema.System,
				Content: fmt.Sprintf("调用工具: %s", tc.Function.Name),
			}
			r.broker.Publish(pubsub.UpdatedEvent, toolMsg)
		}
	}
}

// Broker 获取事件 Broker
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
}

// SetupRuntime 设置 Runtime（从 main.go 调用）
func SetupRuntime(ctx context.Context) (*Runtime, error) {
	// 创建 ChatModel
	chatModel, err := providers.CreateChatModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建 ChatModel 失败: %w", err)
	}

	// 创建工具列表
	toolsList, err := createTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建工具失败: %w", err)
	}

	return NewRuntime(ctx, chatModel, toolsList)
}

// createTools 创建所有工具
func createTools(ctx context.Context) ([]tool.BaseTool, error) {
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
	toolsList = append(toolsList, tools.GetFetchTool())

	// 知识库工具
	toolsList = append(toolsList, tools.GetKnowledgeTool())

	return toolsList, nil
}
