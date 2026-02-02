package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"
)

// ToolExtra keys for ToolInfo.Extra map
const (
	// ToolExtraSilent marks a tool's output as hidden from UI
	// Usage: Extra: map[string]any{tools.ToolExtraSilent: true}
	ToolExtraSilent = "silent"
)

// ResultStatus represents the status of a tool execution
type ResultStatus string

const (
	StatusSuccess ResultStatus = "success"
	StatusError   ResultStatus = "error"
	StatusPartial ResultStatus = "partial" // 部分成功
)

// Metadata contains structured metadata about tool execution
type Metadata struct {
	// File operations
	FilePath  string `json:"file_path,omitempty"`
	LineCount int    `json:"line_count,omitempty"`
	ByteCount int    `json:"byte_count,omitempty"`

	// Bash execution
	Command  string `json:"command,omitempty"`
	Duration int64  `json:"duration_ms,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
	Timeout  bool   `json:"timeout,omitempty"`

	// Search results
	MatchCount int      `json:"match_count,omitempty"`
	Files      []string `json:"files,omitempty"`

	// Network
	URL        string `json:"url,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}

// ToolResult represents a structured tool response
type ToolResult struct {
	Status   ResultStatus `json:"status"`
	Content  string       `json:"content"`
	Metadata *Metadata    `json:"metadata,omitempty"`
}

// String returns the formatted string representation for LLM consumption
func (r *ToolResult) String() string {
	var sb strings.Builder

	// Status indicator
	if r.Status == StatusError {
		sb.WriteString("[ERROR] ")
	} else if r.Status == StatusPartial {
		sb.WriteString("[PARTIAL] ")
	}

	// Content
	sb.WriteString(r.Content)

	// Metadata in XML format (LLLM-friendly)
	if r.Metadata != nil {
		md := r.Metadata
		var attrs []string

		if md.FilePath != "" {
			attrs = append(attrs, fmt.Sprintf("file=%s", md.FilePath))
		}
		if md.LineCount > 0 {
			attrs = append(attrs, fmt.Sprintf("lines=%d", md.LineCount))
		}
		if md.Command != "" {
			attrs = append(attrs, fmt.Sprintf("cmd=%q", md.Command))
		}
		if md.Duration > 0 {
			attrs = append(attrs, fmt.Sprintf("duration=%dms", md.Duration))
		}
		if md.ExitCode != 0 {
			attrs = append(attrs, fmt.Sprintf("exit=%d", md.ExitCode))
		}
		if md.Timeout {
			attrs = append(attrs, "timeout=true")
		}
		if md.MatchCount > 0 {
			attrs = append(attrs, fmt.Sprintf("matches=%d", md.MatchCount))
		}
		if md.URL != "" {
			attrs = append(attrs, fmt.Sprintf("url=%s", md.URL))
		}
		if md.StatusCode > 0 {
			attrs = append(attrs, fmt.Sprintf("status=%d", md.StatusCode))
		}

		if len(attrs) > 0 {
			sb.WriteString(fmt.Sprintf("\n\n<metadata %s />", strings.Join(attrs, " ")))
		}
	}

	return sb.String()
}

// JSON returns the JSON representation (for debugging/logging)
func (r *ToolResult) JSON() string {
	data, _ := json.MarshalIndent(r, "", "  ")
	return string(data)
}

// Helper constructors

// Success creates a successful tool result
func Success(content string, metadata *Metadata) (string, error) {
	return (&ToolResult{
		Status:   StatusSuccess,
		Content:  content,
		Metadata: metadata,
	}).String(), nil
}

// Error creates an error tool result
func Error(content string) (string, error) {
	return (&ToolResult{
		Status:  StatusError,
		Content: content,
	}).String(), nil
}

// Partial creates a partial success tool result
func Partial(content string, metadata *Metadata) (string, error) {
	return (&ToolResult{
		Status:   StatusPartial,
		Content:  content,
		Metadata: metadata,
	}).String(), nil
}

// SuccessWithFile creates a success result with file metadata
func SuccessWithFile(content, filePath string, lineCount int) (string, error) {
	return Success(content, &Metadata{
		FilePath:  filePath,
		LineCount: lineCount,
	})
}

// SuccessWithCommand creates a success result with command metadata
func SuccessWithCommand(content, command string, duration int64, exitCode int) (string, error) {
	return Success(content, &Metadata{
		Command:  command,
		Duration: duration,
		ExitCode: exitCode,
	})
}

// ErrorHandler 是工具错误处理中间件
// 捕获工具调用错误，转换为友好的错误消息
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
						// 将错误转换为成功的工具结果
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
