package main

import (
	"context"
	"cowork-agent/llm/tools"
	"cowork-agent/temp/example2/providers"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// NewSummaryAgent creates a specialized agent for summarizing web pages
func NewSummaryAgent(ctx context.Context) adk.Agent {
	model, err := providers.CreateChatModel(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Get the fetch tool
	fetchTool := tools.GetFetchTool()

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "summarize_url",
		Description: "Fetches a URL and provides a concise summary of its content.",
		Instruction: "You are a web summarizer. The user will provide a URL (and optionally a query). Use the 'fetch_web_content' tool to get the page content, then summarize it relevant to the user's intent. Return ONLY the summary.",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               []tool.BaseTool{fetchTool},
				ToolCallMiddlewares: []compose.ToolMiddleware{ErrorHandler()},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	return agent
}

// NewSummaryTool wraps the Summary Agent as a Tool using standard adk.NewAgentTool
func NewSummaryTool(ctx context.Context) tool.BaseTool {
	summaryAgent := NewSummaryAgent(ctx)
	return adk.NewAgentTool(ctx, summaryAgent)
}
