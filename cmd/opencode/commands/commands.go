package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewRunCommand creates the run subcommand (explicit TUI launch)
func NewRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the interactive TUI",
		Long:  `Launch the interactive terminal user interface for coding assistance.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("(TUI implementation coming in Phase 6)")
			return nil
		},
	}
}

// NewServeCommand creates the serve subcommand
func NewServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP server",
		Long:  `Start the HTTP/WebSocket server for remote access.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("(Server implementation coming in Phase 8)")
			return nil
		},
	}
}

// NewSessionCommand creates the session management subcommand
func NewSessionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "session",
		Short: "Manage sessions",
		Long:  `List, create, delete, and manage coding sessions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("(Session management coming in Phase 2)")
			return nil
		},
	}
}

// NewProvidersCommand creates the providers listing subcommand
func NewProvidersCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "List available AI providers",
		Long:  `List all configured AI providers and their status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("(Provider listing coming in Phase 3)")
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
			fmt.Println("(Model listing coming in Phase 3)")
			return nil
		},
	}
}

// NewConfigCommand creates the config management subcommand
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `View and manage OpenCode configuration.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("(Config display coming in Phase 1)")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set configuration value",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("requires key and value arguments")
			}
			fmt.Printf("Setting %s = %s\n", args[0], args[1])
			fmt.Println("(Config write coming in Phase 1)")
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("requires provider argument")
			}
			fmt.Printf("Logging in to %s\n", args[0])
			fmt.Println("(Auth login coming in Phase 3)")
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
			fmt.Printf("Logging out from %s\n", args[0])
			fmt.Println("(Auth logout coming in Phase 3)")
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
			fmt.Println("OpenCode CLI (Go)")
			fmt.Printf("Version: %s\n", cmd.Root().Version)
			fmt.Println("Implementation: Phase 1 (Core Infrastructure)")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "paths",
		Short: "Show global paths",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("(Paths display coming in Phase 1)")
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
		Short: "Upgrade OpenCode",
		Long:  `Upgrade OpenCode to the latest version.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("(Upgrade coming in Phase 9)")
			return nil
		},
	}
}