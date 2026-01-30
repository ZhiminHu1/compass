package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino-examples/adk/common/tool"
	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/callbacks"
	"github.com/coze-dev/cozeloop-go"
	"github.com/joho/godotenv"
)

func init() {
	// Load .env from root if possible, or assume env vars set
	_ = godotenv.Load("../.env")
	// Also try current dir just in case
	_ = godotenv.Load()
}
func main() {
	ctx := context.Background()
	cozeloopApiToken := os.Getenv("COZE_LOOP_API_TOKEN")
	cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID")

	var handlers []callbacks.Handler
	if cozeloopApiToken != "" && cozeloopWorkspaceID != "" {
		client, err := cozeloop.NewClient(
			cozeloop.WithAPIToken(cozeloopApiToken),
			cozeloop.WithWorkspaceID(cozeloopWorkspaceID),
		)
		if err != nil {
			panic(err)
		}
		defer func() {
			time.Sleep(5 * time.Second)
			client.Close(ctx)
		}()
		handlers = append(handlers, clc.NewLoopHandler(client))
	}
	callbacks.AppendGlobalHandlers(handlers...)

	a := NewTicketBookingAgent()
	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           a,
		CheckPointStore: store.NewInMemoryStore(),
	})
	iter := runner.Query(context.Background(), "book a ticket for Martin, to Beijing, on 2025-12-01, the phone number is 1234567. directly call tool. 使用中文回答", adk.WithCheckPointID("1"))
	var lastEvent *adk.AgentEvent
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatal(event.Err)
		}

		prints.Event(event)

		lastEvent = event
	}
	if lastEvent == nil {
		log.Fatal("last event is nil")
	}
	if lastEvent.Action == nil || lastEvent.Action.Interrupted == nil {
		log.Fatal("last event is not an interrupt event")
	}

	interruptID := lastEvent.Action.Interrupted.InterruptContexts[0].ID

	var apResult *tool.ApprovalResult

	for {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("You inout here: ")
		scanner.Scan()
		fmt.Println()

		nInput := scanner.Text()
		if strings.ToUpper(nInput) == "Y" {
			apResult = &tool.ApprovalResult{Approved: true}
			break
		} else if strings.ToUpper(nInput) == "N" {
			fmt.Print("Please provide a reason for denial: ")
			scanner.Scan()
			reason := scanner.Text()
			fmt.Println()
			apResult = &tool.ApprovalResult{Approved: false, DisapproveReason: &reason}
			break
		}
		fmt.Println("invalid input, please input Y or N")
	}
	iter, err := runner.ResumeWithParams(context.Background(), "1", &adk.ResumeParams{
		Targets: map[string]any{
			interruptID: apResult,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatal(event.Err)
		}
		prints.Event(event)
	}
}
