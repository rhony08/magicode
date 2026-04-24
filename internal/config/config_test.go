package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDefault(t *testing.T) {
	cfg := NewDefault()

	if cfg == nil {
		t.Fatal("Expected config service to be created")
	}

	info := cfg.Get()
	if info == nil {
		t.Fatal("Expected config info to be returned")
	}

	// Check defaults
	if info.LogLevel != "INFO" {
		t.Errorf("Expected LogLevel=INFO, got %s", info.LogLevel)
	}

	if info.Username == "" {
		t.Error("Expected Username to be set")
	}

	if info.Model != "anthropic/claude-sonnet-4-5" {
		t.Errorf("Expected Model=anthropic/claude-sonnet-4-5, got %s", info.Model)
	}

	if info.Snapshot == nil || *info.Snapshot != true {
		t.Error("Expected Snapshot=true")
	}
}

func TestNewWithDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := t.TempDir()

	// Create a config file
	configPath := filepath.Join(tmpDir, "magicode.json")
	configContent := `{
		"model": "anthropic/claude-opus-4",
		"username": "testuser",
		"logLevel": "DEBUG"
	}`
	os.WriteFile(configPath, []byte(configContent), 0644)

	cfg, err := New(tmpDir, globalDir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	info := cfg.Get()

	if info.Model != "anthropic/claude-opus-4" {
		t.Errorf("Expected Model=anthropic/claude-opus-4, got %s", info.Model)
	}

	if info.Username != "testuser" {
		t.Errorf("Expected Username=testuser, got %s", info.Username)
	}

	if info.LogLevel != "DEBUG" {
		t.Errorf("Expected LogLevel=DEBUG, got %s", info.LogLevel)
	}
}

func TestLoadJSONC(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a JSONC config file with comments
	configPath := filepath.Join(tmpDir, "magicode.jsonc")
	configContent := `{
		// This is a comment
		"model": "openai/gpt-4",
		"username": "jsoncuser", /* block comment */
	}`
	os.WriteFile(configPath, []byte(configContent), 0644)

	cfg, err := New(tmpDir, t.TempDir())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	info := cfg.Get()

	if info.Model != "openai/gpt-4" {
		t.Errorf("Expected Model=openai/gpt-4, got %s", info.Model)
	}

	if info.Username != "jsoncuser" {
		t.Errorf("Expected Username=jsoncuser, got %s", info.Username)
	}
}

func TestMergeConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := t.TempDir()

	// Global config
	globalConfig := filepath.Join(globalDir, "magicode.json")
	globalContent := `{
		"model": "anthropic/claude-sonnet-4-5",
		"username": "globaluser",
		"instructions": ["global instruction"]
	}`
	os.WriteFile(globalConfig, []byte(globalContent), 0644)

	// Project config
	projectConfig := filepath.Join(tmpDir, "magicode.json")
	projectContent := `{
		"model": "anthropic/claude-opus-4",
		"instructions": ["project instruction"],
		"provider": {
			"anthropic": {
				"env": ["ANTHROPIC_API_KEY"]
			}
		}
	}`
	os.WriteFile(projectConfig, []byte(projectContent), 0644)

	cfg, err := New(tmpDir, globalDir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	info := cfg.Get()

	// Project should override global for model
	if info.Model != "anthropic/claude-opus-4" {
		t.Errorf("Expected Model from project, got %s", info.Model)
	}

	// Instructions should be merged
	if len(info.Instructions) != 2 {
		t.Errorf("Expected 2 instructions, got %d", len(info.Instructions))
	}

	// Providers should be merged
	if info.Providers == nil || info.Providers["anthropic"].Env == nil {
		t.Error("Expected anthropic provider to be set")
	}
}

func TestModel(t *testing.T) {
	cfg := NewDefault()

	// Test default model
	model := cfg.Model()
	if model != "anthropic/claude-sonnet-4-5" {
		t.Errorf("Expected default model, got %s", model)
	}

	// Test with custom model
	cfg.config.Model = "custom/model"
	model = cfg.Model()
	if model != "custom/model" {
		t.Errorf("Expected custom model, got %s", model)
	}
}

func TestSmallModel(t *testing.T) {
	cfg := NewDefault()

	// Test default small model
	model := cfg.SmallModel()
	if model != "anthropic/claude-3-5-haiku" {
		t.Errorf("Expected default small model, got %s", model)
	}
}

func TestParseModel(t *testing.T) {
	tests := []struct {
		input       string
		expectedProv string
		expectedMod string
		hasError    bool
	}{
		{"anthropic/claude-sonnet-4-5", "anthropic", "claude-sonnet-4-5", false},
		{"openai/gpt-4", "openai", "gpt-4", false},
		{"invalid", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		prov, mod, err := ParseModel(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("ParseModel(%s) expected error", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseModel(%s) unexpected error: %v", tt.input, err)
			}
			if prov != tt.expectedProv {
				t.Errorf("ParseModel(%s) provider=%s, expected %s", tt.input, prov, tt.expectedProv)
			}
			if mod != tt.expectedMod {
				t.Errorf("ParseModel(%s) model=%s, expected %s", tt.input, mod, tt.expectedMod)
			}
		}
	}
}

func TestMergeStringArrays(t *testing.T) {
	a := []string{"a", "b"}
	b := []string{"b", "c", "d"}

	result := mergeStringArrays(a, b)

	// Should dedupe and preserve order
	expected := []string{"a", "b", "c", "d"}
	if len(result) != len(expected) {
		t.Errorf("Expected %d elements, got %d", len(expected), len(result))
	}

	for i, v := range expected {
		if result[i] != v {
			t.Errorf("Expected result[%d]=%s, got %s", i, v, result[i])
		}
	}
}

func TestMergeMaps(t *testing.T) {
	a := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	b := map[string]interface{}{
		"key2": "newvalue",
		"key3": "value3",
	}

	result := mergeMaps(a, b)

	if result["key1"] != "value1" {
		t.Error("Expected key1=value1")
	}
	if result["key2"] != "newvalue" {
		t.Error("Expected key2=newvalue (b should override)")
	}
	if result["key3"] != "value3" {
		t.Error("Expected key3=value3")
	}
}

func TestBoolPtr(t *testing.T) {
	ptr := boolPtr(true)
	if ptr == nil || *ptr != true {
		t.Error("Expected boolPtr(true) to return *true")
	}

	ptr = boolPtr(false)
	if ptr == nil || *ptr != false {
		t.Error("Expected boolPtr(false) to return *false")
	}
}

func TestConfigPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file
	configPath := filepath.Join(tmpDir, "magicode.json")
	os.WriteFile(configPath, []byte("{}"), 0644)

	cfg, err := New(tmpDir, t.TempDir())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	path := cfg.ConfigPath()
	if path != configPath {
		t.Errorf("Expected ConfigPath=%s, got %s", configPath, path)
	}
}

func TestDotOpencodeDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .magicode directory with config
magicodeDir := filepath.Join(tmpDir, ".magicode")

	os.MkdirAll(magicodeDir, 0755)

	configPath := filepath.Join(magicodeDir, "magicode.json")
	configContent := `{
		"model": "anthropic/claude-opus-4",
		"username": "dotmagicodeuser"
	}`
	os.WriteFile(configPath, []byte(configContent), 0644)

	cfg, err := New(tmpDir, t.TempDir())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	info := cfg.Get()
	if info.Model != "anthropic/claude-opus-4" {
		t.Errorf("Expected Model from .magicode, got %s", info.Model)
	}
}