package global

import (
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

// Paths contains global application paths
type Paths struct {
	Data   string // Data directory (sessions, database)
	Config string // Config directory (magicode.json)
	State  string // State directory (logs)
	Cache  string // Cache directory
}

// CustomPaths allows overriding default paths
type CustomPaths struct {
	DataDir    string // Override data directory
	ConfigDir  string // Override config directory
	StateDir   string // Override state directory
	CacheDir   string // Override cache directory
	ConfigFile string // Override config file path
	Database   string // Override database file path
	LogFile    string // Override log file path
}

// Path is the global path instance
var Path Paths

// Custom is the custom paths override
var Custom CustomPaths

// initialized tracks whether Init has been called
var initialized bool

// appName is the application name (can be changed for backward compatibility)
var appName = "magicode"

// SetAppName sets the application name (for backward compatibility)
func SetAppName(name string) {
	appName = name
}

// GetAppName returns the current application name
func GetAppName() string {
	return appName
}

// Init initializes global paths using XDG standards
// Data: ~/.local/share/magicode (or $XDG_DATA_HOME/magicode)
// Config: ~/.config/magicode (or $XDG_CONFIG_HOME/magicode)
// State: ~/.local/state/magicode (or $XDG_STATE_HOME/magicode)
// Cache: ~/.cache/magicode (or $XDG_CACHE_HOME/magicode)
func Init() error {
	if initialized {
		return nil
	}

	// Use custom paths if provided
	dataDir := Custom.DataDir
	configDir := Custom.ConfigDir
	stateDir := Custom.StateDir
	cacheDir := Custom.CacheDir

	// Default to XDG paths if not overridden
	if dataDir == "" {
		dataDir = filepath.Join(xdg.DataHome, appName)
	}
	if configDir == "" {
		configDir = filepath.Join(xdg.ConfigHome, appName)
	}
	if stateDir == "" {
		stateDir = filepath.Join(xdg.StateHome, appName)
	}
	if cacheDir == "" {
		cacheDir = filepath.Join(xdg.CacheHome, appName)
	}

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

// InitWithCustom initializes with custom paths
func InitWithCustom(custom CustomPaths) error {
	Custom = custom
	return Init()
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
	Custom = CustomPaths{}
	appName = "magicode"
}

// IsInitialized returns whether Init has been called
func IsInitialized() bool {
	return initialized
}

// DatabasePath returns the path to the SQLite database
func DatabasePath() string {
	// Use custom path if provided
	if Custom.Database != "" {
		return Custom.Database
	}
	return filepath.Join(Path.Data, appName+".db")
}

// ConfigFile returns the path to the main config file
func ConfigFile() string {
	// Use custom path if provided
	if Custom.ConfigFile != "" {
		return Custom.ConfigFile
	}

	// Check for existing config files in order of preference
	candidates := []string{
		filepath.Join(Path.Config, appName+".jsonc"),
		filepath.Join(Path.Config, appName+".json"),
		filepath.Join(Path.Config, "config.json"),
	}

	for _, file := range candidates {
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}

	// Default to appName.jsonc
	return filepath.Join(Path.Config, appName+".jsonc")
}

// LogFile returns the path to the log file
func LogFile() string {
	// Use custom path if provided
	if Custom.LogFile != "" {
		return Custom.LogFile
	}
	return filepath.Join(Path.State, appName+".log")
}

// PlansPath returns the path for storing plan files
func PlansPath() string {
	return filepath.Join(Path.Data, "plans")
}

// BackwardCompatibility checks for existing opencode files and uses them
// This allows seamless migration from opencode to magicode
func BackwardCompatibility() bool {
	opencodeData := filepath.Join(xdg.DataHome, "opencode")
	opencodeConfig := filepath.Join(xdg.ConfigHome, "opencode")
	opencodeState := filepath.Join(xdg.StateHome, "opencode")

	// Check if opencode directories exist
	dataExists := dirExists(opencodeData)
	configExists := dirExists(opencodeConfig)
	stateExists := dirExists(opencodeState)

	// If opencode exists but magicode doesn't, use opencode paths
	magicodeData := filepath.Join(xdg.DataHome, "magicode")
	magicodeConfig := filepath.Join(xdg.ConfigHome, "magicode")
	magicodeState := filepath.Join(xdg.StateHome, "magicode")

	magicodeDataExists := dirExists(magicodeData)
	magicodeConfigExists := dirExists(magicodeConfig)
	magicodeStateExists := dirExists(magicodeState)

	// If opencode exists and magicode doesn't, switch to opencode
	if (dataExists || configExists || stateExists) &&
		!magicodeDataExists && !magicodeConfigExists && !magicodeStateExists {
		appName = "opencode"
		return true
	}

	return false
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return stat.IsDir()
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}

// GetAllPaths returns all important paths as a map
func GetAllPaths() map[string]string {
	return map[string]string{
		"data_dir":    Path.Data,
		"config_dir":  Path.Config,
		"state_dir":   Path.State,
		"cache_dir":   Path.Cache,
		"database":    DatabasePath(),
		"config_file": ConfigFile(),
		"log_file":    LogFile(),
		"plans_dir":   PlansPath(),
	}
}