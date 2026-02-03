package renderer

import "github.com/charmbracelet/lipgloss"

// Theme 主题样式配置
type Theme struct {
	User       lipgloss.Style
	Assistant  lipgloss.Style
	System     lipgloss.Style
	Thinking   lipgloss.Style
	ToolBorder lipgloss.Style
	Minimal    lipgloss.Style
	Compact    lipgloss.Style
	Result     lipgloss.Style
	Arguments  lipgloss.Style
}

// DefaultTheme 返回默认主题
func DefaultTheme() *Theme {
	return &Theme{
		User: lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")). // Cyan
			Bold(true),

		Assistant: lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")). // Yellow
			Bold(true),

		System: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")). // Gray
			Italic(true),

		Thinking: lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")). // Light gray
			Italic(true),

		ToolBorder: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")), // Dim gray

		Minimal: lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")),

		Compact: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),

		Result: lipgloss.NewStyle().
			Foreground(lipgloss.Color("153")),

		Arguments: lipgloss.NewStyle().
			Foreground(lipgloss.Color("215")),
	}
}

// Icons 图标配置
type Icons struct {
	Tool    string
	File    string
	Search  string
	Clock   string
	Success string
	Error   string
}

// DefaultIcons 返回默认图标
func DefaultIcons() *Icons {
	return &Icons{
		Tool:    "🔧",
		File:    "📄",
		Search:  "🔍",
		Clock:   "⏱",
		Success: "✅",
		Error:   "❌",
	}
}
