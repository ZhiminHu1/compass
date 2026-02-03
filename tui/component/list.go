package component

import (
	"cowork-agent/pubsub"
	"cowork-agent/tui/component/renderer"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cloudwego/eino/adk"
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
}

// NewListModel 创建新的消息列表组件
func NewListModel() ListModel {
	vp := viewport.New(30, 30)
	vp.SetContent(`Welcome to the chat room!Type a message and press Enter to send.`)

	// 创建消息渲染器（内部已包含默认样式和工具渲染逻辑）
	msgRenderer := renderer.NewMessageRenderer()

	return ListModel{
		viewport: vp,
		messages: make([]adk.Message, 0),
		renderer: msgRenderer,
		width:    30,
		height:   5,
		ready:    true,
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
			// 更新消息和索引
			m.messages = append(m.messages, msg.Payload)

			// 索引消息中的工具结果（如果是工具消息）
			m.renderer.IndexMessage(msg.Payload)

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
	// 直接使用 renderer 渲染，不再传递 findToolResult
	content := m.renderer.RenderMessages(m.messages)
	m.viewport.SetContent(content)
}
