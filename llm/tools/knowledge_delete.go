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
	// DeleteDocumentToolName is the name of the document deletion tool
	DeleteDocumentToolName = "delete_document"
)

// deleteKnowledgeDocumentDescription is the detailed tool description
const deleteKnowledgeDocumentDescription = `Delete documents from the knowledge base.

USE CASES:
- Remove outdated documents
- Clean up incorrectly ingested files
- Manage knowledge base contents

PARAMETERS (one required):
- source: Delete all documents from this source file path
- id: Delete a specific document by its ID

WARNING:
- This operation cannot be undone
- Deleting by source removes all chunks from that file
- Use list_documents to find document IDs before deletion

EXAMPLES:
- Delete by source: {"source": "./docs/old.md"}
- Delete by ID: {"id": "doc_abc123..."}

SAFETY:
- Always confirm with the user before deleting
- Check what will be deleted using list_documents first`

// DeleteDocumentParams defines parameters for document deletion
type DeleteDocumentParams struct {
	Source string `json:"source,omitempty" jsonschema:"description=Source file path to delete all documents from"`
	ID     string `json:"id,omitempty" jsonschema:"description=Specific document ID to delete"`
}

// DeleteDocumentFunc deletes documents from the knowledge base
func DeleteDocumentFunc(ctx context.Context, params DeleteDocumentParams) (string, error) {
	if globalKnowledgeVectorStore == nil {
		return Error("vector store is not initialized")
	}

	// Validate that at least one parameter is provided
	if params.Source == "" && params.ID == "" {
		return Error("either 'source' or 'id' parameter is required")
	}

	var deletedCount int
	var err error

	if params.ID != "" {
		// Delete specific document by ID
		err = globalKnowledgeVectorStore.Delete(ctx, params.ID)
		if err != nil {
			return Error(fmt.Sprintf("failed to delete document: %v", err))
		}
		deletedCount = 1
	} else {
		// Delete all documents from source
		source := strings.TrimSpace(params.Source)

		// Check what we're about to delete
		filter := llm.ListFilter{
			Source: source,
			Limit:  1000,
		}
		docs, checkErr := globalKnowledgeVectorStore.List(ctx, filter)
		if checkErr == nil && len(docs) > 0 {
			// Get the title before deleting
			title := docs[0].Title
			fileType := docs[0].FileType

			err = globalKnowledgeVectorStore.DeleteBySource(ctx, source)
			if err != nil {
				return Error(fmt.Sprintf("failed to delete documents: %v", err))
			}
			deletedCount = len(docs)

			// Get updated count
			totalCount, _ := globalKnowledgeVectorStore.Count(ctx)

			return Success(fmt.Sprintf("Deleted document:\n"+
				"  Title: %s\n"+
				"  Source: %s\n"+
				"  Type: %s\n"+
				"  Chunks removed: %d\n"+
				"  Remaining documents: %d",
				title, source, fileType, deletedCount, totalCount),
				&Metadata{
					FilePath:   source,
					MatchCount: deletedCount,
				}, TierCompact)
		}

		return Success(fmt.Sprintf("No documents found for source: %s", source),
			nil, TierCompact)
	}

	// Get updated count
	totalCount, _ := globalKnowledgeVectorStore.Count(ctx)

	return Success(fmt.Sprintf("Deleted %d document(s). Remaining: %d",
		deletedCount, totalCount),
		&Metadata{
			MatchCount: deletedCount,
		}, TierCompact)
}

// GetDeleteDocumentTool returns the document deletion tool
func GetDeleteDocumentTool() tool.InvokableTool {
	t, err := utils.InferTool(
		DeleteDocumentToolName,
		deleteKnowledgeDocumentDescription,
		DeleteDocumentFunc,
	)
	if err != nil {
		return nil
	}
	return t
}
