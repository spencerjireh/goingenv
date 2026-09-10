package cli

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"goingenv/pkg/types"
	"goingenv/pkg/utils"
)

// newPackCommand creates the pack command
func newPackCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack",
		Short: "Pack and encrypt environment files",
		Long: `Scan for environment files in the specified directory and create an encrypted archive.

The pack command will:
- Scan for common environment file patterns (.env, .env.local, etc.)
- Calculate checksums for integrity verification
- Encrypt files using AES-256-GCM with PBKDF2 key derivation
- Store the encrypted archive in the .goingenv directory

Examples:
  goingenv pack                                    # Interactive password prompt
  goingenv pack --password-env MY_PASSWORD        # Read from environment variable
  goingenv pack -d /path/to/project -o backup.enc # Specify directory and output
  goingenv pack -d . --depth 5                    # Custom scan depth
  goingenv pack --env-exclude '\.env\.backup$'    # Skip an env file by name`,
		RunE: runPackCommand,
	}

	cmd.Flags().String("password-env", "", "Read password from environment variable")
	cmd.Flags().StringP("directory", "d", "", "Directory to scan (default: current directory)")
	cmd.Flags().StringP("output", "o", "", "Output archive name (default: auto-generated with timestamp)")
	cmd.Flags().IntP("depth", "", 0, "Maximum directory depth to scan (default: from config)")
	cmd.Flags().StringSliceP("include", "i", nil, "Additional file patterns to include")
	cmd.Flags().StringSliceP("exclude", "e", nil, "Additional directory patterns to exclude (matched against directory paths, not filenames)")
	cmd.Flags().StringSlice("env-exclude", nil, "Additional env-file patterns to exclude (matched against the base filename)")
	cmd.Flags().BoolP("dry-run", "", false, "Show what would be packed without creating archive")
	cmd.Flags().BoolP("verbose", "v", false, "Show detailed information during packing")

	return cmd
}

// runPackCommand executes the pack command
func runPackCommand(cmd *cobra.Command, args []string) error {
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

	opts, err := parsePackOpts(cmd)
	if err != nil {
		return err
	}

	out.Header()
	out.Blank()

	key, cleanup, err := getPass(opts.PassEnv)
	if err != nil {
		out.Error(fmt.Sprintf("Failed to get password: %v", err))
		return err
	}
	defer cleanup()

	files, err := scanPackFiles(out, app, opts)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		out.Warning("No environment files found")
		out.Hint("Use 'goingenv status' to see what files are detected")
		return nil
	}

	displayPackFiles(out, files, opts.Verbose)

	if opts.DryRun {
		out.Success(fmt.Sprintf("Dry run: would create %s", opts.Output))
		return emitPackResult(out, files, opts, 0, true)
	}

	if !confirm(fmt.Sprintf("Proceed with packing to %s?", opts.Output)) {
		out.Skipped("Operation cancelled")
		return nil
	}

	return executePack(out, app, files, opts, key)
}

// scanPackFiles scans for files to pack
func scanPackFiles(out *Output, app *types.App, opts *PackOpts) ([]types.EnvFile, error) {
	scanOpts := buildScanOpts(opts, app.Config)

	if opts.Verbose {
		out.Action(fmt.Sprintf("Scanning %s...", scanOpts.RootPath))
	}

	files, err := app.Scanner.ScanFiles(scanOpts)
	if err != nil {
		out.Error(fmt.Sprintf("Error scanning files: %v", err))
		return nil, err
	}
	return files, nil
}

// displayPackFiles shows the files to be packed
func displayPackFiles(out *Output, files []types.EnvFile, verbose bool) {
	out.Action(fmt.Sprintf("Packing %d files...", len(files)))
	out.Blank()

	for i, file := range files {
		switch {
		case verbose:
			out.ListItem(fmt.Sprintf("%s (%s)", file.RelativePath, utils.FormatSize(file.Size)))
		case i < 5:
			out.ListItem(file.RelativePath)
		case i == 5:
			out.ListItem(fmt.Sprintf("... and %d more files", len(files)-5))
			out.Blank()
			return
		}
	}
	out.Blank()
}

// executePack performs the actual packing operation
func executePack(out *Output, app *types.App, files []types.EnvFile, opts *PackOpts, key string) error { //nolint:unparam // error return kept for consistency
	packOpts := types.PackOptions{
		Files:      files,
		OutputPath: opts.Output,
		Password:   key,
		Description: fmt.Sprintf("Environment files archive created on %s from %s",
			time.Now().Format("2006-01-02 15:04:05"), opts.Dir),
	}

	if opts.Verbose {
		out.Action("Encrypting...")
	}

	start := time.Now()
	err := app.Archiver.Pack(packOpts)
	duration := time.Since(start)

	if err != nil {
		out.Error(fmt.Sprintf("Error packing files: %v", err))
		return err
	}

	out.Success(fmt.Sprintf("Created %s", opts.Output))

	if opts.Verbose {
		if info, statErr := os.Stat(opts.Output); statErr == nil {
			out.Indent(fmt.Sprintf("Files: %d", len(files)))
			out.Indent(fmt.Sprintf("Size: %s", utils.FormatSize(info.Size())))
			out.Indent(fmt.Sprintf("Time: %v", duration.Round(time.Millisecond)))
		}
	}

	out.Blank()
	out.Hint("Store your password securely")

	var archiveSize int64
	if info, statErr := os.Stat(opts.Output); statErr == nil {
		archiveSize = info.Size()
	}
	return emitPackResult(out, files, opts, archiveSize, false)
}

// emitPackResult writes the machine-readable result, and nothing at all in the
// default text format -- the human output has already been printed.
//
// TotalSize is the size of the packed archive rather than the sum of the
// inputs: a script asking about pack wants to know what landed on disk, and
// the per-file sizes are in Files if it wants the other number. On a dry run
// no archive exists, so it is zero.
func emitPackResult(out *Output, files []types.EnvFile, opts *PackOpts, archiveSize int64, dryRun bool) error {
	switch out.Format() {
	case FormatJSON:
		return out.EmitJSON(PackPayload{
			Archive:   opts.Output,
			Files:     toFileInfos(files),
			Count:     len(files),
			TotalSize: archiveSize,
			DryRun:    dryRun,
		})
	case FormatPorcelain:
		records := make([][]string, 0, len(files))
		for _, f := range files {
			records = append(records, []string{
				f.RelativePath,
				strconv.FormatInt(f.Size, 10),
				f.ModTime.UTC().Format(time.RFC3339),
			})
		}
		return out.EmitPorcelain(records)
	default:
		return nil
	}
}
