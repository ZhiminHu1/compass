package tools

import (
	"compass/llm"
	"compass/llm/parser"
	"compass/llm/vector"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

const (
	// IngestDocumentToolName is the name of the document ingestion tool
	IngestDocumentToolName = "ingest_document"
)

var (
	// globalVectorStore holds the vector store instance for knowledge tools
	globalKnowledgeVectorStore vector.VectorStore
	// globalParser holds the parser registry for document ingestion
	globalKnowledgeParser *parser.Registry
	// globalEmbedder holds the embedding model
	globalKnowledgeEmbedder embedding.Embedder
)

// InitKnowledgeVectorStore initializes the knowledge tools with vector store
func InitKnowledgeVectorStore(vs vector.VectorStore, p *parser.Registry, emb embedding.Embedder) {
	globalKnowledgeVectorStore = vs
	globalKnowledgeParser = p
	globalKnowledgeEmbedder = emb
}

// IngestDocumentParams defines parameters for document ingestion
type IngestDocumentParams struct {
	FilePath string `json:"file_path" jsonschema:"description=Path to the file to ingest into the knowledge base"`
	Title    string `json:"title,omitempty" jsonschema:"description=Optional title for the document (defaults to filename)"`
}

// ingestDescription is the detailed tool description for the AI
const ingestDescription = `Ingest a document file into the knowledge base for semantic search.

SUPPORTED FORMATS:
- Text files (.txt)
- Markdown files (.md, .markdown)
- HTML files (.html, .htm)

USE CASES:
- Add reference documents for later retrieval
- Store documentation for context-aware answers
- Build a knowledge base from local files

PARAMETERS:
- file_path (required): Path to the file to ingest
- title (optional): Custom title for the document

PROCESS:
1. File content is parsed according to its type
2. Content is split into chunks for better retrieval
3. Each chunk is converted to a vector embedding
4. Chunks are stored in the vector database

EXAMPLES:
- Ingest markdown: {"file_path": "./docs/api.md"}
- Ingest with title: {"file_path": "./reference.txt", "title": "API Reference"}

NOTES:
- Large files are automatically chunked for optimal retrieval
- Existing documents with the same source path are replaced
- Use list_documents to see what's in the knowledge base`

// IngestDocumentFunc ingests a document into the knowledge base
func IngestDocumentFunc(ctx context.Context, params IngestDocumentParams) (string, error) {
	if globalKnowledgeParser == nil {
		return Error("document parser is not initialized")
	}
	if globalKnowledgeVectorStore == nil {
		return Error("vector store is not initialized")
	}

	filePath := strings.TrimSpace(params.FilePath)
	if filePath == "" {
		return Error("file_path parameter is required")
	}

	// Clean the path
	filePath = filepath.Clean(filePath)

	// Parse the file
	parsedDoc, err := globalKnowledgeParser.ParseFile(ctx, filePath)
	if err != nil {
		return Error(fmt.Sprintf("failed to parse file: %v", err))
	}

	// Use custom title if provided, otherwise use extracted title
	title := params.Title
	if title == "" {
		title = parsedDoc.Title
	}

	// Get file type from extension
	ext := strings.TrimPrefix(filepath.Ext(filePath), ".")
	fileType := parser.FileTypeFromExt(ext).String()

	// Chunk the document
	chunkConfig := vector.DefaultChunkConfig()
	chunks := vector.ChunkDocument(parsedDoc.Content, chunkConfig)

	if len(chunks) == 0 {
		return Error("document content is too short to process")
	}

	// Create documents with embeddings
	docs := make([]llm.Document, len(chunks))
	now := time.Now().Format(time.RFC3339)

	for i, chunk := range chunks {
		// Generate document ID
		docID := fmt.Sprintf("doc_%s_%d", filepath.Base(filePath), i)

		docs[i] = llm.Document{
			ID:         docID,
			Content:    chunk.Content,
			Source:     filePath,
			FileType:   fileType,
			Title:      title,
			ChunkIndex: i,
			CreatedAt:  now,
			Metadata: map[string]interface{}{
				"chunk_count":    len(chunks),
				"chunk_index":    i,
				"original_title": parsedDoc.Title,
				"file_size":      len(parsedDoc.Content),
			},
		}

		// Copy parser metadata
		for k, v := range parsedDoc.Metadata {
			docs[i].Metadata[k] = v
		}
	}

	// Delete existing documents from the same source
	_ = globalKnowledgeVectorStore.DeleteBySource(ctx, filePath)

	// Add documents to vector store
	if err := globalKnowledgeVectorStore.AddBatch(ctx, docs); err != nil {
		return Error(fmt.Sprintf("failed to store documents: %v", err))
	}

	// Get updated count
	count, _ := globalKnowledgeVectorStore.Count(ctx)

	return Success(fmt.Sprintf("Document ingested successfully:\n"+
		"  Title: %s\n"+
		"  Source: %s\n"+
		"  Type: %s\n"+
		"  Chunks: %d\n"+
		"  Total documents in knowledge base: %d",
		title, filePath, fileType, len(chunks), count),
		&Metadata{
			FilePath:   filePath,
			MatchCount: len(chunks),
		}, TierCompact)
}

// GetIngestDocumentTool returns the document ingestion tool
func GetIngestDocumentTool() tool.InvokableTool {
	t, err := utils.InferTool(
		IngestDocumentToolName,
		ingestDescription,
		IngestDocumentFunc,
	)
	if err != nil {
		return nil
	}
	return t
}
