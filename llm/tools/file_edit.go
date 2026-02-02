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

// EditFileParams defines parameters for editing a file using search and replace.
type EditFileParams struct {
	Path    string `json:"path" jsonschema:"description=The path of the file to edit"`
	Search  string `json:"search" jsonschema:"description=The string to search for (must be unique in the file)"`
	Replace string `json:"replace" jsonschema:"description=The string to replace with"`
}

// editDescription is the detailed tool description for the AI
const editDescription = `Edit a file by replacing a specific search string with a replacement string.

BEFORE USING:
- Use view tool to read the file first
- Ensure the search string is unique within the file
- Include enough context for uniqueness

CAPABILITIES:
- Search and replace within a file
- Replaces ALL occurrences of the search string
- Case-sensitive matching

PARAMETERS:
- path (required): The path of the file to edit
- search (required): The string to search for
- replace (required): The string to replace with

OUTPUT FORMAT:
Returns confirmation with the file path edited and replacement count.

EXAMPLES:
- Simple replace: {"path": "main.go", "search": "oldFunc", "replace": "newFunc"}
- Multi-line: {"path": "config.json", "search": "\"port\": 8080", "replace": "\"port\": 3000"}

WARNINGS:
- If search string appears multiple times, ALL occurrences will be replaced
- Search is case-sensitive
- Search must match exactly, including whitespace`

// EditFileFunc edits a file by replacing a string.
func EditFileFunc(ctx context.Context, params EditFileParams) (string, error) {
	data, err := os.ReadFile(params.Path)
	if err != nil {
		return Error(fmt.Sprintf("file not found: %v", err))
	}

	content := string(data)
	if !strings.Contains(content, params.Search) {
		return Error(fmt.Sprintf("search string not found in file: %s", params.Path))
	}

	newContent := strings.ReplaceAll(content, params.Search, params.Replace)
	err = os.WriteFile(params.Path, []byte(newContent), 0644)
	if err != nil {
		return Error(fmt.Sprintf("failed to write file: %v", err))
	}

	absPath, _ := filepath.Abs(params.Path)
	return EditFileSuccess(absPath, strings.Count(newContent, "\n")+1)
}

// GetEditFileTool returns the edit file tool.
func GetEditFileTool() tool.InvokableTool {
	t, err := utils.InferTool(EditToolName, editDescription, EditFileFunc)
	if err != nil {
		log.Fatal(err)
	}
	return t
}
