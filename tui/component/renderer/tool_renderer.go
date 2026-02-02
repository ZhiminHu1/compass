package renderer

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/cloudwego/eino/schema"
)

// ToolResult 工具结果结构（与 llm/tools/types.go 对应）
type ToolResult struct {
	Status   string    `json:"status"`
	Content  string    `json:"content"`
	Metadata *Metadata `json:"metadata,omitempty"`
	Tier     string    `json:"tier"`
	Icon     string    `json:"icon,omitempty"`
}

type Metadata struct {
	FilePath   string `json:"file_path,omitempty"`
	LineCount  int    `json:"line_count,omitempty"`
	ByteCount  int    `json:"byte_count,omitempty"`
	Command    string `json:"command,omitempty"`
	Duration   int64  `json:"duration,omitempty"`
	ExitCode   int    `json:"exit_code,omitempty"`
	Timeout    bool   `json:"timeout,omitempty"`
	MatchCount int    `json:"match_count,omitempty"`
	FileCount  int    `json:"file_count,omitempty"`
	Pattern    string `json:"pattern,omitempty"`
	URL        string `json:"url,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}

// ToolCallStyles 工具调用样式（重命名以避免与 renderer.go 中的 ToolStyles 冲突）
type ToolCallStyles struct {
	Border    lipgloss.Style
	ToolName  lipgloss.Style
	Arguments lipgloss.Style
	Result    lipgloss.Style
	Minimal   lipgloss.Style
	Compact   lipgloss.Style
}

// ToolRenderer 工具渲染器
type ToolRenderer struct {
	styles *ToolCallStyles
}

// NewToolRenderer 创建新的工具渲染器
func NewToolRenderer() *ToolRenderer {
	return &ToolRenderer{
		styles: defaultToolCallStyles(),
	}
}

// defaultToolCallStyles 默认样式
func defaultToolCallStyles() *ToolCallStyles {
	borderColor := lipgloss.Color("#565f89")

	return &ToolCallStyles{
		Border: lipgloss.NewStyle().
			Foreground(borderColor).
			Faint(true),
		ToolName: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e0af68")).
			Bold(true),
		Arguments: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7dcfff")),
		Result: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c0caf5")),
		Minimal: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a9b1d6")),
		Compact: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c0caf5")),
	}
}

// RenderToolCall 渲染工具调用（实现 ToolRendererInterface 接口）
func (r *ToolRenderer) RenderToolCall(tc schema.ToolCall, index int, getResultFunc func(string) (string, bool), styles interface{}) string {
	// 获取工具结果
	result, ok := getResultFunc(tc.ID)
	if !ok {
		return r.styles.Minimal.Render(fmt.Sprintf("│ 🔧 #%d: (%s) (no result)", index, tc.Function.Name))
	}

	// 尝试解析为 ToolResult
	var toolResult ToolResult
	if err := json.Unmarshal([]byte(result), &toolResult); err == nil {
		// 新格式：根据 Tier 渲染
		return r.renderByTier(&toolResult, index)
	}

	// 降级：如果无法解析，显示简化信息
	preview := shortenString(result, 100)
	return r.styles.Minimal.Render(fmt.Sprintf("│ 🔧 #%d: %s", index, preview))
}

// renderByTier 根据展示层级渲染
func (r *ToolRenderer) renderByTier(result *ToolResult, callNum int) string {
	switch result.Tier {
	case "minimal":
		return r.renderMinimal(result, callNum)
	case "compact":
		return r.renderCompact(result, callNum)
	default:
		return r.renderFull(result, callNum)
	}
}

// renderMinimal 最小化渲染（单行）
func (r *ToolRenderer) renderMinimal(result *ToolResult, callNum int) string {
	icon := result.Icon
	if icon == "" {
		icon = "🔧"
	}

	md := result.Metadata
	if md == nil {
		return r.styles.Minimal.Render(fmt.Sprintf("│ %s #%d ✅", icon, callNum))
	}

	var parts []string

	// 文件名
	if md.FilePath != "" {
		parts = append(parts, filepath.Base(md.FilePath))
	}

	// 关键指标
	if md.LineCount > 0 {
		parts = append(parts, fmt.Sprintf("%d行", md.LineCount))
	}
	if md.MatchCount > 0 {
		parts = append(parts, fmt.Sprintf("%d匹配", md.MatchCount))
	}
	if md.FileCount > 0 {
		parts = append(parts, fmt.Sprintf("%d文件", md.FileCount))
	}
	if md.ByteCount > 0 {
		parts = append(parts, formatBytes(md.ByteCount))
	}

	// 状态
	status := "✅"
	if result.Status == "error" {
		status = "❌"
	}

	summary := strings.Join(parts, " · ")
	line := fmt.Sprintf("│ %s #%d: %s %s", icon, callNum, summary, status)

	return r.styles.Minimal.Render(line)
}

// renderCompact 紧凑渲染（2-3行）
func (r *ToolRenderer) renderCompact(result *ToolResult, callNum int) string {
	icon := result.Icon
	if icon == "" {
		icon = "🔧"
	}

	md := result.Metadata
	var lines []string

	// 第1行：头部
	header := r.styles.Border.Render("┌─ ") +
		r.styles.Border.Render(fmt.Sprintf(" #%d", callNum))
	lines = append(lines, header)

	// 第2行：关键信息
	if md != nil {
		var info []string
		if md.Command != "" {
			info = append(info, shortenString(md.Command, 50))
		}
		if md.URL != "" {
			info = append(info, shortenURL(md.URL))
		}
		if md.FilePath != "" {
			info = append(info, filepath.Base(md.FilePath))
		}

		if len(info) > 0 {
			lines = append(lines, r.styles.Border.Render("│ ")+r.styles.Compact.Render(strings.Join(info, " · ")))
		}
	}

	// 第3行：状态和指标
	var metrics []string
	if md != nil {
		if md.Duration > 0 {
			d := time.Duration(md.Duration) * time.Millisecond
			metrics = append(metrics, fmt.Sprintf("⏱️ %v", d))
		}
		if md.ExitCode != 0 {
			metrics = append(metrics, fmt.Sprintf("❌ exit:%d", md.ExitCode))
		} else if md.ExitCode == 0 && md.Command != "" {
			metrics = append(metrics, "✅")
		}
		if md.StatusCode > 0 {
			if md.StatusCode == 200 {
				metrics = append(metrics, "✅ 200")
			} else {
				metrics = append(metrics, fmt.Sprintf("📊 %d", md.StatusCode))
			}
		}
	}

	if len(metrics) > 0 {
		lines = append(lines, r.styles.Border.Render("├─ ")+r.styles.Result.Render(strings.Join(metrics, " · ")))
	}

	lines = append(lines, r.styles.Border.Render("└─"))

	return strings.Join(lines, "\n")
}

// renderFull 完整渲染（传统盒子）
func (r *ToolRenderer) renderFull(result *ToolResult, callNum int) string {
	icon := result.Icon
	if icon == "" {
		icon = "🔧"
	}

	md := result.Metadata
	var lines []string

	// 头部
	header := r.styles.Border.Render("┌─ ") +
		r.styles.Border.Render(fmt.Sprintf(" Tool #%d", callNum))
	lines = append(lines, header)

	// Arguments摘要
	if md != nil && md.FilePath != "" {
		args := r.styles.Arguments.Render(fmt.Sprintf("📁 %s", filepath.Base(md.FilePath)))
		lines = append(lines, r.styles.Border.Render("│ ")+args)
	}

	// Result
	lines = append(lines, r.styles.Border.Render("├─ Result:"))

	// 元数据摘要
	if md != nil {
		summary := r.formatMetadataSummary(md)
		if summary != "" {
			lines = append(lines, r.styles.Border.Render("│  ")+r.styles.Result.Render(summary))
		}
	}

	// 内容预览
	if result.Content != "" {
		preview := shortenString(result.Content, 150)
		lines = append(lines, r.styles.Border.Render("│  ")+r.styles.Result.Render(preview))
	}

	lines = append(lines, r.styles.Border.Render("└─"))

	return strings.Join(lines, "\n")
}

// formatMetadataSummary 格式化元数据摘要
func (r *ToolRenderer) formatMetadataSummary(md *Metadata) string {
	var parts []string

	if md.FilePath != "" {
		parts = append(parts, "📄 "+filepath.Base(md.FilePath))
	}
	if md.LineCount > 0 {
		parts = append(parts, fmt.Sprintf("📏 %d 行", md.LineCount))
	}
	if md.ByteCount > 0 {
		parts = append(parts, "📦 "+formatBytes(md.ByteCount))
	}
	if md.MatchCount > 0 {
		parts = append(parts, fmt.Sprintf("🔍 %d 匹配", md.MatchCount))
	}
	if md.FileCount > 0 {
		parts = append(parts, fmt.Sprintf("📁 %d 文件", md.FileCount))
	}

	return strings.Join(parts, " · ")
}

func formatBytes(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func shortenString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func shortenURL(url string) string {
	if len(url) <= 40 {
		return url
	}
	parts := strings.SplitN(url, "//", 2)
	if len(parts) == 2 {
		domain := strings.SplitN(parts[1], "/", 2)
		if len(domain) == 2 {
			return parts[0] + "//" + domain[0] + "/..."
		}
	}
	return url[:40] + "..."
}

func shortenPath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return filepath.Base(path)
}
