package commands

import (
	"fmt"
	"os"

	"github.com/opencode-ai/opencode-go/internal/config"
	"github.com/opencode-ai/opencode-go/internal/global"
	"github.com/opencode-ai/opencode-go/internal/util/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewRootCommand creates the root CLI command
func NewRootCommand(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "opencode",
		Short: "AI-powered coding assistant CLI",
		Long: `OpenCode is an AI-powered coding assistant that helps you write,
edit, and understand code. It integrates with multiple AI providers
and provides tools for file operations, code search, and more.

Run without arguments to start the interactive TUI.`,
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize global paths
			if err := global.Init(); err != nil {
				return fmt.Errorf("failed to initialize: %w", err)
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

	viper.BindPFlag("print-logs", rootCmd.PersistentFlags().Lookup("print-logs"))
	viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("pure", rootCmd.PersistentFlags().Lookup("pure"))
	viper.BindPFlag("directory", rootCmd.PersistentFlags().Lookup("directory"))
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

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

	// TODO: Initialize TUI (Phase 6)
	fmt.Printf("OpenCode v%s\n", cmd.Root().Version)
	fmt.Printf("Starting in: %s\n", directory)
	fmt.Printf("Model: %s\n", cfg.Model())
	fmt.Println("(TUI implementation coming in Phase 6)")
	return nil
}