package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"cowork-agent/llm/agent"
	"cowork-agent/tui/chat"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func init() {
	// Load .env file if exists
	_ = godotenv.Load()
}

func main() {
	ctx := context.Background()

	// 初始化 Agent Runtime
	runtime, err := agent.SetupRuntime(ctx)
	if err != nil {
		log.Fatalf("初始化 Agent 失败: %v", err)
	}
	defer runtime.Close()

	// 初始化 UI 界面
	model := chat.InitialModel(runtime)
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	if _, err := program.Run(); err != nil {
		fmt.Printf("程序运行出错: %v\n", err)
		os.Exit(1)
	}
}
