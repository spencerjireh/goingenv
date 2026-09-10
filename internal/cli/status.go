package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

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
	out, err := outputFor(cmd, false)
	if err != nil {
		return err
	}

	app, err := initApp()
	if err != nil {
		out.Header()
		out.Blank()
		return err
	}

	verbose, _ := cmd.Flags().GetBool("verbose") //nolint:errcheck // flag always exists

	directory := "."
	if len(args) > 0 {
		directory = args[0]
	}

	if out.Format().IsMachine() {
		return runStatusMachine(out, app, directory)
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
	out.Indent(fmt.Sprintf("Config: %s", config.ResolveConfigPath()))
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
				out.Blank()
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

// runStatusMachine reports status as a machine-readable payload.
//
// Kept separate from the human path rather than threading conditionals through
// it: the two differ in what they collect, not just how they print. The human
// view truncates to ten files and hides configuration unless --verbose, both
// of which would be wrong here -- a script asking for status wants all of it,
// every time.
func runStatusMachine(out *Output, app *types.App, directory string) error {
	cwd, _ := os.Getwd() //nolint:errcheck // best effort
	resolved := cwd
	if directory != "." {
		if abs, absErr := filepath.Abs(directory); absErr == nil {
			resolved = abs
		}
	}

	scanOpts := types.ScanOptions{RootPath: directory, MaxDepth: app.Config.DefaultDepth}
	files, scanErr := app.Scanner.ScanFiles(&scanOpts)
	if scanErr != nil {
		// A scan failure is the whole answer to "what is here", so it is an
		// error rather than an empty list a script would misread as "clean".
		out.EmitError(fmt.Errorf("could not scan files: %w", scanErr))
		return fmt.Errorf("could not scan files: %w", scanErr)
	}

	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}

	archives := collectArchives(app)

	payload := StatusPayload{
		Directory:   resolved,
		Initialized: config.IsInitialized(),
		Config:      describeConfig(app),
		EnvFiles:    toFileInfos(files),
		Archives:    archives,
		TotalSize:   totalSize,
	}

	if out.Format() == FormatJSON {
		return out.EmitJSON(payload)
	}

	records := make([][]string, 0, len(files)+len(archives))
	for _, f := range payload.EnvFiles {
		records = append(records, []string{
			porcelainKindEnv, f.Path,
			strconv.FormatInt(f.Size, 10),
			f.Modified.UTC().Format(time.RFC3339),
		})
	}
	for _, a := range archives {
		records = append(records, []string{
			porcelainKindArchive, a.Name,
			strconv.FormatInt(a.Size, 10),
			a.Modified.UTC().Format(time.RFC3339),
		})
	}
	return out.EmitPorcelain(records)
}

// collectArchives lists archives with their on-disk metadata, skipping any
// that cannot be stat'd -- the same tolerance the human view applies.
func collectArchives(app *types.App) []ArchiveInfo {
	paths, err := app.Archiver.GetAvailableArchives("")
	if err != nil {
		return []ArchiveInfo{}
	}

	archives := make([]ArchiveInfo, 0, len(paths))
	for _, p := range paths {
		info, statErr := os.Stat(p)
		if statErr != nil {
			continue
		}
		archives = append(archives, ArchiveInfo{
			Name:     filepath.Base(p),
			Path:     p,
			Size:     info.Size(),
			Modified: info.ModTime(),
		})
	}
	return archives
}

// describeConfig reports the effective configuration and which tier it came
// from. Project-local config takes whole-file precedence over the per-user
// one, so "which file won" is the thing worth reporting.
func describeConfig(app *types.App) ConfigInfo {
	path := config.ResolveConfigPath()
	source := "user"
	if path == config.ProjectConfigPath() {
		source = "project"
	}

	return ConfigInfo{
		Path:        path,
		Source:      source,
		Depth:       app.Config.DefaultDepth,
		MaxFileSize: app.Config.MaxFileSize,
		Patterns:    app.Config.EnvPatterns,
		Excludes:    app.Config.ExcludePatterns,
		EnvExcludes: app.Config.EnvExcludePatterns,
	}
}
