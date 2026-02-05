# CoWork Agent

> 一个基于 Go 的终端 AI 助手 Demo

## 简介

这是一个用 Go 写的 AI 助手练习项目，主要学习目的：
- CloudWeGo Eino 框架的使用
- Bubble Tea TUI 界面开发
- Agent + Tools 模式的实现

## 功能

- 🤖 **智能对话**: 集成 GLM/OpenAI 等主流模型
- 🛠️ **工具增强**:
  - **文件操作**: 读写、编辑、删除、列出目录、Glob 匹配、Grep 搜索
  - **信息获取**: 网络搜索 (DuckDuckGo)、网页内容抓取、内容摘要
  - **系统控制**: 执行 PowerShell/Bash 命令
- 🧠 **知识库**: 基于 Redis 的向量存储，支持知识摄入与检索
- 📡 **事件驱动**: 内置 PubSub 系统，支持组件间异步通信
- 🖥️ **终端界面**: 基于 Bubble Tea 的交互式 TUI，包含多组件（输入、列表、状态栏）

## 快速运行

```bash
# 1. 复制配置文件
cp .env.example .env

# 2. 修改 .env，填入你的 API Key
# 至少需要配置：API_KEY、BASE_URL、MODEL

# 3. 运行
go run main.go
```

### 配置示例

```bash
# 使用智谱 GLM
API_KEY=你的GLM密钥
BASE_URL=https://open.bigmodel.cn/api/paas/v4
MODEL=glm-4.7-flashx

# 或使用 OpenAI
API_KEY=你的OpenAI密钥
BASE_URL=https://api.openai.com/v1
MODEL=gpt-4
```

## 项目结构

```
cowork-agent/
├── main.go            # 程序入口
├── llm/               # LLM 核心逻辑
│   ├── agent/         # Agent 定义与运行时
│   ├── parser/        # 输出解析器
│   ├── tools/         # 工具集 (File, Search, Bash, Knowledge)
│   └── vector/        # 向量存储实现 (Redis)
├── pubsub/            # 事件发布/订阅系统
└── tui/               # 终端交互界面
    ├── chat/          # 聊天主逻辑
    └── component/     # UI 组件 (List, Edit, Status)
```

## 参考学习

- [CloudWeGo Eino](https://github.com/cloudwego/eino)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea)

## License

MIT
