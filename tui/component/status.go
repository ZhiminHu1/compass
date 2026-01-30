package component

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	if m.running {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View 渲染组件视图
func (m StatusModel) View() string {
	if m.running {
		return fmt.Sprintf("\n\n%s %s\n\n", m.spinner.View(), m.text)
	}
	return fmt.Sprintf("\n\n  %s\n\n", m.text)
}

// Start 启动 spinner
func (m *StatusModel) Start() tea.Cmd {
	m.running = true
	return m.spinner.Tick
}

// Stop 停止 spinner
func (m *StatusModel) Stop() {
	m.running = false
}

// SetText 设置状态文本
func (m *StatusModel) SetText(text string) {
	m.text = text
}

// SetWidth 设置组件宽度
func (m *StatusModel) SetWidth(width int) {
	m.width = width
}

// IsRunning 返回 spinner 是否在运行
func (m *StatusModel) IsRunning() bool {
	return m.running
}
