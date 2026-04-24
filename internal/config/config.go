package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/jsonc"
)

// Info represents the configuration structure
type Info struct {
	Schema         string                 `json:"$schema,omitempty"`
	LogLevel       string                 `json:"logLevel,omitempty"`
	Model          string                 `json:"model,omitempty"`
	SmallModel     string                 `json:"small_model,omitempty"`
	DefaultAgent   string                 `json:"default_agent,omitempty"`
	Username       string                 `json:"username,omitempty"`
	Providers      map[string]Provider    `json:"provider,omitempty"`
	DisabledProviders []string            `json:"disabled_providers,omitempty"`
	EnabledProviders  []string            `json:"enabled_providers,omitempty"`
	Instructions   []string               `json:"instructions,omitempty"`
	Permission     map[string]interface{} `json:"permission,omitempty"`
	Agent          map[string]interface{} `json:"agent,omitempty"`
	MCP            map[string]interface{} `json:"mcp,omitempty"`
	Tools          map[string]bool        `json:"tools,omitempty"`
	Experimental   map[string]interface{} `json:"experimental,omitempty"`
	Compaction     map[string]interface{} `json:"compaction,omitempty"`
	Server         map[string]interface{} `json:"server,omitempty"`
	LSP            map[string]interface{} `json:"lsp,omitempty"`
	Formatter      map[string]interface{} `json:"formatter,omitempty"`
	Share          string                 `json:"share,omitempty"`
	Autoshare      bool                   `json:"autoshare,omitempty"`
	Autoupdate     interface{}            `json:"autoupdate,omitempty"`
	Snapshot       *bool                  `json:"snapshot,omitempty"`
}

// Provider represents provider configuration
type Provider struct {
	Name        string                 `json:"name,omitempty"`
	Env         []string               `json:"env,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
	Models      map[string]interface{} `json:"models,omitempty"`
	Blacklist   []string               `json:"blacklist,omitempty"`
	Whitelist   []string               `json:"whitelist,omitempty"`
}

// Service provides configuration access
type Service struct {
	config      *Info
	configPath  string
	directory   string
	globalDir   string // Global config directory
}

// New creates a new configuration service
func New(directory string, globalDir string) (*Service, error) {
	s := &Service{
		directory: directory,
		globalDir: globalDir,
	}

	if err := s.load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return s, nil
}

// NewDefault creates a configuration service with defaults only
func NewDefault() *Service {
	return &Service{
		config: defaultConfig(),
	}
}

func defaultConfig() *Info {
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME") // Windows
	}
	if username == "" {
		username = "unknown"
	}

	return &Info{
		LogLevel:     "INFO",
		Username:     username,
		Snapshot:     boolPtr(true),
		Model:        "anthropic/claude-sonnet-4-5",
		SmallModel:   "anthropic/claude-3-5-haiku",
		DefaultAgent: "build",
	}
}

func boolPtr(b bool) *bool {
	return &b
}

// load reads and merges configuration from all sources
func (s *Service) load() error {
	// Start with default config
	s.config = defaultConfig()

	// Load global config if directory specified
	if s.globalDir != "" {
		for _, file := range []string{"magicode.jsonc", "magicode.json", "config.json"} {
			path := filepath.Join(s.globalDir, file)
			if _, err := os.Stat(path); err == nil {
				if err := s.loadFile(path); err != nil {
					return fmt.Errorf("failed to load global config: %w", err)
				}
				break
			}
		}
	}

	// Load project config if directory specified
	if s.directory != "" {
		projectConfig := filepath.Join(s.directory, "magicode.json")
		if _, err := os.Stat(projectConfig); err == nil {
			if err := s.loadFile(projectConfig); err != nil {
				return err
			}
		}

		// Also check magicode.jsonc
		projectConfigC := filepath.Join(s.directory, "magicode.jsonc")
		if _, err := os.Stat(projectConfigC); err == nil {
			if err := s.loadFile(projectConfigC); err != nil {
				return err
			}
		}

		// Check .magicode directory
		magicodeDir := filepath.Join(s.directory, ".magicode")
		if stat, err := os.Stat(magicodeDir); err == nil && stat.IsDir() {
			for _, file := range []string{"magicode.json", "magicode.jsonc"} {
				path := filepath.Join(magicodeDir, file)
				if _, err := os.Stat(path); err == nil {
					if err := s.loadFile(path); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// loadFile loads and merges a config file
func (s *Service) loadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Convert JSONC to standard JSON
	jsonData := jsonc.ToJSON(data)

	var fileConfig Info
	if err := json.Unmarshal(jsonData, &fileConfig); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}

	// Merge into existing config
	s.merge(&fileConfig)
	s.configPath = path

	return nil
}

// merge merges a config into the existing one
func (s *Service) merge(other *Info) {
	if other.Schema != "" {
		s.config.Schema = other.Schema
	}
	if other.LogLevel != "" {
		s.config.LogLevel = other.LogLevel
	}
	if other.Model != "" {
		s.config.Model = other.Model
	}
	if other.SmallModel != "" {
		s.config.SmallModel = other.SmallModel
	}
	if other.DefaultAgent != "" {
		s.config.DefaultAgent = other.DefaultAgent
	}
	if other.Username != "" {
		s.config.Username = other.Username
	}
	if other.Share != "" {
		s.config.Share = other.Share
	}
	if other.Autoshare {
		s.config.Autoshare = other.Autoshare
	}
	if other.Autoupdate != nil {
		s.config.Autoupdate = other.Autoupdate
	}
	if other.Snapshot != nil {
		s.config.Snapshot = other.Snapshot
	}

	// Merge arrays
	s.config.Instructions = mergeStringArrays(s.config.Instructions, other.Instructions)
	s.config.DisabledProviders = mergeStringArrays(s.config.DisabledProviders, other.DisabledProviders)
	s.config.EnabledProviders = mergeStringArrays(s.config.EnabledProviders, other.EnabledProviders)

	// Merge maps
	s.config.Providers = mergeProviderMaps(s.config.Providers, other.Providers)
	s.config.Permission = mergeMaps(s.config.Permission, other.Permission)
	s.config.Agent = mergeMaps(s.config.Agent, other.Agent)
	s.config.MCP = mergeMaps(s.config.MCP, other.MCP)
	s.config.Tools = mergeBoolMaps(s.config.Tools, other.Tools)
	s.config.Experimental = mergeMaps(s.config.Experimental, other.Experimental)
	s.config.Compaction = mergeMaps(s.config.Compaction, other.Compaction)
	s.config.Server = mergeMaps(s.config.Server, other.Server)
	s.config.LSP = mergeMaps(s.config.LSP, other.LSP)
	s.config.Formatter = mergeMaps(s.config.Formatter, other.Formatter)
}

// Get returns the current configuration
func (s *Service) Get() *Info {
	return s.config
}

// Model returns the configured model
func (s *Service) Model() string {
	if s.config.Model != "" {
		return s.config.Model
	}
	return "anthropic/claude-sonnet-4-5"
}

// SmallModel returns the configured small model
func (s *Service) SmallModel() string {
	if s.config.SmallModel != "" {
		return s.config.SmallModel
	}
	return "anthropic/claude-3-5-haiku"
}

// ParseModel parses a model string (provider/model format)
func ParseModel(modelStr string) (provider, model string, err error) {
	parts := strings.SplitN(modelStr, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid model format: %s (expected provider/model)", modelStr)
	}
	return parts[0], parts[1], nil
}

// Helper functions for merging

func mergeStringArrays(a, b []string) []string {
	result := make([]string, 0)
	seen := make(map[string]bool)
	for _, s := range a {
		if !seen[s] {
			result = append(result, s)
			seen[s] = true
		}
	}
	for _, s := range b {
		if !seen[s] {
			result = append(result, s)
			seen[s] = true
		}
	}
	return result
}

func mergeProviderMaps(a, b map[string]Provider) map[string]Provider {
	result := make(map[string]Provider)
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		result[k] = v
	}
	return result
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		result[k] = v
	}
	return result
}

func mergeBoolMaps(a, b map[string]bool) map[string]bool {
	result := make(map[string]bool)
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		result[k] = v
	}
	return result
}

// ConfigPath returns the path of the loaded config file
func (s *Service) ConfigPath() string {
	return s.configPath
}