// Package tool provides the Bash tool implementation.
package tool

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	bashDefaultTimeout = 2 * 60 * 1000 // 2 minutes in milliseconds
	maxOutputLength    = 30_000        // characters
)

// BashTool executes shell commands
type BashTool struct{}

// NewBashTool creates a new Bash tool
func NewBashTool() *BashTool {
	return &BashTool{}
}

// ID returns the tool ID
func (t *BashTool) ID() ToolID {
	return ToolBash
}

// Definition returns the tool definition
func (t *BashTool) Definition() ToolDefinition {
	return ToolDefinition{
		ID:          "bash",
		Description: "Executes a bash command in a persistent shell session with optional timeout. Be aware: OS: linux, Shell: bash. Commands run in the current working directory by default. Use the workdir parameter if you need to run a command in a different directory. IMPORTANT: This tool is for terminal operations like git, npm, docker, etc. DO NOT use it for file operations (reading, writing, editing, searching, finding files) - use the specialized tools for this instead.",
		Parameters: map[string]ParameterSchema{
			"command": {
				Type:        "string",
				Description: "The command to execute",
				Required:    true,
			},
			"timeout": {
				Type:        "number",
				Description: "Optional timeout in milliseconds. Defaults to 120000 (2 minutes).",
				Required:    false,
				Default:     bashDefaultTimeout,
			},
			"workdir": {
				Type:        "string",
				Description: "The working directory to run the command in. Defaults to the current directory. Use this instead of 'cd' commands.",
				Required:    false,
			},
			"description": {
				Type:        "string",
				Description: "Clear, concise description of what this command does in 5-10 words. Examples:\nInput: ls\nOutput: Lists files in current directory\n\nInput: git status\nOutput: Shows working tree status",
				Required:    true,
			},
		},
	}
}

// Validate checks parameters
func (t *BashTool) Validate(params map[string]interface{}) error {
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return NewValidationError(ToolBash, "command is required and must be a string")
	}

	description, ok := params["description"].(string)
	if !ok || description == "" {
		return NewValidationError(ToolBash, "description is required and must be a string")
	}

	// Optional timeout
	if timeout, ok := params["timeout"]; ok {
		switch v := timeout.(type) {
		case int:
			if v <= 0 {
				return NewValidationError(ToolBash, "timeout must be positive")
			}
		case float64:
			if v <= 0 {
				return NewValidationError(ToolBash, "timeout must be positive")
			}
		}
	}

	return nil
}

// Execute runs the command
func (t *BashTool) Execute(ctx context.Context, params map[string]interface{}, toolCtx ToolContext) (*ToolResult, error) {
	command := params["command"].(string)
	description := params["description"].(string)

	// Security check: Validate command is not dangerous
	if IsDangerousCommand(command) {
		return nil, NewExecutionError(ToolBash, "command contains potentially dangerous operations and is not allowed")
	}

	// Get timeout
	timeout := bashDefaultTimeout
	if t, ok := params["timeout"]; ok {
		switch v := t.(type) {
		case int:
			timeout = v
		case float64:
			timeout = int(v)
		}
	}

	// Get working directory
	workdir := toolCtx.WorkDir
	if w, ok := params["workdir"].(string); ok && w != "" {
		workdir = w
	}

	start := time.Now()

	// Create command
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	if workdir != "" {
		cmd.Dir = workdir
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	// Wait for completion or timeout
	var execErr error
	select {
	case <-ctx.Done():
		// Context cancelled (abort signal)
		return nil, NewTimeoutError(ToolBash, "command aborted")
	case execErr = <-done:
		// Command finished
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		// Timeout
		cmd.Process.Kill()
		return nil, NewTimeoutError(ToolBash, fmt.Sprintf("command timed out after %d ms", timeout))
	}

	duration := time.Since(start).Milliseconds()

	// Build output
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nstderr:\n" + stderr.String()
	}

	// Truncate if too long
	truncated := false
	if len(output) > maxOutputLength {
		output = output[:maxOutputLength] + "\n... (output truncated)"
		truncated = true
	}

	// Get exit code
	exitCode := 0
	if execErr != nil {
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, NewExecutionError(ToolBash, execErr.Error())
		}
	}

	// Build title
	title := description
	if title == "" {
		title = fmt.Sprintf("Execute: %s", truncateCommand(command, 50))
	}

	return &ToolResult{
		Title:  title,
		Output: output,
		Metadata: Metadata{
			Duration:  duration,
			ExitCode:  exitCode,
			Truncated: truncated,
		},
	}, nil
}

// truncateCommand truncates a command for display
func truncateCommand(cmd string, maxLen int) string {
	if len(cmd) <= maxLen {
		return cmd
	}
	return cmd[:maxLen] + "..."
}

// IsDangerousCommand checks if a command is potentially dangerous
func IsDangerousCommand(command string) bool {
	dangerousPatterns := []string{
		"rm -rf /",
		"rm -rf ~",
		"rm -rf *",
		"mkfs",
		"dd if=",
		":(){ :|:& };:", // fork bomb
		"chmod -R 777 /",
		"> /dev/sda",
		"curl | sh",
		"wget | sh",
		"| sh",
		"| bash",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(command, pattern) {
			return true
		}
	}
	return false
}