package parser

import (
	"context"
	"fmt"
	"io"
	"os"
)

// TxtParser handles plain text files
type TxtParser struct{}

// NewTxtParser creates a new plain text parser
func NewTxtParser() *TxtParser {
	return &TxtParser{}
}

// Parse reads and parses plain text from the reader
func (p *TxtParser) Parse(ctx context.Context, r io.Reader) (*Document, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read text: %w", err)
	}

	content := string(data)
	return &Document{
		Content:  content,
		Title:    ExtractTitle(content, ""),
		Metadata: make(map[string]interface{}),
	}, nil
}

// ParseFile reads and parses a plain text file
func (p *TxtParser) ParseFile(ctx context.Context, filePath string) (*Document, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)
	return &Document{
		Content: content,
		Title:   ExtractTitle(content, filePath),
		Metadata: map[string]interface{}{
			"file_size":  len(data),
			"line_count": len(splitLines(content)),
		},
	}, nil
}

// FileType returns the file type this parser handles
func (p *TxtParser) FileType() FileType {
	return FileTypeTXT
}

// splitLines splits content into lines
func splitLines(content string) []string {
	lines := []string{}
	current := ""
	for _, ch := range content {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
