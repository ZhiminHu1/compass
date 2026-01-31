package component

import (
	"fmt"
	"strings"

	"cowork-agent/pubsub"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// ListModel 封装消息列表组件
type ListModel struct {
	viewport viewport.Model
	messages []adk.Message
	width    int
	height   int
	ready    bool

	// toolResults stores the content of tool messages keyed by ToolCallID
	toolResults map[string]string

	// markdownRenderer 用于渲染 Markdown 内容
	markdownRenderer *glamour.TermRenderer

	// 样式
	userStyle       lipgloss.Style
	assistantStyle  lipgloss.Style
	systemStyle     lipgloss.Style
	toolStyle       lipgloss.Style
	toolNameStyle   lipgloss.Style
	toolBorderStyle lipgloss.Style
	indentStyle     lipgloss.Style
}

// NewListModel 创建新的消息列表组件
func NewListModel() ListModel {
	vp := viewport.New(30, 30)
	vp.SetContent(`Welcome to the chat room!Type a message and press Enter to send.`)

	// 初始化 Markdown 渲染器 (Dracula 主题)
	markdownRenderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dracula"),
		glamour.WithWordWrap(0), // 禁用自动换行，由 viewport 控制
	)
	if err != nil {
		// 如果 dracula 主题失败，回退到 auto
		markdownRenderer, _ = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(0),
		)
	}

	// 初始化示例数据
	//messages := []adk.Message{
	//	{
	//		Role:    schema.System,
	//		Content: "This is a TechTutor demo session.",
	//	},
	//	{
	//		Role:    schema.User,
	//		Content: "How do I search for 'golang'?",
	//	},
	//	{
	//		Role:    schema.Assistant,
	//		Content: "I will help you with that. Let me search for it.",
	//		ToolCalls: []schema.ToolCall{
	//			{
	//				ID: "call_search_01",
	//				Function: schema.FunctionCall{
	//					Name:      "web_search",
	//					Arguments: `{"query": "golang tutorial"}`,
	//				},
	//			},
	//		},
	//	},
	//	{
	//		Role:    schema.Assistant,
	//		Content: "I'll also check local docs.",
	//		ToolCalls: []schema.ToolCall{
	//			{
	//				ID: "call_search_02",
	//				Function: schema.FunctionCall{
	//					Name:      "search_knowledge",
	//					Arguments: `{"query": "golang basics"}`,
	//				},
	//			},
	//		},
	//	},
	//}

	// 模拟已完成的工具结果
	toolResults := make(map[string]string)
	toolResults["call_search_01"] = `
Found 10 results for "golang tutorial":
1. A Tour of Go - https://tour.golang.org
2. Go by Example - https://gobyexample.com
...`

	// 注意：call_search_02 没有结果，将显示为 "Executing..."

	return ListModel{
		viewport:         vp,
		messages:         make([]adk.Message, 0),
		toolResults:      toolResults,
		markdownRenderer: markdownRenderer,
		width:            30,
		height:           5,
		ready:            true,
		userStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff")).Bold(true),
		assistantStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#bb9af7")).Bold(true),
		systemStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Italic(true),
		toolStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a")),
		toolNameStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68")).Bold(true),
		toolBorderStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Faint(true),
		indentStyle:      lipgloss.NewStyle().PaddingLeft(2),
	}
}

// Init 初始化组件
func (m ListModel) Init() tea.Cmd {
	return nil
}

// Update 更新组件状态
func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		// 处理鼠标滚轮事件
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.viewport.ScrollUp(3)
		case tea.MouseButtonWheelDown:
			m.viewport.ScrollDown(3)
		}
	case pubsub.Event[adk.Message]:
		// 处理 Agent 消息事件（跳过 nil payload，如 FinishedEvent）
		if msg.Type != pubsub.FinishedEvent {
			m.messages = append(m.messages, msg.Payload)

			// 如果是 Tool 类型的消息，将其内容存入 map
			if msg.Payload.Role == schema.Tool && msg.Payload.ToolCallID != "" {
				m.toolResults[msg.Payload.ToolCallID] = msg.Payload.Content
			}

			m.updateViewportContent()
			m.viewport.GotoBottom()
		}
		return m, nil
	}

	// 更新 viewport
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View 渲染组件视图
func (m ListModel) View() string {
	if !m.ready {
		return "Initializing..."
	}
	return m.viewport.View()
}

// AddMessage 添加新消息并滚动到底部
func (m *ListModel) AddMessage(msg adk.Message) {
	m.messages = append(m.messages, msg)

	// 如果是 Tool 类型的消息，将其内容存入 map
	if msg.Role == schema.Tool && msg.ToolCallID != "" {
		m.toolResults[msg.ToolCallID] = msg.Content
	}

	m.updateViewportContent()
	m.viewport.GotoBottom()
}

// SetSize 设置组件尺寸
func (m *ListModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	// 确保高度至少为 1，防止负数或零
	if height < 1 {
		height = 1
	}

	m.viewport.Width = width
	m.viewport.Height = height
	m.ready = true

	// 更新内容宽度
	if len(m.messages) > 0 {
		m.updateViewportContent()
	}
	m.viewport.GotoBottom()
}

// updateViewportContent 更新 viewport 内容
func (m *ListModel) updateViewportContent() {
	content := m.renderMessages()
	m.viewport.SetContent(content)
}

// renderMessages 渲染所有消息
func (m *ListModel) renderMessages() string {
	if len(m.messages) == 0 {
		return "Welcome to the chat room!\nType a message and press Enter to send."
	}

	var renderedMessages []string
	for _, msg := range m.messages {
		rendered := m.renderMessage(msg)
		if rendered != "" {
			renderedMessages = append(renderedMessages, rendered)
		}
	}

	content := strings.Join(renderedMessages, "\n\n")
	// 包装内容以适应宽度
	return lipgloss.NewStyle().Width(m.viewport.Width).Render(content)
}

// renderMessage 渲染单条消息
func (m *ListModel) renderMessage(msg adk.Message) string {
	switch msg.Role {
	case schema.User:
		return m.renderUserMessage(msg)
	case schema.Assistant:
		return m.renderAssistantMessage(msg)
	case schema.System:
		return m.renderSystemMessage(msg)
	case schema.Tool:
		return m.renderToolMessage(msg)
	default:
		return m.systemStyle.Render("Unknown message type")
	}
}

// renderMarkdown 渲染 Markdown 内容
func (m *ListModel) renderMarkdown(content string) string {
	if m.markdownRenderer == nil {
		return content
	}
	rendered, err := m.markdownRenderer.Render(content)
	if err != nil {
		// 渲染失败，返回原始内容
		return content
	}
	// 去除首尾空白（glamour 会添加前后换行）
	return strings.TrimSpace(rendered)
}

// renderUserMessage 渲染用户消息
func (m *ListModel) renderUserMessage(msg adk.Message) string {
	if msg.Content == "" {
		return ""
	}
	// 用户消息通常不需要 Markdown 渲染，保持原始文本
	return m.userStyle.Render("User:") + " " + msg.Content
}

// renderAssistantMessage 渲染助手消息
func (m *ListModel) renderAssistantMessage(msg adk.Message) string {
	var parts []string

	// 渲染文本内容（支持 Markdown）
	if msg.Content != "" {
		header := m.assistantStyle.Render("Assistant:")
		renderedContent := m.renderMarkdown(msg.Content)
		parts = append(parts, header+"\n"+renderedContent)
	}

	// 渲染工具调用
	if len(msg.ToolCalls) > 0 {
		if msg.Content == "" {
			header := m.assistantStyle.Render("Assistant:")
			parts = append(parts, header)
		}
		toolCallsRendered := m.renderToolCalls(msg.ToolCalls)
		parts = append(parts, toolCallsRendered)
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n")
}

// renderSystemMessage 渲染系统消息
func (m *ListModel) renderSystemMessage(msg adk.Message) string {
	if msg.Content == "" {
		return ""
	}
	return m.systemStyle.Render("System: " + msg.Content)
}

// renderToolMessage 渲染工具响应消息
// 注意：现在工具响应会直接被 renderAssistantMessage 获取并渲染在调用下方
// 所以这里返回空字符串，避免重复渲染
func (m *ListModel) renderToolMessage(msg adk.Message) string {
	return ""
}

// renderToolCalls 渲染工具调用列表
func (m *ListModel) renderToolCalls(toolCalls []schema.ToolCall) string {
	if len(toolCalls) == 0 {
		return ""
	}

	var parts []string
	for i, tc := range toolCalls {
		renderedCall := m.renderToolCall(tc, i+1)
		if renderedCall != "" {
			parts = append(parts, renderedCall)
		}
	}

	return strings.Join(parts, "\n")
}

// renderToolCall 渲染单个工具调用及结果
func (m *ListModel) renderToolCall(tc schema.ToolCall, index int) string {
	var parts []string

	// 工具调用头部
	header := m.indentStyle.Render(
		m.toolBorderStyle.Render("┌─ ") +
			m.toolNameStyle.Render(fmt.Sprintf("Tool Call #%d: %s", index, tc.Function.Name)),
	)
	parts = append(parts, header)

	// 格式化参数
	if tc.Function.Arguments != "" {
		formattedArgs := m.formatArguments(tc.Function.Arguments)
		argsLine := m.indentStyle.Render(
			m.toolBorderStyle.Render("│ ") +
				m.systemStyle.Render("Arguments: ") +
				formattedArgs,
		)
		parts = append(parts, argsLine)
	}

	// 尝试从 map 中查找结果
	if result, ok := m.toolResults[tc.ID]; ok {
		// 渲染结果（工具结果也支持 Markdown）
		maxLen := 500
		displayResult := result
		if len(result) > maxLen {
			displayResult = result[:maxLen] + "..."
		}

		// 对工具结果也进行 Markdown 渲染
		renderedResult := m.renderMarkdown(displayResult)

		resultHeader := m.indentStyle.Render(m.toolBorderStyle.Render("├─ ") + m.toolStyle.Render("Result:"))
		parts = append(parts, resultHeader)

		resultBody := m.indentStyle.Render(
			m.toolBorderStyle.Render("│  ") + renderedResult,
		)
		parts = append(parts, resultBody)

		footer := m.indentStyle.Render(m.toolBorderStyle.Render("└─"))
		parts = append(parts, footer)

	} else {
		// 没有结果，显示正在执行
		statusLine := m.indentStyle.Render(
			m.toolBorderStyle.Render("│ ") +
				m.systemStyle.Render("Status: ") +
				"Executing...",
		)
		parts = append(parts, statusLine)

		footer := m.indentStyle.Render(m.toolBorderStyle.Render("└─"))
		parts = append(parts, footer)
	}

	return strings.Join(parts, "\n")
}

// formatArguments 格式化参数显示
func (m *ListModel) formatArguments(args string) string {
	// 参考 print-event.go 的风格，直接显示参数
	// 对于过长的参数进行截断以保持界面整洁
	maxLen := 300
	if len(args) > maxLen {
		return args[:maxLen] + "..."
	}
	return args
}

// Clear 清空消息列表
func (m *ListModel) Clear() {
	m.messages = make([]adk.Message, 0)
	// 清空 tool map
	m.toolResults = make(map[string]string)
	m.updateViewportContent()
}
