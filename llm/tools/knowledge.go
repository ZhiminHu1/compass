package tools

import (
	"context"
	"cowork-agent/temp/example4/vectorstore"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

const (
	// KnowledgeToolName is the name of the knowledge base tool
	KnowledgeToolName = "search_knowledge"

	// DefaultTopK is the default number of results to return
	DefaultTopK = 5
	// MaxTopK is the maximum allowed results
	MaxTopK = 10
)

var (
	globalVectorStore *vectorstore.VectorStore
)

// InitKnowledgeTool initializes the knowledge base with a vector store
func InitKnowledgeTool(vs *vectorstore.VectorStore) {
	globalVectorStore = vs
}

// KnowledgeToolParams defines parameters for knowledge base search
type KnowledgeToolParams struct {
	Query string `json:"query" jsonschema:"description=The query to search for in the knowledge base"`
	TopK  int    `json:"top_k,omitempty" jsonschema:"description=Number of results to return (default: 5, max: 10)"`
}

// knowledgeDescription is the detailed tool description for the AI
const knowledgeDescription = `Search the local knowledge base for relevant information.

BEFORE USING:
- Try this tool first for historical research content
- Fall back to web_search for real-time information
- Use specific, focused queries

CAPABILITIES:
- Semantic search across stored documents
- Returns ranked results with relevance scores
- Best for research notes, documentation, and cached content

PARAMETERS:
- query (required): The question or topic to search for
- top_k (optional): Number of results (default: 5, max: 10)

OUTPUT FORMAT:
Returns ranked results with relevance scores and content.

EXAMPLES:
- Search topic: {"query": "Go design patterns"}
- Find concept: {"query": "singleton pattern implementation"}
- Quick lookup: {"query": "goroutine best practices"}`

// KnowledgeToolFunc searches the knowledge base for relevant information
func KnowledgeToolFunc(ctx context.Context, params KnowledgeToolParams) (string, error) {
	if globalVectorStore == nil {
		return Error("knowledge base is not initialized")
	}

	if params.Query == "" {
		return Error("query parameter is required")
	}

	topK := params.TopK
	if topK <= 0 {
		topK = DefaultTopK
	}
	if topK > MaxTopK {
		topK = MaxTopK
	}

	// Search the knowledge base
	results, err := globalVectorStore.Search(ctx, params.Query, topK)
	if err != nil {
		return Error(fmt.Sprintf("knowledge base search failed: %v", err))
	}

	if len(results) == 0 {
		return Success("No relevant content found in the knowledge base. Try using web_search for current information.",
			&Metadata{MatchCount: 0})
	}

	// Format results
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d relevant results in knowledge base:\n\n", len(results)))

	for i, result := range results {
		sb.WriteString(fmt.Sprintf("--- Result %d (score: %.2f) ---\n", i+1, result.Score))
		sb.WriteString(result.Document.Content)
		sb.WriteString("\n")

		// Add metadata if available
		if result.Document.Metadata != nil {
			var meta []string
			if source, ok := result.Document.Metadata["source"].(string); ok && source != "" {
				meta = append(meta, fmt.Sprintf("source: %s", source))
			}
			if timestamp, ok := result.Document.Metadata["timestamp"].(string); ok && timestamp != "" {
				meta = append(meta, fmt.Sprintf("time: %s", timestamp))
			}
			if len(meta) > 0 {
				sb.WriteString(fmt.Sprintf("[metadata: %s]", strings.Join(meta, ", ")))
			}
		}
		sb.WriteString("\n")
	}

	return Success(sb.String(), &Metadata{
		MatchCount: len(results),
	})
}

// GetKnowledgeTool returns the knowledge base search tool with enhanced description
func GetKnowledgeTool() tool.InvokableTool {
	t, err := utils.InferTool(
		KnowledgeToolName,
		knowledgeDescription,
		KnowledgeToolFunc,
	)
	if err != nil {
		log.Fatalf("failed to create knowledge tool: %v", err)
	}
	return t
}
