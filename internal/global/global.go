package global

import (
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

// Paths contains global application paths
type Paths struct {
	Data   string // Data directory (sessions, database)
	Config string // Config directory (opencode.json)
	State  string // State directory (logs, temp files)
	Cache  string // Cache directory
}

var (
	// Path holds the initialized global paths
	Path Paths

	// initialized tracks if Init was called
	initialized bool
)

// Init initializes global paths based on XDG standards
func Init() error {
	if initialized {
		return nil
	}

	// Use XDG Base Directory Specification
	// Data: ~/.local/share/opencode (or $XDG_DATA_HOME/opencode)
	// Config: ~/.config/opencode (or $XDG_CONFIG_HOME/opencode)
	// State: ~/.local/state/opencode (or $XDG_STATE_HOME/opencode)
	// Cache: ~/.cache/opencode (or $XDG_CACHE_HOME/opencode)

	appName := "opencode"

	Path = Paths{
		Data:   filepath.Join(xdg.DataHome, appName),
		Config: filepath.Join(xdg.ConfigHome, appName),
		State:  filepath.Join(xdg.StateHome, appName),
		Cache:  filepath.Join(xdg.CacheHome, appName),
	}

	// Ensure directories exist
	dirs := []string{Path.Data, Path.Config, Path.State, Path.Cache}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	initialized = true
	return nil
}

// InitWithDir initializes with custom directories (for testing)
func InitWithDir(dataDir, configDir, stateDir, cacheDir string) error {
	Path = Paths{
		Data:   dataDir,
		Config: configDir,
		State:  stateDir,
		Cache:  cacheDir,
	}

	// Ensure directories exist
	dirs := []string{Path.Data, Path.Config, Path.State, Path.Cache}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	initialized = true
	return nil
}

// Reset resets the global state (for testing)
func Reset() {
	initialized = false
	Path = Paths{}
}

// IsInitialized returns whether Init has been called
func IsInitialized() bool {
	return initialized
}

// DatabasePath returns the path to the SQLite database
func DatabasePath() string {
	return filepath.Join(Path.Data, "opencode.db")
}

// ConfigFile returns the path to the main config file
func ConfigFile() string {
	// Check for existing config files in order of preference
	candidates := []string{
		filepath.Join(Path.Config, "opencode.jsonc"),
		filepath.Join(Path.Config, "opencode.json"),
		filepath.Join(Path.Config, "config.json"),
	}

	for _, file := range candidates {
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}

	// Default to opencode.jsonc
	return filepath.Join(Path.Config, "opencode.jsonc")
}

// LogFile returns the path to the log file
func LogFile() string {
	return filepath.Join(Path.State, "opencode.log")
}

// PlansPath returns the path for storing plan files
func PlansPath() string {
	return filepath.Join(Path.Data, "plans")
}