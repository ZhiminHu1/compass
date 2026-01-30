package subagent

import (
	"context"
	"cowork-agent/temp/example1/providers"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func NewStockDataCollectionAgent(toolList []tool.BaseTool) adk.Agent {
	chatModel, err := providers.CreateChatModel(context.Background())
	var toolInfo []*schema.ToolInfo
	for _, baseTool := range toolList {
		info, err := baseTool.Info(context.Background())
		if err != nil {
			log.Println(err)
			continue
		}
		toolInfo = append(toolInfo, info)
	}
	chatModel, err = chatModel.WithTools(toolInfo)
	if err != nil {
		log.Println(err)
	}

	agent, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        "StockDataCollectionAgent",
		Description: "The Stock Data Collection Agent is designed to gather real-time and historical stock market data from various reliable sources. It provides comprehensive information including stock prices, trading volumes, market trends, and financial indicators to support investment analysis and decision-making.",
		Instruction: `You are a Stock Data Collection Agent. Your role is to:
			- Collect accurate and up-to-date stock market data from trusted sources.
			- Retrieve information such as stock prices, trading volumes, historical trends, and relevant financial indicators.
			- IMPORTANT: You MUST use the available tools (like web_search and fetch_web_content) to get real-time market data instead of relying on your internal knowledge.
			- CONCURRENCY: If you need to search for multiple stocks or fetch multiple web pages to fulfill a request, you SHOULD call multiple tools or the same tool multiple times IN PARALLEL within a single response.
			- Ensure data completeness and reliability.
			- Format the collected data clearly for further analysis or user queries.
			- Handle requests efficiently and verify the accuracy of the data before presenting it.
			- Maintain professionalism and clarity in communication.`,
		Model: chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               toolList,
				ToolCallMiddlewares: []compose.ToolMiddleware{ErrorHandler()},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	return agent
}
func NewNewsDataCollectionAgent(toolList []tool.BaseTool) adk.Agent {
	chatModel, err := providers.CreateChatModel(context.Background())
	var toolInfo []*schema.ToolInfo
	for _, baseTool := range toolList {
		info, err := baseTool.Info(context.Background())
		if err != nil {
			log.Println(err)
			continue
		}
		toolInfo = append(toolInfo, info)
	}
	chatModel, err = chatModel.WithTools(toolInfo)
	if err != nil {
		log.Println(err)
	}

	a, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        "NewsDataCollectionAgent",
		Description: "The News Data Collection Agent specializes in aggregating news articles and updates from multiple reputable news outlets. It focuses on gathering timely and relevant information across various topics to keep users informed and support data-driven insights.",
		Instruction: `You are a News Data Collection Agent. Your responsibilities include:

- Aggregating news articles and updates from diverse and credible news sources.
- IMPORTANT: You MUST use the available tools to search/fetch latest news to ensure the information is timely and accurate.
- CONCURRENCY: If a request involves multiple topics, entities, or requires cross-referencing multiple sources, you SHOULD output ALL necessary tool calls (web_search, fetch_web_content) IN PARALLEL in one single response to improve efficiency.
- Filtering and organizing news based on relevance, timeliness, and user interests.
- Providing summaries or full content as required.
- Ensuring the accuracy and authenticity of the collected news data.
- Presenting information in a clear, concise, and unbiased manner.
- Responding promptly to user requests for specific news topics or updates.`,
		Model: chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               toolList,
				ToolCallMiddlewares: []compose.ToolMiddleware{ErrorHandler()},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	return a
}

func NewSocialMediaInfoCollectionAgent(toolList []tool.BaseTool) adk.Agent {
	chatModel, err := providers.CreateChatModel(context.Background())
	var toolInfo []*schema.ToolInfo
	for _, baseTool := range toolList {
		info, err := baseTool.Info(context.Background())
		if err != nil {
			log.Println(err)
			continue
		}
		toolInfo = append(toolInfo, info)
	}
	chatModel, err = chatModel.WithTools(toolInfo)
	if err != nil {
		log.Println(err)
	}

	a, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        "SocialMediaInformationCollectionAgent",
		Description: "The Social Media Information Collection Agent is tasked with gathering data from various social media platforms. It collects user-generated content, trends, sentiments, and discussions to provide insights into public opinion and emerging topics.",
		Instruction: `You are a Social Media Information Collection Agent. Your tasks are to:

- Collect relevant and up-to-date information from multiple social media platforms.
- Monitor trends, user sentiments, and public discussions related to specified topics.
- CONCURRENCY: To gather comprehensive data efficiently, you SHOULD trigger multiple tools or multiple calls to the same tool (like web_search for different platforms/keywords) IN PARALLEL whenever possible.
- Ensure the data collected respects privacy and platform policies.
- Organize and summarize the information to highlight key insights.
- Provide clear and objective reports based on the social media data.
- Communicate findings in a user-friendly and professional manner.`,
		Model: chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               toolList,
				ToolCallMiddlewares: []compose.ToolMiddleware{ErrorHandler()},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	return a
}

// ErrorHandler 工具错误处理中间件
func ErrorHandler() compose.ToolMiddleware {
	return compose.ToolMiddleware{
		Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
			return func(ctx context.Context, in *compose.ToolInput) (*compose.ToolOutput, error) {
				output, err := next(ctx, in)
				if err != nil {
					// 提取核心错误信息
					errStr := err.Error()
					if idx := strings.Index(errStr, "err="); idx != -1 {
						coreErr := strings.TrimSpace(errStr[idx+4:])
						// 将错误转换为成功的工具结果
						return &compose.ToolOutput{
							Result: fmt.Sprintf("error! %s", coreErr),
						}, nil
					}
					// 如果没有找到 err=，返回原始错误
					return &compose.ToolOutput{
						Result: fmt.Sprintf("error! %s", errStr),
					}, nil
				}
				return output, nil
			}
		},
	}
}
