package vectorstore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/embedding"
)

// StoreData represents the JSON structure of the knowledge store file
type StoreData struct {
	Version   string     `json:"version"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
	Documents []Document `json:"documents"`
}

// Document represents a single document with its embedding vector
type Document struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Vector   []float32              `json:"vector"`
	Metadata map[string]interface{} `json:"metadata"`
}

// VectorStore manages local vector storage with JSON persistence
type VectorStore struct {
	filePath       string
	mu             sync.RWMutex
	documents      []Document
	embeddingModel embedding.Embedder
	createdAt      time.Time
	updatedAt      time.Time
}

// NewVectorStore creates a new vector store instance
func NewVectorStore(filePath string, model embedding.Embedder) (*VectorStore, error) {
	if model == nil {
		return nil, fmt.Errorf("embedding model is required")
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	return &VectorStore{
		filePath:       filePath,
		documents:      make([]Document, 0),
		embeddingModel: model,
		createdAt:      time.Now(),
		updatedAt:      time.Now(),
	}, nil
}

// Load loads documents from the JSON file
func (vs *VectorStore) Load() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	data, err := os.ReadFile(vs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, that's okay
			vs.documents = make([]Document, 0)
			vs.createdAt = time.Now()
			vs.updatedAt = time.Now()
			return nil
		}
		return fmt.Errorf("failed to read store file: %w", err)
	}

	var storeData StoreData
	if err := json.Unmarshal(data, &storeData); err != nil {
		return fmt.Errorf("failed to parse store data: %w", err)
	}

	vs.documents = storeData.Documents
	if storeData.CreatedAt != "" {
		vs.createdAt, _ = time.Parse(time.RFC3339, storeData.CreatedAt)
	}
	if storeData.UpdatedAt != "" {
		vs.updatedAt, _ = time.Parse(time.RFC3339, storeData.UpdatedAt)
	}

	return nil
}

// Save saves documents to the JSON file
func (vs *VectorStore) Save() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	vs.updatedAt = time.Now()

	storeData := StoreData{
		Version:   "1.0",
		CreatedAt: vs.createdAt.Format(time.RFC3339),
		UpdatedAt: vs.updatedAt.Format(time.RFC3339),
		Documents: vs.documents,
	}

	data, err := json.MarshalIndent(storeData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal store data: %w", err)
	}

	if err := os.WriteFile(vs.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write store file: %w", err)
	}

	return nil
}

// AddDocument adds a new document with embedding to the store
func (vs *VectorStore) AddDocument(ctx context.Context, content string, metadata map[string]interface{}) error {
	if content == "" {
		return fmt.Errorf("content cannot be empty")
	}

	// Generate embedding
	embeddings, err := vs.embeddingModel.EmbedStrings(ctx, []string{content})
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return fmt.Errorf("empty embedding returned")
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Generate unique ID
	timestamp := time.Now().UnixMilli()
	docID := fmt.Sprintf("doc_%d_%d", timestamp, len(vs.documents))

	// Convert float64 to float32 for storage
	vector := make([]float32, len(embeddings[0]))
	for i, v := range embeddings[0] {
		vector[i] = float32(v)
	}

	doc := Document{
		ID:       docID,
		Content:  content,
		Vector:   vector,
		Metadata: metadata,
	}

	vs.documents = append(vs.documents, doc)
	return nil
}

// SearchResult represents a search result with relevance score
type SearchResult struct {
	Document Document
	Score    float32
}

// Search performs semantic search using cosine similarity
func (vs *VectorStore) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if topK <= 0 {
		topK = 5
	}

	vs.mu.RLock()
	docCount := len(vs.documents)
	vs.mu.RUnlock()

	if docCount == 0 {
		return []SearchResult{}, nil
	}

	// Generate query embedding
	queryEmbeddings, err := vs.embeddingModel.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	if len(queryEmbeddings) == 0 || len(queryEmbeddings[0]) == 0 {
		return nil, fmt.Errorf("empty query embedding returned")
	}

	// Convert float64 to float32
	queryVector := make([]float32, len(queryEmbeddings[0]))
	for i, v := range queryEmbeddings[0] {
		queryVector[i] = float32(v)
	}

	vs.mu.RLock()
	defer vs.mu.RUnlock()

	// Calculate cosine similarity for all documents
	results := make([]SearchResult, 0, len(vs.documents))
	for _, doc := range vs.documents {
		score := cosineSimilarity(queryVector, doc.Vector)
		results = append(results, SearchResult{
			Document: doc,
			Score:    score,
		})
	}

	// Sort by score (descending)
	sortResults(results)

	// Return top K
	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK], nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt32(normA) * sqrt32(normB))
}

// sqrt32 is a float32 square root implementation
func sqrt32(x float32) float32 {
	return float32(sqrtFloat64(float64(x)))
}

// sqrtFloat64 wraps math.Sqrt for convenience
func sqrtFloat64(x float64) float64 {
	// Simple implementation using Go's built-in
	// Import would be: "math"
	// Using a simple approximation here to avoid import issues in template
	z := x
	if z == 0 {
		return 0
	}
	for i := 0; i < 20; i++ {
		z = 0.5 * (z + x/z)
	}
	return z
}

// sortResults sorts search results by score in descending order
func sortResults(results []SearchResult) {
	// Simple bubble sort (for small datasets)
	n := len(results)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if results[j].Score < results[j+1].Score {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
}

// GetDocumentCount returns the number of documents in the store
func (vs *VectorStore) GetDocumentCount() int {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return len(vs.documents)
}

// Clear removes all documents from the store
func (vs *VectorStore) Clear() error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	vs.documents = make([]Document, 0)
	vs.updatedAt = time.Now()

	// Also delete the file
	if err := os.Remove(vs.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete store file: %w", err)
	}

	return nil
}

// ListDocuments returns all documents with their metadata
func (vs *VectorStore) ListDocuments() []Document {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	docs := make([]Document, len(vs.documents))
	copy(docs, vs.documents)
	return docs
}
