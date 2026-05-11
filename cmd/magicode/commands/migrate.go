// Package commands provides CLI commands.
package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rhony08/magicode/internal/global"
	"github.com/rhony08/magicode/internal/util/log"
	"github.com/spf13/cobra"
)

// NewMigrateCommand creates the migrate command
func NewMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate configuration from OpenCode to MagiCode",
		Long: `Migrate/copy all OpenCode configuration to MagiCode.

This command copies (not moves) configuration files from OpenCode to MagiCode:
- opencode.json -> magicode.json (converted)
- agents/ -> agents/
- prompts/ -> prompts/
- skills/ -> skills/
- workflows/ -> workflows/

After migration, you can use MagiCode without --use-opencode flag.
The original OpenCode configuration is preserved.`,
		RunE: runMigrate,
	}

	cmd.Flags().Bool("force", false, "Overwrite existing MagiCode configuration")

	return cmd
}

func runMigrate(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	// Ensure global paths are initialized
	if err := global.Init(); err != nil {
		return fmt.Errorf("failed to initialize paths: %w", err)
	}

	// Get source (OpenCode) and destination (MagiCode) paths
	sourceDir := filepath.Join(os.Getenv("HOME"), ".config", "opencode")
	destDir := global.Path.Config

	// Check if OpenCode config exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("OpenCode config directory not found at %s", sourceDir)
	}

	// Create MagiCode config directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create MagiCode config directory: %w", err)
	}

	// Check if already migrated
	if !force {
		magicodeConfig := filepath.Join(destDir, "magicode.json")
		if _, err := os.Stat(magicodeConfig); err == nil {
			fmt.Println("MagiCode configuration already exists.")
			fmt.Println("Use --force to overwrite.")
			return nil
		}
	}

	// Track what was copied
	copied := []string{}
	skipped := []string{}
	failed := []string{}

	// 1. Copy and convert opencode.json -> magicode.json
	if err := migrateConfigFile(sourceDir, destDir); err != nil {
		log.Warn("Failed to migrate config file", "error", err)
		failed = append(failed, "magicode.json")
	} else {
		copied = append(copied, "magicode.json")
	}

	// 2. Copy directories
	dirs := []string{"agents", "prompts", "skills", "workflows"}
	for _, dir := range dirs {
		srcPath := filepath.Join(sourceDir, dir)
		dstPath := filepath.Join(destDir, dir)

		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			skipped = append(skipped, dir+" (not found in OpenCode)")
			continue
		}

		if err := copyDirectory(srcPath, dstPath); err != nil {
			log.Warn("Failed to copy directory", "dir", dir, "error", err)
			failed = append(failed, dir)
		} else {
			copied = append(copied, dir+"/")
		}
	}

	// Print summary
	fmt.Println("\n✅ Migration Complete!")
	fmt.Println("\n📁 Copied:")
	for _, item := range copied {
		fmt.Printf("  ✓ %s\n", item)
	}

	if len(skipped) > 0 {
		fmt.Println("\n⏭️  Skipped:")
		for _, item := range skipped {
			fmt.Printf("  - %s\n", item)
		}
	}

	if len(failed) > 0 {
		fmt.Println("\n❌ Failed:")
		for _, item := range failed {
			fmt.Printf("  ✗ %s\n", item)
		}
		return fmt.Errorf("some items failed to migrate")
	}

	fmt.Printf("\n📂 Configuration copied to: %s\n", destDir)
	fmt.Println("\n📝 Next steps:")
	fmt.Println("  1. Review the migrated magicode.json")
	fmt.Println("  2. Run 'magicode' to start without --use-opencode flag")
	fmt.Println("\n🔒 Your OpenCode configuration is preserved at: ~/.config/opencode/")

	return nil
}

// migrateConfigFile copies and converts opencode.json to magicode.json
func migrateConfigFile(sourceDir, destDir string) error {
	srcFile := filepath.Join(sourceDir, "opencode.json")
	dstFile := filepath.Join(destDir, "magicode.json")

	// Read OpenCode config
	data, err := os.ReadFile(srcFile)
	if err != nil {
		return fmt.Errorf("failed to read opencode.json: %w", err)
	}

	// Parse and convert
	config, err := convertOpenCodeConfig(data)
	if err != nil {
		return fmt.Errorf("failed to convert config: %w", err)
	}

	// Write MagiCode config
	if err := os.WriteFile(dstFile, config, 0644); err != nil {
		return fmt.Errorf("failed to write magicode.json: %w", err)
	}

	return nil
}

// convertOpenCodeConfig converts OpenCode JSON to MagiCode format
func convertOpenCodeConfig(data []byte) ([]byte, error) {
	// For now, just copy the file as-is
	// The config loader already supports both formats
	// We just add a marker to indicate it was migrated

	// If you need specific conversions, add them here
	// e.g., convert provider formats, rename fields, etc.

	return data, nil
}

// copyDirectory recursively copies a directory
func copyDirectory(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dst, err)
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", src, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", src, err)
	}

	// Write destination file
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", dst, err)
	}

	return nil
}