package main

import (
	"bufio"
	"context"
	"cowork-agent/temp/example2/providers"
	vectorstore2 "cowork-agent/temp/example4/vectorstore"
	"cowork-agent/utils"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-examples/adk/common/store"
	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/coze-dev/cozeloop-go"
	"github.com/joho/godotenv"
)

func init() {
	// Load .env from root if possible, or assume env vars set
	_ = godotenv.Load("../.env")
	// Also try current dir just in case
	_ = godotenv.Load()
}

func main() {
	ctx := context.Background()
	cozeloopApiToken := os.Getenv("COZE_LOOP_API_TOKEN")
	cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID")
	var handlers []callbacks.Handler
	if cozeloopApiToken != "" && cozeloopWorkspaceID != "" {
		client, err := cozeloop.NewClient(
			cozeloop.WithAPIToken(cozeloopApiToken),
			cozeloop.WithWorkspaceID(cozeloopWorkspaceID),
		)
		if err != nil {
			panic(err)
		}
		defer func() {
			time.Sleep(5 * time.Second)
			client.Close(ctx)
		}()
		handlers = append(handlers, clc.NewLoopHandler(client))
	}
	callbacks.AppendGlobalHandlers(handlers...)

	// 1. Initialize embedding model
	embeddingModel, err := providers.CreateEmbeddingModel(ctx)
	if err != nil {
		log.Printf("Warning: Failed to create embedding model: %v. Knowledge base features will be disabled.", err)
		embeddingModel = nil
	}

	// 2. Initialize vector store
	var vectorStore *vectorstore2.VectorStore
	if embeddingModel != nil {
		vectorStore, err = vectorstore2.NewVectorStore("./data/knowledge_store.json", embeddingModel)
		if err != nil {
			log.Printf("Warning: Failed to create vector store: %v. Knowledge base features will be disabled.", err)
			vectorStore = nil
		} else {
			if err := vectorStore.Load(); err != nil {
				log.Printf("Warning: Failed to load vector store: %v", err)
				// Continue anyway - will create new store
			}
			docCount := vectorStore.GetDocumentCount()
			if docCount > 0 {
				fmt.Printf("[知识库] 已加载 %d 个文档\n", docCount)
			}
		}
	}

	// 3. Create agent with vector store
	agent := NewResearchAgent(ctx, vectorStore)

	// 4. Setup runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
		CheckPointStore: store.NewInMemoryStore(),
	})

	// 5. Run first query
	firstQuery := "请你分析当前时间股市情况"
	fmt.Printf("\nUser Query: %s\n\n", firstQuery)
	runIteration(ctx, runner, "1", vectorStore, "", firstQuery)

	// 6. Handle interrupts for save to knowledge base
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n输入新查询或 'exit' 退出: ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()
		if strings.ToLower(input) == "exit" {
			break
		}
		if input == "" {
			continue
		}

		fmt.Printf("\nUser Query: %s\n\n", input)
		runIteration(ctx, runner, "1", vectorStore, "", input)
	}

	fmt.Println("\n程序结束。")
}

// runIteration 执行一次查询迭代
func runIteration(ctx context.Context, runner *adk.Runner, runPath string, vectorStore *vectorstore2.VectorStore, toolOption string, query string) {
	iter := runner.Query(ctx, query, adk.WithCheckPointID(runPath))
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatal(event.Err)
		}

		// 处理中断事件
		if event.Action != nil && event.Action.Interrupted != nil {
			handleInterrupt(ctx, runner, event, runPath, vectorStore)
			return // 中断后退出，等待用户输入新查询
		}

		utils.PrintEvent(event)
	}
}

// handleInterrupt 处理中断事件（询问用户是否保存到知识库）
func handleInterrupt(ctx context.Context, runner *adk.Runner, event *adk.AgentEvent, runPath string, vectorStore *vectorstore2.VectorStore) {
	interrupted := event.Action.Interrupted
	if len(interrupted.InterruptContexts) == 0 {
		return
	}

	// 获取中断信息
	ic := interrupted.InterruptContexts[0]

	// 尝试获取 markdown 内容
	var markdownContent string

	// 检查是否是 SaveKnowledgeContext 类型
	if ctx, ok := ic.Info.(SaveKnowledgeContext); ok {
		markdownContent = ctx.Markdown
	}

	// 显示 Markdown 内容预览
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("【研究结果已生成，是否保存到知识库？】")
	fmt.Println(strings.Repeat("=", 70))

	if markdownContent != "" {
		// 显示预览（前 500 字）
		preview := markdownContent
		if len(markdownContent) > 500 {
			preview = markdownContent[:500] + "\n...\n[完整内容较长，仅显示前 500 字]"
		}
		fmt.Println("\n内容预览:")
		fmt.Println(strings.Repeat("-", 70))
		fmt.Println(preview)
		fmt.Println(strings.Repeat("-", 70))
	} else {
		fmt.Println("\n(未显示完整内容)")
	}

	// 获取用户输入
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\n是否保存到知识库？(y/n): ")

	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("读取输入失败: %v\n", err)
		// 继续执行（不保存）
		resumeWithChoice(ctx, runner, runPath, "no", vectorStore, "")
		return
	}

	choice := strings.TrimSpace(strings.ToLower(input))
	var saveChoice string
	var actualMarkdown string

	if choice == "y" || choice == "yes" {
		saveChoice = "yes"
		actualMarkdown = markdownContent
		fmt.Println("\n[✓] 用户选择保存，正在处理...")
	} else {
		saveChoice = "no"
		fmt.Println("\n[✗] 用户选择不保存。")
	}

	// 恢复执行，传入用户选择
	resumeWithChoice(ctx, runner, runPath, saveChoice, vectorStore, actualMarkdown)
}

// resumeWithChoice 恢复执行并传入用户选择
func resumeWithChoice(ctx context.Context, runner *adk.Runner, runPath string, choice string, vectorStore *vectorstore2.VectorStore, markdown string) {
	// 如果用户选择保存，先保存到向量存储
	if choice == "yes" && vectorStore != nil && markdown != "" {
		saveToVectorStore(ctx, vectorStore, markdown)
	}

	// 使用 Resume 继续执行，传入用户选择
	iter, err := runner.Resume(ctx, runPath, adk.WithToolOptions([]tool.Option{WithSaveChoice(choice)}))
	if err != nil {
		log.Printf("Resume failed: %v", err)
		return
	}

	// 继续处理后续事件
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Printf("Event error: %v", event.Err)
			break
		}

		// 如果再次中断（不应该发生），递归处理
		if event.Action != nil && event.Action.Interrupted != nil {
			handleInterrupt(ctx, runner, event, runPath, vectorStore)
			return
		}

		utils.PrintEvent(event)
	}
}

// saveToVectorStore 保存 Markdown 内容到向量存储
func saveToVectorStore(ctx context.Context, vs *vectorstore2.VectorStore, markdown string) {
	fmt.Println("\n正在保存到知识库...")

	config := vectorstore2.DefaultConfig()
	chunks := vectorstore2.SplitDocument(markdown, config)

	savedCount := 0
	for _, chunk := range chunks {
		metadata := map[string]interface{}{
			"source":      "research",
			"timestamp":   time.Now().Format(time.RFC3339),
			"chunk_index": chunk.ChunkIndex,
		}

		if err := vs.AddDocument(ctx, chunk.Content, metadata); err != nil {
			log.Printf("Warning: Failed to add chunk %d: %v", chunk.ChunkIndex, err)
		} else {
			savedCount++
		}
	}

	// 保存到磁盘
	if err := vs.Save(); err != nil {
		log.Printf("Error: Failed to save vector store: %v", err)
		return
	}

	fmt.Printf("[✓] 成功保存 %d 个文档块到知识库 (总共 %d 个文档)\n", savedCount, vs.GetDocumentCount())
}

// ErrorHandler 工具错误处理中间件
func ErrorHandler() compose.ToolMiddleware {
	return compose.ToolMiddleware{
		Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
			return func(ctx context.Context, in *compose.ToolInput) (*compose.ToolOutput, error) {
				output, err := next(ctx, in)
				if err != nil {
					errStr := err.Error()
					// 跳过中断信号
					if strings.Contains(errStr, "interrupt signal") {
						return nil, err
					}

					// 处理普通错误：提取核心错误信息
					if idx := strings.Index(errStr, "err="); idx != -1 {
						coreErr := strings.TrimSpace(errStr[idx+4:])
						// 将错误转换为成功的工具结果
						return &compose.ToolOutput{
							Result: fmt.Sprintf("error! %s", coreErr),
						}, nil
					}
					// 如果没有找到 err=，返回原始错误
					return &compose.ToolOutput{
						Result: fmt.Sprintf("error! %s", errStr),
					}, nil
				}
				return output, nil
			}
		},
	}
}
