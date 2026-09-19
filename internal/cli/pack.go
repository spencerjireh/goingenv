package cli

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"goingenv/internal/manifest"
	"goingenv/pkg/types"
	"goingenv/pkg/utils"
)

// newPackCommand creates the pack command
func newPackCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack",
		Short: "Pack and encrypt environment files",
		Long: `Scan for environment files in the specified directory and create an encrypted archive.

The pack command:
- Scans for common environment file patterns (.env, .env.local, etc.)
- Calculates a checksum for each file
- Encrypts them using AES-256-GCM with an Argon2id-derived key
- Stores the archive in the .goingenv directory
- Optionally writes a manifest beside it: file paths and key names, never values

Examples:
  goingenv pack                                    # Interactive password prompt
  goingenv pack --password-env MY_PASSWORD        # Read from environment variable
  printf 'pw\n' | goingenv pack --password-stdin  # Read from stdin, no prompt
  goingenv pack --env prod                        # Write prod-<timestamp>.enc
  goingenv pack --manifest                        # Also write <archive>.manifest.json
  goingenv pack -d /path/to/project -o backup.enc # Specify directory and output
  goingenv pack -d . --depth 5                    # Custom scan depth
  goingenv pack --env-exclude '\.env\.backup$'    # Skip an env file by name`,
		RunE: runPackCommand,
	}

	addPasswordFlags(cmd)
	addEnvFlag(cmd)
	cmd.Flags().Bool("manifest", false, "Write <archive>.manifest.json listing file paths and key names (overrides config)")
	cmd.Flags().Bool("no-manifest", false, "Do not write a manifest (overrides config)")
	cmd.Flags().StringP("directory", "d", "", "Scan this directory (default: current directory)")
	cmd.Flags().StringP("output", "o", "", "Name the output archive (default: timestamped)")
	cmd.Flags().IntP("depth", "", 0, "Limit how many directories deep to scan (default: from config)")
	cmd.Flags().StringSliceP("include", "i", nil, "Include additional file patterns")
	cmd.Flags().StringSliceP("exclude", "e", nil, "Exclude additional directory patterns (matched against directory paths, not filenames)")
	cmd.Flags().StringSlice("env-exclude", nil, "Exclude additional env-file patterns (matched against the base filename)")
	cmd.Flags().BoolP("dry-run", "", false, "Show what would be packed without creating an archive")
	cmd.Flags().BoolP("verbose", "v", false, "Show detailed output")

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

	opts, err := parsePackOpts(cmd, app.Config)
	if err != nil {
		return err
	}

	out.Header()
	out.Blank()

	key, cleanup, err := getPass(opts.PassOpts, opts.Env, true)
	if err != nil {
		return err
	}
	defer cleanup()

	files, err := scanPackFiles(out, app, opts)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		out.Warning("No environment files found")
		out.Hint("Use 'goingenv status' to see detected files")
		return nil
	}

	displayPackFiles(out, files, opts.Verbose)

	if opts.DryRun {
		out.Success(fmt.Sprintf("Dry run: would create %s", opts.Output))
		return emitPackResult(out, files, opts, 0, true)
	}

	// With the password on stdin there is no terminal conversation to have,
	// and a confirm prompt would read from the same stream.
	if !opts.PassStdin && !confirm(fmt.Sprintf("Proceed with packing to %s?", opts.Output)) {
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
		out.Error(fmt.Sprintf("Failed to scan files: %v", err))
		return nil, err
	}
	return files, nil
}

// displayPackFiles shows every file to be packed; verbose adds the size.
func displayPackFiles(out *Output, files []types.EnvFile, verbose bool) {
	out.Action(fmt.Sprintf("Packing %d files...", len(files)))
	out.Blank()
	listFiles(out, files, verbose)
}

// executePack performs the actual packing operation
func executePack(out *Output, app *types.App, files []types.EnvFile, opts *PackOpts, key string) error { //nolint:unparam // error return kept for consistency
	packOpts := types.PackOptions{
		Files:      files,
		OutputPath: opts.Output,
		Password:   key,
		Description: fmt.Sprintf("Environment files archive created on %s from %s",
			time.Now().Format("2006-01-02 15:04:05"), opts.Dir),
		Env: opts.Env,
	}

	if opts.Verbose {
		out.Action("Encrypting...")
	}

	start := time.Now()
	err := app.Archiver.Pack(packOpts)
	duration := time.Since(start)

	if err != nil {
		return fmt.Errorf("failed to pack files: %w", err)
	}

	out.Success(fmt.Sprintf("Created %s", opts.Output))

	manifestPath, err := writeManifest(files, opts, packOpts.Description)
	if err != nil {
		return err
	}
	if manifestPath != "" {
		out.Success(fmt.Sprintf("Wrote %s", manifestPath))
	}

	if opts.Verbose {
		if info, statErr := os.Stat(opts.Output); statErr == nil {
			out.Indent(fmt.Sprintf("Files: %d", len(files)))
			out.Indent(fmt.Sprintf("Size: %s", utils.FormatSize(info.Size())))
			out.Indent(fmt.Sprintf("Time: %v", duration.Round(time.Millisecond)))
		}
	}

	out.Blank()
	out.Hint("The password is not stored anywhere. Lose it and the archive is unreadable")

	var archiveSize int64
	if info, statErr := os.Stat(opts.Output); statErr == nil {
		archiveSize = info.Size()
	}
	return emitPackResult(out, files, opts, archiveSize, false)
}

// writeManifest writes the manifest beside the archive when the pack asked
// for one and returns its path, or "" when none was requested. A failure is
// an error even though the archive already exists: a CI job that asked for a
// manifest must not go green without one.
func writeManifest(files []types.EnvFile, opts *PackOpts, description string) (string, error) {
	if !opts.Manifest {
		return "", nil
	}
	m, err := manifest.Build(opts.Output, opts.Env, description, time.Now(), files)
	if err != nil {
		return "", fmt.Errorf("archive %s was created, but the manifest could not be built: %w", opts.Output, err)
	}
	path := manifest.PathFor(opts.Output)
	if err := manifest.Write(path, m); err != nil {
		return "", fmt.Errorf("archive %s was created, but the manifest could not be written: %w", opts.Output, err)
	}
	return path, nil
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
		payload := PackPayload{
			Archive:   opts.Output,
			Env:       opts.Env,
			Files:     toFileInfos(files),
			Count:     len(files),
			TotalSize: archiveSize,
			DryRun:    dryRun,
		}
		if opts.Manifest && !dryRun {
			payload.Manifest = manifest.PathFor(opts.Output)
		}
		return out.EmitJSON(payload)
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
