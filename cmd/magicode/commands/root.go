package commands

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rhony08/magicode/internal/config"
	"github.com/rhony08/magicode/internal/global"
	"github.com/rhony08/magicode/internal/tui"
	"github.com/rhony08/magicode/internal/util/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewRootCommand creates the root CLI command
func NewRootCommand(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "magicode",
		Short: "AI-powered coding assistant CLI",
		Long: `MagiCode is an AI-powered coding assistant that helps you write,
edit, and understand code. It integrates with multiple AI providers
and provides tools for file operations, code search, and more.

Run without arguments to start the interactive TUI.

Path Configuration:
  MagiCode uses XDG Base Directory Specification for paths:
  - Data:   ~/.local/share/magicode  (database, sessions)
  - Config: ~/.config/magicode       (magicode.json)
  - State:  ~/.local/state/magicode  (logs)
  - Cache:  ~/.cache/magicode        (cache)

  You can override these paths with flags or config file.
  For backward compatibility with OpenCode, use --use-opencode flag.`,
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Check for backward compatibility first
			useOpenCode := viper.GetBool("use-opencode")
			if useOpenCode {
				global.SetAppName("opencode")
			} else {
				// Auto-detect opencode installation
				global.BackwardCompatibility()
			}

			// Setup custom paths from flags/config
			customPaths := global.CustomPaths{
				DataDir:    viper.GetString("data-dir"),
				ConfigDir:  viper.GetString("config-dir"),
				StateDir:   viper.GetString("state-dir"),
				CacheDir:   viper.GetString("cache-dir"),
				ConfigFile: viper.GetString("config-file"),
				Database:   viper.GetString("database"),
				LogFile:    viper.GetString("log-file"),
			}

			// Initialize global paths with custom overrides
			if err := global.InitWithCustom(customPaths); err != nil {
				return fmt.Errorf("failed to initialize paths: %w", err)
			}

			// Initialize logging
			logLevel := viper.GetString("log-level")
			printLogs := viper.GetBool("print-logs")
			if err := log.Init(logLevel, printLogs, global.Path.State); err != nil {
				// Continue without file logging
				log.InitDefault()
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default behavior: start TUI
			return runTUI(cmd, args)
		},
	}

	// Persistent flags (available to all subcommands)
	rootCmd.PersistentFlags().Bool("print-logs", false, "print logs to stderr")
	rootCmd.PersistentFlags().String("log-level", "INFO", "log level (DEBUG, INFO, WARN, ERROR)")
	rootCmd.PersistentFlags().Bool("pure", false, "run without external plugins")
	rootCmd.PersistentFlags().StringP("directory", "d", "", "working directory (default: current directory)")
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path")

	// Path configuration flags
	rootCmd.PersistentFlags().String("data-dir", "", "custom data directory (database, sessions)")
	rootCmd.PersistentFlags().String("config-dir", "", "custom config directory")
	rootCmd.PersistentFlags().String("state-dir", "", "custom state directory (logs)")
	rootCmd.PersistentFlags().String("cache-dir", "", "custom cache directory")
	rootCmd.PersistentFlags().String("config-file", "", "custom config file path (overrides default lookup)")
	rootCmd.PersistentFlags().String("database", "", "custom database file path")
	rootCmd.PersistentFlags().String("log-file", "", "custom log file path")
	rootCmd.PersistentFlags().Bool("use-opencode", false, "use opencode paths for backward compatibility")

	viper.BindPFlag("print-logs", rootCmd.PersistentFlags().Lookup("print-logs"))
	viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("pure", rootCmd.PersistentFlags().Lookup("pure"))
	viper.BindPFlag("directory", rootCmd.PersistentFlags().Lookup("directory"))
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("data-dir", rootCmd.PersistentFlags().Lookup("data-dir"))
	viper.BindPFlag("config-dir", rootCmd.PersistentFlags().Lookup("config-dir"))
	viper.BindPFlag("state-dir", rootCmd.PersistentFlags().Lookup("state-dir"))
	viper.BindPFlag("cache-dir", rootCmd.PersistentFlags().Lookup("cache-dir"))
	viper.BindPFlag("config-file", rootCmd.PersistentFlags().Lookup("config-file"))
	viper.BindPFlag("database", rootCmd.PersistentFlags().Lookup("database"))
	viper.BindPFlag("log-file", rootCmd.PersistentFlags().Lookup("log-file"))
	viper.BindPFlag("use-opencode", rootCmd.PersistentFlags().Lookup("use-opencode"))

	// Add subcommands
	rootCmd.AddCommand(NewRunCommand())
	rootCmd.AddCommand(NewServeCommand())
	rootCmd.AddCommand(NewSessionCommand())
	rootCmd.AddCommand(NewProvidersCommand())
	rootCmd.AddCommand(NewModelsCommand())
	rootCmd.AddCommand(NewConfigCommand())
	rootCmd.AddCommand(NewAuthCommand())
	rootCmd.AddCommand(NewDebugCommand())
	rootCmd.AddCommand(NewExportCommand())
	rootCmd.AddCommand(NewImportCommand())
	rootCmd.AddCommand(NewUpgradeCommand())
	rootCmd.AddCommand(NewPathsCommand())

	return rootCmd
}

// runTUI starts the interactive terminal UI
func runTUI(cmd *cobra.Command, args []string) error {
	directory := viper.GetString("directory")
	if directory == "" {
		var err error
		directory, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		viper.Set("directory", directory)
	}

	// Initialize config
	cfg, err := config.New(directory, global.Path.Config)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply path config from file if present
	if cfg.Paths() != nil {
		paths := cfg.Paths()
		if paths.UseOpenCode && !viper.IsSet("use-opencode") {
			global.SetAppName("opencode")
		}
	}

	// Create TUI app configuration with title showing model info
	appName := global.GetAppName()
	tuiConfig := tui.Config{
		Title: fmt.Sprintf("%s - %s", appName, cfg.Model()),
	}

	// Create the TUI app
	app := tui.NewApp(tuiConfig)

	// Run the TUI
	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}