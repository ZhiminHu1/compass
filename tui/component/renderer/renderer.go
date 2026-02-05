package renderer

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"cowork-agent/llm/tools"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// MessageRenderer 消息渲染器
type MessageRenderer struct {
	markdownRenderer *glamour.TermRenderer
	theme            *Theme
	icons            *Icons
	toolResults      map[string]string // toolCallID -> JSON string
	viewportWidth    int
}

// NewMessageRenderer 创建消息渲染器
func NewMessageRenderer() *MessageRenderer {
	markdownRenderer, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("dracula"),
		glamour.WithWordWrap(0),
	)
	return &MessageRenderer{
		markdownRenderer: markdownRenderer,
		theme:            DefaultTheme(),
		icons:            DefaultIcons(),
		toolResults:      make(map[string]string),
	}
}

// RenderMessages 渲染所有消息
func (r *MessageRenderer) RenderMessages(messages []adk.Message) string {
	if len(messages) == 0 {
		return "Welcome to the chat room!\nType a message and press Enter to send."
	}

	var lines []string
	for _, msg := range messages {
		rendered := r.RenderMessage(msg)
		if rendered != "" {
			lines = append(lines, rendered)
		}
	}

	content := strings.Join(lines, "\n\n")

	if r.viewportWidth > 0 {
		return lipgloss.NewStyle().Width(r.viewportWidth).Render(content)
	}
	return content
}

// RenderMessage 渲染单条消息
func (r *MessageRenderer) RenderMessage(msg adk.Message) string {
	switch msg.Role {
	case schema.User:
		return r.renderUser(msg)
	case schema.Assistant:
		return r.renderAssistant(msg)
	case schema.System:
		return r.renderSystem(msg)
	}
	return ""
}

// renderUser 渲染用户消息
func (r *MessageRenderer) renderUser(msg adk.Message) string {
	if msg.Content == "" {
		return ""
	}
	return r.theme.User.Render("User:") + " " + msg.Content
}

// renderAssistant 渲染助手消息
func (r *MessageRenderer) renderAssistant(msg adk.Message) string {
	var parts []string

	if msg.ReasoningContent != "" {
		header := r.theme.Thinking.Render("Thinking:")
		content := r.theme.Thinking.Render(msg.ReasoningContent)
		parts = append(parts, header+"\n"+content)
	}

	if msg.Content != "" {
		header := r.theme.Assistant.Render("Assistant:")
		renderedContent := r.renderMarkdown(msg.Content)
		parts = append(parts, header+"\n"+renderedContent)
	}

	if len(msg.ToolCalls) > 0 {
		if msg.Content == "" && msg.ReasoningContent == "" {
			parts = append(parts, r.theme.Assistant.Render("Assistant:"))
		}
		parts = append(parts, r.renderToolCalls(msg.ToolCalls))
	}

	return strings.Join(parts, "\n")
}

// renderSystem 渲染系统消息
func (r *MessageRenderer) renderSystem(msg adk.Message) string {
	if msg.Content == "" {
		return ""
	}
	return r.theme.System.Render("System: " + msg.Content)
}

// renderToolCalls 渲染工具调用列表
func (r *MessageRenderer) renderToolCalls(toolCalls []schema.ToolCall) string {
	var parts []string
	for i, tc := range toolCalls {
		rendered := r.renderToolCall(tc, i+1)
		if rendered != "" {
			parts = append(parts, rendered)
		}
	}
	return strings.Join(parts, "\n")
}

// renderToolCall 渲染单个工具调用
func (r *MessageRenderer) renderToolCall(tc schema.ToolCall, index int) string {
	resultJSON, ok := r.toolResults[tc.ID]
	if !ok {
		return r.theme.Minimal.Render(fmt.Sprintf("│ %s #%d: (%s:%s) (no result)\n",
			r.icons.Tool, index, tc.Function.Name, tc.Function.Arguments))
	}

	// 解析 ToolResult - 使用统一类型
	var result tools.ToolResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		preview := Truncate(resultJSON, 100)
		return r.theme.Minimal.Render(fmt.Sprintf("│ %s #%d: %s",
			r.icons.Tool, index, preview))
	}

	// 根据 Tier 渲染
	switch result.Tier {
	case tools.TierMinimal:
		return r.renderToolMinimal(&result, index)
	case tools.TierCompact:
		return r.renderToolCompact(&result, index)
	default:
		return r.renderToolFull(&result, index)
	}
}

// renderToolMinimal 最小化渲染（单行）
func (r *MessageRenderer) renderToolMinimal(result *tools.ToolResult, callNum int) string {
	icon := r.icons.Tool
	md := result.Metadata
	if md == nil {
		return r.theme.Minimal.Render(fmt.Sprintf("│ %s #%d ✅", icon, callNum))
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
		parts = append(parts, FormatBytes(md.ByteCount))
	}

	status := r.icons.Success
	if result.Status == tools.StatusError {
		status = r.icons.Error
	}

	summary := strings.Join(parts, " · ")
	line := fmt.Sprintf("│ %s #%d: %s %s", icon, callNum, summary, status)

	return r.theme.Minimal.Render(line)
}

// renderToolCompact 紧凑渲染（2-3行）
func (r *MessageRenderer) renderToolCompact(result *tools.ToolResult, callNum int) string {
	md := result.Metadata
	var lines []string

	// 第1行：头部
	header := r.theme.ToolBorder.Render("┌─ ") +
		r.theme.ToolBorder.Render(fmt.Sprintf(" #%d", callNum))
	lines = append(lines, header)

	// 第2行：关键信息
	if md != nil {
		var info []string
		if md.Command != "" {
			info = append(info, Truncate(md.Command, 50))
		}
		if md.URL != "" {
			info = append(info, ShortenURL(md.URL))
		}
		if md.FilePath != "" {
			info = append(info, filepath.Base(md.FilePath))
		}

		if len(info) > 0 {
			lines = append(lines,
				r.theme.ToolBorder.Render("│ ")+r.theme.Compact.Render(strings.Join(info, " · ")))
		}
	}

	// 第3行：状态和指标
	var metrics []string
	if md != nil {
		if md.Duration > 0 {
			if d := FormatDuration(md.Duration); d != "" {
				metrics = append(metrics, fmt.Sprintf("%s %s", r.icons.Clock, d))
			}
		}
		if md.ExitCode != 0 {
			metrics = append(metrics, fmt.Sprintf("%s exit:%d", r.icons.Error, md.ExitCode))
		} else if md.ExitCode == 0 && md.Command != "" {
			metrics = append(metrics, r.icons.Success)
		}
		if md.StatusCode > 0 {
			if md.StatusCode == 200 {
				metrics = append(metrics, fmt.Sprintf("%s 200", r.icons.Success))
			} else {
				metrics = append(metrics, fmt.Sprintf("📊 %d", md.StatusCode))
			}
		}
	}

	if len(metrics) > 0 {
		lines = append(lines,
			r.theme.ToolBorder.Render("├─ ")+r.theme.Result.Render(strings.Join(metrics, " · ")))
	}

	lines = append(lines, r.theme.ToolBorder.Render("└─"))

	return strings.Join(lines, "\n")
}

// renderToolFull 完整渲染（传统盒子）
func (r *MessageRenderer) renderToolFull(result *tools.ToolResult, callNum int) string {
	md := result.Metadata
	var lines []string

	// 头部
	header := r.theme.ToolBorder.Render("┌─ ") +
		r.theme.ToolBorder.Render(fmt.Sprintf(" Tool #%d", callNum))
	lines = append(lines, header)

	// Arguments摘要
	if md != nil && md.FilePath != "" {
		args := r.theme.Arguments.Render(fmt.Sprintf("📁 %s", filepath.Base(md.FilePath)))
		lines = append(lines, r.theme.ToolBorder.Render("│ ")+args)
	}

	// Result
	lines = append(lines, r.theme.ToolBorder.Render("├─ Result:"))

	// 元数据摘要
	if md != nil {
		summary := r.formatMetadataSummary(md)
		if summary != "" {
			lines = append(lines,
				r.theme.ToolBorder.Render("│  ")+r.theme.Result.Render(summary))
		}
	}

	// 内容预览
	if result.Content != "" {
		preview := Truncate(result.Content, 150)
		lines = append(lines,
			r.theme.ToolBorder.Render("│  ")+r.theme.Result.Render(preview))
	}

	lines = append(lines, r.theme.ToolBorder.Render("└─"))

	return strings.Join(lines, "\n")
}

// formatMetadataSummary 格式化元数据摘要
func (r *MessageRenderer) formatMetadataSummary(md *tools.Metadata) string {
	var parts []string

	if md.FilePath != "" {
		parts = append(parts, fmt.Sprintf("%s %s", r.icons.File, filepath.Base(md.FilePath)))
	}
	if md.LineCount > 0 {
		parts = append(parts, fmt.Sprintf("📏 %d 行", md.LineCount))
	}
	if md.ByteCount > 0 {
		parts = append(parts, fmt.Sprintf("📦 %s", FormatBytes(md.ByteCount)))
	}
	if md.MatchCount > 0 {
		parts = append(parts, fmt.Sprintf("%s %d 匹配", r.icons.Search, md.MatchCount))
	}
	if md.FileCount > 0 {
		parts = append(parts, fmt.Sprintf("📁 %d 文件", md.FileCount))
	}

	return strings.Join(parts, " · ")
}

// renderMarkdown 渲染 Markdown 内容
func (r *MessageRenderer) renderMarkdown(content string) string {
	if r.markdownRenderer == nil {
		return content
	}
	rendered, err := r.markdownRenderer.Render(content)
	if err != nil {
		return content
	}
	return strings.TrimSpace(rendered)
}

// IndexMessage 索引工具结果
func (r *MessageRenderer) IndexMessage(msg adk.Message) {
	if msg.Role == schema.Tool && msg.ToolCallID != "" {
		r.toolResults[msg.ToolCallID] = msg.Content
	}
}

// ClearIndex 清空工具结果索引
func (r *MessageRenderer) ClearIndex() {
	r.toolResults = make(map[string]string)
}

// SetViewportWidth 设置视口宽度
func (r *MessageRenderer) SetViewportWidth(width int) {
	r.viewportWidth = width
}
