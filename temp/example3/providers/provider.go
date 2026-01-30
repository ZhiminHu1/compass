package providers

import (
	"context"
	"fmt"
	"os"

	openaiEmbed "github.com/cloudwego/eino-ext/components/embedding/openai"
	openaiModel "github.com/cloudwego/eino-ext/components/model/openai"
	einoEmbedding "github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
)

// CreateChatModel creates an OpenAI-compatible chat model from environment variables.
// Required environment variables:
//   - API_KEY: API key for the LLM provider
//
// Optional environment variables:
//   - BASE_URL: Base URL for OpenAI-compatible API (default: https://open.bigmodel.cn/api/paas/v4)
//   - MODEL: Model name (default: glm-4-flash)
func CreateChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API_KEY environment variable is required")
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://open.bigmodel.cn/api/paas/v4"
	}

	modelName := os.Getenv("MODEL")
	if modelName == "" {
		modelName = "glm-4-flash"
	}

	return openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   modelName,
		ExtraFields: map[string]any{
			"Stream": "true",
		},
	})
}

// CreateEmbeddingModel creates an OpenAI-compatible embedding model from environment variables.
func CreateEmbeddingModel(ctx context.Context) (einoEmbedding.Embedder, error) {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API_KEY environment variable is required")
	}

	baseURL := os.Getenv("EMBEDDING_MODEL_BASE_URL")
	if baseURL == "" {
		baseURL = "https://open.bigmodel.cn/api/paas/v4"
	}

	// Default to embedding-2 for Zhipu AI if not specified
	modelName := os.Getenv("EMBEDDING_MODEL")
	if modelName == "" {
		modelName = "embedding-3"
	}

	return openaiEmbed.NewEmbedder(ctx, &openaiEmbed.EmbeddingConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   modelName,
	})
}
