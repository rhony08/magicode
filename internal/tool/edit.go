// Package tool provides the Edit tool implementation.
package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EditTool edits file contents with string replacement
type EditTool struct{}

// NewEditTool creates a new Edit tool
func NewEditTool() *EditTool {
	return &EditTool{}
}

// ID returns the tool ID
func (t *EditTool) ID() ToolID {
	return ToolEdit
}

// Definition returns the tool definition
func (t *EditTool) Definition() ToolDefinition {
	return ToolDefinition{
		ID:          "edit",
		Description: "Performs exact string replacements in files. This tool edits existing files by replacing specific text. Use this tool when you need to make targeted edits. The filePath parameter should be an absolute path. The oldString parameter must match exactly - include whitespace and indentation. This tool edits existing files in the codebase. NEVER write new files unless explicitly required.",
		Parameters: map[string]ParameterSchema{
			"filePath": {
				Type:        "string",
				Description: "The absolute path to the file to edit",
				Required:    true,
			},
			"oldString": {
				Type:        "string",
				Description: "The exact text to replace (must match exactly, including whitespace)",
				Required:    true,
			},
			"newString": {
				Type:        "string",
				Description: "The text to replace with",
				Required:    true,
			},
			"replaceAll": {
				Type:        "boolean",
				Description: "Replace all occurrences of oldString",
				Required:    false,
				Default:     false,
			},
		},
	}
}

// Validate checks parameters
func (t *EditTool) Validate(params map[string]interface{}) error {
	filePath, ok := params["filePath"].(string)
	if !ok || filePath == "" {
		return NewValidationError(ToolEdit, "filePath is required and must be a string")
	}

	oldString, ok := params["oldString"].(string)
	if !ok || oldString == "" {
		return NewValidationError(ToolEdit, "oldString is required and must be a non-empty string")
	}

	newString, ok := params["newString"].(string)
	if !ok {
		return NewValidationError(ToolEdit, "newString is required and must be a string")
	}

	// Check for absolute path
	if !filepath.IsAbs(filePath) {
		return NewValidationError(ToolEdit, "filePath must be an absolute path")
	}

	// oldString and newString must be different
	if oldString == newString {
		return NewValidationError(ToolEdit, "oldString and newString must be different")
	}

	return nil
}

// Execute performs the edit
func (t *EditTool) Execute(ctx context.Context, params map[string]interface{}, toolCtx ToolContext) (*ToolResult, error) {
	filePath := params["filePath"].(string)
	oldString := params["oldString"].(string)
	newString := params["newString"].(string)

	// Security check: Validate path is within working directory
	if err := ValidatePath(filePath, toolCtx.WorkDir); err != nil {
		return nil, NewPermissionError(ToolEdit, err.Error())
	}

	replaceAll := false
	if r, ok := params["replaceAll"].(bool); ok {
		replaceAll = r
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewExecutionError(ToolEdit, fmt.Sprintf("file not found: %s", filePath))
		}
		return nil, NewExecutionError(ToolEdit, err.Error())
	}

	contentStr := string(content)

	// Check if oldString exists
	if !strings.Contains(contentStr, oldString) {
		return nil, NewExecutionError(ToolEdit, "oldString not found in file. Please ensure it matches exactly, including whitespace.")
	}

	// Check for multiple matches when not replaceAll
	matches := strings.Count(contentStr, oldString)
	if matches > 1 && !replaceAll {
		return nil, NewExecutionError(ToolEdit, fmt.Sprintf("Found %d matches for oldString. Provide more surrounding context to make it unique, or use replaceAll=true.", matches))
	}

	// Perform replacement
	var newContent string
	if replaceAll {
		newContent = strings.ReplaceAll(contentStr, oldString, newString)
	} else {
		// Replace first occurrence only
		idx := strings.Index(contentStr, oldString)
		newContent = contentStr[:idx] + newString + contentStr[idx+len(oldString):]
	}

	// Write file
	err = os.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		return nil, NewExecutionError(ToolEdit, err.Error())
	}

	// Build title
	title := fmt.Sprintf("Edit: %s", filepath.Base(filePath))

	// Build output
	replacements := matches
	if !replaceAll && matches > 0 {
		replacements = 1
	}

	output := fmt.Sprintf("Successfully edited %s", filePath)
	output += fmt.Sprintf("\n%d replacement(s) made", replacements)

	// Show the change
	oldLen := len(oldString)
	newLen := len(newString)
	if oldLen > 50 {
		oldString = oldString[:50] + "..."
	}
	if newLen > 50 {
		newString = newString[:50] + "..."
	}
	output += fmt.Sprintf("\nReplaced: '%s' -> '%s'", oldString, newString)

	return &ToolResult{
		Title:  title,
		Output: output,
		Metadata: Metadata{
			RowsAffected:  replacements,
			FilesChanged:  []string{filePath},
		},
	}, nil
}