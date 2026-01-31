package chat

import (
	"context"

	"cowork-agent/llm/agent"
	"cowork-agent/pubsub"
	"cowork-agent/tui/component"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cloudwego/eino/adk"
)

// Model 聊天界面模型
type Model struct {
	list   component.ListModel
	edit   component.EditModel
	status component.StatusModel

	runtime *agent.Runtime
	sub     <-chan pubsub.Event[adk.Message]
	ctx     context.Context

	width  int
	height int
	err    error
}

// InitialModel 创建初始模型
func InitialModel(runtime *agent.Runtime) Model {
	ctx := context.Background()
	sub := runtime.Broker().Subscribe(ctx)

	return Model{
		list:    component.NewListModel(),
		edit:    component.NewEditModel(),
		status:  component.NewStatusModel(),
		runtime: runtime,
		sub:     sub,
		ctx:     ctx,
		width:   0,
		height:  0,
		err:     nil,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.list.Init(),
		m.edit.Init(),
		m.status.Init(),
		m.waitForAgentMessage(), // 订阅 Agent 消息
	)
}

// waitForAgentMessage 等待 Agent 消息的 Cmd
func (m Model) waitForAgentMessage() tea.Cmd {
	return func() tea.Msg {
		event := <-m.sub
		return event
	}
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
		// 调用 Agent（在 goroutine 中）
		go func() {
			_ = m.runtime.Run(msg.Value)
		}()

	case pubsub.Event[adk.Message]:
		// 继续等待下一条消息
		cmds = append(cmds, m.waitForAgentMessage())
		// list 和 status 会在下面透传处理

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
