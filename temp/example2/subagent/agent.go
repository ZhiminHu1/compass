package subagent

import (
	"context"
	"cowork-agent/temp/example2/providers"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

func NewBookRecommendAgent() adk.Agent {
	chatModel, err := providers.CreateChatModel(context.Background())
	//var toolInfo []*schema.ToolInfo
	//for _, baseTool := range toolList {
	//	info, err := baseTool.Info(context.Background())
	//	if err != nil {
	//		log.Println(err)
	//		continue
	//	}
	//	toolInfo = append(toolInfo, info)
	//}
	ctx := context.Background()
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "BookRecommender",
		Description: "An agent that can recommend books",
		Instruction: `You are an expert book recommender. If the user's request is ambiguous or lacks necessary details (like genre, length, or rating) to perform a search, you MUST use the "ask_for_clarification" tool to ask follow-up questions. Do NOT ask directly in text. Only when you have clear requirements, use the "search_book" tool to find books. Finally, present the results to the user.`,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				// 配备两个tool
				Tools: []tool.BaseTool{NewBookRecommender(), NewAskForClarificationTool()},
			},
		},
	})
	if err != nil {
		log.Printf("NewBookRecommendAgent err: %v\n", err)
	}
	return agent

}
