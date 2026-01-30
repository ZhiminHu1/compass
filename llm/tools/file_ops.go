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

var (
	LSToolName     = "list"
	ViewToolName   = "read"
	WriteToolName  = "write"
	EditToolName   = "edit"
	DeleteToolName = "delete"
)

// ListDirParams defines parameters for listing directory contents.
type ListDirParams struct {
	Path      string `json:"path" jsonschema:"description=The directory path to list contents of."`
	Recursive bool   `json:"recursive,omitempty" jsonschema:"description=Whether to list contents recursively."`
}

// ListDirFunc lists the contents of a directory.
func ListDirFunc(ctx context.Context, params ListDirParams) (string, error) {
	path := params.Path
	if path == "" {
		path = "."
	}

	var results []string
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if p == path {
			return nil
		}
		rel, _ := filepath.Rel(path, p)
		isDir := ""
		if info.IsDir() {
			isDir = "/"
		}
		results = append(results, fmt.Sprintf("%s%s", rel, isDir))
		if !params.Recursive && info.IsDir() && p != path {
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "Directory is empty", nil
	}

	return strings.Join(results, "\n"), nil
}

// ReadFileParams defines parameters for reading a file.
type ReadFileParams struct {
	Path      string `json:"path" jsonschema:"description=The path of the file to read."`
	StartLine int    `json:"start_line,omitempty" jsonschema:"description=The starting line number (1-indexed) to read from."`
	EndLine   int    `json:"end_line,omitempty" jsonschema:"description=The ending line number (1-indexed) to read to."`
}

// ReadFileFunc reads the content of a file.
func ReadFileFunc(ctx context.Context, params ReadFileParams) (string, error) {
	data, err := os.ReadFile(params.Path)
	if err != nil {
		return "", err
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
		return "Start line is beyond file length", nil
	}

	return strings.Join(lines[start-1:end], "\n"), nil
}

// WriteFileParams defines parameters for writing to a file.
type WriteFileParams struct {
	Path    string `json:"path" jsonschema:"description=The path of the file to write to."`
	Content string `json:"content" jsonschema:"description=The content to write to the file."`
}

// WriteFileFunc writes content to a file.
func WriteFileFunc(ctx context.Context, params WriteFileParams) (string, error) {
	err := os.MkdirAll(filepath.Dir(params.Path), 0755)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(params.Path, []byte(params.Content), 0644)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully wrote to %s", params.Path), nil
}

// DeleteFileParams defines parameters for deleting a file.
type DeleteFileParams struct {
	Path string `json:"path" jsonschema:"description=The path of the file to delete."`
}

// DeleteFileFunc deletes a file.
func DeleteFileFunc(ctx context.Context, params DeleteFileParams) (string, error) {
	// Security check: don't allow deleting sensitive files
	base := filepath.Base(params.Path)
	if base == ".env" || base == ".git" {
		return "Error: Deleting sensitive configuration files is not allowed.", nil
	}

	err := os.Remove(params.Path)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully deleted %s", params.Path), nil
}

// EditFileParams defines parameters for editing a file using search and replace.
type EditFileParams struct {
	Path    string `json:"path" jsonschema:"description=The path of the file to edit."`
	Search  string `json:"search" jsonschema:"description=The string to search for."`
	Replace string `json:"replace" jsonschema:"description=The string to replace with."`
}

// EditFileFunc edits a file by replacing a string.
func EditFileFunc(ctx context.Context, params EditFileParams) (string, error) {
	data, err := os.ReadFile(params.Path)
	if err != nil {
		return "", err
	}

	content := string(data)
	if !strings.Contains(content, params.Search) {
		return fmt.Sprintf("Error: Search string not found in %s", params.Path), nil
	}

	newContent := strings.ReplaceAll(content, params.Search, params.Replace)
	err = os.WriteFile(params.Path, []byte(newContent), 0644)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully updated %s", params.Path), nil
}

func GetListDirTool() tool.InvokableTool {
	t, err := utils.InferTool("list_dir", "List files and directories in a given path.", ListDirFunc)
	if err != nil {
		log.Fatal(err)
	}
	return t
}

func GetReadFileTool() tool.InvokableTool {
	t, err := utils.InferTool("read_file", "Read the content of a file, with optional line range support.", ReadFileFunc)
	if err != nil {
		log.Fatal(err)
	}
	return t
}

func GetWriteFileTool() tool.InvokableTool {
	t, err := utils.InferTool("write_file", "Create or overwrite a file with given content.", WriteFileFunc)
	if err != nil {
		log.Fatal(err)
	}
	return t
}

func GetDeleteFileTool() tool.InvokableTool {
	t, err := utils.InferTool("delete_file", "Delete a specific file.", DeleteFileFunc)
	if err != nil {
		log.Fatal(err)
	}
	return t
}

func GetEditFileTool() tool.InvokableTool {
	t, err := utils.InferTool("edit_file", "Edit a file by replacing a specific search string with a replacement string.", EditFileFunc)
	if err != nil {
		log.Fatal(err)
	}
	return t
}
