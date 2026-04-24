// Package tool provides the tool registry.
package tool

import (
	"context"
	"sync"
)

// Registry manages available tools
type Registry struct {
	tools map[ToolID]Tool
	mu    sync.RWMutex
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[ToolID]Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.ID()] = t
}

// Get retrieves a tool by ID
func (r *Registry) Get(id ToolID) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[id]
	return t, ok
}

// List returns all registered tools
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

// ListDefinitions returns all tool definitions for AI
func (r *Registry) ListDefinitions() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t.Definition())
	}
	return result
}

// Execute runs a tool by ID
func (r *Registry) Execute(id ToolID, ctx ToolContext, params map[string]interface{}) (*ToolResult, error) {
	t, ok := r.Get(id)
	if !ok {
		return nil, NewValidationError(id, "tool not found: "+string(id))
	}

	// Validate parameters first
	if err := t.Validate(params); err != nil {
		return nil, err
	}

	// Execute the tool
	execCtx, cancel := context.WithCancel(ctx.Abort)
	defer cancel()

	return t.Execute(execCtx, params, ctx)
}

// DefaultRegistry is the default global registry
var DefaultRegistry = NewRegistry()

// RegisterTool registers a tool in the default registry
func RegisterTool(t Tool) {
	DefaultRegistry.Register(t)
}

// GetTool retrieves a tool from the default registry
func GetTool(id ToolID) (Tool, bool) {
	return DefaultRegistry.Get(id)
}

// ListTools returns all tools from the default registry
func ListTools() []Tool {
	return DefaultRegistry.List()
}

// ExecuteTool runs a tool from the default registry
func ExecuteTool(id ToolID, ctx ToolContext, params map[string]interface{}) (*ToolResult, error) {
	return DefaultRegistry.Execute(id, ctx, params)
}