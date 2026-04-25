// Package opencode provides compatibility with OpenCode configuration files
package opencode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfigReader(t *testing.T) {
	reader := NewConfigReader("/config", "/state")

	if reader.configDir != "/config" {
		t.Errorf("configDir = %s, want /config", reader.configDir)
	}

	if reader.stateDir != "/state" {
		t.Errorf("stateDir = %s, want /state", reader.stateDir)
	}
}

func TestDefaultConfigReader(t *testing.T) {
	reader := DefaultConfigReader()

	if reader.configDir == "" {
		t.Error("configDir should not be empty")
	}

	if reader.stateDir == "" {
		t.Error("stateDir should not be empty")
	}
}

func TestReadAgentFile(t *testing.T) {
	// Create temp directory with test agent file
	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agentsDir, 0755)

	// Create test agent file
	agentContent := `---
description: "Test agent for testing"
mode: subagent
temperature: 0.5
color: "#FF0000"
tools:
  write: true
  edit: true
permission:
  bash:
    "*": allow
---

# Test Agent

This is the test agent content.
`
	agentPath := filepath.Join(agentsDir, "test.md")
	os.WriteFile(agentPath, []byte(agentContent), 0644)

	reader := NewConfigReader(tmpDir, tmpDir)
	agents, err := reader.ReadAgents()
	if err != nil {
		t.Fatalf("ReadAgents() error = %v", err)
	}

	if len(agents) != 1 {
		t.Errorf("ReadAgents() returned %d agents, want 1", len(agents))
	}

	agent := agents[0]
	if agent.Name != "test" {
		t.Errorf("agent.Name = %s, want test", agent.Name)
	}

	if agent.Description != "Test agent for testing" {
		t.Errorf("agent.Description = %s, want 'Test agent for testing'", agent.Description)
	}

	if agent.Mode != "subagent" {
		t.Errorf("agent.Mode = %s, want subagent", agent.Mode)
	}

	if agent.Temperature != 0.5 {
		t.Errorf("agent.Temperature = %f, want 0.5", agent.Temperature)
	}

	if agent.Color != "#FF0000" {
		t.Errorf("agent.Color = %s, want #FF0000", agent.Color)
	}

	if len(agent.Tools) != 2 {
		t.Errorf("agent.Tools has %d entries, want 2", len(agent.Tools))
	}
}

func TestReadAgentsEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	reader := NewConfigReader(tmpDir, tmpDir)

	agents, err := reader.ReadAgents()
	if err != nil {
		t.Fatalf("ReadAgents() error = %v", err)
	}

	if agents != nil {
		t.Errorf("ReadAgents() returned %v, want nil for non-existent dir", agents)
	}
}

func TestReadConfig(t *testing.T) {
	// Create temp directory with test config
	tmpDir := t.TempDir()

	configContent := `{
		"$schema": "https://opencode.ai/config.json",
		"provider": {
			"test-provider": {
				"npm": "@ai-sdk/test",
				"name": "Test Provider",
				"baseURL": "https://test.example.com",
				"apiKey": "test-key",
				"models": {
					"test-model": {
						"name": "Test Model",
						"modalities": {
							"input": ["text"],
							"output": ["text"]
						},
						"limit": {
							"context": 100000,
							"output": 4096
						}
					}
				}
			}
		}
	}`

	configPath := filepath.Join(tmpDir, "opencode.json")
	os.WriteFile(configPath, []byte(configContent), 0644)

	reader := NewConfigReader(tmpDir, tmpDir)
	config, err := reader.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() error = %v", err)
	}

	if config == nil {
		t.Fatal("ReadConfig() returned nil")
	}

	if len(config.Providers) != 1 {
		t.Errorf("ReadConfig() returned %d providers, want 1", len(config.Providers))
	}

	provider, ok := config.Providers["test-provider"]
	if !ok {
		t.Error("Provider 'test-provider' not found")
	}

	if provider.Name != "Test Provider" {
		t.Errorf("provider.Name = %s, want 'Test Provider'", provider.Name)
	}

	if len(provider.Models) != 1 {
		t.Errorf("provider has %d models, want 1", len(provider.Models))
	}

	model, ok := provider.Models["test-model"]
	if !ok {
		t.Error("Model 'test-model' not found")
	}

	if model.Name != "Test Model" {
		t.Errorf("model.Name = %s, want 'Test Model'", model.Name)
	}
}

func TestReadConfigNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	reader := NewConfigReader(tmpDir, tmpDir)

	config, err := reader.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() error = %v", err)
	}

	if config != nil {
		t.Errorf("ReadConfig() returned %v, want nil for non-existent file", config)
	}
}

func TestReadProviders(t *testing.T) {
	// Create temp directory with test config
	tmpDir := t.TempDir()

	configContent := `{
		"$schema": "https://opencode.ai/config.json",
		"provider": {
			"provider-1": {
				"npm": "@ai-sdk/one",
				"name": "Provider One",
				"models": {}
			},
			"provider-2": {
				"npm": "@ai-sdk/two",
				"name": "Provider Two",
				"models": {}
			}
		}
	}`

	configPath := filepath.Join(tmpDir, "opencode.json")
	os.WriteFile(configPath, []byte(configContent), 0644)

	reader := NewConfigReader(tmpDir, tmpDir)
	providers, err := reader.ReadProviders()
	if err != nil {
		t.Fatalf("ReadProviders() error = %v", err)
	}

	if len(providers) != 2 {
		t.Errorf("ReadProviders() returned %d providers, want 2", len(providers))
	}
}

func TestReadModelState(t *testing.T) {
	// Create temp directory with test state
	tmpDir := t.TempDir()

	stateContent := `{
		"recent": [
			{"providerID": "test", "modelID": "model1"},
			{"providerID": "test", "modelID": "model2"}
		],
		"favorite": [
			{"providerID": "test", "modelID": "fav1"}
		],
		"variant": {
			"session1": "variant1"
		}
	}`

	stateDir := filepath.Join(tmpDir, "state")
	os.MkdirAll(stateDir, 0755)
	statePath := filepath.Join(stateDir, "model.json")
	os.WriteFile(statePath, []byte(stateContent), 0644)

	reader := NewConfigReader(tmpDir, stateDir)
	state, err := reader.ReadModelState()
	if err != nil {
		t.Fatalf("ReadModelState() error = %v", err)
	}

	if state == nil {
		t.Fatal("ReadModelState() returned nil")
	}

	if len(state.Recent) != 2 {
		t.Errorf("state.Recent has %d items, want 2", len(state.Recent))
	}

	if len(state.Favorite) != 1 {
		t.Errorf("state.Favorite has %d items, want 1", len(state.Favorite))
	}

	if len(state.Variant) != 1 {
		t.Errorf("state.Variant has %d items, want 1", len(state.Variant))
	}
}

func TestReadModelStateNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	reader := NewConfigReader(tmpDir, tmpDir)

	state, err := reader.ReadModelState()
	if err != nil {
		t.Fatalf("ReadModelState() error = %v", err)
	}

	if state == nil {
		t.Fatal("ReadModelState() returned nil")
	}

	// Should return empty state, not nil
	if len(state.Recent) != 0 {
		t.Errorf("state.Recent has %d items, want 0", len(state.Recent))
	}
}

func TestGetFirstValidModel(t *testing.T) {
	// Create temp directory with test config
	tmpDir := t.TempDir()

	configContent := `{
		"$schema": "https://opencode.ai/config.json",
		"provider": {
			"test-provider": {
				"npm": "@ai-sdk/test",
				"name": "Test Provider",
				"models": {
					"test-model": {
						"name": "Test Model",
						"modalities": {
							"input": ["text"],
							"output": ["text"]
						},
						"limit": {
							"context": 100000,
							"output": 4096
						}
					}
				}
			}
		}
	}`

	configPath := filepath.Join(tmpDir, "opencode.json")
	os.WriteFile(configPath, []byte(configContent), 0644)

	reader := NewConfigReader(tmpDir, tmpDir)
	modelRef, err := reader.GetFirstValidModel()
	if err != nil {
		t.Fatalf("GetFirstValidModel() error = %v", err)
	}

	if modelRef == nil {
		t.Fatal("GetFirstValidModel() returned nil")
	}

	if modelRef.ProviderID != "test-provider" {
		t.Errorf("modelRef.ProviderID = %s, want test-provider", modelRef.ProviderID)
	}

	if modelRef.ModelID != "test-model" {
		t.Errorf("modelRef.ModelID = %s, want test-model", modelRef.ModelID)
	}
}

func TestConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	reader := NewConfigReader(tmpDir, tmpDir)

	// Should return false when no config exists
	if reader.ConfigExists() {
		t.Error("ConfigExists() should return false when no config exists")
	}

	// Create config file
	configPath := filepath.Join(tmpDir, "opencode.json")
	os.WriteFile(configPath, []byte("{}"), 0644)

	// Should return true when config exists
	if !reader.ConfigExists() {
		t.Error("ConfigExists() should return true when config exists")
	}
}
