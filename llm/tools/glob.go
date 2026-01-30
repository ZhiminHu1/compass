package tools

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

var (
	GlobToolName = "Glob"
)

// GlobToolParams contains parameters for the glob tool.
type GlobToolParams struct {
	Pattern string `json:"pattern" jsonschema:"description=The glob pattern to match files (e.g., *.go, **/*.json)"`
	Path    string `json:"path,omitempty" jsonschema:"description=The directory to search in (defaults to current working directory)"`
}

// GlobToolFunc executes the glob search.
func GlobToolFunc(_ context.Context, params *GlobToolParams) (string, error) {
	searchPath := params.Path
	if searchPath == "" {
		searchPath = "."
	}

	absPath, err := filepath.Abs(searchPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("directory not found: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory")
	}

	pattern := filepath.Join(absPath, params.Pattern)
	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return "", fmt.Errorf("glob matching failed: %w", err)
	}
	if len(matches) == 0 {
		return "No matches found", nil
	}

	var relPaths []string
	for _, match := range matches {
		rel, err := filepath.Rel(absPath, match)
		if err != nil {
			rel = match
		}
		relPaths = append(relPaths, rel)
	}

	return fmt.Sprintf("Found %d matching files:\n%s", len(relPaths), strings.Join(relPaths, "\n")), nil
}

// GetGlobTool returns the glob tool.
func GetGlobTool() tool.InvokableTool {
	globTool, err := utils.InferTool(
		"glob",
		"Match and list filesystem paths using wildcard patterns (e.g., *.py, **/*.json) within a specified directory.",
		GlobToolFunc,
	)
	if err != nil {
		log.Fatal(err)
	}
	return globTool
}
