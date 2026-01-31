# AGENTS.md

This file contains guidelines and commands for agentic coding agents working in this repository.

## Project Overview

This is a Go-based AI agent application called "cowork-agent" that provides a terminal UI for interacting with AI models. The project uses the Eino framework (ByteDance's Go-based LLM application framework) and includes:
- TUI interface built with Bubbletea and Lipgloss
- Agent runtime using Eino's ADK (Agent Development Kit)
- Pub/Sub system for message passing
- Various file and web tools
- Support for multiple AI providers (OpenAI, Gemini, etc.)

## Build & Development Commands

### Build Commands
```bash
# Build the main application
go build -o cowork-agent .

# Build with specific output
go build -o cowork-agent.exe .

# Build for different platforms
go build -o cowork-agent-linux .
go build -o cowork-agent-darwin .
```

### Test Commands
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test file
go test -v ./pubsub -run TestBrokerFlow

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

### Lint & Format Commands
```bash
# Format code
go fmt ./...

# Run vet for static analysis
go vet ./...

# Run golangci-lint (if installed)
golangci-lint run

# Run misspell for spelling
gomisspell -w .
```

### Run Commands
```bash
# Run the application
go run main.go

# Run with environment variables
.env go run main.go

# Run specific example
cd example_tutor && go run main.go
```

## Code Style Guidelines

### Import Organization
- Group imports in this order:
  1. Standard library packages
  2. External packages (github.com, etc.)
  3. Internal packages (cowork-agent/)
- Use blank lines between groups
- Avoid unused imports

```go
import (
    "context"
    "fmt"
    "log"
    
    "cowork-agent/llm/agent"
    "cowork-agent/tui/chat"
    
    tea "github.com/charmbracelet/bubbletea"
    "github.com/joho/godotenv"
)
```

### Naming Conventions
- **Package names**: lowercase, short, descriptive (e.g., `chat`, `pubsub`, `agent`)
- **Struct names**: PascalCase (e.g., `Runtime`, `Broker`, `Model`)
- **Interface names**: PascalCase, often ending with `-er` (e.g., `ConversationStore`)
- **Function names**: PascalCase for exported, camelCase for unexported
- **Variable names**: camelCase, descriptive but concise
- **Constants**: PascalCase for exported, camelCase for unexported

### Error Handling
- Always handle errors explicitly
- Use `fmt.Errorf` with `%w` for error wrapping
- Return errors as the last return value
- Use `log.Fatalf` for unrecoverable errors in main
- Log errors with context using `log.Printf`

```go
if err != nil {
    return fmt.Errorf("创建 ChatModel 失败: %w", err)
}

log.Printf("获取消息失败: %v", err)
```

### Context Usage
- Always pass context as first parameter
- Use `context.Background()` for root contexts
- Use `context.WithCancel()` for cancellable operations
- Propagate context through function calls

### Struct Design
- Use composition over inheritance
- Keep structs focused and cohesive
- Use embedded types when appropriate
- Provide clear field documentation

```go
// Runtime Agent 运行时
type Runtime struct {
    agent      adk.Agent
    runner     *adk.Runner
    store      ConversationStore
    broker     *pubsub.Broker[adk.Message]
    ctx        context.Context
    cancelFunc context.CancelFunc
}
```

### Concurrency Patterns
- Use goroutines for concurrent operations
- Protect shared state with mutexes
- Use channels for communication
- Follow the context cancellation pattern
- Prefer buffered channels when appropriate

### Testing Guidelines
- Write table-driven tests for multiple scenarios
- Use descriptive test names
- Test both success and error cases
- Use `t.Helper()` for helper functions
- Clean up resources in `defer` statements

```go
func TestBrokerFlow(t *testing.T) {
    // Test setup
    broker := NewBroker[string]()
    defer broker.Shutdown()
    
    // Test execution
    // ...
    
    // Assertions
    if msg != testMsg {
        t.Errorf("期望得到 %s, 实际得到 %s", testMsg, msg)
    }
}
```

### Documentation
- Use Go doc comments for exported types and functions
- Write comments in Chinese for Chinese audience
- Use `//` for inline comments, `/* */` for block comments
- Document error conditions and return values

```go
// NewRuntime 创建新的 Agent 运行时
func NewRuntime(ctx context.Context, chatModel model.ToolCallingChatModel, toolsList []tool.BaseTool) (*Runtime, error) {
    // 创建 TechTutor Agent
    agt, err := NewTechTutorAgent(ctx, &TechTutorConfig{
        ChatModel: chatModel,
        Tools:     toolsList,
    })
    if err != nil {
        return nil, fmt.Errorf("创建 Agent 失败: %w", err)
    }
    
    // 创建 Runner
    runner := adk.NewRunner(ctx, adk.RunnerConfig{
        Agent:           agt,
        EnableStreaming: false, // 非流式
    })
    
    return &Runtime{agent: agt, runner: runner}, nil
}
```

## Project Structure

```
cowork-agent/
├── cmd/                    # CLI commands (currently empty)
├── llm/                   # LLM-related code
│   ├── agent/             # Agent runtime and configuration
│   ├── providers/         # AI provider implementations
│   └── tools/             # Various tools (file, web, search, etc.)
├── tui/                   # Terminal UI components
│   ├── chat/              # Chat interface
│   ├── component/         # Reusable UI components
│   └── spinner/           # Loading spinner
├── pubsub/                # Publish/subscribe system
├── utils/                 # Utility functions
├── temp/                  # Temporary examples and experiments
├── data/                  # Data files (knowledge store, etc.)
├── go.mod                 # Go module definition
├── go.sum                 # Go dependency checksums
├── main.go                # Application entry point
└── eino_framework_study_notes.md  # Eino framework documentation
```

## Dependencies

### Core Dependencies
- **Eino Framework**: `github.com/cloudwego/eino` - Main LLM framework
- **Bubbletea**: `github.com/charmbracelet/bubbletea` - TUI framework
- **Lipgloss**: `github.com/charmbracelet/lipgloss` - Styling for TUI
- **Godotenv**: `github.com/joho/godotenv` - Environment variable loading

### AI Provider Dependencies
- **OpenAI**: `github.com/cloudwego/eino-ext/components/model/openai`
- **Gemini**: `github.com/cloudwego/eino-ext/components/model/gemini`
- **Cloudwego eino-examples**: `github.com/cloudwego/eino-examples`

### Tool Dependencies
- **Goquery**: `github.com/PuerkitoBio/goquery` - HTML parsing
- **HTML to Markdown**: `github.com/JohannesKaufmann/html-to-markdown`
- **File operations**: `github.com/bmatcuk/doublestar/v4`

## Development Notes

### Environment Variables
- Use `.env` file for configuration
- Load environment variables in `init()` function
- Never commit sensitive data to version control

### Error Handling Strategy
- Use structured error handling with wrapping
- Log errors with sufficient context
- Return errors to callers for proper handling
- Use panic only for truly unrecoverable errors

### Performance Considerations
- Use buffered channels for pub/sub
- Implement non-blocking publish for backpressure
- Use context cancellation for graceful shutdown
- Optimize memory allocations in hot paths

### Testing Strategy
- Unit tests for individual components
- Integration tests for system interactions
- Mock external dependencies
- Test both success and failure scenarios
- Use table-driven tests for multiple cases

## Code Review Checklist

- [ ] Imports are properly organized
- [ ] Error handling is comprehensive
- [ ] Context is properly propagated
- [ ] Concurrency is safely managed
- [ ] Tests cover both success and error cases
- [ ] Documentation is clear and accurate
- [ ] Code follows naming conventions
- [ ] No sensitive data in version control
- [ ] Build and tests pass locally

## Common Patterns

### Resource Cleanup
```go
func (r *Runtime) Close() {
    r.cancelFunc()
    r.broker.Shutdown()
}
```

### Channel Operations
```go
// Non-blocking publish
select {
case sub <- event:
default:
    // Channel full, skip message
}
```

### Context Usage
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
```

### Error Wrapping
```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```