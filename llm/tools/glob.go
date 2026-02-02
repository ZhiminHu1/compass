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

const (
	// GlobToolName is the name of the glob tool
	GlobToolName = "glob"

	// DefaultMaxResults is the default maximum number of results
	DefaultMaxResults = 100
	// MaxMaxResults is the maximum allowed results
	MaxMaxResults = 1000
)

// GlobToolParams contains parameters for the glob tool.
type GlobToolParams struct {
	Pattern    string `json:"pattern" jsonschema:"description=The glob pattern to match files (e.g., *.go, **/*.json)"`
	Path       string `json:"path,omitempty" jsonschema:"description=The directory to search in (defaults to current working directory)"`
	MaxResults int    `json:"max_results,omitempty" jsonschema:"description=Maximum number of results to return (default: 100, max: 1000)"`
}

// globDescription is the detailed tool description for the AI
const globDescription = `Match and list filesystem paths using wildcard patterns.

BEFORE USING:
- Use this tool to discover files before reading or editing them
- Check the current directory structure first

CAPABILITIES:
- Match files by name pattern (*.go, *.md, etc.)
- Recursive search with ** pattern
- Search in specific directories
- Returns relative paths from the search directory

SUPPORTED PATTERNS:
- *.go           - Match Go files in current directory
- **/*.go         - Match Go files recursively
- test_*.go       - Match Go files starting with test_
- **/*.{go,md}    - Match .go and .md files recursively

PARAMETERS:
- pattern (required): The glob pattern to match files
- path (optional): Directory to search in (default: current directory)
- max_results (optional): Maximum results (default: 100, max: 1000)

OUTPUT FORMAT:
Returns a list of matching file paths, one per line.

EXAMPLES:
- Find Go files: {"pattern": "*.go"}
- Find all Markdown: {"pattern": "**/*.md"}
- Find test files: {"pattern": "**/*_test.go"}`

// GlobToolFunc executes the glob search with structured response.
func GlobToolFunc(_ context.Context, params GlobToolParams) (string, error) {
	searchPath := params.Path
	if searchPath == "" {
		searchPath = "."
	}

	absPath, err := filepath.Abs(searchPath)
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

	pattern := filepath.Join(absPath, params.Pattern)
	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return Error(fmt.Sprintf("glob matching failed: %v", err))
	}

	if len(matches) == 0 {
		return GlobSuccess("No matches found", 0)
	}

	// Apply max results limit
	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = DefaultMaxResults
	}
	if maxResults > MaxMaxResults {
		maxResults = MaxMaxResults
	}

	truncated := false
	if len(matches) > maxResults {
		matches = matches[:maxResults]
		truncated = true
	}

	var relPaths []string
	for _, match := range matches {
		rel, err := filepath.Rel(absPath, match)
		if err != nil {
			rel = match
		}
		relPaths = append(relPaths, rel)
	}

	content := strings.Join(relPaths, "\n")
	if truncated {
		content += fmt.Sprintf("\n\n... (showing first %d of %d matches)",
			maxResults, maxResults)
	}

	return GlobSuccess(content, len(matches))
}

// GetGlobTool returns the glob tool with enhanced description.
func GetGlobTool() tool.InvokableTool {
	globTool, err := utils.InferTool(
		GlobToolName,
		globDescription,
		GlobToolFunc,
	)
	if err != nil {
		log.Fatal(err)
	}
	return globTool
}
