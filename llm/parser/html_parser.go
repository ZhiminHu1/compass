package parser

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// HTMLParser handles HTML files
type HTMLParser struct {
	// preserveStructure whether to preserve heading structure
	preserveStructure bool
}

// NewHTMLParser creates a new HTML parser
func NewHTMLParser() *HTMLParser {
	return &HTMLParser{
		preserveStructure: true,
	}
}

// Parse reads and parses HTML from the reader
func (p *HTMLParser) Parse(ctx context.Context, r io.Reader) (*Document, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTML: %w", err)
	}

	return p.parse(string(data), ""), nil
}

// ParseFile reads and parses an HTML file
func (p *HTMLParser) ParseFile(ctx context.Context, filePath string) (*Document, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.parse(string(data), filePath), nil
}

// parse processes the HTML content
func (p *HTMLParser) parse(content, filePath string) *Document {
	// Extract title from <title> tag
	title := p.extractTitle(content, filePath)

	// Remove script and style elements
	content = p.removeScripts(content)

	// Remove HTML comments
	content = p.removeComments(content)

	// Extract text content
	textContent := p.extractText(content)

	// Clean up whitespace
	textContent = p.cleanWhitespace(textContent)

	return &Document{
		Content: textContent,
		Title:   title,
		Metadata: map[string]interface{}{
			"file_size":      len(content),
			"html_tag_count": countTags(content),
		},
	}
}

// extractTitle extracts the title from HTML
func (p *HTMLParser) extractTitle(content, filePath string) string {
	// Try <title> tag first
	re := regexp.MustCompile(`<title[^>]*>(.*?)</title>`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		title := strings.TrimSpace(matches[1])
		if title != "" {
			return title
		}
	}

	// Try <h1> tag
	re = regexp.MustCompile(`<h1[^>]*>(.*?)</h1>`)
	matches = re.FindStringSubmatch(content)
	if len(matches) > 1 {
		title := stripHTMLTags(matches[1])
		title = strings.TrimSpace(title)
		if title != "" {
			return title
		}
	}

	// Fall back to filename
	if filePath != "" {
		return extractFileName(filePath)
	}
	return "Untitled"
}

// removeScripts removes script and style elements
func (p *HTMLParser) removeScripts(content string) string {
	// Remove script tags
	re := regexp.MustCompile(`<script[^>]*>[\s\S]*?</script>`)
	content = re.ReplaceAllString(content, "")

	// Remove style tags
	re = regexp.MustCompile(`<style[^>]*>[\s\S]*?</style>`)
	content = re.ReplaceAllString(content, "")

	return content
}

// removeComments removes HTML comments
func (p *HTMLParser) removeComments(content string) string {
	re := regexp.MustCompile(`<!--[\s\S]*?-->`)
	return re.ReplaceAllString(content, "")
}

// extractText extracts readable text from HTML
func (p *HTMLParser) extractText(content string) string {
	// Replace block elements with newlines
	content = p.replaceBlockElements(content)

	// Remove all remaining HTML tags
	re := regexp.MustCompile(`<[^>]+>`)
	content = re.ReplaceAllString(content, " ")

	// Decode HTML entities
	content = p.decodeEntities(content)

	return content
}

// replaceBlockElements replaces block elements with appropriate whitespace
func (p *HTMLParser) replaceBlockElements(content string) string {
	// Replace block-level tags with newlines
	blockTags := []string{
		"div", "p", "h1", "h2", "h3", "h4", "h5", "h6",
		"br", "hr", "li", "tr", "th", "td",
		"header", "footer", "main", "section", "article",
		"ul", "ol", "table", "blockquote", "pre", "code",
	}

	result := content
	for _, tag := range blockTags {
		// Match both opening and closing tags
		re := regexp.MustCompile(fmt.Sprintf(`</?%s[^>]*>`, tag))
		result = re.ReplaceAllString(result, "\n")
	}

	return result
}

// decodeEntities decodes common HTML entities
func (p *HTMLParser) decodeEntities(content string) string {
	// Common HTML entities
	entities := map[string]string{
		"&nbsp;":  " ",
		"&lt;":    "<",
		"&gt;":    ">",
		"&amp;":   "&",
		"&quot;":  "\"",
		"&apos;":  "'",
		"&copy;":  "(c)",
		"&reg;":   "(r)",
		"&mdash;": "-",
		"&ndash;": "-",
	}

	result := content
	for entity, replacement := range entities {
		result = strings.ReplaceAll(result, entity, replacement)
	}

	// Handle numeric entities
	re := regexp.MustCompile(`&#(\d+);`)
	result = re.ReplaceAllStringFunc(result, func(match string) string {
		numStr := match[2 : len(match)-1]
		// Simple handling for common numeric entities
		if numStr == "8217" {
			return "'"
		}
		if numStr == "8220" || numStr == "8221" {
			return "\""
		}
		return " "
	})

	return result
}

// cleanWhitespace cleans up extra whitespace
func (p *HTMLParser) cleanWhitespace(content string) string {
	// Replace multiple spaces with single space
	re := regexp.MustCompile(`[ \t]+`)
	content = re.ReplaceAllString(content, " ")

	// Replace multiple newlines with double newline
	re = regexp.MustCompile(`\n\s*\n\s*\n+`)
	content = re.ReplaceAllString(content, "\n\n")

	// Trim leading/trailing whitespace
	content = strings.TrimSpace(content)

	return content
}

// FileType returns the file type this parser handles
func (p *HTMLParser) FileType() FileType {
	return FileTypeHTML
}

// stripHTMLTags removes all HTML tags from a string
func stripHTMLTags(s string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return re.ReplaceAllString(s, "")
}

// countTags counts the number of HTML tags
func countTags(content string) int {
	re := regexp.MustCompile(`<[^>]+>`)
	return len(re.FindAllString(content, -1))
}
