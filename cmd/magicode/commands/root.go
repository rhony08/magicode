// Package commands provides CLI commands.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rhony08/magicode/internal/config"
	"github.com/rhony08/magicode/internal/database"
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
  For backward compatibility with OpenCode, use --use-opencode flag.

Session Continuation:
  Use --session <id> to continue an existing session.
  Use 'magicode session list --all' to see all session IDs.`,
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

	// Session flag for continuing existing session
	rootCmd.PersistentFlags().StringP("session", "s", "", "continue existing session by ID")

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
	viper.BindPFlag("session", rootCmd.PersistentFlags().Lookup("session"))
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
	sessionID := viper.GetString("session")
	var initialMessages []tui.Message
	var sessionTitle string
	var sessionDirectory string
	var cursor int64  // Pagination cursor (timestamp in milliseconds)
	var complete bool // Whether all messages loaded

	// If session ID provided, load it to get the directory and messages
	if sessionID != "" {
		ctx := context.Background()
		dbPath := global.DatabasePath()
		db, err := database.New(ctx, database.Config{Path: dbPath})
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		sessionStorage := database.NewSessionStorage(db)
		session, err := sessionStorage.Get(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("session not found: %s\n\nUse 'magicode session list --all --use-opencode' to see available sessions", sessionID)
		}

		sessionTitle = session.Title
		sessionDirectory = session.Directory

		// Use session's directory if not specified
		if directory == "" {
			directory = session.Directory
		}

		log.Info("Continuing session", "sessionID", sessionID)
		log.Info("Directory", "path", directory)
		log.Info("Title", "title", session.Title)

		// Load messages from database with pagination (matching OpenCode)
		messageStorage := database.NewMessageStorage(db)
		partStorage := database.NewPartStorage(db)

		// Use InitialMessagePageSize for initial load (80 messages, matching OpenCode)
		dbMessages, cursor, complete, err := messageStorage.ListPaginated(ctx, sessionID, tui.InitialMessagePageSize, 0)
		if err != nil {
			log.Warn("Failed to load messages", "error", err.Error())
		} else {
			log.Info("Loaded messages from session", "count", len(dbMessages), "cursor", cursor, "complete", complete)

			// Convert database messages to TUI messages
			for _, dbMsg := range dbMessages {
				tuiMsg := convertDBMessageToTUI(dbMsg)

				// Load parts for ALL messages (user and assistant)
				parts, err := partStorage.ListByMessage(ctx, dbMsg.ID)
				if err == nil && len(parts) > 0 {
					tuiMsg.Parts = convertPartsToTUI(parts)
					// For user messages, extract text from parts if available
					if dbMsg.Data.Role == "user" {
						for _, part := range parts {
							if part.Data.Type == "text" && part.Data.Text != "" {
								tuiMsg.Content = part.Data.Text
								break
							}
						}
					}
				}

				initialMessages = append(initialMessages, tuiMsg)
			}
		}
	}

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
	title := fmt.Sprintf("%s - %s", appName, cfg.Model())
	if sessionTitle != "" {
		title = fmt.Sprintf("%s - %s", appName, sessionTitle)
	}

	tuiConfig := tui.Config{
		Title:           title,
		InitialMessages: initialMessages,
		SessionID:       sessionID,
		MessageMeta: tui.MessageMeta{
			Cursor:   cursor,
			Complete: complete,
			Limit:    len(initialMessages),
			Loading:  false,
		},
		DatabasePath: global.DatabasePath(),
	}

	// If continuing a session, pass it to TUI
	if sessionID != "" {
		tuiConfig.Session = tui.Session{
			ID:        sessionID,
			Title:     sessionTitle,
			Directory: sessionDirectory,
			CreatedAt: time.Now(),
			Active:    true,
		}
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

// convertDBMessageToTUI converts a database message to a TUI message
// Note: Content is stored in parts, not in message.data
func convertDBMessageToTUI(dbMsg database.Message) tui.Message {
	role := tui.RoleUser
	if dbMsg.Data.Role == "assistant" {
		role = tui.RoleAssistant
	} else if dbMsg.Data.Role == "system" {
		role = tui.RoleSystem
	}

	// Extract timestamp from data.time.created if available, otherwise use message timestamp
	var timestamp time.Time
	if dbMsg.Data.Time != nil {
		if created, ok := dbMsg.Data.Time["created"]; ok {
			// Handle different numeric types from JSON
			switch v := created.(type) {
			case float64:
				timestamp = time.UnixMilli(int64(v))
			case int64:
				timestamp = time.UnixMilli(v)
			case int:
				timestamp = time.UnixMilli(int64(v))
			default:
				timestamp = time.UnixMilli(dbMsg.Timestamps.TimeCreated)
			}
		} else {
			timestamp = time.UnixMilli(dbMsg.Timestamps.TimeCreated)
		}
	} else {
		timestamp = time.UnixMilli(dbMsg.Timestamps.TimeCreated)
	}

	return tui.Message{
		ID:        dbMsg.ID,
		Role:      role,
		Content:   "", // Content is in parts, not here
		Timestamp: timestamp,
		Model:     dbMsg.Data.ModelID,
		Provider:  dbMsg.Data.ProviderID,
	}
}

// convertPartsToTUI converts database parts to TUI parts
// Note: Filters out internal parts (patch, step-start, step-finish) that shouldn't be displayed
// This matches OpenCode's SKIP_PARTS behavior in sync.tsx
func convertPartsToTUI(parts []database.Part) []tui.Part {
	// SKIP_PARTS - internal parts that shouldn't be displayed (matches OpenCode)
	skipParts := map[string]bool{
		"patch":       true,
		"step-start":  true,
		"step-finish": true,
	}

	var result []tui.Part
	for _, p := range parts {
		// Skip internal parts that shouldn't be rendered
		if skipParts[p.Data.Type] {
			continue
		}

		tuiPart := tui.Part{
			ID:     p.ID,
			Type:   p.Data.Type,
			Text:   p.Data.Text,
			Status: p.Data.Status,
		}

		// Handle file parts
		if p.Data.Type == "file" {
			tuiPart.Text = fmt.Sprintf("[File: %s]", p.Data.Filename)
		}

		// Handle tool parts
		if p.Data.Type == "tool_use" || p.Data.Type == "tool_result" {
			tuiPart.ToolName = p.Data.ToolName
			tuiPart.ToolID = p.Data.ToolID
			if p.Data.Type == "tool_result" {
				tuiPart.ToolResult = p.Data.ToolResult
			}
			if p.Data.ToolInput != nil {
				// Convert map to JSON string for display
				inputJSON, _ := json.Marshal(p.Data.ToolInput)
				tuiPart.ToolInput = string(inputJSON)
			}
		}

		result = append(result, tuiPart)
	}
	return result
}
