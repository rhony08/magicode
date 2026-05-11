// Package agent provides agent registry for managing specialized AI agents.
package agent

import (
	"context"
	"sync"

	"github.com/rhony08/magicode/internal/opencode"
	"github.com/rhony08/magicode/internal/provider"
	"github.com/rhony08/magicode/internal/util/log"
)

// Info represents agent configuration
type Info struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description,omitempty"`
	Mode        string                       `json:"mode"` // "subagent", "primary", "all"
	Hidden      bool                         `json:"hidden,omitempty"`
	Temperature float64                      `json:"temperature,omitempty"`
	Color       string                       `json:"color,omitempty"`
	Permission  []PermissionRule             `json:"permission,omitempty"`
	Model       *ModelRef                    `json:"model,omitempty"`
	Prompt      string                       `json:"prompt,omitempty"`
	Tools       map[string]bool              `json:"tools,omitempty"`
	Options     map[string]interface{}       `json:"options,omitempty"`
}

// PermissionRule represents a permission rule for an agent
type PermissionRule struct {
	Permission string `json:"permission"`
	Pattern    string `json:"pattern"`
	Action     string `json:"action"` // "allow", "deny", "ask"
}

// ModelRef references a specific model for an agent
type ModelRef struct {
	ModelID    provider.ModelID    `json:"model_id"`
	ProviderID provider.ProviderID `json:"provider_id"`
}

// Registry manages available agents
type Registry struct {
	agents  map[string]*Info
	config  *opencode.ConfigReader
	mu      sync.RWMutex
	logger  *log.Logger
}

// NewRegistry creates a new agent registry
func NewRegistry(config *opencode.ConfigReader) *Registry {
	return &Registry{
		agents: make(map[string]*Info),
		config: config,
		logger: log.Create(map[string]string{"service": "agent.registry"}),
	}
}

// Load loads agents from OpenCode config
func (r *Registry) Load(ctx context.Context) error {
	if r.config == nil {
		return nil
	}

	agents, err := r.config.ReadAgents()
	if err != nil {
		r.logger.Warn("Failed to load agents from OpenCode config", map[string]interface{}{
			"error": err.Error(),
		})
		return nil // Non-fatal, continue with empty registry
	}

	r.mu.Lock()
	for _, agent := range agents {
		info := r.convertAgentToInfo(agent.Name, agent)
		r.agents[agent.Name] = info
	}
	r.mu.Unlock()

	r.logger.Info("Loaded agents", map[string]interface{}{
		"count": len(agents),
	})
	return nil
}

// convertAgentToInfo converts OpenCode agent to agent.Info
func (r *Registry) convertAgentToInfo(name string, agent opencode.Agent) *Info {
	info := &Info{
		Name:        name,
		Description: agent.Description,
		Mode:        agent.Mode,
		Hidden:      agent.Hidden,
		Temperature: agent.Temperature,
		Color:       agent.Color,
		Prompt:      agent.Content,
		Tools:       agent.Tools,
	}

	// Convert permissions
	info.Permission = make([]PermissionRule, 0)
	for perm, patterns := range agent.Permission {
		for pattern, action := range patterns {
			info.Permission = append(info.Permission, PermissionRule{
				Permission: perm,
				Pattern:    pattern,
				Action:     action,
			})
		}
	}

	return info
}

// Get retrieves an agent by name
func (r *Registry) Get(name string) (*Info, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, ok := r.agents[name]
	return info, ok
}

// List returns all available agents
func (r *Registry) List() []*Info {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Info, 0, len(r.agents))
	for _, info := range r.agents {
		if !info.Hidden {
			result = append(result, info)
		}
	}
	return result
}

// ListAll returns all agents including hidden ones
func (r *Registry) ListAll() []*Info {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Info, 0, len(r.agents))
	for _, info := range r.agents {
		result = append(result, info)
	}
	return result
}

// ListSubagents returns agents that can be used as subagents
func (r *Registry) ListSubagents() []*Info {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Info, 0)
	for _, info := range r.agents {
		if !info.Hidden && (info.Mode == "subagent" || info.Mode == "all") {
			result = append(result, info)
		}
	}
	return result
}

// DefaultAgent returns the default agent name
func (r *Registry) DefaultAgent() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Find first non-hidden primary agent
	for name, info := range r.agents {
		if !info.Hidden && info.Mode == "primary" {
			return name
		}
	}

	// Fallback to first available agent
	for name, info := range r.agents {
		if !info.Hidden {
			return name
		}
	}

	return "code" // Ultimate fallback
}

// Add adds a new agent to the registry
func (r *Registry) Add(info *Info) {
	r.mu.Lock()
	r.agents[info.Name] = info
	r.mu.Unlock()
}

// Remove removes an agent from the registry
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	delete(r.agents, name)
	r.mu.Unlock()
}

// SetModel sets the model for an agent
func (r *Registry) SetModel(name string, model *ModelRef) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, ok := r.agents[name]
	if !ok {
		return nil // Agent not found, non-fatal
	}

	info.Model = model
	return nil
}

// HasPermission checks if an agent has a specific permission
func (r *Registry) HasPermission(agentName, permission string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, ok := r.agents[agentName]
	if !ok {
		return false
	}

	for _, rule := range info.Permission {
		if rule.Permission == permission && rule.Action == "allow" {
			return true
		}
	}

	return false
}

// CanTask checks if agent can spawn subtasks
func (r *Registry) CanTask(agentName string) bool {
	return r.HasPermission(agentName, "task")
}

// CanTodo checks if agent can use todowrite
func (r *Registry) CanTodo(agentName string) bool {
	return r.HasPermission(agentName, "todowrite")
}