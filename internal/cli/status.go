package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"goingenv/internal/config"
	"goingenv/internal/constants"
	"goingenv/pkg/types"
	"goingenv/pkg/utils"
)

// newStatusCommand creates the status command
func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [directory]",
		Short: "Show current status and available archives",
		Long: `Display comprehensive status information about the current environment.

The status command shows:
- Current directory information
- Available archives in .goingenv directory
- Detected environment files
- Configuration settings (with --verbose)

Examples:
  goingenv status
  goingenv status --verbose
  goingenv status /path/to/project`,
		RunE: runStatusCommand,
	}

	cmd.Flags().BoolP("verbose", "v", false, "Show detailed information")

	return cmd
}

// runStatusCommand executes the status command
func runStatusCommand(cmd *cobra.Command, args []string) error {
	out := NewOutput(appVersion)

	app, err := initApp()
	if err != nil {
		out.Header()
		out.Blank()
		out.Error(err.Error())
		return err
	}

	verbose, _ := cmd.Flags().GetBool("verbose") //nolint:errcheck // flag always exists

	directory := "."
	if len(args) > 0 {
		directory = args[0]
	}

	out.Header()
	out.Blank()

	displayDirectory(out, directory)
	displayConfig(out, app, verbose)
	files := displayFiles(out, app, directory, verbose)
	archives := displayArchives(out, app, verbose)

	// Hint for next steps
	if len(files) > 0 && len(archives) == 0 {
		out.Hint("Run 'goingenv pack' to create a new archive")
	}

	return nil
}

// displayDirectory shows the current directory section
func displayDirectory(out *Output, directory string) {
	cwd, _ := os.Getwd() //nolint:errcheck // best effort
	out.Section("Directory")
	if directory == "." {
		out.Indent(cwd)
	} else {
		absDir, _ := filepath.Abs(directory) //nolint:errcheck // best effort
		out.Indent(absDir)
	}
	out.Blank()
}

// displayConfig shows the configuration section (verbose only)
func displayConfig(out *Output, app *types.App, verbose bool) {
	if !verbose {
		return
	}
	out.Section("Configuration")
	out.Indent(fmt.Sprintf("Scan depth: %d", app.Config.DefaultDepth))
	out.Indent(fmt.Sprintf("Max file size: %s", utils.FormatSize(app.Config.MaxFileSize)))
	out.Indent(fmt.Sprintf("Config: %s", config.GetGoingEnvDir()))
	out.Blank()
}

// displayFiles shows the environment files section and returns found files
func displayFiles(out *Output, app *types.App, directory string, verbose bool) []types.EnvFile {
	scanOpts := types.ScanOptions{
		RootPath: directory,
		MaxDepth: app.Config.DefaultDepth,
	}

	files, err := app.Scanner.ScanFiles(&scanOpts)
	switch {
	case err != nil:
		out.Warning(fmt.Sprintf("Could not scan files: %v", err))
	case len(files) == 0:
		out.Section("Environment Files (0)")
		out.MutedPrint("  No environment files detected")
		out.Blank()
	default:
		out.Section(fmt.Sprintf("Environment Files (%d)", len(files)))
		for i, file := range files {
			switch {
			case verbose:
				out.Indent(fmt.Sprintf("%-25s %10s   %s",
					file.RelativePath,
					utils.FormatSize(file.Size),
					file.ModTime.Format(constants.DateTimeFormat)))
			case i < 10:
				out.Indent(file.RelativePath)
			case i == 10:
				out.Indent(fmt.Sprintf("... and %d more", len(files)-10))
				return files
			}
		}
		out.Blank()
	}
	return files
}

// displayArchives shows the archives section and returns found archives
func displayArchives(out *Output, app *types.App, verbose bool) []string {
	archives, err := app.Archiver.GetAvailableArchives("")
	switch {
	case err != nil:
		out.Warning(fmt.Sprintf("Could not read archives: %v", err))
	case len(archives) == 0:
		out.Section("Archives (0)")
		out.MutedPrint("  No archives found")
		out.Blank()
	default:
		out.Section(fmt.Sprintf("Archives (%d)", len(archives)))
		for _, archivePath := range archives {
			info, statErr := os.Stat(archivePath)
			if statErr == nil {
				if verbose {
					out.Indent(fmt.Sprintf("%-25s %10s   %s",
						filepath.Base(archivePath),
						utils.FormatSize(info.Size()),
						info.ModTime().Format(constants.DateTimeFormat)))
				} else {
					out.Indent(fmt.Sprintf("%s    %s    %s",
						filepath.Base(archivePath),
						utils.FormatSize(info.Size()),
						utils.FormatTimeAgo(info.ModTime())))
				}
			}
		}
		out.Blank()
	}
	return archives
}
