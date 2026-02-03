package parser

import (
	"context"
	"fmt"
	"io"
)

// DocxParser handles Word documents (.docx)
// Note: This requires a docx parsing library such as:
// - github.com/fumiama/go-docx (MIT licensed)
// - github.com/nguyenthenguyen/docx (MIT licensed)
//
// To enable DOCX parsing:
// 1. Add the dependency: go get github.com/fumiama/go-docx
// 2. Uncomment the implementation below
type DocxParser struct {
	// preserveFormatting whether to preserve text formatting info
	preserveFormatting bool
}

// NewDocxParser creates a new DOCX parser
func NewDocxParser() *DocxParser {
	return &DocxParser{
		preserveFormatting: false,
	}
}

// Parse reads and parses DOCX from the reader
func (p *DocxParser) Parse(ctx context.Context, r io.Reader) (*Document, error) {
	// Placeholder implementation
	return nil, fmt.Errorf("DOCX parser not enabled - requires go-docx library")
}

// ParseFile reads and parses a DOCX file
func (p *DocxParser) ParseFile(ctx context.Context, filePath string) (*Document, error) {
	// Placeholder implementation
	return nil, fmt.Errorf("DOCX parser not enabled - requires go-docx library (github.com/fumiama/go-docx)")
}

// FileType returns the file type this parser handles
func (p *DocxParser) FileType() FileType {
	return FileTypeDocx
}

/*
// Example implementation with go-docx:

import (
	"github.com/fumiama/go-docx"
)

func (p *DocxParser) ParseFile(ctx context.Context, filePath string) (*Document, error) {
	// Open the DOCX file
	doc, err := docx.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DOCX: %w", err)
	}
	defer doc.Close()

	// Extract all paragraphs
	paragraphs := doc.Paras()

	var contentBuilder strings.Builder
	for _, para := range paragraphs {
		text := para.Text()
		contentBuilder.WriteString(text)
		contentBuilder.WriteString("\n")
	}

	content := contentBuilder.String()

	return &Document{
		Content:  content,
		Title:    ExtractTitle(content, filePath),
		Metadata: map[string]interface{}{
			"paragraph_count": len(paragraphs),
			"file_size": getFileSize(filePath),
		},
	}, nil
}
*/
