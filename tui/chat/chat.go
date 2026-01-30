package chat

// A simple program demonstrating the text area component from the Bubbles
// component library.

import (
	"cowork-agent/tui/component"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type Model struct {
	list   component.ListModel
	edit   component.EditModel
	status component.StatusModel

	width  int
	height int
	err    error
}

func InitialModel() Model {
	return Model{
		list:   component.NewListModel(),
		edit:   component.NewEditModel(),
		status: component.NewStatusModel(),
		width:  0,
		height: 0,
		err:    nil,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.list.Init(),
		m.edit.Init(),
		m.status.Init(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// 计算各组件高度
		statusHeight := lipgloss.Height(m.status.View())
		editHeight := m.edit.Height()
		listHeight := m.height - statusHeight - editHeight

		// 更新各组件尺寸
		m.list.SetSize(m.width, listHeight)
		m.edit.SetWidth(m.width)
		m.status.SetWidth(m.width)

	case component.EditorSubmitMsg:
		// 处理用户输入提交
		userMsg := createUserMsg(msg.Value)
		m.list.AddMessage(userMsg)

		// 启动 status spinner
		if !m.status.IsRunning() {
			m.status.SetText("Processing...")
			cmd := m.status.Start()
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	}

	// 更新各子组件
	var cmd tea.Cmd

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	m.edit, cmd = m.edit.Update(msg)
	cmds = append(cmds, cmd)

	m.status, cmd = m.status.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.list.View(),
		m.status.View(),
		m.edit.View(),
	)
}

func createUserMsg(value string) adk.Message {
	return &schema.Message{
		Role:    schema.User,
		Content: value,
	}
}
