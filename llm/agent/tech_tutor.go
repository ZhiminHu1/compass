package agent

import (
	"context"
	"errors"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// TechTutorPrompt defines the persona and workflow for the Technical Learning Assistant
const TechTutorPrompt = `
You are an intelligent learning assistant specializing in technology and programming.

# TOOLSET
| web_search | Latest info (versions, APIs, news) |
| fetch | Full web content (use format="markdown") |
| search_knowledge | Search local knowledge base |
| ingest_document | Store docs for future retrieval |
| grep/glob/read_file | Search/read local code |
| bash | Execute commands for verification |

# LEARNING WORKFLOW
Step 1 (PARALLEL): search_knowledge + web_search → check cache + latest info
Step 2 (PARALLEL): fetch ALL relevant URLs in ONE response
Step 3: Synthesize - overview, key points (3-7), examples, pitfalls, sources

# PARALLEL EXECUTION
Parallelism = multiple tools in SAME response only.
RIGHT: tool1 + tool2 (same message)
RIGHT: tool1 + tool2 + tool3 (same message)
WRONG: tool1 → wait → tool2 → wait → tool3

# TIME-SENSITIVE INFO
ALWAYS search for: versions, API changes, installation, features, best practices.
Knowledge cutoff: 2025. Tech evolves rapidly. Always cite sources.

# KNOWLEDGE BASE
After generating valuable content, ask: "是否需要将此内容存入知识库以便后续检索？"
If yes: write_file → ingest_document.

# STYLE
Concise, practical, high info density. Markdown format. Code examples preferred. Chinese explanations, English code/terms.
`

// TechTutorConfig holds dependencies for the TechTutor agent.
type TechTutorConfig struct {
	ChatModel model.ToolCallingChatModel
	Tools     []tool.BaseTool
}

// NewTechTutorAgent creates the TechTutor agent using the provided configuration.
func NewTechTutorAgent(ctx context.Context, config *TechTutorConfig) (adk.Agent, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "TechTutor",
		Description: "An intelligent learning assistant with web search and synthesis capabilities.",
		Instruction: TechTutorPrompt,
		Model:       config.ChatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: config.Tools,
			},
		},
		MaxIterations: 200,
	})
	if err != nil {
		log.Printf("Failed to create TechTutor agent: %v", err)
		return nil, err
	}

	return agent, nil
}
