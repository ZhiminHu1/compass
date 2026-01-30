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

// SearchToolParams 定义了搜索工具接收的参数，符合 Eino 工具调用规范
type SearchToolParams struct {
	Query      string `json:"query" jsonschema:"description=The search keywords or question to look for on the web."`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"description=Maximum number of search results to return (default: 10, max: 20)"`
}

// SearchResult 内部定义的搜索结果结构体，用于解析 HTML 时存储中间数据
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

// setRandomizedHeaders 这个函数非常关键：
// 它通过随机设置 User-Agent 和其他 HTTP 头信息，模拟真实浏览器的指纹。
// 这样可以有效绕过搜索引擎对机器人（Bot）的初级屏蔽，提高抓取成功率。
func setRandomizedHeaders(req *http.Request) {
	req.Header.Set("User-Agent", userAgents[rand.IntN(len(userAgents))])
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}

// parseLiteSearchResults 这是处理核心逻辑：
// 1. 使用 golang.org/x/net/html 将 HTML 字符串解析成 DOM 树节点（Node）。
// 2. 使用递归函数 traverse 深度优先遍历所有节点。
// 3. 寻找带有 "result-link" 类的 <a> 标签提取标题和链接。
// 4. 寻找带有 "result-snippet" 类的 <td> 标签提取摘要。
// 这种基于“特征类名”的抓取方式比正则表达式更稳健。
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
			// 查找鸭鸭走 Lite 版的特定 DOM 结构
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

// cleanDuckDuckGoURL 负责清洗“外联重定向”。
// DuckDuckGo 会在原始链接前包裹一层自己的统计链接（如 //duckduckgo.com/l/?uddg=...）。
// 该函数通过 URL 解码提取出最终的目标地址，让 Agent 能直接访问真实网站。
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

var (
	lastSearchMu   sync.Mutex
	lastSearchTime time.Time
)

// maybeDelaySearch 流量哨兵：
// 搜索引擎对高频请求极其敏感。这个函数强制在两次搜索之间保持 0.5s ~ 2s 的随机间隔。
// 这是保护你的服务器 IP 不被搜索引擎加入黑名单的关键。
func maybeDelaySearch() {
	lastSearchMu.Lock()
	defer lastSearchMu.Unlock()

	minGap := time.Duration(500+rand.IntN(1500)) * time.Millisecond
	elapsed := time.Since(lastSearchTime)
	if elapsed < minGap {
		time.Sleep(minGap - elapsed)
	}
	lastSearchTime = time.Now()
}

// SearchToolFunc 是 Eino 工具的核心回调函数
func SearchToolFunc(ctx context.Context, params SearchToolParams) (string, error) {
	if params.Query == "" {
		return "", fmt.Errorf("query parameter is required")
	}

	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 20 {
		maxResults = 20
	}

	// 1. 流量控制
	maybeDelaySearch()

	// 2. 构造 Lite 版搜索 URL（Lite 版 HTML 结构简单，解析更快且不易变动）
	searchURL := "https://lite.duckduckgo.com/lite/?q=" + url.QueryEscape(params.Query)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", err
	}

	setRandomizedHeaders(req)

	// 3. 执行网络请求
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 4. 解析结果
	results, err := parseLiteSearchResults(string(body), maxResults)
	if err != nil {
		return "", err
	}

	// 5. 格式化成 LLM 易读的 Markdown 文本
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d search results for '%s':\n\n", len(results), params.Query))
	for _, res := range results {
		sb.WriteString(fmt.Sprintf("- **%s**\n", res.Title))
		sb.WriteString(fmt.Sprintf("  URL: %s\n", res.Link))
		sb.WriteString(fmt.Sprintf("  Snippet: %s\n\n", res.Snippet))
	}
	return sb.String(), nil
}

// GetSearchTool 返回 Eino 可调用的工具实例
func GetSearchTool() tool.InvokableTool {
	t, err := utils.InferTool(
		"web_search",
		"Performs a web search to find latest information, news, or links. Essential for tasks requiring real-time data.",
		SearchToolFunc,
	)
	if err != nil {
		log.Fatalf("failed to create search tool: %v", err)
	}
	return t
}

// 辅助函数：检查 HTML 节点是否包含特定 CSS 类
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

// 辅助函数：提取节点及其子节点的递归纯文本
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
