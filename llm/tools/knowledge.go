package tools

import (
	"context"
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
	if globalKnowledgeVectorStore == nil {
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
	results, err := globalKnowledgeVectorStore.Search(ctx, params.Query, topK)
	if err != nil {
		return Error(fmt.Sprintf("knowledge base search failed: %v", err))
	}

	if len(results) == 0 {
		return Success("No relevant content found in the knowledge base. Try using web_search for current information.",
			&Metadata{MatchCount: 0}, TierCompact)
	}

	// Format results
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d relevant results in knowledge base:\n\n", len(results)))

	for i, result := range results {
		sb.WriteString(fmt.Sprintf("--- Result %d (score: %.2f) ---\n", i+1, result.Score))
		sb.WriteString(result.Document.Content)
		sb.WriteString("\n")

		// Add metadata if available
		if result.Document.Source != "" {
			sb.WriteString(fmt.Sprintf("[source: %s]", result.Document.Source))
		}
		if result.Document.Title != "" {
			sb.WriteString(fmt.Sprintf(" [title: %s]", result.Document.Title))
		}
		sb.WriteString("\n")
	}

	return Success(sb.String(), &Metadata{
		MatchCount: len(results),
	}, TierCompact)
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
