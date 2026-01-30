package component

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EditorSubmitMsg 自定义消息：用户提交输入
type EditorSubmitMsg struct {
	Value string
}

// EditModel 封装输入框组件
type EditModel struct {
	textarea textarea.Model
	width    int
}

// NewEditModel 创建新的输入框组件
func NewEditModel() EditModel {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "> "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(1)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	// 禁用换行，Enter 用于提交
	ta.KeyMap.InsertNewline.SetEnabled(false)

	return EditModel{
		textarea: ta,
		width:    30,
	}
}

// Init 初始化组件
func (m EditModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update 更新组件状态
func (m EditModel) Update(msg tea.Msg) (EditModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// 获取输入值并提交
			value := m.textarea.Value()
			if value != "" {
				m.textarea.Reset()
				// 发送自定义提交消息
				return m, func() tea.Msg {
					return EditorSubmitMsg{Value: value}
				}
			}
			return m, nil
		}
	}

	// 更新 textarea
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View 渲染组件视图
func (m *EditModel) View() string {
	return m.textarea.View()
}

// SetWidth 设置组件宽度
func (m *EditModel) SetWidth(width int) {
	m.width = width
	m.textarea.SetWidth(width)
}

// Focus 聚焦输入框
func (m *EditModel) Focus() tea.Cmd {
	return m.textarea.Focus()
}

// Blur 失焦输入框
func (m *EditModel) Blur() {
	m.textarea.Blur()
}

// Reset 清空输入
func (m *EditModel) Reset() {
	m.textarea.Reset()
}

// Height 返回组件高度
func (m *EditModel) Height() int {
	return m.textarea.Height()
}
