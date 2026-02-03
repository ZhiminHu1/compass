package parser

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// MarkdownParser handles markdown files
type MarkdownParser struct {
	// stripCodeBlocks whether to remove code blocks from content
	stripCodeBlocks bool
}

// NewMarkdownParser creates a new markdown parser
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{
		stripCodeBlocks: false, // Keep code blocks by default
	}
}

// Parse reads and parses markdown from the reader
func (p *MarkdownParser) Parse(ctx context.Context, r io.Reader) (*Document, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read markdown: %w", err)
	}

	return p.parse(string(data), ""), nil
}

// ParseFile reads and parses a markdown file
func (p *MarkdownParser) ParseFile(ctx context.Context, filePath string) (*Document, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.parse(string(data), filePath), nil
}

// parse processes the markdown content
func (p *MarkdownParser) parse(content, filePath string) *Document {
	// Extract metadata from YAML frontmatter
	metadata := p.extractFrontmatter(content)
	processedContent := p.removeFrontmatter(content)

	// Optionally strip code blocks
	if p.stripCodeBlocks {
		processedContent = p.removeCodeBlocks(processedContent)
	}

	// Clean up markdown formatting for better embedding
	processedContent = p.cleanMarkdown(processedContent)

	// Extract title
	title := p.extractTitle(processedContent, filePath)
	if frontmatterTitle, ok := metadata["title"].(string); ok {
		title = frontmatterTitle
	}

	// Add parsing metadata
	metadata["file_size"] = len(content)
	metadata["line_count"] = countLines(content)
	metadata["has_frontmatter"] = hasFrontmatter(content)

	return &Document{
		Content:  processedContent,
		Title:    title,
		Metadata: metadata,
	}
}

// extractFrontmatter extracts YAML frontmatter from content
func (p *MarkdownParser) extractFrontmatter(content string) map[string]interface{} {
	metadata := make(map[string]interface{})

	if !hasFrontmatter(content) {
		return metadata
	}

	// Find the closing ---
	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		return metadata
	}

	// Skip first line (opening ---)
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "---" {
			break
		}

		// Parse simple key: value pairs
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			// Remove quotes if present
			value = strings.Trim(value, `"`)
			metadata[key] = value
		}
	}

	return metadata
}

// removeFrontmatter removes YAML frontmatter from content
func (p *MarkdownParser) removeFrontmatter(content string) string {
	if !hasFrontmatter(content) {
		return content
	}

	lines := strings.Split(content, "\n")
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[i+1:], "\n")
		}
	}
	return content
}

// hasFrontmatter checks if content has YAML frontmatter
func hasFrontmatter(content string) bool {
	lines := strings.Split(content, "\n")
	return len(lines) >= 2 && strings.TrimSpace(lines[0]) == "---"
}

// removeCodeBlocks removes markdown code blocks
func (p *MarkdownParser) removeCodeBlocks(content string) string {
	// Remove fenced code blocks
	re := regexp.MustCompile("```[\\s\\S]*?```")
	content = re.ReplaceAllString(content, "")

	// Remove inline code
	re = regexp.MustCompile("`[^`]+`")
	content = re.ReplaceAllString(content, "")

	return content
}

// cleanMarkdown cleans up markdown formatting for better embedding
func (p *MarkdownParser) cleanMarkdown(content string) string {
	// Remove markdown headers but keep the text
	re := regexp.MustCompile(`^#+\s+(.*)$`)
	content = re.ReplaceAllString(content, "$1")

	// Remove bold/italic markers
	content = strings.ReplaceAll(content, "**", "")
	content = strings.ReplaceAll(content, "__", "")
	content = strings.ReplaceAll(content, "*", "")
	content = strings.ReplaceAll(content, "_", "")

	// Remove links but keep the text
	re = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
	content = re.ReplaceAllString(content, "$1")

	// Remove image references
	re = regexp.MustCompile(`!\[([^\]]*)\]\([^\)]+\)`)
	content = re.ReplaceAllString(content, "$1")

	// Clean up extra whitespace
	lines := strings.Split(content, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "<") { // Skip HTML tags
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n\n")
}

// extractTitle extracts the title from markdown content
func (p *MarkdownParser) extractTitle(content, filePath string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Check for heading
			if strings.HasPrefix(line, "#") {
				// Remove heading markers
				title := strings.TrimLeft(line, "#")
				title = strings.TrimSpace(title)
				if title != "" && len(title) < 100 {
					return title
				}
			} else {
				// Use first non-empty line
				if len(line) < 100 {
					return line
				}
			}
			break
		}
	}

	if filePath != "" {
		return extractFileName(filePath)
	}
	return "Untitled"
}

// extractFileName extracts the base name without extension
func extractFileName(path string) string {
	// Get just the filename
	parts := strings.Split(path, "/")
	parts = strings.Split(parts[len(parts)-1], "\\")
	filename := parts[len(parts)-1]

	// Remove extension
	if idx := strings.LastIndex(filename, "."); idx > 0 {
		filename = filename[:idx]
	}

	// Convert kebab-case to Title Case
	filename = strings.ReplaceAll(filename, "-", " ")
	filename = strings.ReplaceAll(filename, "_", " ")

	return strings.ToTitle(filename)
}

// FileType returns the file type this parser handles
func (p *MarkdownParser) FileType() FileType {
	return FileTypeMD
}

// countLines counts the number of lines in content
func countLines(content string) int {
	return len(strings.Split(content, "\n"))
}
