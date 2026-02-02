package component

import (
	"cowork-agent/pubsub"
	"cowork-agent/tui/component/renderer"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// ListModel 封装消息列表组件
// 负责消息存储和 viewport 管理，渲染逻辑委托给 MessageRenderer
type ListModel struct {
	viewport viewport.Model
	messages []adk.Message
	width    int
	height   int
	ready    bool

	// renderer 消息渲染器
	renderer *renderer.MessageRenderer

	// toolRenderer 工具调用渲染器
	toolRenderer *ToolRenderer
}

// NewListModel 创建新的消息列表组件
func NewListModel() ListModel {
	vp := viewport.New(30, 30)
	vp.SetContent(`Welcome to the chat room!Type a message and press Enter to send.`)

	// 创建样式
	styles := renderer.DefaultMessageStyles()

	// 创建消息渲染器
	msgRenderer := renderer.NewMessageRenderer(styles)

	// 创建工具渲染器
	toolRend := NewToolRenderer()

	// 设置工具渲染器到消息渲染器
	msgRenderer.SetToolRenderer(toolRend)

	// 设置工具结果获取函数
	msgRenderer.SetToolResultsFunc(func(id string) (string, bool) {
		// 这里需要在运行时获取，暂时返回空
		return "", false
	})

	return ListModel{
		viewport:     vp,
		messages:     make([]adk.Message, 0),
		renderer:     msgRenderer,
		toolRenderer: toolRend,
		width:        30,
		height:       5,
		ready:        true,
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
		if msg.Type != pubsub.FinishedEvent {
			m.messages = append(m.messages, msg.Payload)
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

	// 更新渲染器宽度
	m.renderer.SetViewportWidth(width)

	// 更新内容
	if len(m.messages) > 0 {
		m.updateViewportContent()
	}
	m.viewport.GotoBottom()
}

// updateViewportContent 更新 viewport 内容
func (m *ListModel) updateViewportContent() {
	// 将 messages 传递给渲染器，渲染器从中提取工具结果
	m.renderer.SetToolResultsFunc(m.findToolResult(m.messages))
	content := m.renderer.RenderMessages(m.messages)
	m.viewport.SetContent(content)
}

// findToolResult 从消息列表中查找工具结果
func (m *ListModel) findToolResult(messages []adk.Message) func(string) (string, bool) {
	return func(toolCallID string) (string, bool) {
		for _, msg := range messages {
			if msg.Role == schema.Tool && msg.ToolCallID == toolCallID {
				return msg.Content, true
			}
		}
		return "", false
	}
}

// GetRenderer 获取消息渲染器（用于外部配置）
func (m *ListModel) GetRenderer() *renderer.MessageRenderer {
	return m.renderer
}

// GetToolRenderer 获取工具渲染器（用于外部配置）
func (m *ListModel) GetToolRenderer() *ToolRenderer {
	return m.toolRenderer
}

// Clear 清空消息列表
func (m *ListModel) Clear() {
	m.messages = make([]adk.Message, 0)
	m.updateViewportContent()
}

// GetMessages 获取所有消息
func (m *ListModel) GetMessages() []adk.Message {
	return m.messages
}
