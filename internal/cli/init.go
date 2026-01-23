package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"goingenv/internal/config"
)

// newInitCommand creates the init command
func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize goingenv in the current directory",
		Long: `Initialize goingenv in the current directory by creating the .goingenv folder
and generating configuration files.

This command will:
- Create the .goingenv directory for storing encrypted archives
- Create a default configuration file in your home directory if it doesn't exist

Encrypted archives (.enc files) can be safely committed to git for sharing
with team members.

This must be run before using any other goingenv commands.

Examples:
  goingenv init`,
		RunE: runInitCommand,
	}

	cmd.Flags().BoolP("force", "f", false, "Force initialization even if already initialized")

	return cmd
}

// runInitCommand executes the init command
func runInitCommand(cmd *cobra.Command, args []string) error {
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return fmt.Errorf("failed to get force flag: %w", err)
	}

	// Check if already initialized
	if config.IsInitialized() && !force {
		fmt.Println("goingenv is already initialized in this directory.")
		fmt.Println("Use 'goingenv init --force' to reinitialize.")
		return nil
	}

	fmt.Println("Initializing goingenv in current directory...")

	// Create .goingenv directory for storing encrypted archives
	if initErr := config.InitializeProject(); initErr != nil {
		return fmt.Errorf("failed to initialize project: %w", initErr)
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

	fmt.Println("goingenv successfully initialized!")
	fmt.Println()
	fmt.Println("What's been created:")
	fmt.Printf("  - .goingenv/ directory for storing encrypted archives\n")
	fmt.Printf("  - Configuration file in your home directory\n")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  - Run 'goingenv pack' to create your first encrypted archive")
	fmt.Println("  - Run 'goingenv status' to see what environment files are detected")
	fmt.Println("  - Use the TUI mode by running 'goingenv' without arguments")
	fmt.Println()
	fmt.Println("Encrypted archives (.enc files) are safe to commit to git for sharing.")

	return nil
}
