package llm

// Document represents a document with content and metadata for vector storage
type Document struct {
	ID         string                 `json:"id"`
	Content    string                 `json:"content"`
	Source     string                 `json:"source"`
	FileType   string                 `json:"file_type"`
	Title      string                 `json:"title"`
	ChunkIndex int                    `json:"chunk_index"`
	Vector     []float32              `json:"vector,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  string                 `json:"created_at"`
}

// SearchResult represents a search result with relevance score
type SearchResult struct {
	Document Document
	Score    float32
}

// ListFilter defines filters for listing documents
type ListFilter struct {
	Source   string // Filter by source file path
	FileType string // Filter by file type (pdf, docx, md, txt, html)
	Limit    int    // Maximum number of results
	Offset   int    // Offset for pagination
}
