package main

import (
	"context"
	"cowork-agent/temp/example2/providers"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// NewTravelAgent creates the main agent that uses the Translator tool
func NewTravelAgent(ctx context.Context) adk.Agent {
	model, err := providers.CreateChatModel(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Create the tool that wraps the sub-agent
	translatorTool := NewTranslatorTool(ctx)

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "TravelGuide",
		Description: "A helpful travel guide.",
		Instruction: "You are a travel guide. Help users with travel tips. If the user asks for a translation, use the 'translate_text' tool. CRITICAL: Once you get the translation from the tool, STOP calling tools and immediately present the translation to the user.",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{translatorTool},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	return agent
}
