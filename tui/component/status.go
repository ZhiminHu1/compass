package component

import (
	"fmt"

	"compass/pubsub"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cloudwego/eino/adk"
)

// StatusModel 封装状态显示组件（spinner + 状态文本）
type StatusModel struct {
	spinner spinner.Model
	running bool
	text    string
	width   int
}

// NewStatusModel 创建新的状态组件
func NewStatusModel() StatusModel {
	s := spinner.New()
	s.Spinner = spinner.Jump
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return StatusModel{
		spinner: s,
		running: false,
		text:    "Ready",
		width:   0,
	}
}

// Init 初始化组件
func (m StatusModel) Init() tea.Cmd {
	// 不自动启动 spinner，等待外部控制
	return nil
}

// Update 更新组件状态
func (m StatusModel) Update(msg tea.Msg) (StatusModel, tea.Cmd) {
	switch msg := msg.(type) {
	case pubsub.Event[adk.Message]:
		switch msg.Type {
		case pubsub.CreatedEvent:
			// 用户发送消息，启动 spinner
			if !m.running {
				m.running = true
				m.text = "Processing..."
				return m, m.spinner.Tick
			}
		case pubsub.FinishedEvent:
			// Agent 完成，停止 spinner
			if m.running {
				m.running = false
				m.text = "Ready"
				return m, nil
			}
		}
	}

	// Spinner 动画帧更新
	if m.running {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View 渲染组件视图
func (m StatusModel) View() string {
	style := lipgloss.NewStyle().Padding(1, 0)
	content := m.text
	if m.running {
		content = fmt.Sprintf("%s %s", m.spinner.View(), m.text)
	}
	return style.Render(content)
}

// Start 启动 spinner
func (m StatusModel) Start() (StatusModel, tea.Cmd) {
	m.running = true
	m.text = "Processing..."
	return m, m.spinner.Tick
}

// Stop 停止 spinner
func (m StatusModel) Stop() {
	m.running = false
}

// SetText 设置状态文本
func (m StatusModel) SetText(text string) {
	m.text = text
}

// SetWidth 设置组件宽度
func (m StatusModel) SetWidth(width int) {
	m.width = width
}

// IsRunning 返回 spinner 是否在运行
func (m StatusModel) IsRunning() bool {
	return m.running
}
