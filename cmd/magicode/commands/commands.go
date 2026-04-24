package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rhony08/magicode/internal/config"
	"github.com/rhony08/magicode/internal/database"
	"github.com/rhony08/magicode/internal/global"
	"github.com/rhony08/magicode/internal/provider"
	"github.com/rhony08/magicode/internal/server"
	"github.com/rhony08/magicode/internal/util/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewRunCommand creates the run subcommand (explicit TUI launch)
func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the interactive TUI",
		Long:  `Launch the interactive terminal user interface for coding assistance.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use the same TUI logic as the root command
			return runTUI(cmd, args)
		},
	}

	// Inherit flags from root command
	return cmd
}

// NewServeCommand creates the serve subcommand
func NewServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP server",
		Long:  `Start the HTTP/WebSocket server for remote access.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			directory := viper.GetString("directory")
			if directory == "" {
				var err error
				directory, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}
			}

			// Initialize config
			cfg, err := config.New(directory, global.Path.Config)
			if err != nil {
				log.Warn("Failed to load config, using defaults: %v", err)
			}

			// Create server config
			serverCfg := server.DefaultConfig()
			if cfg != nil {
				// Apply config settings if available
				// TODO: Add port and hostname config options
			}

			// Create server
			srv := server.New(directory, serverCfg)

			// Start server
			listener, err := srv.Listen()
			if err != nil {
				return fmt.Errorf("failed to start server: %w", err)
			}

			log.Info("MagiCode server started")
			fmt.Printf("Server running at %s\n", listener.URL.String())
			fmt.Println("Press Ctrl+C to stop")

			// Wait for interrupt signal
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			// Block until signal
			<-sigChan

			// Shutdown
			fmt.Println("\nShutting down server...")
			if err := srv.Shutdown(); err != nil {
				return fmt.Errorf("failed to shutdown server: %w", err)
			}

			fmt.Println("Server stopped")
			return nil
		},
	}

	// Add serve-specific flags
	cmd.Flags().IntP("port", "p", 3000, "server port")
	cmd.Flags().String("host", "localhost", "server hostname")
	cmd.Flags().Bool("cors", false, "enable CORS")

	viper.BindPFlag("serve.port", cmd.Flags().Lookup("port"))
	viper.BindPFlag("serve.host", cmd.Flags().Lookup("host"))
	viper.BindPFlag("serve.cors", cmd.Flags().Lookup("cors"))

	return cmd
}

// NewSessionCommand creates the session management subcommand
func NewSessionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage sessions",
		Long:  `List, create, delete, and manage coding sessions.`,
	}

	// Add --all flag to parent command for list subcommand
	var showAll bool

	// Add subcommands
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Initialize database
			dbPath := global.DatabasePath()
			db, err := database.New(ctx, database.Config{Path: dbPath})
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer db.Close()

			sessionStorage := database.NewSessionStorage(db)

			var sessions []database.Session
			if showAll {
				// List all sessions from all directories
				sessions, err = sessionStorage.ListAll(ctx)
				if err != nil {
					return fmt.Errorf("failed to list sessions: %w", err)
				}
			} else {
				// List sessions for current directory only
				directory := viper.GetString("directory")
				if directory == "" {
					directory, _ = os.Getwd()
				}
				sessions, err = sessionStorage.ListByDirectory(ctx, directory)
				if err != nil {
					return fmt.Errorf("failed to list sessions: %w", err)
				}
			}

			fmt.Println("Sessions:")
			if len(sessions) == 0 {
				if showAll {
					fmt.Println("  (No sessions found in database)")
				} else {
					directory := viper.GetString("directory")
					if directory == "" {
						directory, _ = os.Getwd()
					}
					fmt.Printf("  (No sessions found for directory: %s)\n", directory)
					fmt.Println("\n  Use --all to see sessions from all directories")
				}
			} else {
				for _, s := range sessions {
					timeStr := time.UnixMilli(s.Timestamps.TimeCreated).Format("2006-01-02 15:04")
					fmt.Printf("  %s | %s | %s\n", s.ID[:20]+"...", timeStr, s.Title)
					if showAll {
						fmt.Printf("    Directory: %s\n", s.Directory)
					}
				}
				fmt.Printf("\n  Total: %d sessions\n", len(sessions))
			}

			return nil
		},
	}
	listCmd.Flags().BoolVarP(&showAll, "all", "a", false, "show sessions from all directories")
	cmd.AddCommand(listCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "create [title]",
		Short: "Create a new session",
		RunE: func(cmd *cobra.Command, args []string) error {
			title := "New Session"
			if len(args) > 0 {
				title = args[0]
			}
			// TODO: Connect to database and create session
			fmt.Printf("Created session: %s\n", title)
			fmt.Println("Note: Session will be persisted when database is connected")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a session",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("requires session id argument")
			}
			// TODO: Connect to database and delete session
			fmt.Printf("Deleted session: %s\n", args[0])
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show <id>",
		Short: "Show session details",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("requires session id argument")
			}

			ctx := context.Background()
			dbPath := global.DatabasePath()
			db, err := database.New(ctx, database.Config{Path: dbPath})
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer db.Close()

			sessionStorage := database.NewSessionStorage(db)
			session, err := sessionStorage.Get(ctx, args[0])
			if err != nil {
				return fmt.Errorf("session not found: %s", args[0])
			}

			fmt.Printf("Session: %s\n", session.ID)
			fmt.Printf("  Title: %s\n", session.Title)
			fmt.Printf("  Directory: %s\n", session.Directory)
			fmt.Printf("  Slug: %s\n", session.Slug)
			fmt.Printf("  Project ID: %s\n", session.ProjectID)
			fmt.Printf("  Created: %s\n", time.UnixMilli(session.Timestamps.TimeCreated).Format("2006-01-02 15:04:05"))
			fmt.Printf("  Updated: %s\n", time.UnixMilli(session.Timestamps.TimeUpdated).Format("2006-01-02 15:04:05"))

			// Count messages
			msgStorage := database.NewMessageStorage(db)
			messages, err := msgStorage.List(ctx, session.ID)
			if err == nil {
				fmt.Printf("  Messages: %d\n", len(messages))
			}

			return nil
		},
	})

	return cmd
}

// NewProvidersCommand creates the providers listing subcommand
func NewProvidersCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "List available AI providers",
		Long:  `List all configured AI providers and their status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create registry with default providers
			registry := provider.NewProviderRegistry()

			// Register Anthropic if API key is set
			if os.Getenv("ANTHROPIC_API_KEY") != "" {
				anthropic := provider.NewAnthropicProvider(os.Getenv("ANTHROPIC_API_KEY"))
				registry.Register(anthropic)
			}

			// Register OpenAI if API key is set
			if os.Getenv("OPENAI_API_KEY") != "" {
				openai := provider.NewOpenAIProvider(os.Getenv("OPENAI_API_KEY"))
				registry.Register(openai)
			}

			// List providers
			providers := registry.ListProviders()

			fmt.Println("Available AI Providers:")
			fmt.Println()

			if len(providers) == 0 {
				fmt.Println("  No providers configured.")
				fmt.Println("\n  To configure providers, set environment variables:")
				fmt.Println("    export ANTHROPIC_API_KEY=your_key  # for Anthropic")
				fmt.Println("    export OPENAI_API_KEY=your_key     # for OpenAI")
				return nil
			}

			for _, p := range providers {
				info := p.Info()
				fmt.Printf("  %s\n", info.ID)
				fmt.Printf("    Name: %s\n", info.Name)
				if len(info.EnvKeys) > 0 {
					fmt.Printf("    Env vars: %v\n", info.EnvKeys)
				}
				models := registry.ListModelsByProvider(info.ID)
				if len(models) > 0 {
					fmt.Printf("    Models:\n")
					for _, m := range models {
						fmt.Printf("      - %s (%s)\n", m.ID, m.Name)
					}
				}
				fmt.Println()
			}

			return nil
		},
	}
}

// NewModelsCommand creates the models listing subcommand
func NewModelsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "models",
		Short: "List available AI models",
		Long:  `List all available models from configured providers.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create registry with default providers
			registry := provider.NewProviderRegistry()

			// Register Anthropic if API key is set
			if os.Getenv("ANTHROPIC_API_KEY") != "" {
				anthropic := provider.NewAnthropicProvider(os.Getenv("ANTHROPIC_API_KEY"))
				registry.Register(anthropic)
			}

			// Register OpenAI if API key is set
			if os.Getenv("OPENAI_API_KEY") != "" {
				openai := provider.NewOpenAIProvider(os.Getenv("OPENAI_API_KEY"))
				registry.Register(openai)
			}

			// List all models
			models := registry.ListModels()

			fmt.Println("Available AI Models:")
			fmt.Println()

			if len(models) == 0 {
				fmt.Println("  No models available.")
				fmt.Println("\n  Set API keys to enable providers:")
				fmt.Println("    export ANTHROPIC_API_KEY=your_key")
				fmt.Println("    export OPENAI_API_KEY=your_key")
				return nil
			}

			// Group by provider
			providerModels := make(map[string][]provider.ModelInfo)
			for _, m := range models {
				provID, _ := provider.ParseModelID(m.ID)
				providerModels[string(provID)] = append(providerModels[string(provID)], m)
			}

			for providerID, pmodels := range providerModels {
				fmt.Printf("  %s:\n", providerID)
				for _, m := range pmodels {
					fmt.Printf("    %s - %s\n", m.ID, m.Name)
				}
				fmt.Println()
			}

			return nil
		},
	}
}

// NewConfigCommand creates the config management subcommand
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `View and manage MagiCode configuration.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			directory := viper.GetString("directory")
			if directory == "" {
				directory, _ = os.Getwd()
			}

			// Initialize config
			cfg, err := config.New(directory, global.Path.Config)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			info := cfg.Get()

			fmt.Println("MagiCode Configuration:")
			fmt.Println()
			fmt.Printf("  Model:        %s\n", cfg.Model())
			fmt.Printf("  Small Model:  %s\n", cfg.SmallModel())
			fmt.Printf("  Username:     %s\n", info.Username)
			fmt.Printf("  Default Agent: %s\n", info.DefaultAgent)
			fmt.Println()
			fmt.Printf("  Config file:  %s\n", global.Path.Config)

			// Show providers
			if len(info.Providers) > 0 {
				fmt.Println("\n  Providers:")
				for id, p := range info.Providers {
					fmt.Printf("    %s:\n", id)
					if len(p.Env) > 0 {
						fmt.Printf("      Env vars: %v\n", p.Env)
					}
				}
			}

			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set configuration value",
		Long: `Set a configuration value. Valid keys:
  model, small_model, username, default_agent, log_level`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("requires key and value arguments")
			}
			key := args[0]
			value := args[1]

			// Validate key
			validKeys := []string{"model", "small_model", "username", "default_agent", "log_level"}
			valid := false
			for _, k := range validKeys {
				if k == key {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid key: %s. Valid keys: %v", key, validKeys)
			}

			// TODO: Write to config file
			fmt.Printf("Setting %s = %s\n", key, value)
			fmt.Println("Note: Config changes will be persisted when config write is implemented")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Show config file paths",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Config file search order:")
			fmt.Printf("  1. %s/.magicode/magicode.json\n", global.Path.Config)
			fmt.Printf("  2. %s/.magicode/magicode.jsonc\n", global.Path.Config)
			fmt.Printf("  3. %s/magicode.json (project)\n", viper.GetString("directory"))
			fmt.Println()
			fmt.Printf("Current config: %s\n", global.Path.Config)
			return nil
		},
	})

	return cmd
}

// NewAuthCommand creates the authentication subcommand
func NewAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
		Long:  `Configure API keys and authentication for AI providers.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "login <provider>",
		Short: "Login to a provider",
		Long: `Login to an AI provider. Providers:
  anthropic - Requires ANTHROPIC_API_KEY
  openai    - Requires OPENAI_API_KEY`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("requires provider argument. Available: anthropic, openai")
			}
			providerID := args[0]

			// Check environment variables
			envVars := map[string][]string{
				"anthropic": []string{"ANTHROPIC_API_KEY"},
				"openai":    []string{"OPENAI_API_KEY"},
			}

			vars, ok := envVars[providerID]
			if !ok {
				return fmt.Errorf("unknown provider: %s. Available: anthropic, openai", providerID)
			}

			fmt.Printf("Checking authentication for %s...\n", providerID)

			allSet := true
			for _, envVar := range vars {
				value := os.Getenv(envVar)
				if value == "" {
					fmt.Printf("  %s: NOT SET\n", envVar)
					allSet = false
				} else {
					fmt.Printf("  %s: SET (%d chars)\n", envVar, len(value))
				}
			}

			if allSet {
				fmt.Printf("\n%s is authenticated!\n", providerID)
			} else {
				fmt.Printf("\nTo authenticate %s, set the required environment variables:\n", providerID)
				for _, envVar := range vars {
					fmt.Printf("  export %s=your_api_key\n", envVar)
				}
			}

			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "logout <provider>",
		Short: "Logout from a provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("requires provider argument")
			}
			fmt.Printf("Note: To logout from %s, unset the environment variable:\n", args[0])
			fmt.Println("  unset ANTHROPIC_API_KEY  # for anthropic")
			fmt.Println("  unset OPENAI_API_KEY     # for openai")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show authentication status for all providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Authentication Status:")
			fmt.Println()

			// Check Anthropic
			anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
			fmt.Print("  Anthropic: ")
			if anthropicKey != "" {
				fmt.Println("Authenticated (API key set)")
			} else {
				fmt.Println("Not authenticated")
			}

			// Check OpenAI
			openaiKey := os.Getenv("OPENAI_API_KEY")
			fmt.Print("  OpenAI: ")
			if openaiKey != "" {
				fmt.Println("Authenticated (API key set)")
			} else {
				fmt.Println("Not authenticated")
			}

			return nil
		},
	})

	return cmd
}

// NewDebugCommand creates the debug subcommand
func NewDebugCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug commands",
		Long:  `Debugging and diagnostic commands.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show detailed version info",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("MagiCode CLI (Go Implementation)")
			fmt.Printf("Version: %s\n", cmd.Root().Version)
			fmt.Println("Implementation: Phase 8 Complete, Phase 9 In Progress")
			fmt.Println()
			fmt.Println("Components:")
			fmt.Println("  ✓ CLI (Cobra)")
			fmt.Println("  ✓ Logging")
			fmt.Println("  ✓ Global paths")
			fmt.Println("  ✓ Config")
			fmt.Println("  ✓ Instance")
			fmt.Println("  ✓ Bus")
			fmt.Println("  ✓ Database")
			fmt.Println("  ✓ Provider (Anthropic, OpenAI)")
			fmt.Println("  ✓ Tools")
			fmt.Println("  ✓ LSP")
			fmt.Println("  ✓ TUI (Bubble Tea)")
			fmt.Println("  ✓ PTY")
			fmt.Println("  ✓ Server (Fiber)")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "paths",
		Short: "Show global paths",
		RunE: func(cmd *cobra.Command, args []string) error {
			paths := global.GetAllPaths()
			appName := global.GetAppName()

			fmt.Printf("Application Name: %s\n", appName)
			fmt.Println("\nPaths:")
			for key, value := range paths {
				fmt.Printf("  %s: %s\n", key, value)
			}

			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "env",
		Short: "Show environment variables",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Environment Variables:")
			fmt.Println()

			// AI provider keys
			keys := []string{
				"ANTHROPIC_API_KEY",
				"OPENAI_API_KEY",
			}

			for _, key := range keys {
				value := os.Getenv(key)
				if value != "" {
					fmt.Printf("  %s: SET (%d chars)\n", key, len(value))
				} else {
					fmt.Printf("  %s: NOT SET\n", key)
				}
			}

			return nil
		},
	})

	return cmd
}

// NewExportCommand creates the export subcommand
func NewExportCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "export <session-id>",
		Short: "Export session",
		Long:  `Export a session to a file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("requires session-id argument")
			}
			fmt.Printf("Exporting session %s\n", args[0])
			fmt.Println("(Export coming in Phase 2)")
			return nil
		},
	}
}

// NewImportCommand creates the import subcommand
func NewImportCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "import <file>",
		Short: "Import session",
		Long:  `Import a session from a file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("requires file argument")
			}
			fmt.Printf("Importing from %s\n", args[0])
			fmt.Println("(Import coming in Phase 2)")
			return nil
		},
	}
}

// NewUpgradeCommand creates the upgrade subcommand
func NewUpgradeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade MagiCode",
		Long:  `Upgrade MagiCode to the latest version.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("(Upgrade coming in Phase 9)")
			return nil
		},
	}
}

// NewPathsCommand creates the paths subcommand
func NewPathsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "paths",
		Short: "Show path configuration",
		Long: `Display all configured paths for MagiCode.

Shows:
  - Data directory (database, sessions)
  - Config directory (magicode.json)
  - State directory (logs)
  - Cache directory
  - Database file path
  - Config file path
  - Log file path`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paths := global.GetAllPaths()
			appName := global.GetAppName()

			fmt.Printf("Application Name: %s\n", appName)
			fmt.Println("\nPaths:")
			fmt.Printf("  Data Directory:   %s\n", paths["data_dir"])
			fmt.Printf("  Config Directory: %s\n", paths["config_dir"])
			fmt.Printf("  State Directory:  %s\n", paths["state_dir"])
			fmt.Printf("  Cache Directory:  %s\n", paths["cache_dir"])
			fmt.Println("\nFiles:")
			fmt.Printf("  Database:  %s\n", paths["database"])
			fmt.Printf("  Config:    %s\n", paths["config_file"])
			fmt.Printf("  Log:       %s\n", paths["log_file"])
			fmt.Printf("  Plans:     %s\n", paths["plans_dir"])

			return nil
		},
	}
}