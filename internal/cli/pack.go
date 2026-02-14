package cli

import (
	"fmt"
	"os"
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
  goingenv pack -d . --depth 5                    # Custom scan depth`,
		RunE: runPackCommand,
	}

	cmd.Flags().String("password-env", "", "Read password from environment variable")
	cmd.Flags().StringP("directory", "d", "", "Directory to scan (default: current directory)")
	cmd.Flags().StringP("output", "o", "", "Output archive name (default: auto-generated with timestamp)")
	cmd.Flags().IntP("depth", "", 0, "Maximum directory depth to scan (default: from config)")
	cmd.Flags().StringSliceP("include", "i", nil, "Additional file patterns to include")
	cmd.Flags().StringSliceP("exclude", "e", nil, "Additional patterns to exclude")
	cmd.Flags().BoolP("dry-run", "", false, "Show what would be packed without creating archive")
	cmd.Flags().BoolP("verbose", "v", false, "Show detailed information during packing")

	return cmd
}

// runPackCommand executes the pack command
func runPackCommand(cmd *cobra.Command, args []string) error {
	out := NewOutput(appVersion)

	app, err := initApp()
	if err != nil {
		out.Header()
		out.Blank()
		out.Error(err.Error())
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
		return nil
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

	return nil
}
