package vector

import (
	"context"
	"cowork-agent/llm"
)

// VectorStore defines the interface for vector storage operations
type VectorStore interface {
	// Add adds a single document to the store
	Add(ctx context.Context, doc llm.Document) error

	// AddBatch adds multiple documents in a single operation
	AddBatch(ctx context.Context, docs []llm.Document) error

	// Search performs semantic search and returns top-k results
	Search(ctx context.Context, query string, topK int) ([]llm.SearchResult, error)

	// Delete removes a document by its ID
	Delete(ctx context.Context, id string) error

	// DeleteBySource removes all documents from a specific source file
	DeleteBySource(ctx context.Context, source string) error

	// List returns documents matching the filter criteria
	List(ctx context.Context, filter llm.ListFilter) ([]llm.Document, error)

	// Count returns the total number of documents in the store
	Count(ctx context.Context) (int64, error)

	// Close closes any connections or resources
	Close() error
}

// StoreConfig holds configuration for vector store implementations
type StoreConfig struct {
	// Embedding dimension (must match the embedding model)
	EmbeddingDim int

	// Index name for the vector index
	IndexName string

	// Key prefix for stored documents
	KeyPrefix string
}

// DefaultStoreConfig returns default configuration
func DefaultStoreConfig() StoreConfig {
	return StoreConfig{
		EmbeddingDim: 1024,
		IndexName:    "cowork-knowledge",
		KeyPrefix:    "vec:",
	}
}
