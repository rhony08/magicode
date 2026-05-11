// Package tool provides tool implementations for the AI agent.
package tool

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// ToolID is a unique identifier for a tool
type ToolID string

// Well-known tool IDs
const (
	ToolBash      ToolID = "bash"
	ToolRead      ToolID = "read"
	ToolWrite     ToolID = "write"
	ToolEdit      ToolID = "edit"
	ToolGrep      ToolID = "grep"
	ToolGlob      ToolID = "glob"
	ToolWebFetch  ToolID = "webfetch"
	ToolWebSearch ToolID = "websearch"
	ToolLSP       ToolID = "lsp"
	ToolTask      ToolID = "task"
	ToolQuestion  ToolID = "question"
	ToolTodo      ToolID = "todo"
	ToolPlan      ToolID = "plan"
)

// ToolContext provides context for tool execution
type ToolContext struct {
	SessionID  string
	MessageID  string
	Agent      string
	CallID     string
	WorkDir    string
	Abort      context.Context
	Extra      map[string]interface{}
}

// ToolResult is the result of a tool execution
type ToolResult struct {
	Title       string      `json:"title"`
	Output      string      `json:"output"`
	Metadata    Metadata    `json:"metadata"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Metadata holds tool execution metadata
type Metadata struct {
	Truncated   bool   `json:"truncated,omitempty"`
	OutputPath  string `json:"output_path,omitempty"`
	Duration    int64  `json:"duration,omitempty"` // milliseconds
	ExitCode    int    `json:"exit_code,omitempty"`
	RowsAffected int   `json:"rows_affected,omitempty"`
	FilesChanged []string `json:"files_changed,omitempty"`
}

// Attachment represents a file attachment in tool result
type Attachment struct {
	Type     string `json:"type"` // "image", "file"
	Name     string `json:"name"`
	Path     string `json:"path"`
	MimeType string `json:"mime_type,omitempty"`
}

// ParameterSchema describes a tool parameter
type ParameterSchema struct {
	Type        string      `json:"type"`                  // "string", "number", "boolean", "array", "object"
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Properties  map[string]ParameterSchema `json:"properties,omitempty"` // for object type
	Items       *ParameterSchema `json:"items,omitempty"` // for array type
}

// ToolDefinition describes a tool
type ToolDefinition struct {
	ID          string                    `json:"name"`
	Description string                    `json:"description"`
	Parameters  map[string]ParameterSchema `json:"parameters"`
}

// Tool is the interface for tool implementations
type Tool interface {
	// ID returns the tool's unique identifier
	ID() ToolID
	
	// Definition returns the tool's schema definition for the AI
	Definition() ToolDefinition
	
	// Execute runs the tool with given parameters
	Execute(ctx context.Context, params map[string]interface{}, toolCtx ToolContext) (*ToolResult, error)
	
	// Validate checks if parameters are valid
	Validate(params map[string]interface{}) error
}

// ToolError represents an error from tool execution
type ToolError struct {
	ToolID   string `json:"tool_id"`
	Message  string `json:"message"`
	Type     string `json:"type"` // "validation_error", "execution_error", "permission_denied", "timeout"
	Details  map[string]interface{} `json:"details,omitempty"`
}

func (e *ToolError) Error() string {
	return e.Message
}

// NewToolError creates a new tool error
func NewToolError(toolID ToolID, typ string, message string) *ToolError {
	return &ToolError{
		ToolID:  string(toolID),
		Type:    typ,
		Message: message,
	}
}

// NewValidationError creates a validation error
func NewValidationError(toolID ToolID, message string) *ToolError {
	return NewToolError(toolID, "validation_error", message)
}

// NewExecutionError creates an execution error
func NewExecutionError(toolID ToolID, message string) *ToolError {
	return NewToolError(toolID, "execution_error", message)
}

// NewPermissionError creates a permission denied error
func NewPermissionError(toolID ToolID, message string) *ToolError {
	return NewToolError(toolID, "permission_denied", message)
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(toolID ToolID, message string) *ToolError {
	return NewToolError(toolID, "timeout", message)
}

// ValidatePath validates that a file path stays within the working directory
// This prevents path traversal attacks (e.g., ../../../etc/passwd)
func ValidatePath(filePath, workDir string) error {
	// Clean and get absolute paths
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// If workDir is not set, allow the path (for backward compatibility)
	if workDir == "" {
		return nil
	}

	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("invalid working directory: %w", err)
	}

	// Ensure the path is within the working directory
	// Add trailing separator to prevent partial matches
	if !strings.HasPrefix(absPath, absWorkDir+string(filepath.Separator)) && absPath != absWorkDir {
		return fmt.Errorf("path %s is outside working directory %s", absPath, absWorkDir)
	}

	return nil
}