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

// ListDirParams defines parameters for listing directory contents.
type ListDirParams struct {
	Path      string `json:"path" jsonschema:"description=The directory path to list contents of (default: current directory)"`
	Recursive bool   `json:"recursive,omitempty" jsonschema:"description=Whether to list contents recursively"`
}

// listDescription is the detailed tool description for the AI
const listDescription = `List files and directories at a given path.

BEFORE USING:
- Use absolute paths when possible
- Check parent directory exists before creating new files

CAPABILITIES:
- List directory contents
- Show files and subdirectories
- Recursive listing support
- Directories marked with trailing "/"

PARAMETERS:
- path (optional): Directory path to list (default: current directory)
- recursive (optional): Include all subdirectories if true

OUTPUT FORMAT:
Returns a list of files and directories, one per line. Directories end with "/".

EXAMPLES:
- List current: {"path": "."}
- List recursive: {"path": "src", "recursive": true}`

// ListDirFunc lists the contents of a directory.
func ListDirFunc(ctx context.Context, params ListDirParams) (string, error) {
	path := params.Path
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return Error(fmt.Sprintf("invalid path: %v", err))
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return Error(fmt.Sprintf("directory not found: %v", err))
	}
	if !info.IsDir() {
		return Error("path is not a directory")
	}

	var results []string
	err = filepath.Walk(absPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if p == absPath {
			return nil
		}
		rel, _ := filepath.Rel(absPath, p)
		isDir := ""
		if info.IsDir() {
			isDir = "/"
		}
		results = append(results, fmt.Sprintf("%s%s", rel, isDir))
		if !params.Recursive && info.IsDir() && p != absPath {
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		return Error(fmt.Sprintf("failed to list directory: %v", err))
	}

	if len(results) == 0 {
		return Success("Directory is empty", &Metadata{FilePath: absPath})
	}

	return Success(strings.Join(results, "\n"), &Metadata{
		FilePath:   absPath,
		MatchCount: len(results),
	})
}

// GetListDirTool returns the list directory tool.
func GetListDirTool() tool.InvokableTool {
	t, err := utils.InferTool(ListToolName, listDescription, ListDirFunc)
	if err != nil {
		log.Fatal(err)
	}
	return t
}
