package tools

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// ReadFileParams defines parameters for reading a file.
type ReadFileParams struct {
	Path      string `json:"path" jsonschema:"description=The path of the file to read"`
	StartLine int    `json:"start_line,omitempty" jsonschema:"description=The starting line number (1-indexed) to read from"`
	EndLine   int    `json:"end_line,omitempty" jsonschema:"description=The ending line number (1-indexed) to read to"`
}

// viewDescription is the detailed tool description for the AI
const viewDescription = `Read the content of a file with optional line range support.

BEFORE USING:
- Verify the file path is correct
- Use list tool to find files first
- For large files, consider using start_line and end_line

CAPABILITIES:
- Read entire file or specific line ranges
- Supports text files
- 1-indexed line numbers

PARAMETERS:
- path (required): The path of the file to read
- start_line (optional): Starting line number (1-indexed, default: 1)
- end_line (optional): Ending line number (1-indexed, default: end of file)

OUTPUT FORMAT:
Returns the file content as plain text.

EXAMPLES:
- Read whole file: {"path": "main.go"}
- Read specific range: {"path": "main.go", "start_line": 1, "end_line": 50}`

// ReadFileFunc reads the content of a file.
func ReadFileFunc(ctx context.Context, params ReadFileParams) (string, error) {
	data, err := os.ReadFile(params.Path)
	if err != nil {
		return Error(fmt.Sprintf("file not found: %v", err))
	}

	lines := strings.Split(string(data), "\n")
	start := params.StartLine
	end := params.EndLine

	if start <= 0 {
		start = 1
	}
	if end <= 0 || end > len(lines) {
		end = len(lines)
	}
	if start > len(lines) {
		return Error(fmt.Sprintf("start line %d exceeds file length %d", start, len(lines)))
	}

	content := strings.Join(lines[start-1:end], "\n")

	absPath, _ := filepath.Abs(params.Path)
	return ReadFileSuccess(content, absPath, len(lines), len(data))
}

// GetReadFileTool returns the read file tool.
func GetReadFileTool() tool.InvokableTool {
	t, err := utils.InferTool(ViewToolName, viewDescription, ReadFileFunc)
	if err != nil {
		log.Fatal(err)
	}
	return t
}
