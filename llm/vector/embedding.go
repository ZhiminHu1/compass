package vector

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/cloudwego/eino/components/embedding"
)

// EmbeddingService wraps an embedding model for vector generation
type EmbeddingService struct {
	embedder embedding.Embedder
	dim      int
	mu       sync.RWMutex
}

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService(embedder embedding.Embedder, dim int) *EmbeddingService {
	if dim <= 0 {
		dim = 1024 // Default dimension for many models
	}
	return &EmbeddingService{
		embedder: embedder,
		dim:      dim,
	}
}

// Embed generates an embedding vector for a single text
func (s *EmbeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	vectors, err := s.embedder.EmbedStrings(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(vectors) == 0 || len(vectors[0]) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	// Convert float64 to float32
	result := make([]float32, len(vectors[0]))
	for i, v := range vectors[0] {
		result[i] = float32(v)
	}

	return result, nil
}

// EmbedBatch generates embedding vectors for multiple texts
func (s *EmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("texts cannot be empty")
	}

	// Filter out empty texts
	var validTexts []string
	var indices []int
	for i, text := range texts {
		if text != "" {
			validTexts = append(validTexts, text)
			indices = append(indices, i)
		}
	}

	if len(validTexts) == 0 {
		return nil, fmt.Errorf("no valid texts to embed")
	}

	vectors, err := s.embedder.EmbedStrings(ctx, validTexts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Convert all vectors to float32
	result := make([][]float32, len(texts))
	for i, vec := range vectors {
		result[indices[i]] = make([]float32, len(vec))
		for j, v := range vec {
			result[indices[i]][j] = float32(v)
		}
	}

	return result, nil
}

// Dimension returns the embedding dimension
func (s *EmbeddingService) Dimension() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dim
}

// GetEmbeddingDimFromEnv reads embedding dimension from environment variable
func GetEmbeddingDimFromEnv() int {
	dim := 1024 // Default
	if val := os.Getenv("VECTOR_DIM"); val != "" {
		if n, err := parseDim(val); err == nil && n > 0 {
			dim = n
		}
	}
	return dim
}

// parseDim parses dimension string to integer
func parseDim(s string) (int, error) {
	var dim int
	if _, err := fmt.Sscanf(s, "%d", &dim); err != nil {
		return 0, err
	}
	return dim, nil
}
