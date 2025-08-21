package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"goingenv/internal/config"
)

// newInitCommand creates the init command
func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize GoingEnv in the current directory",
		Long: `Initialize GoingEnv in the current directory by creating the .goingenv folder
and generating configuration files.

This command will:
- Create the .goingenv directory
- Generate a .gitignore file inside .goingenv (allowing *.enc files for safe transfer)
- Create a default configuration file in your home directory if it doesn't exist
- Ensure the project root .gitignore includes .goingenv/

This must be run before using any other GoingEnv commands.

Examples:
  goingenv init`,
		RunE: runInitCommand,
	}

	cmd.Flags().BoolP("force", "f", false, "Force initialization even if already initialized")

	return cmd
}

// runInitCommand executes the init command
func runInitCommand(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	// Check if already initialized
	if config.IsInitialized() && !force {
		fmt.Println("GoingEnv is already initialized in this directory.")
		fmt.Println("Use 'goingenv init --force' to reinitialize.")
		return nil
	}

	fmt.Println("Initializing GoingEnv in current directory...")

	// Create .goingenv directory with proper gitignore
	if err := config.InitializeProject(); err != nil {
		return fmt.Errorf("failed to initialize project: %w", err)
	}

	// Ensure configuration exists in home directory
	configMgr := config.NewManager()
	cfg, err := configMgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Save default config if it was newly created
	if err := configMgr.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Update project root .gitignore to include .goingenv/
	if err := ensureProjectGitignore(); err != nil {
		fmt.Printf("Warning: Could not update project .gitignore: %v\n", err)
		fmt.Println("Please manually add '.goingenv/' to your project's .gitignore file.")
	}

	fmt.Println("✅ GoingEnv successfully initialized!")
	fmt.Println()
	fmt.Println("What's been created:")
	fmt.Printf("  • .goingenv/ directory\n")
	fmt.Printf("  • .goingenv/.gitignore (allows *.enc files for safe transfer)\n")
	fmt.Printf("  • Configuration file in your home directory\n")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  • Run 'goingenv pack' to create your first encrypted archive")
	fmt.Println("  • Run 'goingenv status' to see what environment files are detected")
	fmt.Println("  • Use the TUI mode by running 'goingenv' without arguments")

	return nil
}

// ensureProjectGitignore ensures the project root .gitignore includes .goingenv/
func ensureProjectGitignore() error {
	gitignorePath := ".gitignore"
	
	// Check if .gitignore exists
	content := ""
	if _, err := os.Stat(gitignorePath); err == nil {
		// Read existing content
		data, err := os.ReadFile(gitignorePath)
		if err != nil {
			return fmt.Errorf("failed to read .gitignore: %w", err)
		}
		content = string(data)
	}

	// Check if .goingenv/ is already in gitignore
	if filepath.Base(content) != "" {
		// Simple check - if .goingenv appears anywhere in the file, assume it's handled
		if len(content) > 0 && (contains(content, ".goingenv/") || contains(content, ".goingenv")) {
			return nil
		}
	}

	// Add .goingenv/ to gitignore
	if content != "" && content[len(content)-1] != '\n' {
		content += "\n"
	}
	content += "\n# GoingEnv directory\n.goingenv/\n"

	// Write back to .gitignore
	if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	return nil
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}()
}