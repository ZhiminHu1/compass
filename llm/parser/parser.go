package parser

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileType represents the type of document file
type FileType string

const (
	FileTypePDF     FileType = "pdf"
	FileTypeDocx    FileType = "docx"
	FileTypeMD      FileType = "md"
	FileTypeHTML    FileType = "html"
	FileTypeHTM     FileType = "htm"
	FileTypeTXT     FileType = "txt"
	FileTypeUnknown FileType = "unknown"
)

// Document represents a parsed document with its content and metadata
type Document struct {
	Content  string
	Title    string
	Metadata map[string]interface{}
}

// Parser defines the interface for document parsers
type Parser interface {
	// Parse reads and parses a document from the reader
	Parse(ctx context.Context, r io.Reader) (*Document, error)

	// ParseFile reads and parses a document from a file path
	ParseFile(ctx context.Context, filePath string) (*Document, error)

	// FileType returns the file type this parser handles
	FileType() FileType
}

// Registry holds all registered parsers
type Registry struct {
	parsers map[FileType]Parser
}

// NewRegistry creates a new parser registry
func NewRegistry() *Registry {
	return &Registry{
		parsers: make(map[FileType]Parser),
	}
}

// Register adds a parser to the registry
func (r *Registry) Register(p Parser) {
	r.parsers[p.FileType()] = p
}

// GetParser returns a parser for the given file type
func (r *Registry) GetParser(ft FileType) (Parser, bool) {
	p, ok := r.parsers[ft]
	return p, ok
}

// GetParserForPath returns a parser for the given file path
func (r *Registry) GetParserForPath(filePath string) (Parser, bool) {
	ext := strings.TrimPrefix(filepath.Ext(filePath), ".")
	ft := FileTypeFromExt(ext)
	return r.GetParser(ft)
}

// ParseFile parses a file using the appropriate parser
func (r *Registry) ParseFile(ctx context.Context, filePath string) (*Document, error) {
	parser, ok := r.GetParserForPath(filePath)
	if !ok {
		return nil, fmt.Errorf("no parser found for file: %s", filePath)
	}

	return parser.ParseFile(ctx, filePath)
}

// FileTypeFromExt converts a file extension to FileType
func FileTypeFromExt(ext string) FileType {
	switch strings.ToLower(ext) {
	case "pdf":
		return FileTypePDF
	case "docx", "doc":
		return FileTypeDocx
	case "md", "markdown":
		return FileTypeMD
	case "html", "htm":
		return FileTypeHTML
	case "txt":
		return FileTypeTXT
	default:
		return FileTypeUnknown
	}
}

// String returns the string representation of the FileType
func (ft FileType) String() string {
	return string(ft)
}

// DefaultRegistry returns a registry with all default parsers registered
func DefaultRegistry() *Registry {
	reg := NewRegistry()
	reg.Register(NewTxtParser())
	reg.Register(NewMarkdownParser())
	reg.Register(NewHTMLParser())

	// Note: PDF and DOCX parsers require additional dependencies
	// and should be registered explicitly if the dependencies are available
	return reg
}

// ReadFileContent reads file content for basic parsers
func ReadFileContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(data), nil
}

// ExtractTitle extracts a title from content (first line or heading)
func ExtractTitle(content, filePath string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return filepath.Base(filePath)
	}

	// Try to get first non-empty line as title
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Remove markdown heading markers
			line = strings.TrimPrefix(line, "#")
			line = strings.TrimSpace(line)
			if line != "" && len(line) < 100 {
				return line
			}
			break
		}
	}

	return filepath.Base(filePath)
}
