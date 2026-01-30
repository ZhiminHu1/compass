package main

import (
	"context"
	"cowork-agent/utils"
	"fmt"
	"log"

	"github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino/adk"
	"github.com/joho/godotenv"
)

func init() {
	// Load .env file if exists
	_ = godotenv.Load()
}
func main() {
	ctx := context.Background()

	fmt.Println("Initializing Agent-as-Tool Example...")

	// Create the main agent (which internally creates the sub-agent tool)
	agent := NewTravelAgent(ctx)

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
		Agent:           agent,
		CheckPointStore: store.NewInMemoryStore(),
	})

	// Simulate a user request
	query := "我要去旧金山。英语的‘最近的地铁站在哪里’怎么说？"
	fmt.Printf("\nUser Query: %s\n\n", query)

	iter := runner.Query(ctx, query)
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatal(event.Err)
		}

		utils.PrintEvent(event)
	}
	fmt.Println("\nExample finished.")
}
