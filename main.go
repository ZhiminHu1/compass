package main

import (
	"cowork-agent/tui/chat"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func init() {
	// Load .env file if exists
	_ = godotenv.Load()
}
func main() {

	// 初始化UI界面
	model := chat.InitialModel()
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)
	program.Run()
	//

}
