// Package tool provides the Write tool implementation.
package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteTool writes content to a file
type WriteTool struct{}

// NewWriteTool creates a new Write tool
func NewWriteTool() *WriteTool {
	return &WriteTool{}
}

// ID returns the tool ID
func (t *WriteTool) ID() ToolID {
	return ToolWrite
}

// Definition returns the tool definition
func (t *WriteTool) Definition() ToolDefinition {
	return ToolDefinition{
		ID:          "write",
		Description: "Writes a file to the local filesystem. This tool will overwrite the existing file if there is one at the provided path. Use this tool when you need to create new files or completely replace existing file contents. Prefer editing existing files in the codebase. NEVER write new files unless explicitly required.",
		Parameters: map[string]ParameterSchema{
			"filePath": {
				Type:        "string",
				Description: "The absolute path to the file to write (must be absolute)",
				Required:    true,
			},
			"content": {
				Type:        "string",
				Description: "The content to write to the file",
				Required:    true,
			},
		},
	}
}

// Validate checks parameters
func (t *WriteTool) Validate(params map[string]interface{}) error {
	filePath, ok := params["filePath"].(string)
	if !ok || filePath == "" {
		return NewValidationError(ToolWrite, "filePath is required and must be a string")
	}

	_, ok = params["content"].(string)
	if !ok {
		return NewValidationError(ToolWrite, "content is required and must be a string")
	}

	// Check for absolute path requirement
	if !filepath.IsAbs(filePath) {
		return NewValidationError(ToolWrite, "filePath must be an absolute path")
	}

	return nil
}

// Execute writes the file
func (t *WriteTool) Execute(ctx context.Context, params map[string]interface{}, toolCtx ToolContext) (*ToolResult, error) {
	filePath := params["filePath"].(string)
	content := params["content"].(string)

	// Security check: Validate path is within working directory
	if err := ValidatePath(filePath, toolCtx.WorkDir); err != nil {
		return nil, NewPermissionError(ToolWrite, err.Error())
	}

	// Verify parent directory exists
	parentDir := filepath.Dir(filePath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		return nil, NewExecutionError(ToolWrite, fmt.Sprintf("parent directory does not exist: %s", parentDir))
	}

	// Check if file exists (for metadata)
	existing := false
	if _, err := os.Stat(filePath); err == nil {
		existing = true
	}

	// Write file with proper permissions
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return nil, NewExecutionError(ToolWrite, err.Error())
	}

	// Build title
	title := fmt.Sprintf("Write: %s", filepath.Base(filePath))

	// Build output
	output := fmt.Sprintf("Successfully wrote to %s", filePath)
	output += fmt.Sprintf("\n%d bytes written", len(content))
	if existing {
		output += " (overwritten)"
	} else {
		output += " (new file)"
	}

	// Count lines
	lines := len(strings.Split(content, "\n"))
	output += fmt.Sprintf("\n%d lines", lines)

	return &ToolResult{
		Title:  title,
		Output: output,
		Metadata: Metadata{
			RowsAffected:  1,
			FilesChanged:  []string{filePath},
		},
	}, nil
}