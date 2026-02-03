package tools

import (
	"context"
	"cowork-agent/llm"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

const (
	// ListDocumentsToolName is the name of the document listing tool
	ListDocumentsToolName = "list_documents"
)

// listKnowledgeDocumentsDescription is the detailed tool description for the AI
const listKnowledgeDocumentsDescription = `List documents in the knowledge base.

USE CASES:
- See what documents are stored in the knowledge base
- Filter documents by type or source
- Check knowledge base contents before searching

PARAMETERS:
- file_type (optional): Filter by file type (pdf, docx, md, txt, html)
- source (optional): Filter by source file path
- limit (optional): Maximum results to return (default: 100)

OUTPUT FORMAT:
Returns a list of documents with their metadata:
- Document ID
- Title
- Source file path
- File type
- Chunk index
- Creation time

EXAMPLES:
- List all: {}
- List markdown: {"file_type": "md"}
- List from source: {"source": "./docs/api.md"}
- Limited results: {"limit": 10}`

// ListDocumentsParams defines parameters for listing documents
type ListDocumentsParams struct {
	FileType string `json:"file_type,omitempty" jsonschema:"description=Filter by file type (pdf, docx, md, txt, html)"`
	Source   string `json:"source,omitempty" jsonschema:"description=Filter by source file path"`
	Limit    int    `json:"limit,omitempty" jsonschema:"description=Maximum number of documents to return (default: 100)"`
}

// ListDocumentsFunc lists documents in the knowledge base
func ListDocumentsFunc(ctx context.Context, params ListDocumentsParams) (string, error) {
	if globalKnowledgeVectorStore == nil {
		return Error("vector store is not initialized")
	}

	// Build filter
	filter := llm.ListFilter{
		Source:   params.Source,
		FileType: params.FileType,
		Limit:    params.Limit,
	}
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}

	// List documents
	docs, err := globalKnowledgeVectorStore.List(ctx, filter)
	if err != nil {
		return Error(fmt.Sprintf("failed to list documents: %v", err))
	}

	if len(docs) == 0 {
		// Check if knowledge base is empty
		count, _ := globalKnowledgeVectorStore.Count(ctx)
		if count == 0 {
			return Success("Knowledge base is empty. Use ingest_document to add documents.",
				nil, TierCompact)
		}
		return Success(fmt.Sprintf("No documents match the specified filters.\nTotal documents in knowledge base: %d",
			count), nil, TierCompact)
	}

	// Group documents by source
	grouped := make(map[string][]llm.Document)
	for _, doc := range docs {
		grouped[doc.Source] = append(grouped[doc.Source], doc)
	}

	// Format results
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d documents from %d source(s):\n\n", len(docs), len(grouped)))

	for source, sourceDocs := range grouped {
		sb.WriteString(fmt.Sprintf("📄 %s\n", source))
		sb.WriteString(fmt.Sprintf("   Title: %s\n", sourceDocs[0].Title))
		sb.WriteString(fmt.Sprintf("   Type: %s\n", sourceDocs[0].FileType))
		sb.WriteString(fmt.Sprintf("   Chunks: %d\n", len(sourceDocs)))

		// Show first chunk preview
		if len(sourceDocs[0].Content) > 0 {
			preview := sourceDocs[0].Content
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			sb.WriteString(fmt.Sprintf("   Preview: %s\n", preview))
		}
		sb.WriteString("\n")
	}

	return Success(sb.String(), &Metadata{
		FileCount:  len(grouped),
		MatchCount: len(docs),
	}, TierCompact)
}

// GetListDocumentsTool returns the document listing tool
func GetListDocumentsTool() tool.InvokableTool {
	t, err := utils.InferTool(
		ListDocumentsToolName,
		listKnowledgeDocumentsDescription,
		ListDocumentsFunc,
	)
	if err != nil {
		return nil
	}
	return t
}
