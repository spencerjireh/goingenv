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

// showScanOpts displays scan options in verbose mode
func showScanOpts(opts *types.ScanOptions, verbose bool) {
	if !verbose {
		return
	}
	fmt.Printf("Scanning directory: %s\n", opts.RootPath)
	fmt.Printf("Maximum depth: %d\n", opts.MaxDepth)
	fmt.Printf("Include patterns: %v\n", opts.Patterns)
	fmt.Printf("Exclude patterns: %v\n", opts.ExcludePatterns)
	fmt.Println()
}

// showFiles displays found files and returns total size
func showFiles(files []types.EnvFile, verbose bool) int64 {
	fmt.Printf("Found %d environment files:\n", len(files))
	var totalSize int64
	for _, file := range files {
		totalSize += file.Size
		if verbose {
			fmt.Printf("  - %s (%s) - %s - %s\n",
				file.RelativePath,
				utils.FormatSize(file.Size),
				file.ModTime.Format("2006-01-02 15:04:05"),
				file.Checksum[:8]+"...")
		} else {
			fmt.Printf("  - %s (%s)\n", file.RelativePath, utils.FormatSize(file.Size))
		}
	}
	fmt.Printf("\nTotal size: %s\n", utils.FormatSize(totalSize))
	return totalSize
}

// doPack performs the actual packing
func doPack(app *types.App, opts types.PackOptions) (time.Duration, error) {
	start := time.Now()
	err := app.Archiver.Pack(opts)
	return time.Since(start), err
}

// showPackResult displays pack result
func showPackResult(output string, count int, totalSize int64, duration time.Duration, verbose bool) {
	fmt.Printf("Successfully packed %d files to %s\n", count, output)

	if verbose {
		fmt.Printf("Operation completed in %v\n", duration)

		if info, err := os.Stat(output); err == nil {
			compressionRatio := float64(info.Size()) / float64(totalSize) * 100
			fmt.Printf("Archive size: %s (%.1f%% of original)\n",
				utils.FormatSize(info.Size()), compressionRatio)
		}

		fmt.Printf("Archive checksum: calculating...\n")
		if checksum, err := utils.CalculateFileChecksum(output); err == nil {
			fmt.Printf("Archive SHA-256: %s\n", checksum)
		}
	}

	fmt.Println("\nSecurity reminder:")
	fmt.Println("   - Store your password securely")
	fmt.Println("   - Consider backing up the archive to a secure location")
	fmt.Println("   - Use 'goingenv list' to verify archive contents")
}

// runPackCommand executes the pack command
func runPackCommand(cmd *cobra.Command, args []string) error {
	app, err := initApp()
	if err != nil {
		return err
	}

	opts, err := parsePackOpts(cmd)
	if err != nil {
		return err
	}

	key, cleanup, err := getPass(opts.PassEnv)
	if err != nil {
		return err
	}
	defer cleanup()

	scanOpts := buildScanOpts(opts, app.Config)
	showScanOpts(scanOpts, opts.Verbose)

	files, err := app.Scanner.ScanFiles(scanOpts)
	if err != nil {
		return fmt.Errorf("error scanning files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No environment files found matching the specified criteria.")
		if opts.Verbose {
			fmt.Println("\nTip: Use 'goingenv status' to see what files are detected with current settings.")
		}
		return nil
	}

	totalSize := showFiles(files, opts.Verbose)

	if opts.DryRun {
		fmt.Printf("\nDry run completed. Archive would be created at: %s\n", opts.Output)
		return nil
	}

	if !confirm(fmt.Sprintf("Proceed with packing to %s?", opts.Output)) {
		fmt.Println("Operation cancelled.")
		return nil
	}

	packOpts := types.PackOptions{
		Files:      files,
		OutputPath: opts.Output,
		Password:   key,
		Description: fmt.Sprintf("Environment files archive created on %s from %s",
			time.Now().Format("2006-01-02 15:04:05"), opts.Dir),
	}

	if opts.Verbose {
		fmt.Printf("\nPacking files to %s...\n", opts.Output)
	}

	duration, err := doPack(app, packOpts)
	if err != nil {
		return fmt.Errorf("error packing files: %w", err)
	}

	showPackResult(opts.Output, len(files), totalSize, duration, opts.Verbose)

	return nil
}
