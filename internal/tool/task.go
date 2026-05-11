// Package tool provides the task tool for subagent delegation.
package tool

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rhony08/magicode/internal/agent"
	"github.com/rhony08/magicode/internal/database"
	"github.com/rhony08/magicode/internal/provider"
	"github.com/rhony08/magicode/internal/util/log"
)

// TaskTool implements the task tool for spawning subagents
type TaskTool struct {
	registry    *agent.Registry
	db          *database.Database
	processor   ProcessorInterface // Interface for processing subagent tasks
	logger      *log.Logger
}

// ProcessorInterface defines the interface for processing subagent tasks
type ProcessorInterface interface {
	Process(ctx context.Context, req ProcessRequestInterface) error
}

// ProcessRequestInterface defines the interface for process requests
type ProcessRequestInterface interface {
	GetSessionID() string
	GetUserMessage() string
	GetModel() provider.ModelID
	GetAgent() string
	GetSystemPrompt() string
}

// TaskToolParams represents the parameters for the task tool
type TaskToolParams struct {
	Description    string `json:"description"`              // Short description of the task
	Prompt         string `json:"prompt"`                   // The task for the agent to perform
	SubagentType   string `json:"subagent_type"`            // The type of specialized agent
	TaskID         string `json:"task_id,omitempty"`        // Optional task ID to resume
	Command        string `json:"command,omitempty"`        // Optional command that triggered this
}

// NewTaskTool creates a new task tool
func NewTaskTool(registry *agent.Registry, db *database.Database) *TaskTool {
	return &TaskTool{
		registry: registry,
		db:       db,
		logger:   log.Create(map[string]string{"service": "tool.task"}),
	}
}

// SetProcessor sets the processor for handling subagent tasks
func (t *TaskTool) SetProcessor(processor ProcessorInterface) {
	t.processor = processor
}

// ID returns the tool ID
func (t *TaskTool) ID() ToolID {
	return ToolTask
}

// Definition returns the tool definition
func (t *TaskTool) Definition() ToolDefinition {
	return ToolDefinition{
		ID:          "task",
		Description: "Launch a specialized subagent to handle a specific task. Use this tool when you need to delegate work to a specialized agent like code-review, architect, or debugger.",
		Parameters: map[string]ParameterSchema{
			"description": {
				Type:        "string",
				Description: "A short (3-5 words) description of the task",
				Required:    true,
			},
			"prompt": {
				Type:        "string",
				Description: "The task for the agent to perform",
				Required:    true,
			},
			"subagent_type": {
				Type:        "string",
				Description: "The type of specialized agent to use for this task",
				Required:    true,
				Enum:        t.getAvailableSubagents(),
			},
			"task_id": {
				Type:        "string",
				Description: "This should only be set if you mean to resume a previous task (pass prior task_id to continue the same subagent session)",
				Required:    false,
			},
			"command": {
				Type:        "string",
				Description: "The command that triggered this task",
				Required:    false,
			},
		},
	}
}

// getAvailableSubagents returns list of available subagent types
func (t *TaskTool) getAvailableSubagents() []string {
	if t.registry == nil {
		return []string{"code", "debugger", "reviewer"}
	}

	subagents := t.registry.ListSubagents()
	result := make([]string, 0, len(subagents))
	for _, info := range subagents {
		result = append(result, info.Name)
	}

	if len(result) == 0 {
		return []string{"code", "debugger", "reviewer"}
	}
	return result
}

// Validate validates the parameters
func (t *TaskTool) Validate(params map[string]interface{}) error {
	desc, ok := params["description"].(string)
	if !ok || desc == "" {
		return NewValidationError(ToolTask, "description is required")
	}

	prompt, ok := params["prompt"].(string)
	if !ok || prompt == "" {
		return NewValidationError(ToolTask, "prompt is required")
	}

	subagentType, ok := params["subagent_type"].(string)
	if !ok || subagentType == "" {
		return NewValidationError(ToolTask, "subagent_type is required")
	}

	// Validate subagent type exists
	available := t.getAvailableSubagents()
	valid := false
	for _, sa := range available {
		if sa == subagentType {
			valid = true
			break
		}
	}
	if !valid {
		return NewValidationError(ToolTask, fmt.Sprintf("unknown subagent type: %s. Available: %v", subagentType, available))
	}

	return nil
}

// Execute runs the task tool
func (t *TaskTool) Execute(ctx context.Context, params map[string]interface{}, toolCtx ToolContext) (*ToolResult, error) {
	// Parse parameters
	taskParams := TaskToolParams{}
	if desc, ok := params["description"].(string); ok {
		taskParams.Description = desc
	}
	if prompt, ok := params["prompt"].(string); ok {
		taskParams.Prompt = prompt
	}
	if subagentType, ok := params["subagent_type"].(string); ok {
		taskParams.SubagentType = subagentType
	}
	if taskID, ok := params["task_id"].(string); ok {
		taskParams.TaskID = taskID
	}
	if command, ok := params["command"].(string); ok {
		taskParams.Command = command
	}

	// Get agent info
	agentInfo, ok := t.registry.Get(taskParams.SubagentType)
	if !ok {
		return nil, NewExecutionError(ToolTask, fmt.Sprintf("unknown agent type: %s", taskParams.SubagentType))
	}

	t.logger.Info("Executing task tool", map[string]interface{}{
		"description":    taskParams.Description,
		"subagent_type":  taskParams.SubagentType,
		"task_id":        taskParams.TaskID,
		"session_id":     toolCtx.SessionID,
	})

	// Create or resume sub-session
	subSessionID := taskParams.TaskID
	if subSessionID == "" {
		subSessionID = fmt.Sprintf("task_%s_%d", toolCtx.SessionID, time.Now().UnixNano())
	}

	// Get model from agent or use default
	modelID := provider.ModelID("claude-3-5-sonnet")
	if agentInfo.Model != nil {
		modelID = agentInfo.Model.ModelID
	}

	// Build system prompt from agent
	systemPrompt := agentInfo.Prompt
	if systemPrompt == "" {
		systemPrompt = fmt.Sprintf("You are a specialized %s agent. Focus on the task at hand.", taskParams.SubagentType)
	}

	// Build user prompt
	userPrompt := taskParams.Prompt

	// Execute subagent task
	var output string
	var err error

	if t.processor != nil {
		// Create process request
		req := &TaskProcessRequest{
			SessionID:    subSessionID,
			UserMessage:  userPrompt,
			Model:        modelID,
			Agent:        taskParams.SubagentType,
			SystemPrompt: systemPrompt,
		}

		// Process with subagent
		err = t.processor.Process(ctx, req)
		if err != nil {
			t.logger.Error("Subagent processing failed", map[string]interface{}{
				"error":        err.Error(),
				"subagent":     taskParams.SubagentType,
				"session_id":   subSessionID,
			})
			output = fmt.Sprintf("Subagent task failed: %s", err.Error())
		} else {
			// Get result from last assistant message
			output = t.getSubagentResult(ctx, subSessionID)
		}
	} else {
		// No processor available - return placeholder
		output = fmt.Sprintf("Task '%s' would be delegated to %s agent.\nPrompt: %s",
			taskParams.Description, taskParams.SubagentType, userPrompt)
	}

	// Build result
	result := &ToolResult{
		Title: taskParams.Description,
		Output: strings.Join([]string{
			fmt.Sprintf("task_id: %s (for resuming to continue this task if needed)", subSessionID),
			"",
			"<task_result>",
			output,
			"</task_result>",
		}, "\n"),
		Metadata: Metadata{
			OutputPath: subSessionID,
		},
	}

	return result, nil
}

// getSubagentResult retrieves the result from the subagent session
func (t *TaskTool) getSubagentResult(ctx context.Context, sessionID string) string {
	if t.db == nil {
		return "Result not available (no database)"
	}

	// Get last assistant message
	msgStorage := database.NewMessageStorage(t.db)
	messages, err := msgStorage.List(ctx, sessionID)
	if err != nil {
		return fmt.Sprintf("Failed to get messages: %s", err.Error())
	}

	// Find last assistant message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Data.Role == "assistant" {
			// Get parts for this message
			partStorage := database.NewPartStorage(t.db)
			parts, err := partStorage.ListByMessage(ctx, messages[i].ID)
			if err != nil {
				continue
			}

			// Find last text part
			for j := len(parts) - 1; j >= 0; j-- {
				if parts[j].Data.Type == "text" {
					return parts[j].Data.Text
				}
			}
		}
	}

	return "No result text found"
}

// TaskProcessRequest implements ProcessRequestInterface
type TaskProcessRequest struct {
	SessionID    string
	UserMessage  string
	Model        provider.ModelID
	Agent        string
	SystemPrompt string
}

func (r *TaskProcessRequest) GetSessionID() string       { return r.SessionID }
func (r *TaskProcessRequest) GetUserMessage() string     { return r.UserMessage }
func (r *TaskProcessRequest) GetModel() provider.ModelID { return r.Model }
func (r *TaskProcessRequest) GetAgent() string           { return r.Agent }
func (r *TaskProcessRequest) GetSystemPrompt() string    { return r.SystemPrompt }