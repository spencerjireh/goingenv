package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"goingenv/internal/archive"
	"goingenv/internal/config"
	"goingenv/internal/crypto"
	"goingenv/internal/scanner"
	"goingenv/internal/tui"
	"goingenv/pkg/types"
)

// NewApp creates a new application instance with all dependencies
func NewApp() (*types.App, error) {
	// Initialize configuration manager
	configMgr := config.NewManager()

	// Load configuration
	cfg, err := configMgr.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize services
	cryptoService := crypto.NewService()
	scannerService := scanner.NewService(cfg)
	archiverService := archive.NewService(cryptoService)

	return &types.App{
		Config:    cfg,
		Scanner:   scannerService,
		Archiver:  archiverService,
		Crypto:    cryptoService,
		ConfigMgr: configMgr,
	}, nil
}

// NewRootCommand creates and returns the root command
func NewRootCommand(version string) *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "goingenv",
		Short: "Environment File Manager with Encryption",
		Long: `goingenv is a CLI tool for managing environment files with encryption capabilities.
		
It can scan, encrypt, and archive your .env files securely, making it easy to
backup, transfer, and restore your environment configurations.`,
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			return runInteractiveMode(verbose)
		},
	}

	// Add global verbose flag
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose debug logging for TUI mode")

	// Add subcommands
	rootCmd.AddCommand(newInitCommand())
	rootCmd.AddCommand(newPackCommand())
	rootCmd.AddCommand(newUnpackCommand())
	rootCmd.AddCommand(newListCommand())
	rootCmd.AddCommand(newStatusCommand())

	return rootCmd
}

// runInteractiveMode launches the TUI interface
func runInteractiveMode(verbose bool) error {
	// Check if GoingEnv is initialized
	if !config.IsInitialized() {
		fmt.Println("goingenv is not initialized in this directory.")
		fmt.Println("Run 'goingenv init' first to set up goingenv.")
		return nil
	}

	// Initialize application
	app, err := NewApp()
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	// Create and run TUI
	model := tui.NewModel(app, verbose)
	program := tea.NewProgram(model, tea.WithAltScreen())

	// Ensure cleanup happens even if there's an error
	defer model.Cleanup()

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}