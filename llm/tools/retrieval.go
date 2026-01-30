package tools

//
//import (
//	"context"
//	"encoding/json"
//	"fmt"
//	"os"
//	"strconv"
//
//	"github.com/cloudwego/eino-ext/components/retriever/redis"
//	"github.com/cloudwego/eino/components/tool"
//	"github.com/cloudwego/eino/components/tool/utils"
//	"github.com/cloudwego/eino/schema"
//	redisCli "github.com/redis/go-redis/v9"
//)
//
//var (
//	RetrievalToolName = "retrieval"
//)
//
//const (
//	RedisPrefix   = "eino:doc:"
//	IndexName     = "knowledge_index"
//	ContentField  = "content"
//	MetadataField = "metadata"
//	VectorField   = "vector_field"
//	DistanceField = "dist"
//)
//
//type RetrieveRequest struct {
//	Query string `json:"query" jsonschema:"description=The query string to search for in the knowledge base"`
//}
//
//type RetrieveResponse struct {
//	Documents []string `json:"documents"`
//}
//
//// GetRetrieveTool creates a tool for retrieving knowledge from Redis.
//func GetRetrieveTool(ctx context.Context) (tool.BaseTool, error) {
//	// 1. Create Embedding Model
//	emb, err := provider.CreateEmbeddingModel(ctx)
//	if err != nil {
//		return nil, fmt.Errorf("failed to create embedding model: %w", err)
//	}
//
//	// 2. Setup Redis Client
//	redisAddr := os.Getenv("REDIS_ADDR")
//	if redisAddr == "" {
//		redisAddr = "localhost:6379"
//	}
//	redisPassword := os.Getenv("REDIS_PASSWORD") // Optional, empty for no auth
//	redisClient := redisCli.NewClient(&redisCli.Options{
//		Addr:     redisAddr,
//		Password: redisPassword,
//		Protocol: 2, // Force Protocol 2 to avoid RESP3 issues
//	})
//
//	// Verify Redis connection
//	if err := redisClient.Ping(ctx).Err(); err != nil {
//		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", redisAddr, err)
//	}
//
//	// 3. Create Retriever
//	retriever, err := redis.NewRetriever(ctx, &redis.RetrieverConfig{
//		Client:       redisClient,
//		Index:        IndexName,
//		Dialect:      2,
//		ReturnFields: []string{ContentField, MetadataField, DistanceField},
//		TopK:         3,
//		VectorField:  VectorField,
//		Embedding:    emb,
//		DocumentConverter: func(ctx context.Context, doc redisCli.Document) (*schema.Document, error) {
//			resp := &schema.Document{
//				ID:       doc.ID,
//				Content:  "",
//				MetaData: map[string]any{},
//			}
//			for field, val := range doc.Fields {
//				if field == ContentField {
//					resp.Content = val
//				} else if field == MetadataField {
//					// Deserialize JSON metadata
//					var metadata map[string]any
//					if err := json.Unmarshal([]byte(val), &metadata); err == nil {
//						for k, v := range metadata {
//							resp.MetaData[k] = v
//						}
//					} else {
//						// Fallback to raw value if unmarshal fails
//						resp.MetaData[field] = val
//					}
//				} else if field == DistanceField {
//					if dist, err := strconv.ParseFloat(val, 64); err == nil {
//						resp.WithScore(1 - dist)
//					}
//				}
//			}
//			return resp, nil
//		},
//	})
//	if err != nil {
//		return nil, fmt.Errorf("failed to create redis retriever: %w", err)
//	}
//
//	// 4. Create Tool
//	t, err := utils.InferTool("retrieve_knowledge", "Retrieve technical documentation, implementation guides, and project knowledge. Use this whenever the user asks about system concepts, configuration, or 'how to'.", func(ctx context.Context, req *RetrieveRequest) (*RetrieveResponse, error) {
//		docs, err := retriever.Retrieve(ctx, req.Query)
//		if err != nil {
//			return nil, fmt.Errorf("retrieval failed: %w", err)
//		}
//
//		result := make([]string, 0, len(docs))
//		for _, doc := range docs {
//			result = append(result, doc.Content)
//		}
//		return &RetrieveResponse{Documents: result}, nil
//	})
//	if err != nil {
//		return nil, fmt.Errorf("failed to infer tool: %w", err)
//	}
//	return t, nil
//}
