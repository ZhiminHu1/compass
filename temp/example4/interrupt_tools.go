package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

// SaveKnowledgeContext 定义中断上下文，用于传递需要保存的内容
// 必须是可序列化的结构体，以便 gob 编码
type SaveKnowledgeContext struct {
	Question string // 询问问题
	Markdown string // Markdown 内容
}

func init() {
	// 注册自定义类型到 gob，使其可以在 checkpoint 中序列化
	gob.Register(SaveKnowledgeContext{})
}

// askToSaveKnowledgeOptions 定义工具选项
type askToSaveKnowledgeOptions struct {
	UserChoice *string // 用户选择: "yes" 或 "no"
}

// WithSaveChoice 设置用户选择（用于 Resume 时传递用户输入）
func WithSaveChoice(choice string) tool.Option {
	consume := false
	return tool.WrapImplSpecificOptFn(func(o *askToSaveKnowledgeOptions) {
		if !consume {
			o.UserChoice = &choice
			consume = true
		}
	})
}

// AskToSaveKnowledgeInput 定义输入参数
type AskToSaveKnowledgeInput struct {
	// MarkdownContent 是要保存的 Markdown 内容
	MarkdownContent string `json:"markdown_content" jsonschema:"description=要保存到知识库的 Markdown 格式研究报告内容"`
}

// NewAskToSaveKnowledgeTool 创建一个工具，用于询问用户是否保存研究结果到知识库
// 工作流程：
// 1. Agent 在研究完成后调用此工具，传入 Markdown 内容
// 2. 工具触发 Interrupt，暂停 Agent 执行
// 3. 主程序捕获中断事件，向用户展示内容并询问是否保存
// 4. 用户输入后，主程序调用 runner.Resume 并传入用户选择
// 5. 工具获取用户选择，如果是 "yes" 则返回确认信息，Agent 将在最终回复中告知用户
func NewAskToSaveKnowledgeTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"ask_to_save_knowledge",
		"在研究完成后调用此工具，将生成的 Markdown 内容展示给用户，并询问是否保存到知识库。只有当用户确认后，内容才会被保存。",
		func(ctx context.Context, input *AskToSaveKnowledgeInput, opts ...tool.Option) (output string, err error) {
			// 获取工具选项
			o := tool.GetImplSpecificOptions[askToSaveKnowledgeOptions](nil, opts...)

			// 如果没有用户选择，说明是第一次调用，触发中断
			if o.UserChoice == nil {
				// 触发中断，暂停 Agent 执行
				// input.MarkdownContent 会被传递到中断事件中，供主程序显示给用户
				return "", compose.Interrupt(
					context.Background(),
					// 使用可序列化的结构体，而不是 map[string]interface{}
					SaveKnowledgeContext{
						Question: "是否保存到知识库？",
						Markdown: input.MarkdownContent,
					},
				)
			}

			// 用户已做出选择（通过 Resume 传入）
			choice := *o.UserChoice
			o.UserChoice = nil // 重置

			if choice == "yes" || choice == "y" {
				return "用户已确认保存。研究报告将被保存到知识库。", nil
			}

			return "用户选择不保存。研究结果未被保存。", nil
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	return t
}

// NewListKnowledgeTool 创建列出知识库内容的工具
func NewListKnowledgeTool() tool.InvokableTool {
	t, err := utils.InferTool(
		"list_knowledge",
		"列出知识库中所有已保存的研究文档。",
		func(ctx context.Context, input struct{}) (string, error) {
			if globalVectorStore == nil {
				return "知识库未初始化。", nil
			}

			docs := globalVectorStore.ListDocuments()
			if len(docs) == 0 {
				return "知识库为空。", nil
			}

			return fmt.Sprintf("知识库共有 %d 个文档。", len(docs)), nil
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	return t
}

// NewClearKnowledgeTool 创建清空知识库的工具
func NewClearKnowledgeTool() tool.InvokableTool {
	t, err := utils.InferTool(
		"clear_knowledge",
		"清空知识库中的所有文档。此操作不可恢复。",
		func(ctx context.Context, input struct{}) (string, error) {
			if globalVectorStore == nil {
				return "知识库未初始化。", nil
			}

			if err := globalVectorStore.Clear(); err != nil {
				return "", fmt.Errorf("清空知识库失败: %w", err)
			}

			return "知识库已清空。", nil
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	return t
}
