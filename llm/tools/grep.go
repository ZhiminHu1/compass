package tools

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

const (
	// GrepToolName is the name of the grep tool
	GrepToolName = "grep"

	// DefaultMaxMatches is the default maximum number of matches
	DefaultMaxMatches = 100
	// MaxMaxMatches is the maximum allowed matches
	MaxMaxMatches = 500
)

// GrepToolParams contains parameters for the grep tool.
type GrepToolParams struct {
	Pattern    string   `json:"pattern" jsonschema:"description=The regex pattern to search for in file contents"`
	Files      []string `json:"files" jsonschema:"description=List of file paths to search in"`
	MaxMatches int      `json:"max_matches,omitempty" jsonschema:"description=Maximum number of matches to return (default: 100)"`
}

// grepDescription is the detailed tool description for the AI
const grepDescription = `Search file contents using regular expressions to find specific patterns.

BEFORE USING:
- Use the glob tool to find files first if you don't know the exact paths
- For large codebases, consider limiting the search scope

CAPABILITIES:
- Search for text patterns across multiple files
- Supports full regular expression syntax
- Returns file path, line number, and matching content
- Case-sensitive by default (use (?i) flag for case-insensitive)

PARAMETERS:
- pattern (required): The regex pattern to search for
- files (required): List of file paths to search in
- max_matches (optional): Maximum number of matches (default: 100, max: 500)

OUTPUT FORMAT:
Returns matching lines with file paths and line numbers, grouped by file.

EXAMPLES:
- Find function definitions: {"pattern": "func\s+\w+\(", "files": ["*.go"]}
- Case-insensitive search: {"pattern": "(?i)error", "files": ["main.go"]}
- Find TODO comments: {"pattern": "TODO|FIXME", "files": ["*.go", "*.js"]}`

// GrepMatch represents a single grep result.
type GrepMatch struct {
	File    string
	Line    int
	Content string
}

// GrepToolFunc executes the grep search with structured response.
func GrepToolFunc(ctx context.Context, params GrepToolParams) (string, error) {
	if params.Pattern == "" {
		return Error("pattern parameter is required")
	}

	re, err := regexp.Compile(params.Pattern)
	if err != nil {
		return Error(fmt.Sprintf("invalid regex pattern: %v", err))
	}

	maxMatches := params.MaxMatches
	if maxMatches <= 0 {
		maxMatches = DefaultMaxMatches
	}
	if maxMatches > MaxMaxMatches {
		maxMatches = MaxMaxMatches
	}

	if len(params.Files) == 0 {
		return Error("files parameter is required")
	}

	// Convert to absolute paths and validate
	absFiles := make([]string, 0, len(params.Files))
	for _, f := range params.Files {
		absPath, err := filepath.Abs(f)
		if err != nil {
			continue
		}
		if info, err := os.Stat(absPath); err == nil && !info.IsDir() {
			absFiles = append(absFiles, absPath)
		}
	}

	if len(absFiles) == 0 {
		return Error("no valid files to search")
	}

	// Search files
	var matches []GrepMatch
	for _, file := range absFiles {
		if len(matches) >= maxMatches {
			break
		}

		select {
		case <-ctx.Done():
			return Partial("search cancelled", &Metadata{MatchCount: len(matches)})
		default:
			fileMatches, err := searchFile(file, re, maxMatches-len(matches))
			if err == nil {
				matches = append(matches, fileMatches...)
			}
		}
	}

	if len(matches) == 0 {
		return Success(fmt.Sprintf("No matches found for pattern '%s'", params.Pattern),
			&Metadata{MatchCount: 0})
	}

	// Format results
	var sb strings.Builder
	baseDir := findCommonDir(absFiles)
	currentFile := ""

	for _, m := range matches {
		relPath, _ := filepath.Rel(baseDir, m.File)
		if relPath == "." {
			relPath = filepath.Base(m.File)
		}

		if relPath != currentFile {
			if currentFile != "" {
				sb.WriteString("\n")
			}
			sb.WriteString(fmt.Sprintf("%s:\n", relPath))
			currentFile = relPath
		}
		sb.WriteString(fmt.Sprintf("  %4d: %s\n", m.Line, strings.TrimSpace(m.Content)))
	}

	if len(matches) >= maxMatches {
		sb.WriteString(fmt.Sprintf("\n... (showing first %d matches)\n", maxMatches))
	}

	var files []string
	for _, f := range absFiles {
		files = append(files, filepath.Base(f))
	}

	return Success(sb.String(), &Metadata{
		MatchCount: len(matches),
		Files:      files,
	})
}

// findCommonDir finds the common parent directory of multiple files.
func findCommonDir(files []string) string {
	if len(files) == 0 {
		return "."
	}
	if len(files) == 1 {
		return filepath.Dir(files[0])
	}

	common := filepath.Dir(files[0])
	for _, f := range files[1:] {
		dir := filepath.Dir(f)
		for !strings.HasPrefix(dir, common) && common != "." && common != "/" {
			common = filepath.Dir(common)
		}
	}
	return common
}

// searchFile searches a single file for regex matches.
func searchFile(path string, re *regexp.Regexp, limit int) ([]GrepMatch, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []GrepMatch
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() && len(matches) < limit {
		lineNum++
		line := scanner.Text()
		if re.MatchString(line) {
			matches = append(matches, GrepMatch{
				File:    path,
				Line:    lineNum,
				Content: line,
			})
		}
	}

	return matches, scanner.Err()
}

// GetGrepTool returns the grep tool with enhanced description.
func GetGrepTool() tool.InvokableTool {
	grepTool, err := utils.InferTool(
		GrepToolName,
		grepDescription,
		GrepToolFunc,
	)
	if err != nil {
		log.Fatal(err)
	}
	return grepTool
}
