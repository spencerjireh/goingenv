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

// appVersion stores the version for use in CLI output
var appVersion string

// buildTime and gitCommit hold build metadata injected via -ldflags -X in
// cmd/goingenv. They are reported by --version; the short appVersion is what
// the command output header uses.
var (
	buildTime = "unknown"
	gitCommit = "unknown"
)

// SetBuildInfo records build metadata for the --version output.
func SetBuildInfo(bt, commit string) {
	if bt != "" {
		buildTime = bt
	}
	if commit != "" {
		gitCommit = commit
	}
}

// NewRootCommand creates and returns the root command
func NewRootCommand(version string) *cobra.Command {
	appVersion = version

	var rootCmd = &cobra.Command{
		Use:   "goingenv",
		Short: "Manage encrypted archives of your .env files",
		Long: `Bundle your .env files into one AES-256-GCM encrypted archive, commit it
alongside your code, and let teammates restore it with a shared password.

Run goingenv with no arguments for the terminal UI, which covers the same
operations as the subcommands below.`,
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose") //nolint:errcheck // flag always exists
			return runInteractiveMode(verbose, version)
		},
	}

	rootCmd.SetVersionTemplate(fmt.Sprintf(
		"goingenv {{.Version}}\ncommit: %s\nbuilt:  %s\n", gitCommit, buildTime))

	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	// Add global verbose flag
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose debug logging for TUI mode")

	// Output format, global so scripts get the same contract from every
	// command. `list` also accepts csv, which it supported before this flag
	// existed; it validates that itself.
	rootCmd.PersistentFlags().String("format", string(FormatText),
		"Set the output format: text, json, porcelain")

	// Add subcommands
	rootCmd.AddCommand(newInitCommand())
	rootCmd.AddCommand(newPackCommand())
	rootCmd.AddCommand(newUnpackCommand())
	rootCmd.AddCommand(newListCommand())
	rootCmd.AddCommand(newStatusCommand())

	return rootCmd
}

// runInteractiveMode launches the TUI interface
func runInteractiveMode(verbose bool, version string) error {
	// Initialize application
	app, err := NewApp()
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	// Create and run TUI with mouse support
	model := tui.NewModel(app, verbose, version)
	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Ensure cleanup happens even if there's an error
	defer model.Cleanup()

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// activeFormat records the --format a command resolved, so a failure can be
// reported in the shape the caller asked for. It is set by outputFor before
// any work happens.
var activeFormat = FormatText

// ReportError prints a failed command's error, once.
//
// SilenceErrors is set on the root command, so this is the only place failures
// surface. Commands deliberately do NOT report their own errors: doing both
// printed every failure twice, and in JSON that meant a structured payload
// followed by a prose duplicate.
//
// Always stderr, never stdout, so a machine-readable stdout stays parseable
// (or empty) on failure. The exit code remains the primary signal.
func ReportError(err error) {
	if err == nil {
		return
	}
	NewOutputFormat(appVersion, activeFormat).EmitError(err)
}
