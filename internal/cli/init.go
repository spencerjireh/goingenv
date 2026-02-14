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
	cmd.Flags().BoolP("verbose", "v", false, "Show detailed information")

	return cmd
}

// runInitCommand executes the init command
func runInitCommand(cmd *cobra.Command, args []string) error {
	out := NewOutput(appVersion)

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}

	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		return err
	}

	out.Header()
	out.Blank()

	// Check if already initialized
	if config.IsInitialized() && !force {
		out.Warning("goingenv is already initialized in this directory")
		out.Hint("Use 'goingenv init --force' to reinitialize")
		return nil
	}

	if verbose {
		out.Action("Initializing goingenv...")
	}

	// Create .goingenv directory for storing encrypted archives
	if initErr := config.InitializeProject(); initErr != nil {
		out.Error("Failed to initialize project")
		return fmt.Errorf("initialization failed: %w", initErr)
	}

	// Ensure configuration exists in home directory
	configMgr := config.NewManager()
	cfg, err := configMgr.Load()
	if err != nil {
		out.Error("Failed to load configuration")
		return fmt.Errorf("configuration failed: %w", err)
	}

	// Save default config if it was newly created
	if err := configMgr.Save(cfg); err != nil {
		out.Error("Failed to save configuration")
		return fmt.Errorf("save failed: %w", err)
	}

	if verbose {
		out.Success("Created .goingenv/")
		out.Blank()
		out.Hint("Next steps:")
		out.Indent("Run 'goingenv status' to see detected files")
		out.Indent("Run 'goingenv pack' to create encrypted archive")
	} else {
		out.Success("Initialized")
		out.Blank()
		out.Hint("Run 'goingenv status' to see detected files")
	}

	return nil
}
