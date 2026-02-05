package tools

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

const (
	// FetchToolName is the name of the fetch tool
	FetchToolName = "fetch"

	// DefaultTimeout is the default request timeout
	DefaultTimeout = 30
	// MaxTimeout is the maximum allowed timeout
	MaxTimeout = 120
	// MaxReadSize is the maximum response size (5MB)
	MaxReadSize = int64(5 * 1024 * 1024)
)

// FetchToolParams defines the arguments for the FetchTool.
type FetchToolParams struct {
	URL     string `json:"url" jsonschema:"description=The URL to fetch content from. Must start with http:// or https://"`
	Format  string `json:"format,omitempty" jsonschema:"description=The format to return the content in (text, markdown, or html). Default is text.,enum=text,enum=markdown,enum=html"`
	Timeout int    `json:"timeout,omitempty" jsonschema:"description=Optional timeout in seconds (default: 30, max: 120)"`
}

// fetchDescription is the detailed tool description for the AI
const fetchDescription = `Fetch content from a URL and convert it to text, markdown, or HTML.

BEFORE USING:
- Verify the URL is accessible
- Prefer markdown format for better readability
- Consider timeout for large pages

CAPABILITIES:
- Fetch web pages and extract content
- Convert HTML to readable text or markdown
- Handle redirects automatically
- Size limit: 5MB

SUPPORTED FORMATS:
- text:     Plain text extraction (default)
- markdown: HTML converted to markdown
- html:     Raw HTML content

PARAMETERS:
- url (required): The URL to fetch (must start with http:// or https://)
- format (optional): Output format - text, markdown, or html (default: text)
- timeout (optional): Timeout in seconds (default: 30, max: 120)

OUTPUT FORMAT:
Returns the fetched and formatted content.

EXAMPLES:
- Fetch as markdown: {"url": "https://example.com", "format": "markdown"}
- Quick text: {"url": "https://example.com", "format": "text"}
- With timeout: {"url": "https://example.com", "timeout": 60}`

// FetchToolFunc implements the logic for fetching and converting web content.
func FetchToolFunc(ctx context.Context, params FetchToolParams) (string, error) {
	// 1. Validation
	if params.URL == "" {
		return Error("URL parameter is required")
	}
	if !strings.HasPrefix(params.URL, "http://") && !strings.HasPrefix(params.URL, "https://") {
		return Error("URL must start with http:// or https://")
	}

	format := strings.ToLower(params.Format)
	if format == "" {
		format = "text"
	}
	if format != "text" && format != "markdown" && format != "html" {
		return Error("format must be one of: text, markdown, html")
	}

	// 2. Setup Client with Timeout
	timeout := params.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if timeout > MaxTimeout {
		timeout = MaxTimeout
	}

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// 3. Prepare Request
	req, err := http.NewRequestWithContext(ctx, "GET", params.URL, nil)
	if err != nil {
		return Error(fmt.Sprintf("failed to create request: %v", err))
	}
	req.Header.Set("User-Agent", "compass-fetch-tool/1.0")

	// 4. Execute Request
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return Error(fmt.Sprintf("failed to fetch URL: %v", err))
	}
	defer resp.Body.Close()

	// 5. Read Body with Size Limit
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, MaxReadSize))
	if err != nil {
		return Error(fmt.Sprintf("failed to read response: %v", err))
	}

	content := string(bodyBytes)
	truncated := int64(len(content)) >= MaxReadSize

	// 6. Format Conversion
	contentType := resp.Header.Get("Content-Type")
	switch format {
	case "text":
		if strings.Contains(contentType, "text/html") {
			text, err := extractTextFromHTML(content)
			if err != nil {
				return Error(fmt.Sprintf("failed to extract text: %v", err))
			}
			content = text
		}

	case "markdown":
		if strings.Contains(contentType, "text/html") {
			markdown, err := convertHTMLToMarkdown(content)
			if err != nil {
				return Error(fmt.Sprintf("failed to convert to markdown: %v", err))
			}
			content = markdown
		}

	case "html":
		if strings.Contains(contentType, "text/html") {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
			if err != nil {
				return Error(fmt.Sprintf("failed to parse HTML: %v", err))
			}
			body, err := doc.Find("body").Html()
			if err == nil && body != "" {
				content = "<html>\n<body>\n" + body + "\n</body>\n</html>"
			}
		}
	}

	if truncated {
		content += fmt.Sprintf("\n\n[Content truncated to %d bytes]", MaxReadSize)
	}

	duration := time.Since(startTime)

	if resp.StatusCode != http.StatusOK {
		return Partial(content, &Metadata{
			URL:        params.URL,
			StatusCode: resp.StatusCode,
			Duration:   duration.Milliseconds(),
		})
	}

	return FetchSuccess(content, params.URL, resp.StatusCode)
}

func extractTextFromHTML(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	text := doc.Find("body").Text()
	text = strings.Join(strings.Fields(text), " ")
	return text, nil
}

func convertHTMLToMarkdown(html string) (string, error) {
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(html)
	if err != nil {
		return "", err
	}

	// Clean up excessive blank lines
	lines := strings.Split(markdown, "\n")
	var result []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return strings.Join(result, "\n"), nil
}

// GetFetchTool returns the fetch tool with enhanced description.
func GetFetchTool() tool.InvokableTool {
	t, err := utils.InferTool(
		FetchToolName,
		fetchDescription,
		FetchToolFunc,
	)
	if err != nil {
		log.Fatalf("failed to create fetch tool: %v", err)
	}
	return t
}
