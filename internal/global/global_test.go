package global

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	// Reset before testing
	Reset()

	err := Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if !IsInitialized() {
		t.Error("Expected IsInitialized to be true")
	}

	// Check that paths are set
	if Path.Data == "" {
		t.Error("Expected Data path to be set")
	}
	if Path.Config == "" {
		t.Error("Expected Config path to be set")
	}
	if Path.State == "" {
		t.Error("Expected State path to be set")
	}
	if Path.Cache == "" {
		t.Error("Expected Cache path to be set")
	}

	// Check that directories exist
	for _, dir := range []string{Path.Data, Path.Config, Path.State, Path.Cache} {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Expected directory %s to exist", dir)
		}
	}

	// Call Init again - should not fail
	err = Init()
	if err != nil {
		t.Fatalf("Second Init failed: %v", err)
	}
}

func TestInitWithDir(t *testing.T) {
	Reset()

	// Create temp directories
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	configDir := filepath.Join(tmpDir, "config")
	stateDir := filepath.Join(tmpDir, "state")
	cacheDir := filepath.Join(tmpDir, "cache")

	err := InitWithDir(dataDir, configDir, stateDir, cacheDir)
	if err != nil {
		t.Fatalf("InitWithDir failed: %v", err)
	}

	// Check that custom paths are set
	if Path.Data != dataDir {
		t.Errorf("Expected Data=%s, got %s", dataDir, Path.Data)
	}
	if Path.Config != configDir {
		t.Errorf("Expected Config=%s, got %s", configDir, Path.Config)
	}
	if Path.State != stateDir {
		t.Errorf("Expected State=%s, got %s", stateDir, Path.State)
	}
	if Path.Cache != cacheDir {
		t.Errorf("Expected Cache=%s, got %s", cacheDir, Path.Cache)
	}
}

func TestDatabasePath(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()

	err := InitWithDir(tmpDir, tmpDir, tmpDir, tmpDir)
	if err != nil {
		t.Fatalf("InitWithDir failed: %v", err)
	}

	dbPath := DatabasePath()
	expected := filepath.Join(tmpDir, "magicode.db")
	if dbPath != expected {
		t.Errorf("Expected DatabasePath=%s, got %s", expected, dbPath)
	}
}

func TestConfigFile(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()

	err := InitWithDir(tmpDir, tmpDir, tmpDir, tmpDir)
	if err != nil {
		t.Fatalf("InitWithDir failed: %v", err)
	}

	// Test default path when no config exists
	configPath := ConfigFile()
	expected := filepath.Join(tmpDir, "magicode.jsonc")
	if configPath != expected {
		t.Errorf("Expected ConfigFile=%s, got %s", expected, configPath)
	}

	// Test with existing magicode.json
	jsonPath := filepath.Join(tmpDir, "magicode.json")
	os.WriteFile(jsonPath, []byte("{}"), 0644)

	configPath = ConfigFile()
	if configPath != jsonPath {
		t.Errorf("Expected ConfigFile=%s, got %s", jsonPath, configPath)
	}
}

func TestLogFile(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()

	err := InitWithDir(tmpDir, tmpDir, tmpDir, tmpDir)
	if err != nil {
		t.Fatalf("InitWithDir failed: %v", err)
	}

	logPath := LogFile()
	expected := filepath.Join(tmpDir, "magicode.log")
	if logPath != expected {
		t.Errorf("Expected LogFile=%s, got %s", expected, logPath)
	}
}

func TestPlansPath(t *testing.T) {
	Reset()
	tmpDir := t.TempDir()

	err := InitWithDir(tmpDir, tmpDir, tmpDir, tmpDir)
	if err != nil {
		t.Fatalf("InitWithDir failed: %v", err)
	}

	plansPath := PlansPath()
	expected := filepath.Join(tmpDir, "plans")
	if plansPath != expected {
		t.Errorf("Expected PlansPath=%s, got %s", expected, plansPath)
	}
}

func TestReset(t *testing.T) {
	// First initialize
	err := Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Then reset
	Reset()

	if IsInitialized() {
		t.Error("Expected IsInitialized to be false after Reset")
	}

	if Path.Data != "" || Path.Config != "" || Path.State != "" || Path.Cache != "" {
		t.Error("Expected all paths to be empty after Reset")
	}
}