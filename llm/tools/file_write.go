package tools

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// WriteFileParams defines parameters for writing to a file.
type WriteFileParams struct {
	Path    string `json:"path" jsonschema:"description=The path of the file to write to"`
	Content string `json:"content" jsonschema:"description=The content to write to the file"`
}

// writeDescription is the detailed tool description for the AI
const writeDescription = `Create or overwrite a file with given content.

BEFORE USING:
- Use list tool to verify parent directory exists
- Be careful not to accidentally overwrite existing files

CAPABILITIES:
- Create new files
- Overwrite existing files completely
- Automatically creates parent directories

PARAMETERS:
- path (required): The path of the file to write to
- content (required): The content to write to the file

OUTPUT FORMAT:
Returns confirmation with the file path written.

EXAMPLES:
- Create file: {"path": "main.go", "content": "package main\n\nfunc main() {}"}
- Overwrite: {"path": "config.json", "content": "{\"key\": \"value\"}"}`

// WriteFileFunc writes content to a file.
func WriteFileFunc(ctx context.Context, params WriteFileParams) (string, error) {
	err := os.MkdirAll(filepath.Dir(params.Path), 0755)
	if err != nil {
		return Error(fmt.Sprintf("failed to create parent directories: %v", err))
	}

	err = os.WriteFile(params.Path, []byte(params.Content), 0644)
	if err != nil {
		return Error(fmt.Sprintf("failed to write file: %v", err))
	}

	absPath, _ := filepath.Abs(params.Path)
	return WriteFileSuccess(absPath, len(params.Content))
}

// GetWriteFileTool returns the write file tool.
func GetWriteFileTool() tool.InvokableTool {
	t, err := utils.InferTool(WriteToolName, writeDescription, WriteFileFunc)
	if err != nil {
		log.Fatal(err)
	}
	return t
}
