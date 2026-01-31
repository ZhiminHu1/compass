package tools

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"golang.org/x/net/html"
)

const (
	// SearchToolName is the name of the search tool
	SearchToolName = "web_search"

	// DefaultSearchMaxResults is the default maximum number of search results
	DefaultSearchMaxResults = 10
	// MaxSearchMaxResults is the maximum allowed results
	MaxSearchMaxResults = 20
	// SearchTimeout is the timeout for search requests
	SearchTimeout = 30 * time.Second
	// MinSearchInterval is the minimum interval between searches
	MinSearchInterval = 500 * time.Millisecond
)

// SearchToolParams defines the parameters for the search tool
type SearchToolParams struct {
	Query      string `json:"query" jsonschema:"description=The search keywords or question to look for on the web"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"description=Maximum number of search results to return (default: 10, max: 20)"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Title    string
	Link     string
	Snippet  string
	Position int
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
}

var (
	lastSearchMu   sync.Mutex
	lastSearchTime time.Time
)

// searchDescription is the detailed tool description for the AI
const searchDescription = `Performs a web search to find latest information, news, or links.

BEFORE USING:
- Use this tool for real-time information
- Prefer knowledge base for historical data
- Use specific, focused queries

CAPABILITIES:
- Search the web for current information
- Get news, documentation, and resources
- Returns title, URL, and snippet for each result

RATE LIMITING:
- Minimum 500ms between searches
- Random delay to avoid blocking

PARAMETERS:
- query (required): The search keywords or question
- max_results (optional): Maximum results (default: 10, max: 20)

OUTPUT FORMAT:
Returns formatted search results with titles, URLs, and snippets.

EXAMPLES:
- Search news: {"query": "Golang 1.23 release notes"}
- Find docs: {"query": "CloudWeGo Eino documentation"}
- Quick info: {"query": "PowerShell Get-ChildItem examples"}`

// SearchToolFunc performs a web search using DuckDuckGo Lite
func SearchToolFunc(ctx context.Context, params SearchToolParams) (string, error) {
	if params.Query == "" {
		return Error("query parameter is required")
	}

	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = DefaultSearchMaxResults
	}
	if maxResults > MaxSearchMaxResults {
		maxResults = MaxSearchMaxResults
	}

	// Rate limiting
	maybeDelaySearch()

	// Build search URL
	searchURL := "https://lite.duckduckgo.com/lite/?q=" + url.QueryEscape(params.Query)

	client := &http.Client{Timeout: SearchTimeout}
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return Error(fmt.Sprintf("failed to create request: %v", err))
	}

	setRandomizedHeaders(req)

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return Error(fmt.Sprintf("search request failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Error(fmt.Sprintf("search failed with status code: %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Error(fmt.Sprintf("failed to read response: %v", err))
	}

	// Parse results
	results, err := parseLiteSearchResults(string(body), maxResults)
	if err != nil {
		return Error(fmt.Sprintf("failed to parse results: %v", err))
	}

	if len(results) == 0 {
		return Success(fmt.Sprintf("No results found for '%s'", params.Query),
			&Metadata{MatchCount: 0})
	}

	// Format output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d search results for '%s':\n\n", len(results), params.Query))
	for _, res := range results {
		sb.WriteString(fmt.Sprintf("- **%s**\n", res.Title))
		sb.WriteString(fmt.Sprintf("  URL: %s\n", res.Link))
		sb.WriteString(fmt.Sprintf("  Snippet: %s\n\n", res.Snippet))
	}

	var files []string
	for _, r := range results {
		files = append(files, r.Link)
	}

	return Success(sb.String(), &Metadata{
		MatchCount: len(results),
		Files:      files,
	})
}

// setRandomizedHeaders sets randomized HTTP headers to mimic a real browser
func setRandomizedHeaders(req *http.Request) {
	req.Header.Set("User-Agent", userAgents[rand.IntN(len(userAgents))])
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}

// parseLiteSearchResults parses DuckDuckGo Lite HTML results
func parseLiteSearchResults(htmlContent string, maxResults int) ([]SearchResult, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var results []SearchResult
	var currentResult *SearchResult

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "a" && hasClass(n, "result-link") {
				if currentResult != nil && currentResult.Link != "" {
					currentResult.Position = len(results) + 1
					results = append(results, *currentResult)
					if len(results) >= maxResults {
						return
					}
				}
				currentResult = &SearchResult{Title: getTextContent(n)}
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						currentResult.Link = cleanDuckDuckGoURL(attr.Val)
						break
					}
				}
			}
			if n.Data == "td" && hasClass(n, "result-snippet") && currentResult != nil {
				currentResult.Snippet = getTextContent(n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if len(results) >= maxResults {
				return
			}
			traverse(c)
		}
	}

	traverse(doc)

	if currentResult != nil && currentResult.Link != "" && len(results) < maxResults {
		currentResult.Position = len(results) + 1
		results = append(results, *currentResult)
	}

	return results, nil
}

// cleanDuckDuckGoURL extracts the final URL from DuckDuckGo's redirect link
func cleanDuckDuckGoURL(rawURL string) string {
	if strings.HasPrefix(rawURL, "//duckduckgo.com/l/?uddg=") || strings.Contains(rawURL, "uddg=") {
		if idx := strings.Index(rawURL, "uddg="); idx != -1 {
			encoded := rawURL[idx+5:]
			if ampIdx := strings.Index(encoded, "&"); ampIdx != -1 {
				encoded = encoded[:ampIdx]
			}
			if decoded, err := url.QueryUnescape(encoded); err == nil {
				return decoded
			}
		}
	}
	return rawURL
}

// maybeDelaySearch enforces a minimum interval between searches
func maybeDelaySearch() {
	lastSearchMu.Lock()
	defer lastSearchMu.Unlock()

	minGap := MinSearchInterval + time.Duration(rand.IntN(1500))*time.Millisecond
	elapsed := time.Since(lastSearchTime)
	if elapsed < minGap {
		time.Sleep(minGap - elapsed)
	}
	lastSearchTime = time.Now()
}

// hasClass checks if an HTML node has a specific CSS class
func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			for _, c := range strings.Fields(attr.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

// getTextContent recursively extracts text content from a node
func getTextContent(n *html.Node) string {
	var text strings.Builder
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(n)
	return strings.TrimSpace(text.String())
}

// GetSearchTool returns the search tool with enhanced description
func GetSearchTool() tool.InvokableTool {
	t, err := utils.InferTool(
		SearchToolName,
		searchDescription,
		SearchToolFunc,
	)
	if err != nil {
		log.Fatalf("failed to create search tool: %v", err)
	}
	return t
}
