package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

const (
	// BashToolName is the name of the bash tool
	BashToolName = "bash"

	// DefaultTimeoutMs is the default timeout for command execution
	DefaultTimeoutMs = 30000
	// MaxTimeoutMs is the maximum allowed timeout
	MaxTimeoutMs = 300000
	// MaxOutputLength is the maximum output length before truncation
	MaxOutputLength = 10000
)

// BashToolParams contains parameters for the bash (PowerShell) tool.
type BashToolParams struct {
	Command   string `json:"command" jsonschema:"description=PowerShell command to execute."`
	TimeoutMs uint64 `json:"timeout_ms,omitempty" jsonschema:"description=Timeout in milliseconds (default: 30000, max: 300000)."`
}

// dangerousCommands is a blacklist of dangerous PowerShell commands.
var dangerousCommands = []string{
	"Remove-Item -Recurse -Force \\",
	"Remove-Item -Recurse -Force /",
	"Format-Volume",
	"Remove-Partition",
	"Stop-Computer",
	"Restart-Computer",
	"Remove-ADDomainController",
}

// bashDescription is the detailed tool description for the AI
const bashDescription = `Execute PowerShell commands in a Windows environment.

BEFORE USING:
1. Verify the command is safe before execution
2. Use absolute paths when working with files
3. For file operations, prefer using dedicated tools (read_file, write_file, edit_file)

CAPABILITIES:
- Run any PowerShell command
- Get system information (Get-Process, Get-Service, etc.)
- List files and directories (Get-ChildItem, Get-Location)
- Run build commands (go build, npm install, etc.)
- Git operations (git status, git log, etc.)

SECURITY:
- Dangerous system commands are blocked
- Commands with destructive potential will be rejected

PARAMETERS:
- command (required): The PowerShell command to execute
- timeout_ms (optional): Timeout in milliseconds (default: 30000, max: 300000)

OUTPUT FORMAT:
Returns command output with execution metadata including duration and exit code.

EXAMPLES:
- List files: {"command": "Get-ChildItem"}
- Get processes: {"command": "Get-Process | Select-Object -First 5"}
- Current directory: {"command": "Get-Location"}`

// BashToolFunc executes a PowerShell command with structured response.
func BashToolFunc(ctx context.Context, params BashToolParams) (string, error) {
	command := strings.TrimSpace(params.Command)
	if command == "" {
		return Error("command cannot be empty")
	}

	// Security check
	for _, dangerous := range dangerousCommands {
		if strings.Contains(command, dangerous) {
			return Error(fmt.Sprintf("dangerous command detected and blocked: %s", dangerous))
		}
	}

	// Validate and set timeout
	timeoutMs := params.TimeoutMs
	if timeoutMs == 0 {
		timeoutMs = DefaultTimeoutMs
	}
	if timeoutMs > MaxTimeoutMs {
		timeoutMs = MaxTimeoutMs
	}

	timeout := time.Duration(timeoutMs) * time.Millisecond

	// Execute command
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "powershell", "-NoProfile", "-Command", command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	// Build output
	var output []string

	// Check for timeout
	if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
		return Partial(fmt.Sprintf("Command timed out after %v\n\nPartial output:\n%s",
			timeout, truncateOutput(stdoutStr)), &Metadata{
			Command:  command,
			Duration: duration.Milliseconds(),
			Timeout:  true,
		})
	}

	// Build result content
	if stdoutStr != "" {
		output = append(output, truncateOutput(stdoutStr))
	}

	exitCode := 0
	if err != nil {
		exitCode = 1
		if stderrStr != "" {
			output = append(output, fmt.Sprintf("stderr: %s", truncateOutput(stderrStr)))
		}
		// Don't include the error message for exit code, just metadata
	}

	// No output case
	if len(output) == 0 {
		output = append(output, "Command completed with no output")
	}

	return SuccessWithCommand(
		strings.Join(output, "\n"),
		command,
		duration.Milliseconds(),
		exitCode,
	)
}

// truncateOutput truncates output if it exceeds MaxOutputLength
func truncateOutput(s string) string {
	if len(s) <= MaxOutputLength {
		return s
	}
	half := MaxOutputLength / 2
	truncated := len(s) - MaxOutputLength
	return fmt.Sprintf("%s\n\n... [~%d chars truncated] ...\n\n%s",
		s[:half], truncated, s[len(s)-half:])
}

// GetBashTool returns the PowerShell tool with enhanced description.
func GetBashTool() tool.InvokableTool {
	bashTool, err := utils.InferTool(
		BashToolName,
		bashDescription,
		BashToolFunc,
	)
	if err != nil {
		log.Fatal(err)
	}
	return bashTool
}
