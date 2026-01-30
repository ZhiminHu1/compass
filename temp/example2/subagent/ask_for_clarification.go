package subagent

import (
	"context"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

type askForClarificationOptions struct {
	NewInput *string
}

func WithNewInput(input string) tool.Option {
	// 包装
	//consumed := false
	return tool.WrapImplSpecificOptFn(func(t *askForClarificationOptions) {
		//if !consumed {
		t.NewInput = &input
		//consumed = true
		//}
	})
}

type AskForClarificationInput struct {
	Question string `json:"question" jsonschema_description:"The specific question you want to ask the user to get the missing information"`
}

func NewAskForClarificationTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"ask_for_clarification",
		"Call this tool when the user's request is ambiguous or lacks the necessary information to proceed. Use it to ask a follow-up question to get the details you need, such as the book's genre, before you can use other tools effectively.",
		func(ctx context.Context, input *AskForClarificationInput, opts ...tool.Option) (output string, err error) {
			// get 获取实例
			o := tool.GetImplSpecificOptions[askForClarificationOptions](nil, opts...)
			if o.NewInput == nil {
				return "", compose.Interrupt(context.Background(), input.Question)
			}
			output = *o.NewInput
			o.NewInput = nil

			return output, nil
		})
	if err != nil {
		log.Fatal(err)
	}
	return t
}
