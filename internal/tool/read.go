// Package tool provides the Read tool implementation.
package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxReadFileSize = 2 * 1024 * 1024 // 2MB
	maxReadLines    = 2000
)

// ReadTool reads file contents
type ReadTool struct{}

// NewReadTool creates a new Read tool
func NewReadTool() *ReadTool {
	return &ReadTool{}
}

// ID returns the tool ID
func (t *ReadTool) ID() ToolID {
	return ToolRead
}

// Definition returns the tool definition
func (t *ReadTool) Definition() ToolDefinition {
	return ToolDefinition{
		ID:          "read",
		Description: "Reads a file or directory from the local filesystem. Use this when you need to examine file contents. You can use this tool to read image files and PDFs. The filePath parameter should be an absolute path. By default, this tool returns up to 2000 lines from the start of the file. Use the offset parameter to read later sections.",
		Parameters: map[string]ParameterSchema{
			"filePath": {
				Type:        "string",
				Description: "The absolute path to the file or directory to read",
				Required:    true,
			},
			"offset": {
				Type:        "number",
				Description: "The line number to start reading from (1-indexed)",
				Required:    false,
				Default:     1,
			},
			"limit": {
				Type:        "number",
				Description: "The maximum number of lines to read",
				Required:    false,
				Default:     maxReadLines,
			},
		},
	}
}

// Validate checks parameters
func (t *ReadTool) Validate(params map[string]interface{}) error {
	filePath, ok := params["filePath"].(string)
	if !ok || filePath == "" {
		return NewValidationError(ToolRead, "filePath is required and must be a string")
	}

	// Validate offset
	if offset, ok := params["offset"]; ok {
		switch v := offset.(type) {
		case int:
			if v < 1 {
				return NewValidationError(ToolRead, "offset must be >= 1")
			}
		case float64:
			if v < 1 {
				return NewValidationError(ToolRead, "offset must be >= 1")
			}
		}
	}

	// Validate limit
	if limit, ok := params["limit"]; ok {
		switch v := limit.(type) {
		case int:
			if v < 1 {
				return NewValidationError(ToolRead, "limit must be >= 1")
			}
		case float64:
			if v < 1 {
				return NewValidationError(ToolRead, "limit must be >= 1")
			}
		}
	}

	return nil
}

// Execute reads the file
func (t *ReadTool) Execute(ctx context.Context, params map[string]interface{}, toolCtx ToolContext) (*ToolResult, error) {
	filePath := params["filePath"].(string)

	// Resolve path relative to workdir if not absolute
	if !filepath.IsAbs(filePath) && toolCtx.WorkDir != "" {
		filePath = filepath.Join(toolCtx.WorkDir, filePath)
	}

	// Security check: Validate path is within working directory
	if err := ValidatePath(filePath, toolCtx.WorkDir); err != nil {
		return nil, NewPermissionError(ToolRead, err.Error())
	}

	// Get offset and limit
	offset := 1
	if o, ok := params["offset"]; ok {
		switch v := o.(type) {
		case int:
			offset = v
		case float64:
			offset = int(v)
		}
	}

	limit := maxReadLines
	if l, ok := params["limit"]; ok {
		switch v := l.(type) {
		case int:
			limit = v
		case float64:
			limit = int(v)
		}
	}

	// Check if path exists
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewExecutionError(ToolRead, fmt.Sprintf("file not found: %s", filePath))
		}
		return nil, NewExecutionError(ToolRead, err.Error())
	}

	// Handle directories
	if info.IsDir() {
		return t.readDirectory(filePath)
	}

	// Check file size
	if info.Size() > maxReadFileSize {
		return nil, NewExecutionError(ToolRead, fmt.Sprintf("file too large (%d bytes > %d limit)", info.Size(), maxReadFileSize))
	}

	// Read file content
	content, err := readFileLines(filePath, offset, limit)
	if err != nil {
		return nil, NewExecutionError(ToolRead, err.Error())
	}

	// Build output with line numbers
	output := formatWithLineNumbers(content, offset)

	return &ToolResult{
		Title:  fmt.Sprintf("Read: %s", filepath.Base(filePath)),
		Output: output,
		Metadata: Metadata{
			RowsAffected: len(content),
		},
	}, nil
}

// readDirectory lists directory contents
func (t *ReadTool) readDirectory(dirPath string) (*ToolResult, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, NewExecutionError(ToolRead, err.Error())
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Directory: %s", dirPath))
	lines = append(lines, "")

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		info, err := entry.Info()
		if err != nil {
			lines = append(lines, fmt.Sprintf("%s (error: %v)", name, err))
		} else {
			lines = append(lines, fmt.Sprintf("%s  %d bytes", name, info.Size()))
		}
	}

	return &ToolResult{
		Title:  fmt.Sprintf("List: %s", filepath.Base(dirPath)),
		Output: strings.Join(lines, "\n"),
		Metadata: Metadata{
			RowsAffected: len(entries),
		},
	}, nil
}

// readFileLines reads specific lines from a file
func readFileLines(filePath string, offset, limit int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read all content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Split into lines
	allLines := strings.Split(string(content), "\n")

	// Adjust for 1-indexed offset
	startIdx := offset - 1
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx >= len(allLines) {
		return []string{}, nil
	}

	endIdx := startIdx + limit
	if endIdx > len(allLines) {
		endIdx = len(allLines)
	}

	return allLines[startIdx:endIdx], nil
}

// formatWithLineNumbers formats lines with line number prefix
func formatWithLineNumbers(lines []string, startOffset int) string {
	var result []string
	for i, line := range lines {
		lineNum := startOffset + i
		result = append(result, fmt.Sprintf("%d: %s", lineNum, line))
	}
	return strings.Join(result, "\n")
}