// Package opencode provides compatibility with OpenCode configuration files
package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rhony08/magicode/internal/util/log"
	"gopkg.in/yaml.v3"
)

// Agent represents an OpenCode agent definition
type Agent struct {
	Name        string                       `json:"name" yaml:"-"`
	Description string                       `json:"description" yaml:"description"`
	Mode        string                       `json:"mode" yaml:"mode"`
	Temperature float64                      `json:"temperature" yaml:"temperature"`
	Color       string                       `json:"color" yaml:"color"`
	Tools       map[string]bool              `json:"tools" yaml:"tools"`
	Permission  map[string]map[string]string `json:"permission" yaml:"permission"`
	Content     string                       `json:"content" yaml:"-"` // The markdown content after frontmatter
	Hidden      bool                         `json:"hidden" yaml:"hidden"`
}

// Provider represents an OpenCode provider configuration
type Provider struct {
	ID      string           `json:"id"`
	NPM     string           `json:"npm"`
	Name    string           `json:"name"`
	BaseURL string           `json:"baseURL"`
	APIKey  string           `json:"apiKey"`
	Models  map[string]Model `json:"models"`
}

// Model represents a model configuration from OpenCode
type Model struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Modalities ModelModalities `json:"modalities"`
	Options    ModelOptions    `json:"options"`
	Limit      ModelLimit      `json:"limit"`
}

// ModelModalities represents input/output modalities
type ModelModalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

// ModelOptions represents model-specific options
type ModelOptions struct {
	Thinking *ThinkingOptions `json:"thinking,omitempty"`
}

// ThinkingOptions represents thinking/reasoning options
type ThinkingOptions struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budgetTokens"`
}

// ModelLimit represents model limits
type ModelLimit struct {
	Context int `json:"context"`
	Output  int `json:"output"`
}

// Config represents the OpenCode opencode.json configuration
type Config struct {
	Schema    string              `json:"$schema"`
	Providers map[string]Provider `json:"provider"`
	// Other fields can be added as needed
}

// ModelState represents the persisted model state (recent/favorite models)
type ModelState struct {
	Recent   []ModelRef        `json:"recent"`
	Favorite []ModelRef        `json:"favorite"`
	Variant  map[string]string `json:"variant"` // sessionID -> variant
}

// ModelRef represents a model reference
type ModelRef struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

// ConfigReader reads OpenCode configuration files
type ConfigReader struct {
	configDir string
	stateDir  string
}

// NewConfigReader creates a new OpenCode config reader
func NewConfigReader(configDir, stateDir string) *ConfigReader {
	return &ConfigReader{
		configDir: configDir,
		stateDir:  stateDir,
	}
}

// DefaultConfigReader creates a config reader with default paths
func DefaultConfigReader() *ConfigReader {
	homeDir, _ := os.UserHomeDir()
	return &ConfigReader{
		configDir: filepath.Join(homeDir, ".config", "opencode"),
		stateDir:  filepath.Join(homeDir, ".local", "share", "opencode", "state"),
	}
}

// ReadAgents reads all agent definitions from the agents directory
func (r *ConfigReader) ReadAgents() ([]Agent, error) {
	agentsDir := filepath.Join(r.configDir, "agents")

	// Check if directory exists
	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		log.Info("OpenCode agents directory not found", "path", agentsDir)
		return nil, nil
	}

	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read agents directory: %w", err)
	}

	var agents []Agent
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		agent, err := r.readAgentFile(filepath.Join(agentsDir, entry.Name()))
		if err != nil {
			log.Warn("Failed to read agent file", "file", entry.Name(), "error", err)
			continue
		}

		// Extract name from filename
		agent.Name = strings.TrimSuffix(entry.Name(), ".md")
		agents = append(agents, *agent)
	}

	log.Info("Read OpenCode agents", "count", len(agents))
	return agents, nil
}

// readAgentFile reads a single agent markdown file
func (r *ConfigReader) readAgentFile(path string) (*Agent, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse YAML frontmatter
	var agent Agent
	if err := yaml.Unmarshal(content, &agent); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	// Extract content after frontmatter
	parts := strings.SplitN(string(content), "---", 3)
	if len(parts) >= 3 {
		agent.Content = strings.TrimSpace(parts[2])
	}

	return &agent, nil
}

// ReadConfig reads the main opencode.json configuration
func (r *ConfigReader) ReadConfig() (*Config, error) {
	configPath := filepath.Join(r.configDir, "opencode.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("OpenCode config not found", "path", configPath)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set provider IDs from keys
	providers := make(map[string]Provider)
	for id, provider := range config.Providers {
		provider.ID = id
		// Set model IDs from keys
		models := make(map[string]Model)
		for modelID, model := range provider.Models {
			model.ID = modelID
			models[modelID] = model
		}
		provider.Models = models
		providers[id] = provider
	}
	config.Providers = providers

	log.Info("Read OpenCode config", "providers", len(config.Providers))
	return &config, nil
}

// ReadProviders returns providers as a slice
func (r *ConfigReader) ReadProviders() ([]Provider, error) {
	config, err := r.ReadConfig()
	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, nil
	}

	providers := make([]Provider, 0, len(config.Providers))
	for _, provider := range config.Providers {
		providers = append(providers, provider)
	}

	return providers, nil
}

// ReadModelState reads the persisted model state
func (r *ConfigReader) ReadModelState() (*ModelState, error) {
	statePath := filepath.Join(r.stateDir, "model.json")

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("OpenCode model state not found", "path", statePath)
			return &ModelState{
				Recent:   []ModelRef{},
				Favorite: []ModelRef{},
				Variant:  make(map[string]string),
			}, nil
		}
		return nil, fmt.Errorf("failed to read model state: %w", err)
	}

	var state ModelState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse model state: %w", err)
	}

	if state.Variant == nil {
		state.Variant = make(map[string]string)
	}

	log.Info("Read OpenCode model state", "recent", len(state.Recent), "favorite", len(state.Favorite))
	return &state, nil
}

// GetFirstValidModel returns the first valid model from the config
func (r *ConfigReader) GetFirstValidModel() (*ModelRef, error) {
	config, err := r.ReadConfig()
	if err != nil {
		return nil, err
	}

	if config == nil || len(config.Providers) == 0 {
		return nil, nil
	}

	// Get first provider
	for providerID, provider := range config.Providers {
		if len(provider.Models) > 0 {
			// Get first model
			for modelID := range provider.Models {
				return &ModelRef{
					ProviderID: providerID,
					ModelID:    modelID,
				}, nil
			}
		}
	}

	return nil, nil
}

// ConfigExists checks if OpenCode configuration exists
func (r *ConfigReader) ConfigExists() bool {
	configPath := filepath.Join(r.configDir, "opencode.json")
	_, err := os.Stat(configPath)
	return !os.IsNotExist(err)
}
