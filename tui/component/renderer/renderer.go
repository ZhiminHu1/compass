package renderer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// MessageRenderer 消息渲染器
type MessageRenderer struct {
	markdownRenderer *glamour.TermRenderer
	styles           *MessageStyles
	toolRenderer     ToolRendererInterface
	// getResultFunc func(string) (string, bool) // Removed in favor of toolResults
	toolResults   map[string]string // 内部维护索引
	renderedCache []string          // 已渲染消息的缓存
	viewportWidth int
}

// ToolRendererInterface 工具渲染器接口
type ToolRendererInterface interface {
	RenderToolCall(tc schema.ToolCall, index int, getResultFunc func(string) (string, bool), styles interface{}) string
}

// ToolStyles 工具渲染样式（需要与 tool_renderer.go 中的定义兼容）
type ToolStyles struct {
	Indent   lipgloss.Style
	Border   lipgloss.Style
	System   lipgloss.Style
	Tool     lipgloss.Style
	ToolName lipgloss.Style
}

// NewMessageRenderer 创建消息渲染器
func NewMessageRenderer(styles *MessageStyles) *MessageRenderer {
	if styles == nil {
		styles = DefaultMessageStyles()
	}

	// 初始化 Markdown 渲染器 (Dracula 主题)
	markdownRenderer, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("dracula"),
		glamour.WithWordWrap(0), // 禁用自动换行，由外部控制
	)
	return &MessageRenderer{
		markdownRenderer: markdownRenderer,
		styles:           styles,
		toolResults:      make(map[string]string),
		renderedCache:    make([]string, 0),
	}
}

// SetToolRenderer 设置工具渲染器
func (r *MessageRenderer) SetToolRenderer(renderer ToolRendererInterface) {
	r.toolRenderer = renderer
}

// IndexMessage 索引消息中的工具结果
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

// RenderMessages 渲染所有消息
func (r *MessageRenderer) RenderMessages(messages []adk.Message) string {
	if len(messages) == 0 {
		return "Welcome to the chat room!\nType a message and press Enter to send."
	}

	// 1. 检测是否发生回退（例如清空列表），如果是则重置缓存
	if len(messages) < len(r.renderedCache) {
		r.renderedCache = r.renderedCache[:0]
	}

	for i := len(r.renderedCache); i < len(messages)-1; i++ {
		rendered := r.RenderMessage(messages[i])
		r.renderedCache = append(r.renderedCache, rendered)
	}

	// 3. 拼接内容
	var sb strings.Builder

	// 添加缓存的历史消息
	for _, cached := range r.renderedCache {
		if cached != "" {
			sb.WriteString(cached)
			sb.WriteString("\n\n")
		}
	}

	// 渲染并添加当前最后一条消息 (不缓存，因为它可能还在变)
	if len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		renderedLast := r.RenderMessage(lastMsg)
		if renderedLast != "" {
			sb.WriteString(renderedLast)
		}
	}

	content := sb.String()

	// 4. 包装内容以适应宽度
	if r.viewportWidth > 0 {
		return lipgloss.NewStyle().Width(r.viewportWidth).Render(content)
	}
	return content
}

// RenderMessage 渲染单条消息
func (r *MessageRenderer) RenderMessage(msg adk.Message) string {
	switch msg.Role {
	case schema.User:
		return r.renderUserMessage(msg)
	case schema.Assistant:
		return r.renderAssistantMessage(msg)
	case schema.System:
		return r.renderSystemMessage(msg)
	}
	return ""
}

// renderMarkdown 渲染 Markdown 内容
func (r *MessageRenderer) renderMarkdown(content string) string {
	if r.markdownRenderer == nil {
		return content
	}
	rendered, err := r.markdownRenderer.Render(content)
	if err != nil {
		// 渲染失败，返回原始内容
		return content
	}
	// 去除首尾空白（glamour 会添加前后换行）
	return strings.TrimSpace(rendered)
}

// renderUserMessage 渲染用户消息
func (r *MessageRenderer) renderUserMessage(msg adk.Message) string {
	if msg.Content == "" {
		return ""
	}
	// 用户消息通常不需要 Markdown 渲染，保持原始文本
	return r.styles.User.Render("User:") + " " + msg.Content
}

// renderAssistantMessage 渲染助手消息
func (r *MessageRenderer) renderAssistantMessage(msg adk.Message) string {
	var parts []string

	// 渲染思考内容
	if msg.ReasoningContent != "" {
		header := r.styles.Thinking.Render("Thinking:")
		// 使用斜体样式渲染思考内容
		content := r.styles.Thinking.Render(msg.ReasoningContent)
		parts = append(parts, header+"\n"+content)
	}
	// 渲染文本内容（支持 Markdown）
	if msg.Content != "" {
		header := r.styles.Assistant.Render("Assistant:")
		renderedContent := r.renderMarkdown(msg.Content)
		parts = append(parts, header+"\n"+renderedContent)
	}

	// 渲染工具调用
	if len(msg.ToolCalls) > 0 {
		if msg.Content == "" {
			header := r.styles.Assistant.Render("Assistant:")
			parts = append(parts, header)
		}
		toolCallsRendered := r.renderToolCalls(msg.ToolCalls)
		parts = append(parts, toolCallsRendered)
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n")
}

// renderSystemMessage 渲染系统消息
func (r *MessageRenderer) renderSystemMessage(msg adk.Message) string {
	if msg.Content == "" {
		return ""
	}
	return r.styles.System.Render("System: " + msg.Content)
}

// renderToolCalls 渲染工具调用列表
func (r *MessageRenderer) renderToolCalls(toolCalls []schema.ToolCall) string {
	if len(toolCalls) == 0 {
		return ""
	}

	var parts []string
	for i, tc := range toolCalls {
		renderedCall := r.renderToolCall(tc, i+1)
		if renderedCall != "" {
			parts = append(parts, renderedCall)
		}
	}

	return strings.Join(parts, "\n")
}

// renderToolCall 渲染单个工具调用及结果
func (r *MessageRenderer) renderToolCall(tc schema.ToolCall, index int) string {
	// 使用工具渲染器
	if r.toolRenderer != nil {
		styles := &ToolStyles{
			Indent:   r.styles.Indent,
			Border:   r.styles.ToolBorder,
			System:   r.styles.System,
			Tool:     r.styles.Tool,
			ToolName: r.styles.ToolName,
		}
		// 使用内部索引查找结果
		getResult := func(id string) (string, bool) {
			res, ok := r.toolResults[id]
			return res, ok
		}
		return r.toolRenderer.RenderToolCall(tc, index, getResult, styles)
	}

	// 没有设置工具渲染器，返回简单提示
	return r.styles.Indent.Render(
		r.styles.ToolBorder.Render("┌─ ") +
			r.styles.ToolName.Render(fmt.Sprintf("Tool Call #%d: %s", index, tc.Function.Name)) +
			"\n" +
			r.styles.ToolBorder.Render("│ ") +
			r.styles.System.Render("(Tool renderer not configured)") +
			"\n" +
			r.styles.ToolBorder.Render("└─"),
	)
}
