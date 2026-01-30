package tools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

var (
	GrepToolName = "grep"
)

// GrepToolParams contains parameters for the grep tool.
type GrepToolParams struct {
	Pattern    string   `json:"pattern" jsonschema:"description=The regex pattern to search for in file contents"`
	Files      []string `json:"files" jsonschema:"description=List of file paths to search in"`
	MaxMatches int      `json:"max_matches,omitempty" jsonschema:"description=Maximum number of matches to return (default: 100)"`
}

// GrepMatch represents a single grep result.
type GrepMatch struct {
	File    string
	Line    int
	Content string
}

// GrepToolFunc executes the grep search.
func GrepToolFunc(ctx context.Context, params GrepToolParams) (string, error) {
	if params.Pattern == "" {
		return "", errors.New("pattern is required")
	}

	re, err := regexp.Compile(params.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	maxMatches := params.MaxMatches
	if maxMatches <= 0 {
		maxMatches = 100
	}

	if len(params.Files) == 0 {
		return "", fmt.Errorf("files parameter is required")
	}

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
		return "No valid files to search", nil
	}

	var matches []GrepMatch
	for _, file := range absFiles {
		if len(matches) >= maxMatches {
			break
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		fileMatches, err := searchFile(file, re, maxMatches-len(matches))
		if err != nil {
			continue
		}
		matches = append(matches, fileMatches...)
	}

	if len(matches) == 0 {
		return fmt.Sprintf("No matches found for pattern '%s'", params.Pattern), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d matches for pattern '%s':\n\n", len(matches), params.Pattern))

	baseDir := "."
	if len(absFiles) > 0 {
		baseDir = filepath.Dir(absFiles[0])
		if len(absFiles) > 1 {
			baseDir = findCommonDir(absFiles)
		}
	}

	currentFile := ""
	for _, m := range matches {
		relPath, _ := filepath.Rel(baseDir, m.File)
		if relPath == "" {
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

	return sb.String(), nil
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

// GetGrepTool returns the grep tool.
func GetGrepTool() tool.InvokableTool {
	grepTool, err := utils.InferTool(
		"grep",
		"Search file contents using regular expressions. Returns matching lines with file paths and line numbers.",
		GrepToolFunc,
	)
	if err != nil {
		log.Fatal(err)
	}
	return grepTool
}
