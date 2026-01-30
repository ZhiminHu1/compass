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

var (
	globalVectorStore *vectorstore.VectorStore
)

// InitKnowledgeTool initializes the knowledge base with a vector store
func InitKnowledgeTool(vs *vectorstore.VectorStore) {
	globalVectorStore = vs
}

// KnowledgeToolParams defines parameters for knowledge base search
type KnowledgeToolParams struct {
	Query string `json:"query" jsonschema:"description=要查询的内容"`
	TopK  int    `json:"top_k,omitempty" jsonschema:"description=返回结果数量，默认5，范围1-10"`
}

// KnowledgeToolFunc searches the knowledge base for relevant information
func KnowledgeToolFunc(ctx context.Context, params KnowledgeToolParams) (string, error) {
	if globalVectorStore == nil {
		return "知识库未初始化。", nil
	}

	if params.Query == "" {
		return "", fmt.Errorf("query 参数不能为空")
	}

	topK := params.TopK
	if topK <= 0 {
		topK = 5
	}
	if topK > 10 {
		topK = 10
	}

	// Search the knowledge base
	results, err := globalVectorStore.Search(ctx, params.Query, topK)
	if err != nil {
		return "", fmt.Errorf("知识库搜索失败: %w", err)
	}

	if len(results) == 0 {
		return "知识库中未找到相关内容。建议使用 web_search 工具搜索最新信息。", nil
	}

	// Format results
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("从知识库找到 %d 条相关结果:\n\n", len(results)))

	for i, result := range results {
		sb.WriteString(fmt.Sprintf("--- 结果 %d (相关度: %.2f) ---\n", i+1, result.Score))
		sb.WriteString(result.Document.Content)
		sb.WriteString("\n\n")

		// Add metadata info if available
		if result.Document.Metadata != nil {
			if source, ok := result.Document.Metadata["source"].(string); ok && source != "" {
				sb.WriteString(fmt.Sprintf("来源: %s", source))
			}
			if timestamp, ok := result.Document.Metadata["timestamp"].(string); ok && timestamp != "" {
				sb.WriteString(fmt.Sprintf(" | 时间: %s", timestamp))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

// GetKnowledgeTool returns the knowledge base search tool
func GetKnowledgeTool() tool.InvokableTool {
	t, err := utils.InferTool(
		"search_knowledge",
		"在本地知识库中搜索相关信息。优先使用此工具查询历史研究内容。如果知识库中没有相关内容，再使用 web_search。",
		KnowledgeToolFunc,
	)
	if err != nil {
		log.Fatalf("failed to create knowledge tool: %v", err)
	}
	return t
}
