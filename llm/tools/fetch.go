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

var (
	FetchToolName = "fetch"
)

// FetchToolParams defines the arguments for the FetchTool.
type FetchToolParams struct {
	URL     string `json:"url" jsonschema:"description=The URL to fetch content from. Must start with http:// or https://"`
	Format  string `json:"format,omitempty" jsonschema:"description=The format to return the content in (text, markdown, or html). Default is text.,enum=text,enum=markdown,enum=html"`
	Timeout int    `json:"timeout,omitempty" jsonschema:"description=Optional timeout in seconds (default: 30, max: 120)"`
}

// FetchToolFunc implements the logic for fetching and converting web content.
// It includes safety mechanisms like timeouts and size limits.
func FetchToolFunc(ctx context.Context, params FetchToolParams) (string, error) {
	// 1. Validation
	if params.URL == "" {
		return "", fmt.Errorf("URL parameter is required")
	}
	if !strings.HasPrefix(params.URL, "http://") && !strings.HasPrefix(params.URL, "https://") {
		return "", fmt.Errorf("URL must start with http:// or https://")
	}

	format := strings.ToLower(params.Format)
	if format == "" {
		format = "text"
	}
	if format != "text" && format != "markdown" && format != "html" {
		return "", fmt.Errorf("format must be one of: text, markdown, html")
	}

	// 2. Setup Client with Timeout Safety
	timeout := params.Timeout
	if timeout <= 0 {
		timeout = 30 // Default 30s
	}
	if timeout > 120 {
		timeout = 120 // Hard limit 120s
	}

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// 3. Prepare Request
	req, err := http.NewRequestWithContext(ctx, "GET", params.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "opencode-fetch-tool/1.0")

	// 4. Execute Request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	// 5. Read Body with Size Limit Safety (Max 5MB)
	MaxReadSize := int64(5 * 1024 * 1024)

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, MaxReadSize))
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	content := string(bodyBytes)
	// 6. Format Conversion
	contentType := resp.Header.Get("Content-Type")
	switch params.Format {
	case "text":
		if strings.Contains(contentType, "text/html") {
			text, err := extractTextFromHTML(content)
			if err != nil {
				return "", fmt.Errorf("Failed to extract text from HTML: " + err.Error())
			}
			content = text
		}

	case "markdown":
		if strings.Contains(contentType, "text/html") {
			markdown, err := convertHTMLToMarkdown(content)
			if err != nil {
				return "", fmt.Errorf("Failed to convert HTML to Markdown: %s" + err.Error())
			}
			content = markdown
		}

		content = "```\n" + content + "\n```"

	case "html":
		// return only the body of the HTML document
		if strings.Contains(contentType, "text/html") {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
			if err != nil {
				return "", fmt.Errorf("Failed to parse HTML: " + err.Error())
			}
			body, err := doc.Find("body").Html()
			if err != nil {
				return "", fmt.Errorf("Failed to extract body from HTML: " + err.Error())
			}
			if body == "" {
				return "", fmt.Errorf("no body content found in HTML")
			}
			content = "<html>\n<body>\n" + body + "\n</body>\n</html>"
		}
	}
	// truncate content if it exceeds max read size
	if int64(len(content)) > MaxReadSize {
		content = content[:MaxReadSize]
		content += fmt.Sprintf("\n\n[Content truncated to %d bytes]", MaxReadSize)
	}

	return content, nil
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

	// 1. 去除多余空行
	lines := strings.Split(markdown, "\n")
	var result []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	// 2. 用单个空格连接（去除多余空格）
	return strings.Join(result, " "), nil
}

// GetFetchTool returns the Eino InvokableTool construction.
func GetFetchTool() tool.InvokableTool {
	t, err := utils.InferTool(
		"fetch_web_content",
		"Fetches content from a URL and converts it to text, markdown, or keeps it as HTML. Useful for reading documentation or external web pages.",
		FetchToolFunc,
	)
	if err != nil {
		log.Fatalf("failed to create fetch tool: %v", err)
	}
	return t
}
