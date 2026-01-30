package providers

import (
	"context"
	"fmt"
	"os"

	geminiModel "github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino/components/model"
	"google.golang.org/genai"
)

// CreateGeminiModel creates a Google Gemini chat model from environment variables.
// Required environment variables:
//   - GEMINI_API_KEY: API key for Google Gemini
//
// Optional environment variables:
//   - GEMINI_MODEL: Model name (default: gemini-1.5-pro)
func CreateGeminiModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is required when using google provider")
	}

	modelName := os.Getenv("GEMINI_MODEL")
	if modelName == "" {
		modelName = "gemini-3-flash"
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemai client: %w", err)
	}

	return geminiModel.NewChatModel(ctx, &geminiModel.Config{
		Client: client,
		Model:  modelName,
	})
}
