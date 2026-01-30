package main

import (
	"context"
	tools2 "cowork-agent/llm/tools"
	"cowork-agent/temp/example1/subagent"
	"fmt"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/common/trace"
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
	traceCloseFn, startSpanFn := trace.AppendCozeLoopCallbackIfConfigured(ctx)
	defer traceCloseFn(ctx)

	toolList := []tool.BaseTool{
		tools2.GetFetchTool(),
		tools2.GetSearchTool(),
	}
	agent, err := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        "DataCollectionAgent",
		Description: "Data Collection Agent could collect data from multiple sources.",
		SubAgents: []adk.Agent{
			subagent.NewStockDataCollectionAgent(toolList),
			subagent.NewNewsDataCollectionAgent(toolList),
			subagent.NewSocialMediaInfoCollectionAgent(toolList),
		},
	})
	if err != nil {
		panic(err)
	}

	query := "give me today's market research，请你使用中文回答"
	ctx, endSpanFn := startSpanFn(ctx, "layered-supervisor", query)
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true, // you can disable streaming here
		Agent:           agent,
	})
	iter := runner.Query(ctx, query)

	var lastMessage adk.Message

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			fmt.Printf("Error: %v\n", event.Err)
			break
		}

		prints.Event(event)
		if event.Output != nil {
			lastMessage, _, err = adk.GetMessage(event)
		}
	}

	endSpanFn(ctx, lastMessage)

}
