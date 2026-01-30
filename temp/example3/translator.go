package main

import (
	"context"
	"cowork-agent/temp/example2/providers"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
)

// NewTranslatorAgent creates a specialized agent for translation
func NewTranslatorAgent(ctx context.Context) adk.Agent {
	model, err := providers.CreateChatModel(ctx)
	if err != nil {
		log.Fatal(err)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "translate_text", // The name of the agent becomes the name of the tool
		Description: "Translate text to a specified target language.",
		Instruction: "You are a professional translator. The user request will contain the text and target language (e.g. 'Translate hello to French'). Translate the text accurately. Return ONLY the translated text, no explanations.",
		Model:       model,
	})
	if err != nil {
		log.Fatal(err)
	}
	return agent
}

// NewTranslatorTool wraps the Translator Agent as a Tool using standard adk.NewAgentTool
func NewTranslatorTool(ctx context.Context) tool.BaseTool {
	// Initialize the sub-agent
	translatorAgent := NewTranslatorAgent(ctx)

	// Use the official NewAgentTool capability
	return adk.NewAgentTool(ctx, translatorAgent)
}
