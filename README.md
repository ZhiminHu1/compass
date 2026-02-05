# CoWork Agent

[中文文档](./README_zh.md)


> A terminal AI assistant demo built with Go

## Description

A learning project to explore:
- CloudWeGo Eino framework
- Bubble Tea TUI development
- Agent + Tools pattern

## Features

- 🤖 **Smart Chat**: Integrated with GLM/OpenAI models
- 🛠️ **Enhanced Tools**:
  - **File Operations**: Read, Write, Edit, Delete, List, Glob, Grep
  - **Information Retrieval**: Web Search (DuckDuckGo), Content Fetching, Summarization
  - **System Control**: PowerShell/Bash Command Execution
- 🧠 **Knowledge Base**: Redis-based vector store for knowledge ingestion and retrieval
    > ⚠️ **Note**: The Knowledge Base module is currently under active optimization and refactoring. APIs and storage formats may change.
- 📡 **Event Driven**: Built-in PubSub system for asynchronous component communication
- 🖥️ **Terminal UI**: Bubble Tea-based interactive TUI with multiple components (Input, List, Status)

## Quick Start

```bash
cp .env.example .env
# Edit .env with your API key
go run main.go
```

### Configuration

```bash
# Use GLM (智谱 AI)
API_KEY=your_glm_api_key
BASE_URL=https://open.bigmodel.cn/api/paas/v4
MODEL=glm-4.7-flashx

# Or use OpenAI
API_KEY=your_openai_api_key
BASE_URL=https://api.openai.com/v1
MODEL=gpt-4
```

## Project Structure

```
cowork-agent/
├── main.go            # Entry point
├── llm/               # LLM core logic
│   ├── agent/         # Agent definition & runtime
│   ├── parser/        # Output parsers
│   ├── tools/         # Toolset (File, Search, Bash, Knowledge)
│   └── vector/        # Vector storage (Redis)
├── pubsub/            # PubSub event system
└── tui/               # Terminal User Interface
    ├── chat/          # Chat logic
    └── component/     # UI components (List, Edit, Status)
```

## References

- [CloudWeGo Eino](https://github.com/cloudwego/eino)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea)

## License

MIT
