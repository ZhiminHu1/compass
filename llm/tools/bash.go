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

var (
	BashToolName = "bash"
)

// BashToolParams contains parameters for the bash (PowerShell) tool.
type BashToolParams struct {
	Command   string `json:"command" jsonschema:"description=PowerShell command to run, e.g., 'Get-ChildItem', 'Get-Location', 'Get-Content file.txt'"`
	TimeoutMs uint64 `json:"timeout_ms,omitempty" jsonschema:"description=Timeout in milliseconds (default: 30000ms, max: 300000ms)"`
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

// BashToolFunc executes a PowerShell command.
func BashToolFunc(ctx context.Context, params BashToolParams) (string, error) {
	command := strings.TrimSpace(params.Command)
	if command == "" {
		return "", fmt.Errorf("command cannot be empty")
	}

	for _, dangerous := range dangerousCommands {
		if strings.Contains(command, dangerous) {
			return "Warning: Potentially dangerous command detected. Execution blocked for safety.", nil
		}
	}

	timeoutMs := params.TimeoutMs
	if timeoutMs == 0 {
		timeoutMs = 30000
	}
	if timeoutMs > 300000 {
		timeoutMs = 300000
	}

	timeout := time.Duration(timeoutMs) * time.Millisecond

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "powershell", "-NoProfile", "-Command", command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	output := stdout.String()
	errOutput := stderr.String()

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Command: %s\n", command))
	result.WriteString(fmt.Sprintf("Duration: %v\n", duration))

	if err != nil {
		if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
			result.WriteString(fmt.Sprintf("\nTimeout after %v", timeout))
		} else {
			result.WriteString(fmt.Sprintf("\nError: %v", err))
		}
	}

	if output != "" {
		result.WriteString(fmt.Sprintf("\nOutput:%s", output))
	}
	if errOutput != "" {
		result.WriteString(fmt.Sprintf("\nStderr:%s", errOutput))
	}

	if output == "" && errOutput == "" && err == nil {
		result.WriteString("\nCommand completed successfully with no output")
	}

	return result.String(), nil
}

// GetBashTool returns the PowerShell tool.
func GetBashTool() tool.InvokableTool {
	bashTool, err := utils.InferTool(
		"powershell",
		"Execute PowerShell commands like 'Get-ChildItem', 'Get-Location', 'Get-Content file.txt'. Returns command output, duration, and any errors.",
		BashToolFunc)
	if err != nil {
		log.Fatal(err)
	}
	return bashTool
}
