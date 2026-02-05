package providers

import (
	"context"
	"fmt"
	"os"

	openaiEmbed "github.com/cloudwego/eino-ext/components/embedding/openai"
	openaiModel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	einoEmbedding "github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
)

// ChatModelConfig defines the configuration for creating a chat model.
type ChatModelConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// NewChatModel creates an OpenAI-compatible chat model from specific configuration.
func NewChatModel(ctx context.Context, config *ChatModelConfig) (model.ToolCallingChatModel, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required in config")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://open.bigmodel.cn/api/paas/v4"
	}

	modelName := config.Model
	if modelName == "" {
		modelName = "glm-4-flash"
	}

	return openaiModel.NewChatModel(ctx, &openaiModel.ChatModelConfig{
		APIKey:  config.APIKey,
		BaseURL: baseURL,
		Model:   modelName,
	})
}

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

	return NewChatModel(ctx, &ChatModelConfig{
		APIKey:  apiKey,
		BaseURL: os.Getenv("BASE_URL"),
		Model:   os.Getenv("MODEL"),
	})
}

func CreateSummaryModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	apiKey := os.Getenv("SUMMARY_MODEL_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API_KEY environment variable is required")
	}

	return qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		APIKey:  apiKey,
		BaseURL: os.Getenv("SUMMARY_MODEL_BASE_URL"),
		Model:   os.Getenv("SUMMARY_MODEL"),
	})
}

// EmbeddingConfig defines the configuration for creating an embedding model.
type EmbeddingConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// NewEmbeddingModel creates an OpenAI-compatible embedding model from specific configuration.
func NewEmbeddingModel(ctx context.Context, config *EmbeddingConfig) (einoEmbedding.Embedder, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required in config")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://open.bigmodel.cn/api/paas/v4"
	}

	modelName := config.Model
	if modelName == "" {
		modelName = "embedding-3"
	}

	return openaiEmbed.NewEmbedder(ctx, &openaiEmbed.EmbeddingConfig{
		APIKey:  config.APIKey,
		BaseURL: baseURL,
		Model:   modelName,
	})
}

// CreateEmbeddingModel creates an OpenAI-compatible embedding model from environment variables.
func CreateEmbeddingModel(ctx context.Context) (einoEmbedding.Embedder, error) {
	apiKey := os.Getenv("EMBEDDING_MODEL_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("API_KEY environment variable is required")
	}

	return NewEmbeddingModel(ctx, &EmbeddingConfig{
		APIKey:  apiKey,
		BaseURL: os.Getenv("EMBEDDING_MODEL_BASE_URL"),
		Model:   os.Getenv("EMBEDDING_MODEL"),
	})
}
