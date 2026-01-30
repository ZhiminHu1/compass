package main

import (
	"bufio"
	"context"
	subagent2 "cowork-agent/temp/example2/subagent"
	"cowork-agent/utils"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/joho/godotenv"
)

func init() {
	// Load .env file if exists
	_ = godotenv.Load()
}
func main() {
	ctx := context.Background()

	agent := subagent2.NewBookRecommendAgent()
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
		Agent:           agent,
		CheckPointStore: store.NewInMemoryStore(),
	})
	iter := runner.Query(ctx, "请你推荐一本书给我 请中文回答，必要时，请你调用中断tool", adk.WithCheckPointID("1"))
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatal(event.Err)
		}

		utils.PrintEvent(event)
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("\nyour input here: ")
	for {
		scanner.Scan()
		fmt.Println()
		nInput := scanner.Text()
		if nInput == "exit" {
			break
		}
		//iter, err := runner.Resume(ctx, "1", adk.WithToolOptions([]tool.Option{subagent.WithNewInput(nInput)}))
		iter, err := runner.Resume(ctx, "1", adk.WithToolOptions([]tool.Option{subagent2.WithNewInput(nInput)}))
		if err != nil {
			log.Fatal(err)
		}
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}

			if event.Err != nil {
				log.Fatal(event.Err)
			}

			//prints.Event(event)
			utils.PrintEvent(event)
		}
	}

}
