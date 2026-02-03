package vector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"cowork-agent/llm"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/redis/go-redis/v9"
)

const (
	// Default index configuration
	defaultEFConstruction = 200
	defaultM              = 16

	// Field names in Redis hash
	fieldContent    = "content"
	fieldVector     = "vector"
	fieldSource     = "source"
	fieldFileType   = "file_type"
	fieldTitle      = "title"
	fieldChunkIndex = "chunk_index"
	fieldCreatedAt  = "created_at"
	fieldMetadata   = "metadata"
)

// RedisStore implements VectorStore using Redis with RediSearch vector search
type RedisStore struct {
	client         *redis.Client
	embeddingSvc   *EmbeddingService
	config         StoreConfig
	indexCreated   bool
	mu             sync.RWMutex
	efConstruction int
	m              int
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Addr           string
	Password       string
	DB             int
	PoolSize       int
	IndexName      string
	VectorDim      int
	EFConstruction int
	M              int
}

// DefaultRedisConfig returns default Redis configuration from environment
func DefaultRedisConfig() RedisConfig {
	efConstruction := getEnvInt("HNSW_EF_CONSTRUCTION", defaultEFConstruction)
	m := getEnvInt("HNSW_M", defaultM)

	return RedisConfig{
		Addr:           getEnvString("REDIS_ADDR", "localhost:6379"),
		Password:       getEnvString("REDIS_PASSWORD", ""),
		DB:             getEnvInt("REDIS_DB", 0),
		PoolSize:       getEnvInt("REDIS_POOL_SIZE", 10),
		IndexName:      getEnvString("VECTOR_INDEX_NAME", "cowork-knowledge"),
		VectorDim:      GetEmbeddingDimFromEnv(),
		EFConstruction: efConstruction,
		M:              m,
	}
}

// getEnvString reads a string from environment variable
func getEnvString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// NewRedisStore creates a new Redis-based vector store
func NewRedisStore(ctx context.Context, embedder embedding.Embedder, cfg RedisConfig) (*RedisStore, error) {
	if embedder == nil {
		return nil, fmt.Errorf("embedding model is required")
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	store := &RedisStore{
		client:       client,
		embeddingSvc: NewEmbeddingService(embedder, cfg.VectorDim),
		config: StoreConfig{
			EmbeddingDim: cfg.VectorDim,
			IndexName:    cfg.IndexName,
			KeyPrefix:    "vec:",
		},
		efConstruction: cfg.EFConstruction,
		m:              cfg.M,
	}

	// Create the vector index
	if err := store.ensureIndex(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create vector index: %w", err)
	}

	return store, nil
}

// ensureIndex creates the HNSW vector index if it doesn't exist
func (s *RedisStore) ensureIndex(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if index already exists
	indexName := s.config.IndexName
	_, err := s.client.Do(ctx, "FT.INFO", indexName).Result()
	if err == nil {
		// Index exists
		s.indexCreated = true
		return nil
	}

	// Create index with HNSW algorithm
	dim := s.config.EmbeddingDim
	ef := s.efConstruction
	m := s.m

	// FT.CREATE cowork-knowledge
	//   ON HASH PREFIX 1 "vec:"
	//   SCHEMA vector VECTOR HNSW 6 TYPE FLOAT32 DIM 1024 DISTANCE_METRIC COSINE EF_CONSTRUCTION 200 M 16
	//          content TEXT
	//          source TAG
	//          file_type TAG
	//          title TEXT
	//          chunk_index NUMERIC
	//          created_at NUMERIC

	_, err = s.client.Do(ctx, "FT.CREATE", indexName,
		"ON", "HASH",
		"PREFIX", "1", s.config.KeyPrefix,
		"SCHEMA",
		fieldVector, "VECTOR", "HNSW", "6",
		"TYPE", "FLOAT32",
		"DIM", strconv.Itoa(dim),
		"DISTANCE_METRIC", "COSINE",
		"EF_CONSTRUCTION", strconv.Itoa(ef),
		"M", strconv.Itoa(m),
		fieldContent, "TEXT",
		fieldSource, "TAG",
		fieldFileType, "TAG",
		fieldTitle, "TEXT",
		fieldChunkIndex, "NUMERIC",
		fieldCreatedAt, "NUMERIC",
	).Result()

	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	s.indexCreated = true
	return nil
}

// generateID generates a unique document ID
func (s *RedisStore) generateID(source string, chunkIndex int) string {
	h := sha256.New()
	h.Write([]byte(source))
	h.Write([]byte(fmt.Sprintf("%d", chunkIndex)))
	h.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// Add adds a single document to the store
func (s *RedisStore) Add(ctx context.Context, doc llm.Document) error {
	return s.AddBatch(ctx, []llm.Document{doc})
}

// AddBatch adds multiple documents in a single operation
func (s *RedisStore) AddBatch(ctx context.Context, docs []llm.Document) error {
	if len(docs) == 0 {
		return nil
	}

	// Generate embeddings for all documents
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}

	vectors, err := s.embeddingSvc.EmbedBatch(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Use pipeline for batch insert
	pipe := s.client.Pipeline()

	now := time.Now().Unix()
	for i, doc := range docs {
		if doc.ID == "" {
			doc.ID = s.generateID(doc.Source, doc.ChunkIndex)
		}
		if doc.CreatedAt == "" {
			doc.CreatedAt = time.Now().Format(time.RFC3339)
		}

		key := s.config.KeyPrefix + doc.ID

		// Encode vector as bytes for storage
		vectorBytes, err := encodeVector(vectors[i])
		if err != nil {
			return fmt.Errorf("failed to encode vector: %w", err)
		}

		// Encode metadata
		metadataJSON, _ := json.Marshal(doc.Metadata)

		// Set all fields in hash
		pipe.HSet(ctx, key,
			fieldContent, doc.Content,
			fieldVector, vectorBytes,
			fieldSource, escapeTagValue(doc.Source),
			fieldFileType, doc.FileType,
			fieldTitle, doc.Title,
			fieldChunkIndex, doc.ChunkIndex,
			fieldCreatedAt, now,
			fieldMetadata, metadataJSON,
		)
	}

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to insert documents: %w", err)
	}

	return nil
}

// encodeVector encodes a float32 vector as bytes for Redis storage
func encodeVector(vector []float32) ([]byte, error) {
	// Use JSON encoding for simplicity
	// For production, consider using binary encoding for efficiency
	return json.Marshal(vector)
}

// decodeVector decodes a float32 vector from Redis storage
func decodeVector(data []byte) ([]float32, error) {
	var vector []float32
	if err := json.Unmarshal(data, &vector); err != nil {
		return nil, err
	}
	return vector, nil
}

// escapeTagValue escapes special characters in TAG field values
func escapeTagValue(value string) string {
	// Redis TAG fields use comma as separator, escape commas
	// Also escape spaces if needed
	return escapeTag(value)
}

// escapeTag escapes special TAG characters
func escapeTag(s string) string {
	// Replace commas with escaped version
	// Replace spaces with underscores
	result := s
	for i, c := range result {
		if c == ',' {
			result = result[:i] + "\\," + result[i+1:]
		}
	}
	return result
}

// Search performs semantic search using vector similarity
func (s *RedisStore) Search(ctx context.Context, query string, topK int) ([]llm.SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if topK <= 0 {
		topK = 5
	}
	if topK > 100 {
		topK = 100 // Reasonable limit
	}

	// Generate query embedding
	queryVector, err := s.embeddingSvc.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	queryBytes, err := encodeVector(queryVector)
	if err != nil {
		return nil, fmt.Errorf("failed to encode query vector: %w", err)
	}

	// Execute vector search query
	// FT.SEARCH cowork-knowledge "*=>[KNN 5 @vector $query_vector AS score]"
	//   PARAMS 2 query_vector "<bytes>"
	//   RETURN 3 content source title
	//   SORT BY score
	//   LIMIT 0 5

	indexName := s.config.IndexName

	// Build the search query with KNN
	queryStr := fmt.Sprintf("*=>[KNN %d @vector $query_vector AS score]", topK)

	result, err := s.client.Do(ctx, "FT.SEARCH", indexName, queryStr,
		"PARAMS", "2", "query_vector", queryBytes,
		"RETURN", "6", fieldContent, fieldSource, fieldFileType, fieldTitle, fieldChunkIndex, fieldMetadata,
		"SORTBY", "score",
		"LIMIT", "0", strconv.Itoa(topK),
		"NOCONTENT",
	).Result()

	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Parse results
	results, err := s.parseSearchResults(ctx, result, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	return results, nil
}

// parseSearchResults parses Redis search results
func (s *RedisStore) parseSearchResults(ctx context.Context, result interface{}, topK int) ([]llm.SearchResult, error) {
	// Result format from FT.SEARCH is a list
	// First element is count, followed by pairs of (id, fields)
	values, ok := result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result format")
	}

	if len(values) == 0 {
		return []llm.SearchResult{}, nil
	}

	// First element is the count
	// _, ok = values[0].(int64)
	// if !ok {
	// 	return nil, fmt.Errorf("expected count as first element")
	// }

	var results []llm.SearchResult

	// Process results in pairs (id, fields)
	for i := 1; i < len(values); i += 2 {
		if i+1 >= len(values) {
			break
		}

		docID, ok := values[i].(string)
		if !ok {
			continue
		}

		fields, ok := values[i+1].([]interface{})
		if !ok {
			continue
		}

		doc, err := s.parseDocumentFields(docID, fields)
		if err != nil {
			continue
		}

		// Extract score from the search result - Redis FT.SEARCH with KNN
		// includes the score in a special way
		// For simplicity, we'll use the order as relevance indicator

		results = append(results, llm.SearchResult{
			Document: doc,
			Score:    1.0 - float32(len(results))/float32(topK+1), // Simple decay based on position
		})
	}

	return results, nil
}

// parseDocumentFields parses document fields from Redis result
func (s *RedisStore) parseDocumentFields(id string, fields []interface{}) (llm.Document, error) {
	doc := llm.Document{
		ID:       id,
		Metadata: make(map[string]interface{}),
	}

	for i := 0; i < len(fields); i += 2 {
		if i+1 >= len(fields) {
			break
		}

		fieldName, ok := fields[i].(string)
		if !ok {
			continue
		}

		fieldValue := fields[i+1]

		switch fieldName {
		case fieldContent:
			if val, ok := fieldValue.(string); ok {
				doc.Content = val
			}
		case fieldSource:
			if val, ok := fieldValue.(string); ok {
				doc.Source = val
			}
		case fieldFileType:
			if val, ok := fieldValue.(string); ok {
				doc.FileType = val
			}
		case fieldTitle:
			if val, ok := fieldValue.(string); ok {
				doc.Title = val
			}
		case fieldChunkIndex:
			if val, ok := fieldValue.(int64); ok {
				doc.ChunkIndex = int(val)
			}
		case fieldMetadata:
			if val, ok := fieldValue.(string); ok {
				json.Unmarshal([]byte(val), &doc.Metadata)
			}
		}
	}

	return doc, nil
}

// Delete removes a document by its ID
func (s *RedisStore) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("document ID cannot be empty")
	}

	key := s.config.KeyPrefix + id
	return s.client.Del(ctx, key).Err()
}

// DeleteBySource removes all documents from a specific source file
func (s *RedisStore) DeleteBySource(ctx context.Context, source string) error {
	if source == "" {
		return fmt.Errorf("source cannot be empty")
	}

	// First, find all documents with this source
	indexName := s.config.IndexName
	escapedSource := escapeTagValue(source)

	// Use FT.SEARCH to find documents by source tag
	result, err := s.client.Do(ctx, "FT.SEARCH", indexName,
		fmt.Sprintf("@source:{%s}", escapedSource),
		"NOCONTENT",
		"LIMIT", "0", "1000",
	).Result()

	if err != nil {
		// If index doesn't exist or no results, return success
		return nil
	}

	// Extract document IDs
	values, ok := result.([]interface{})
	if !ok || len(values) < 2 {
		return nil
	}

	var keys []string
	for i := 1; i < len(values); i += 2 {
		if docID, ok := values[i].(string); ok {
			keys = append(keys, s.config.KeyPrefix+docID)
		}
	}

	// Delete all found documents
	if len(keys) > 0 {
		return s.client.Del(ctx, keys...).Err()
	}

	return nil
}

// List returns documents matching the filter criteria
func (s *RedisStore) List(ctx context.Context, filter llm.ListFilter) ([]llm.Document, error) {
	indexName := s.config.IndexName

	// Build query
	var queryParts []string
	if filter.Source != "" {
		escapedSource := escapeTagValue(filter.Source)
		queryParts = append(queryParts, fmt.Sprintf("@source:{%s}", escapedSource))
	}
	if filter.FileType != "" {
		queryParts = append(queryParts, fmt.Sprintf("@file_type:{%s}", filter.FileType))
	}

	query := "*"
	if len(queryParts) > 0 {
		query = strings.Join(queryParts, " ")
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	// Execute search
	result, err := s.client.Do(ctx, "FT.SEARCH", indexName, query,
		"RETURN", "7", fieldContent, fieldSource, fieldFileType, fieldTitle, fieldChunkIndex, fieldCreatedAt, fieldMetadata,
		"LIMIT", strconv.Itoa(offset), strconv.Itoa(limit),
	).Result()

	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Parse results
	docs, err := s.parseListResults(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse results: %w", err)
	}

	return docs, nil
}

// parseListResults parses list results
func (s *RedisStore) parseListResults(result interface{}) ([]llm.Document, error) {
	values, ok := result.([]interface{})
	if !ok || len(values) < 2 {
		return []llm.Document{}, nil
	}

	var docs []llm.Document

	for i := 1; i < len(values); i += 2 {
		if i+1 >= len(values) {
			break
		}

		docID, ok := values[i].(string)
		if !ok {
			continue
		}

		fields, ok := values[i+1].([]interface{})
		if !ok {
			continue
		}

		doc, err := s.parseDocumentFields(docID, fields)
		if err != nil {
			continue
		}

		docs = append(docs, doc)
	}

	return docs, nil
}

// Count returns the total number of documents in the store
func (s *RedisStore) Count(ctx context.Context) (int64, error) {
	indexName := s.config.IndexName

	// Get index info
	info, err := s.client.Do(ctx, "FT.INFO", indexName).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get index info: %w", err)
	}

	// Parse document count from info
	values, ok := info.([]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpected info format")
	}

	// Look for num_docs field
	for i := 0; i < len(values)-1; i += 2 {
		if key, ok := values[i].(string); ok && key == "num_docs" {
			if count, ok := values[i+1].(int64); ok {
				return count, nil
			}
		}
	}

	return 0, nil
}

// Close closes the Redis connection
func (s *RedisStore) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
