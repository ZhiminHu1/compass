package main

import (
	"bufio"
	"context"
	"cowork-agent/llm/agent"
	"cowork-agent/llm/tools"
	"cowork-agent/pubsub"
	"cowork-agent/temp/example2/providers"
	"cowork-agent/temp/example4/vectorstore"
	"cowork-agent/utils"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
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

	// 1. Initialize embedding model (for Knowledge Tool)
	embeddingModel, err := providers.CreateEmbeddingModel(ctx)
	if err != nil {
		log.Printf("Warning: Failed to create embedding model: %v. Knowledge base features will be disabled.", err)
		embeddingModel = nil
	}

	// 2. Initialize vector store
	var vectorStore *vectorstore.VectorStore
	if embeddingModel != nil {
		// Use a local knowledge store for this example
		vectorStore, err = vectorstore.NewVectorStore("./data/knowledge_store.json", embeddingModel)
		if err != nil {
			log.Printf("Warning: Failed to create vector store: %v. Knowledge base features will be disabled.", err)
			vectorStore = nil
		} else {
			if err := vectorStore.Load(); err != nil {
				log.Printf("Warning: Failed to load vector store: %v", err)
			}
			docCount := vectorStore.GetDocumentCount()
			if docCount > 0 {
				fmt.Printf("[Knowledge] Loaded %d documents\n", docCount)
			}
		}
	}

	// 3. Initialize Chat Model
	chatModel, err := providers.CreateChatModel(ctx)
	if err != nil {
		log.Fatal("Failed to create chat model: ", err)
	}

	// 4. Initialize Tools
	// Research Tools
	searchTool := tools.GetSearchTool()
	fetchTool := tools.GetFetchTool()
	// Knowledge Tool
	var knowledgeTool tool.InvokableTool
	if vectorStore != nil {
		knowledgeTool = tools.GetKnowledgeTool()
	}
	// File System Tools
	listDirTool := tools.GetListDirTool()
	readFileTool := tools.GetReadFileTool()
	writeFileTool := tools.GetWriteFileTool()
	editFileTool := tools.GetEditFileTool()
	deleteFileTool := tools.GetDeleteFileTool()
	// Execution Tools
	bashTool := tools.GetBashTool()

	allTools := []tool.BaseTool{
		searchTool, fetchTool, listDirTool, readFileTool,
		writeFileTool, editFileTool, deleteFileTool, bashTool,
	}
	if knowledgeTool != nil {
		allTools = append(allTools, knowledgeTool)
	}

	// 5. Initialize PubSub Broker
	eventBus := pubsub.NewBroker[adk.Message]()
	defer eventBus.Shutdown()

	// 6. Create TechTutor agent with dependencies
	techTutorAgent, err := agent.NewTechTutorAgent(ctx, &agent.TechTutorConfig{
		ChatModel: chatModel,
		Tools:     allTools,
	})
	if err != nil {
		log.Printf("Failed to create TechTutor agent: %v", err)
		return
	}

	// 4. Setup runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           techTutorAgent,
		CheckPointStore: store.NewInMemoryStore(),
	})

	fmt.Println("==================================================")
	fmt.Println("  TechTutor - Your Technical Learning Assistant  ")
	fmt.Println("==================================================")
	fmt.Println("Type 'exit' to quit.")

	// 5. Interactive Loop
	scanner := bufio.NewScanner(os.Stdin)
	runID := "session_1"

	for {
		fmt.Print("\nYou: ")
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

		fmt.Println("\nTechTutor is thinking...")
		runIteration(ctx, runner, runID, input)
	}

	fmt.Println("\nGoodbye!")
}

// runIteration 执行一次查询迭代
func runIteration(ctx context.Context, runner *adk.Runner, runPath string, query string) {
	iter := runner.Query(ctx, query, adk.WithCheckPointID(runPath))
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Printf("Error: %v", event.Err)
			break
		}

		utils.PrintEvent(event)
	}
}
