package component

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/cloudwego/eino/schema"
)

// ToolRenderer 负责工具调用的渲染逻辑
type ToolRenderer struct {
	// 配置：只显示元数据的工具列表
	metadataOnlyTools map[string]bool
}

// NewToolRenderer 创建工具渲染器
func NewToolRenderer() *ToolRenderer {
	return &ToolRenderer{
		metadataOnlyTools: map[string]bool{
			"write":      false, //write -写入文件
			"edit":       false, // edit - 编辑文件
			"delete":     false,
			"list":       false, //list  - 目录列表
			"bash":       false, // bash - 命令执行
			"read":       true,  // read - 文件读取
			"grep":       true,  // grep - 内容搜索
			"glob":       true,  // glob - 文件匹配
			"web_search": true,  // web_search - 网络搜索
			"fetch":      true,  // fetch - 网页获取
		},
	}
}

// ToolStyles 样式配置（直接使用 lipgloss.Style）
type ToolStyles struct {
	Indent   lipgloss.Style
	Border   lipgloss.Style
	System   lipgloss.Style
	Tool     lipgloss.Style
	ToolName lipgloss.Style
}

// NewToolStylesFromDefaultStyles 从 renderer.DefaultStyles() 创建工具样式
func NewToolStylesFromDefaultStyles(styles interface{}) *ToolStyles {
	// 类型断言获取 renderer.ToolStyles
	type ToolStylesLike interface {
		GetIndent() interface{}
		GetBorder() interface{}
		GetSystem() interface{}
		GetTool() interface{}
		GetToolName() interface{}
	}

	// 如果是 renderer.ToolStyles 类型，直接转换
	if ts, ok := styles.(struct {
		Indent   lipgloss.Style
		Border   lipgloss.Style
		System   lipgloss.Style
		Tool     lipgloss.Style
		ToolName lipgloss.Style
	}); ok {
		return &ToolStyles{
			Indent:   ts.Indent,
			Border:   ts.Border,
			System:   ts.System,
			Tool:     ts.Tool,
			ToolName: ts.ToolName,
		}
	}

	// 默认样式
	return &ToolStyles{
		Indent:   lipgloss.NewStyle().PaddingLeft(2),
		Border:   lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Faint(true),
		System:   lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Italic(true),
		Tool:     lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a")),
		ToolName: lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68")).Bold(true),
	}
}

// RenderToolCall 渲染单个工具调用及结果（实现接口）
func (r *ToolRenderer) RenderToolCall(tc schema.ToolCall, index int, getResultFunc func(string) (string, bool), styles interface{}) string {
	// 将 interface{} 转换为 ToolStyles
	toolStyles := NewToolStylesFromDefaultStyles(styles)
	var parts []string

	// 工具调用头部
	header := toolStyles.Indent.Render(
		toolStyles.Border.Render("┌─ ") +
			toolStyles.ToolName.Render(fmt.Sprintf("Tool Call #%d: %s", index, tc.Function.Name)),
	)
	parts = append(parts, header)

	// 格式化参数
	if tc.Function.Arguments != "" {
		formattedArgs := r.formatArguments(tc.Function.Arguments, tc.Function.Name)
		if formattedArgs != "" {
			argsLine := toolStyles.Indent.Render(
				toolStyles.Border.Render("│ ") +
					toolStyles.System.Render("Arguments: ") +
					formattedArgs,
			)
			parts = append(parts, argsLine)
		}
	}

	// 获取工具结果
	if result, ok := getResultFunc(tc.ID); ok {
		renderedResult := r.renderResult(tc.Function.Name, result, toolStyles)
		if renderedResult != "" {
			parts = append(parts, renderedResult)
		}

		footer := toolStyles.Indent.Render(toolStyles.Border.Render("└─"))
		parts = append(parts, footer)
	} else {
		// 没有结果，显示正在执行
		statusLine := toolStyles.Indent.Render(
			toolStyles.Border.Render("│ ") +
				toolStyles.System.Render("Status: ") +
				"Executing...",
		)
		parts = append(parts, statusLine)

		footer := toolStyles.Indent.Render(toolStyles.Border.Render("└─"))
		parts = append(parts, footer)
	}

	return strings.Join(parts, "\n")
}

// renderResult 渲染工具结果
func (r *ToolRenderer) renderResult(toolName, result string, styles *ToolStyles) string {
	// 检查是否为只显示元数据的工具
	if r.metadataOnlyTools[toolName] {
		// 尝试解析结构化元数据
		if summary := r.extractMetadataSummary(toolName, result); summary != "" {
			resultHeader := styles.Indent.Render(styles.Border.Render("├─ ") + styles.Tool.Render("Result:"))
			resultBody := styles.Indent.Render(
				styles.Border.Render("│  ") + summary,
			)
			return resultHeader + "\n" + resultBody
		}
	}

	// 默认：显示完整结果（截断长内容）
	maxLen := 500
	displayResult := result
	if len(result) > maxLen {
		displayResult = result[:maxLen] + "..."
	}

	resultHeader := styles.Indent.Render(styles.Border.Render("├─ ") + styles.Tool.Render("Result:"))
	resultBody := styles.Indent.Render(
		styles.Border.Render("│  ") + displayResult,
	)
	return resultHeader + "\n" + resultBody
}

// extractMetadataSummary 从工具结果中提取元数据摘要
func (r *ToolRenderer) extractMetadataSummary(toolName, result string) string {
	// 1. 首先尝试解析 <metadata /> 标签
	if metadata := r.parseMetadataTag(result); metadata != "" {
		return r.formatMetadataSummary(toolName, metadata)
	}

	// 2. 尝试解析 JSON 格式的 ToolResult
	if jsonSummary := r.parseToolResultJSON(toolName, result); jsonSummary != "" {
		return jsonSummary
	}

	// 3. 针对特定工具的后备解析
	return r.fallbackSummary(toolName, result)
}

// parseMetadataTag 解析 <metadata /> 标签
func (r *ToolRenderer) parseMetadataTag(result string) string {
	// 匹配 <metadata key=value ... />
	re := regexp.MustCompile(`<metadata\s+(.+?)\s*/>`)
	matches := re.FindStringSubmatch(result)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// parseToolResultJSON 解析 JSON 格式的 ToolResult
func (r *ToolRenderer) parseToolResultJSON(toolName, result string) string {
	// 尝试解析 JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		return ""
	}

	// 检查是否是 ToolResult 格式
	status, hasStatus := data["status"].(string)
	_, hasContent := data["content"]
	metadata, hasMetadata := data["metadata"].(map[string]interface{})

	if !hasStatus || !hasContent {
		return ""
	}

	// 从 metadata 中提取信息
	var parts []string
	if hasMetadata {
		if file, ok := metadata["file_path"].(string); ok {
			parts = append(parts, fmt.Sprintf("📄 %s", r.shortenPath(file, 30)))
		}
		if lines, ok := metadata["line_count"].(float64); ok {
			parts = append(parts, fmt.Sprintf("📏 %d 行", int(lines)))
		}
		if matches, ok := metadata["match_count"].(float64); ok {
			parts = append(parts, fmt.Sprintf("🔍 %d 个匹配", int(matches)))
		}
		if cmd, ok := metadata["command"].(string); ok {
			parts = append(parts, fmt.Sprintf("⚡ %s", r.shortenString(cmd, 30)))
		}
		if duration, ok := metadata["duration_ms"].(float64); ok {
			parts = append(parts, fmt.Sprintf("⏱️ %dms", int(duration)))
		}
		if exitCode, ok := metadata["exit_code"].(float64); ok {
			if exitCode == 0 {
				parts = append(parts, "✅ 成功")
			} else {
				parts = append(parts, fmt.Sprintf("❌ 退出码: %d", int(exitCode)))
			}
		}
		if url, ok := metadata["url"].(string); ok {
			parts = append(parts, fmt.Sprintf("🔗 %s", r.shortenPath(url, 30)))
		}
		if statusCode, ok := metadata["status_code"].(float64); ok {
			parts = append(parts, fmt.Sprintf("📊 HTTP %d", int(statusCode)))
		}
		if byteCount, ok := metadata["byte_count"].(float64); ok {
			parts = append(parts, fmt.Sprintf("📦 %s", r.formatBytes(int(byteCount))))
		}
		if files, ok := metadata["files"].([]interface{}); ok {
			parts = append(parts, fmt.Sprintf("📁 %d 个文件", len(files)))
		}
	}

	if status == "error" {
		parts = append(parts, "❌ 错误")
	} else if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " · ")
}

// fallbackSummary 后备摘要生成
func (r *ToolRenderer) fallbackSummary(toolName, result string) string {
	switch toolName {
	case "fetch":
		// fetch 工具：尝试从参数中提取 URL（需要在调用时传入）
		return "📄 网页内容已获取"

	case "web_search":
		// 搜索结果：统计行数
		if strings.Contains(result, "Found ") && strings.Contains(result, "search results") {
			// 提取结果数量
			re := regexp.MustCompile(`Found (\d+) search results`)
			if matches := re.FindStringSubmatch(result); len(matches) > 1 {
				return fmt.Sprintf("🔍 %s 个结果", matches[1])
			}
		}
		lines := strings.Count(result, "\n")
		return fmt.Sprintf("🔍 约 %d 行结果", lines)

	case "bash":
		// bash 命令：显示输出行数
		lines := strings.Count(result, "\n")
		if lines > 10 {
			return fmt.Sprintf("⚡ 输出 %d 行", lines)
		}
		return ""

	case "read", "view", "read_file":
		// 文件读取：显示行数
		lines := strings.Count(result, "\n")
		return fmt.Sprintf("📄 %d 行", lines+1)

	case "list", "list_dir":
		// 目录列表：统计文件数量
		files := strings.Count(result, "\n")
		return fmt.Sprintf("📁 %d 个项目", files)

	case "grep":
		// grep 搜索：统计匹配数
		matches := strings.Count(result, "\n")
		return fmt.Sprintf("🔍 %d 个匹配", matches)

	case "glob":
		// glob 匹配：统计文件数
		files := strings.Count(strings.TrimSpace(result), "\n")
		if files > 0 {
			return fmt.Sprintf("📁 %d 个文件", files+1)
		}
		return ""

	default:
		return ""
	}
}

// formatMetadataSummary 格式化元数据为摘要
func (r *ToolRenderer) formatMetadataSummary(toolName, metadataStr string) string {
	// 解析 key=value 格式
	parts := strings.Fields(metadataStr)

	// fetch 特殊处理：优先显示 URL
	var fetchURL string
	var otherSummaries []string
	var summaries []string // 声明在外层，避免作用域问题

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := kv[0]
		value := strings.Trim(kv[1], `"'`)

		switch key {
		case "url":
			fetchURL = value
		case "file", "file_path":
			otherSummaries = append(otherSummaries, fmt.Sprintf("📄 %s", r.shortenPath(value, 30)))
		case "lines", "line_count":
			otherSummaries = append(otherSummaries, fmt.Sprintf("📏 %s 行", value))
		case "bytes", "byte_count":
			if bytes, err := parseBytes(value); err == nil {
				otherSummaries = append(otherSummaries, fmt.Sprintf("📦 %s", r.formatBytes(bytes)))
			}
		case "matches", "match_count":
			otherSummaries = append(otherSummaries, fmt.Sprintf("🔍 %s 个匹配", value))
		case "cmd", "command":
			otherSummaries = append(otherSummaries, fmt.Sprintf("⚡ %s", r.shortenString(value, 30)))
		case "duration", "duration_ms":
			if d, err := time.ParseDuration(value + "ms"); err == nil {
				otherSummaries = append(otherSummaries, fmt.Sprintf("⏱️ %v", d.Round(time.Millisecond)))
			} else {
				otherSummaries = append(otherSummaries, fmt.Sprintf("⏱️ %s", value))
			}
		case "exit", "exit_code":
			if value == "0" {
				otherSummaries = append(otherSummaries, "✅")
			} else {
				otherSummaries = append(otherSummaries, fmt.Sprintf("❌ exit:%s", value))
			}
		case "status", "status_code":
			if value == "200" {
				otherSummaries = append(otherSummaries, "✅")
			} else {
				otherSummaries = append(otherSummaries, fmt.Sprintf("📊 %s", value))
			}
		case "files":
			otherSummaries = append(otherSummaries, fmt.Sprintf("📁 %s 个文件", value))
		case "timeout":
			if value == "true" {
				otherSummaries = append(otherSummaries, "⏰ 超时")
			}
		}
	}

	// fetch 工具特殊显示
	if toolName == "fetch" && fetchURL != "" {
		summaries = []string{fmt.Sprintf("🔗 %s", r.shortenURL(fetchURL))}
		summaries = append(summaries, otherSummaries...)
		return strings.Join(summaries, " · ")
	}

	summaries = append(summaries, otherSummaries...)
	if len(summaries) == 0 {
		return ""
	}
	return strings.Join(summaries, " · ")
}

// formatArguments 格式化参数显示
func (r *ToolRenderer) formatArguments(args, toolName string) string {
	// 尝试解析 JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(args), &data); err != nil {
		// 不是 JSON，返回原始
		maxLen := 300
		if len(args) > maxLen {
			return args[:maxLen] + "..."
		}
		return args
	}

	// 根据工具类型格式化参数
	switch toolName {
	case "fetch":
		// fetch 只显示 URL
		if url, ok := data["url"].(string); ok {
			return fmt.Sprintf(`{"url": "%s"}`, r.shortenURL(url))
		}
	}

	// 默认：截断长参数
	maxLen := 300
	if len(args) > maxLen {
		return args[:maxLen] + "..."
	}
	return args
}

// shortenPath 缩短路径显示
func (r *ToolRenderer) shortenPath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	// 尝试保留文件名
	base := filepath.Base(path)
	if len(base) >= maxLen-3 {
		// 文件名本身就太长，只保留文件名
		if len(base) > maxLen {
			return "..." + base[len(base)-(maxLen-3):]
		}
		return base
	}
	// 保留开头和结尾
	return path[:maxLen/2] + "..." + path[len(path)-maxLen/3:]
}

// shortenURL 缩短URL显示
func (r *ToolRenderer) shortenURL(url string) string {
	maxLen := 50
	if len(url) <= maxLen {
		return url
	}
	// 保留协议和域名
	if strings.HasPrefix(url, "http://") {
		return url[:7] + "..." + url[len(url)-(maxLen-10):]
	}
	if strings.HasPrefix(url, "https://") {
		return url[:8] + "..." + url[len(url)-(maxLen-11):]
	}
	return r.shortenString(url, maxLen)
}

// shortenString 通用字符串缩短
func (r *ToolRenderer) shortenString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen/2] + "..." + s[len(s)-maxLen/3:]
}

// formatBytes 格式化字节数
func (r *ToolRenderer) formatBytes(bytes int) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// parseBytes 解析字节数字符串
func parseBytes(s string) (int, error) {
	var bytes int
	_, err := fmt.Sscanf(s, "%d", &bytes)
	return bytes, err
}

// SetMetadataOnlyTools 设置只显示元数据的工具列表
func (r *ToolRenderer) SetMetadataOnlyTools(tools map[string]bool) {
	r.metadataOnlyTools = tools
}

// AddMetadataOnlyTool 添加一个只显示元数据的工具
func (r *ToolRenderer) AddMetadataOnlyTool(toolName string) {
	r.metadataOnlyTools[toolName] = true
}

// RemoveMetadataOnlyTool 移除一个工具的元数据-only 模式
func (r *ToolRenderer) RemoveMetadataOnlyTool(toolName string) {
	delete(r.metadataOnlyTools, toolName)
}
