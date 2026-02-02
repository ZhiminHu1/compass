package renderer

import (
	"github.com/charmbracelet/lipgloss"
)

// MessageStyles 消息渲染样式配置
type MessageStyles struct {
	// 消息角色样式
	User      lipgloss.Style
	Assistant lipgloss.Style
	System    lipgloss.Style
	Tool      lipgloss.Style
	Thinking  lipgloss.Style // 新增 Thinking 样式

	// 工具调用样式
	ToolName   lipgloss.Style
	ToolBorder lipgloss.Style
	Indent     lipgloss.Style
}

// DefaultMessageStyles 返回默认消息样式配置
func DefaultMessageStyles() *MessageStyles {
	return &MessageStyles{
		User:       lipgloss.NewStyle().Foreground(lipgloss.Color("#7dcfff")).Bold(true),
		Assistant:  lipgloss.NewStyle().Foreground(lipgloss.Color("#bb9af7")).Bold(true),
		System:     lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Italic(true),
		Tool:       lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a")),
		Thinking:   lipgloss.NewStyle().Foreground(lipgloss.Color("#6272a4")).Italic(true), // 初始化 Thinking 样式 (灰色斜体)
		ToolName:   lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68")).Bold(true),
		ToolBorder: lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")).Faint(true),
		Indent:     lipgloss.NewStyle().PaddingLeft(2),
	}
}
