package main

import (
	"context"
	tools2 "cowork-agent/llm/tools"
	"cowork-agent/temp/example2/providers"
	"cowork-agent/temp/example4/vectorstore"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// globalVectorStore 全局向量存储，供工具使用
var globalVectorStore *vectorstore.VectorStore

// NewResearchAgent creates the main agent that uses Knowledge Search + Search + Summary(Parallel)
func NewResearchAgent(ctx context.Context, vectorStore *vectorstore.VectorStore) adk.Agent {
	// 保存到全局变量，供工具使用
	globalVectorStore = vectorStore

	model, err := providers.CreateChatModel(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize knowledge tools with vector store
	tools2.InitKnowledgeTool(vectorStore)

	// 1. Knowledge Base Search Tool (优先使用)
	knowledgeTool := tools2.GetKnowledgeTool()
	// 2. Web Search Tool
	searchTool := tools2.GetSearchTool()
	// 3. Summary Tool (Agent-as-Tool)
	summaryTool := NewSummaryTool(ctx)
	// 4. Ask to Save Knowledge Tool (使用 Interrupt 机制)
	askToSaveTool := NewAskToSaveKnowledgeTool()
	// 5. Knowledge Management Tools
	listKnowledgeTool := NewListKnowledgeTool()
	clearKnowledgeTool := NewClearKnowledgeTool()

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ResearchAgent",
		Description: "A researcher that searches knowledge base, web, and summarizes web pages concurrently.",
		Instruction: `You are a web researcher specializing in information gathering with a local knowledge base.

WORKFLOW:
1. **FIRST**: Always try 'search_knowledge' tool to search the local knowledge base for relevant information.
   - The knowledge base contains previous research results and summaries
   - If the knowledge base has relevant content, use it to answer the user's question

2. **IF** knowledge base has insufficient or irrelevant results:
   - Use the 'web_search' tool to find relevant URLs for the user's query
   - After receiving search results, immediately identify the top 2-4 most relevant URLs

3. **CRITICAL - PARALLEL EXECUTION**: Call the 'summarize_url' tool for ALL selected URLs IN A SINGLE RESPONSE.
   - Format your response with multiple tool calls: one for each URL
   - Do NOT wait for one summary to complete before requesting the next
   - All summarize_url calls must be grouped together in the same message

4. **After ALL summaries are returned**, synthesize them into a comprehensive final report.

5. **🚨 MANDATORY - MUST CALL TOOL 🚨**:
   ⚠️ IF you performed NEW research using 'web_search' (not just querying knowledge base):
   
   YOU MUST CALL THE 'ask_to_save_knowledge' TOOL!
   
   ❌ WRONG - DO NOT DO THIS:
   "您希望我将这份详细的研究报告保存到知识库中吗？" (just asking in text)
   
   ✅ CORRECT - DO THIS:
   Call the tool: ask_to_save_knowledge
   With parameter: {markdown_content: "# Your Complete Research Report\n..."}
   
   Critical points:
   - You MUST call the tool - asking in text is NOT sufficient
   - Pass your COMPLETE research report in Markdown format
   - Include: query, search findings, URL summaries, your analysis
   - The tool itself will handle asking the user - you don't need to ask
   - DO NOT skip this step - it is MANDATORY after web research

KNOWLEDGE BASE TOOLS:
- search_knowledge: Search local knowledge base (USE THIS FIRST)
- list_knowledge: List all documents in knowledge base
- clear_knowledge: Clear the knowledge base
- ask_to_save_knowledge: ⚠️ MANDATORY to call after web research - saves your report

EXAMPLE WORKFLOW:
User: "研究2026年就业率"
1. Call: search_knowledge("2026年就业率") → No results
2. Call: web_search("2026年就业率") → Get URLs
3. Call: summarize_url(URL1), summarize_url(URL2), summarize_url(URL3) IN ONE MESSAGE
4. Receive all summaries, synthesize final report
5. ⚠️ MUST Call: ask_to_save_knowledge({markdown_content: "# 2026年就业率研究\n\n## 数据来源\n..."})
6. Wait for tool to handle user interaction

REMEMBER: Step 5 is NOT optional. If you did web research, you MUST call ask_to_save_knowledge.`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					knowledgeTool,
					searchTool,
					summaryTool,
					askToSaveTool,
					listKnowledgeTool,
					clearKnowledgeTool,
				},
				ToolCallMiddlewares: []compose.ToolMiddleware{ErrorHandler()},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	return agent
}
