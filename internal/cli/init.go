package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"goingenv/internal/config"
	"goingenv/pkg/types"
)

// newInitCommand creates the init command
func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize goingenv in the current directory",
		Long: `Initialize goingenv in the current directory.

The init command:
- Creates the .goingenv directory, where encrypted archives are stored
- Creates a default configuration file in your home directory if none exists

Encrypted archives (.enc files) are meant to be committed to git and shared
with your team.

With --project-config, a .goingenv/config.json is written as well. Commit it to
pin how this repository is scanned: when present it takes precedence over
~/.goingenv.json, so everyone packing this project gets the same file set
regardless of their personal configuration.

This must be run before using any other goingenv commands.

Examples:
  goingenv init
  goingenv init --project-config   # also pin scan rules for this repository`,
		RunE: runInitCommand,
	}

	cmd.Flags().BoolP("force", "f", false, "Reinitialize even if this directory is already set up")
	cmd.Flags().Bool("project-config", false, "Also write a committed .goingenv/config.json pinning this project's scan rules")
	cmd.Flags().BoolP("verbose", "v", false, "Show detailed output")

	return cmd
}

// maybeWriteProjectConfig writes .goingenv/config.json when the caller asked
// for one and none exists yet, reporting whether it created the file.
//
// An existing project config is never overwritten -- it is a committed file, so
// --force is about re-initializing the directory, not discarding the
// repository's pinned scan rules.
func maybeWriteProjectConfig(enabled bool, cfg *types.Config) (bool, error) {
	if !enabled {
		return false, nil
	}

	projMgr := config.NewManagerWithPath(config.ProjectConfigPath())
	if projMgr.Exists() {
		return false, nil
	}

	if saveErr := projMgr.Save(cfg); saveErr != nil {
		return false, saveErr
	}

	return true, nil
}

// runInitCommand executes the init command
func runInitCommand(cmd *cobra.Command, args []string) error {
	out, err := outputFor(cmd, false)
	if err != nil {
		return err
	}

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}

	projectConfig, err := cmd.Flags().GetBool("project-config")
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
		return emitInitResult(out, false, false)
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

	// Only write the user-level config when it does not exist yet. Saving
	// unconditionally rewrote the user's global config on every `init`.
	if !configMgr.Exists() {
		if err := configMgr.Save(cfg); err != nil {
			out.Error("Failed to save configuration")
			return fmt.Errorf("save failed: %w", err)
		}
	}

	wroteProjectConfig, projErr := maybeWriteProjectConfig(projectConfig, cfg)
	if projErr != nil {
		out.Error("Failed to save project configuration")
		return fmt.Errorf("project config save failed: %w", projErr)
	}

	if verbose {
		out.Success("Created .goingenv/")
		if wroteProjectConfig {
			out.Success("Created " + config.ProjectConfigPath())
		}
		out.Blank()
		out.Hint("Next steps:")
		out.Indent("Run 'goingenv status' to see detected files")
		out.Indent("Run 'goingenv pack' to create an archive")
	} else {
		out.Success("Initialized")
		out.Blank()
		out.Hint("Run 'goingenv status' to see detected files")
	}

	return emitInitResult(out, true, wroteProjectConfig)
}

// emitInitResult writes the machine-readable result, and nothing in the
// default text format -- the human output has already been printed.
//
// created is false when the project was already initialised. That is a
// success, so the exit code stays 0 and the distinction lives in the payload,
// where a script can act on it.
func emitInitResult(out *Output, created, wroteProjectConfig bool) error {
	projectConfig := ""
	if wroteProjectConfig {
		projectConfig = config.ProjectConfigPath()
	}

	switch out.Format() {
	case FormatJSON:
		return out.EmitJSON(InitPayload{
			Created:       created,
			GoingEnvDir:   config.GetGoingEnvDir(),
			ProjectConfig: projectConfig,
		})
	case FormatPorcelain:
		return out.EmitPorcelain([][]string{{
			config.GetGoingEnvDir(),
			strconv.FormatBool(created),
		}})
	default:
		return nil
	}
}
