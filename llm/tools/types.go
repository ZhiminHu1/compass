package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/compose"
)

// ResultStatus represents the status of a tool execution
type ResultStatus string

const (
	StatusSuccess ResultStatus = "success"
	StatusError   ResultStatus = "error"
	StatusPartial ResultStatus = "partial"
)

// DisplayTier 展示层级（控制UI显示详细程度）
type DisplayTier string

const (
	TierMinimal DisplayTier = "minimal" // 单行摘要
	TierCompact DisplayTier = "compact" // 紧凑显示
	TierFull    DisplayTier = "full"    // 完整显示
)

// Metadata contains structured metadata about tool execution
type Metadata struct {
	// File operations
	FilePath  string `json:"file_path,omitempty"`
	LineCount int    `json:"line_count,omitempty"`
	ByteCount int    `json:"byte_count,omitempty"`

	// Bash execution
	Command  string `json:"command,omitempty"`
	Duration int64  `json:"duration,omitempty"` // 毫秒
	ExitCode int    `json:"exit_code,omitempty"`
	Timeout  bool   `json:"timeout,omitempty"`

	// Search results
	MatchCount int    `json:"match_count,omitempty"`
	FileCount  int    `json:"file_count,omitempty"`
	Pattern    string `json:"pattern,omitempty"`

	// Network
	URL        string `json:"url,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}

// ToolResult represents a structured tool response
type ToolResult struct {
	Status   ResultStatus `json:"status"`
	Content  string       `json:"content"`
	Metadata *Metadata    `json:"metadata,omitempty"`
	Tier     DisplayTier  `json:"tier"` // UI展示层级
}

// String returns the formatted string representation for LLM consumption
func (r *ToolResult) String() string {
	var sb strings.Builder

	// Status indicator
	if r.Status == StatusError {
		sb.WriteString("❌ ERROR: ")
	} else if r.Status == StatusPartial {
		sb.WriteString("⚠️  PARTIAL: ")
	}

	// Content
	sb.WriteString(r.Content)

	// Metadata summary for LLM (简洁文本)
	if r.Metadata != nil {
		sb.WriteString("\n\n")
		sb.WriteString(r.formatLLMMetadata())
	}

	return sb.String()
}

// formatLLMMetadata 格式化给LLM看的元数据摘要
func (r *ToolResult) formatLLMMetadata() string {
	var parts []string
	md := r.Metadata

	if md.FilePath != "" {
		parts = append(parts, fmt.Sprintf("📄 %s", filepath.Base(md.FilePath)))
	}
	if md.LineCount > 0 {
		parts = append(parts, fmt.Sprintf("%d lines", md.LineCount))
	}
	if md.MatchCount > 0 {
		parts = append(parts, fmt.Sprintf("🔍 %d matches", md.MatchCount))
	}
	if md.Command != "" {
		parts = append(parts, fmt.Sprintf("⚡ %s", md.Command))
	}

	if len(parts) == 0 {
		return ""
	}
	return "[" + strings.Join(parts, " | ") + "]"
}

// ============================================
// Helper constructors
// ============================================

// Success creates a successful tool result
func Success(content string, metadata *Metadata, tier DisplayTier) (string, error) {
	return (&ToolResult{
		Status:   StatusSuccess,
		Content:  content,
		Metadata: metadata,
		Tier:     tier,
	}).String(), nil
}

// Error creates an error tool result
func Error(content string) (string, error) {
	return (&ToolResult{
		Status:  StatusError,
		Content: content,
		Tier:    TierCompact,
	}).String(), nil
}

// Partial creates a partial success tool result
func Partial(content string, metadata *Metadata) (string, error) {
	return (&ToolResult{
		Status:   StatusPartial,
		Content:  content,
		Metadata: metadata,
		Tier:     TierCompact,
	}).String(), nil
}

// ReadFileSuccess 文件读取成功（最小化显示）
func ReadFileSuccess(content, filePath string, lineCount, byteCount int) (string, error) {
	return Success(content, &Metadata{
		FilePath:  filePath,
		LineCount: lineCount,
		ByteCount: byteCount,
	}, TierMinimal)
}

// GrepSuccess grep搜索成功（最小化显示）
func GrepSuccess(content string, pattern string, matchCount, fileCount int) (string, error) {
	return Success(content, &Metadata{
		Pattern:    pattern,
		MatchCount: matchCount,
		FileCount:  fileCount,
	}, TierMinimal)
}

// GlobSuccess 文件匹配成功（最小化显示）
func GlobSuccess(content string, fileCount int) (string, error) {
	return Success(content, &Metadata{
		FileCount: fileCount,
	}, TierMinimal)
}

// BashSuccess bash执行成功（紧凑显示）
func BashSuccess(content, command string, duration int64, exitCode int) (string, error) {
	return Success(content, &Metadata{
		Command:  command,
		Duration: duration,
		ExitCode: exitCode,
	}, TierCompact)
}

// FetchSuccess 网页获取成功（紧凑显示）
func FetchSuccess(content, url string, statusCode int) (string, error) {
	return Success(content, &Metadata{
		URL:        url,
		StatusCode: statusCode,
	}, TierCompact)
}

// WriteFileSuccess 文件写入成功（完整显示）
func WriteFileSuccess(filePath string, byteCount int) (string, error) {
	content := fmt.Sprintf("File written: %s", filePath)
	return Success(content, &Metadata{
		FilePath:  filePath,
		ByteCount: byteCount,
	}, TierFull)
}

// EditFileSuccess 文件编辑成功（完整显示）
func EditFileSuccess(filePath string, lineCount int) (string, error) {
	content := fmt.Sprintf("File edited: %s", filePath)
	return Success(content, &Metadata{
		FilePath:  filePath,
		LineCount: lineCount,
	}, TierFull)
}

// DeleteFileSuccess 文件删除成功（完整显示）
func DeleteFileSuccess(filePath string) (string, error) {
	content := fmt.Sprintf("File deleted: %s", filePath)
	return Success(content, &Metadata{
		FilePath: filePath,
	}, TierFull)
}

// ErrorHandler 是工具错误处理中间件
func ErrorHandler() compose.ToolMiddleware {
	return compose.ToolMiddleware{
		Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
			return func(ctx context.Context, in *compose.ToolInput) (*compose.ToolOutput, error) {
				output, err := next(ctx, in)
				if err != nil {
					errStr := err.Error()
					// 跳过中断信号（正常流程）
					if strings.Contains(errStr, "interrupt signal") {
						return nil, err
					}

					// 处理普通错误：提取核心错误信息
					if idx := strings.Index(errStr, "err="); idx != -1 {
						coreErr := strings.TrimSpace(errStr[idx+4:])
						return &compose.ToolOutput{
							Result: fmt.Sprintf("Error: %s", coreErr),
						}, nil
					}

					// 默认错误处理
					return &compose.ToolOutput{
						Result: fmt.Sprintf("Error: %s", errStr),
					}, nil
				}
				return output, nil
			}
		},
	}
}
