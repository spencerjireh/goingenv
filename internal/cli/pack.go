package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"goingenv/internal/config"
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
  goingenv pack -k "mypassword"
  goingenv pack -k "mypassword" -d /path/to/project
  goingenv pack -k "mypassword" -o backup-prod.enc
  goingenv pack -d . --depth 5`,
		RunE: runPackCommand,
	}

	// Add flags
	cmd.Flags().StringP("key", "k", "", "Encryption password (will prompt if not provided)")
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
	// Check if GoingEnv is initialized
	if !config.IsInitialized() {
		return fmt.Errorf("GoingEnv is not initialized in this directory. Run 'goingenv init' first")
	}

	// Initialize application
	app, err := NewApp()
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	// Parse flags
	directory, _ := cmd.Flags().GetString("directory")
	if directory == "" {
		directory = "."
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "" {
		output = config.GetDefaultArchivePath()
	} else {
		// Ensure output is in .goingenv directory
		if !filepath.IsAbs(output) {
			output = filepath.Join(config.GetGoingEnvDir(), output)
		}
	}

	key, _ := cmd.Flags().GetString("key")
	depth, _ := cmd.Flags().GetInt("depth")
	includePatterns, _ := cmd.Flags().GetStringSlice("include")
	excludePatterns, _ := cmd.Flags().GetStringSlice("exclude")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Prompt for password if not provided
	if key == "" {
		fmt.Print("Enter encryption password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		key = string(passwordBytes)
	}

	if key == "" {
		return fmt.Errorf("password cannot be empty")
	}

	// Prepare scan options
	scanOpts := types.ScanOptions{
		RootPath: directory,
		MaxDepth: depth,
		Patterns: includePatterns,
		ExcludePatterns: excludePatterns,
	}

	// Use config defaults if not specified
	if scanOpts.MaxDepth == 0 {
		scanOpts.MaxDepth = app.Config.DefaultDepth
	}
	if len(scanOpts.Patterns) == 0 {
		scanOpts.Patterns = app.Config.EnvPatterns
	}
	if len(scanOpts.ExcludePatterns) == 0 {
		scanOpts.ExcludePatterns = app.Config.ExcludePatterns
	} else {
		// Merge with config excludes
		scanOpts.ExcludePatterns = append(scanOpts.ExcludePatterns, app.Config.ExcludePatterns...)
	}

	if verbose {
		fmt.Printf("Scanning directory: %s\n", directory)
		fmt.Printf("Maximum depth: %d\n", scanOpts.MaxDepth)
		fmt.Printf("Include patterns: %v\n", scanOpts.Patterns)
		fmt.Printf("Exclude patterns: %v\n", scanOpts.ExcludePatterns)
		fmt.Println()
	}

	// Scan for files
	files, err := app.Scanner.ScanFiles(scanOpts)
	if err != nil {
		return fmt.Errorf("error scanning files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No environment files found matching the specified criteria.")
		if verbose {
			fmt.Println("\nTip: Use 'goingenv status' to see what files are detected with current settings.")
		}
		return nil
	}

	// Display found files
	fmt.Printf("Found %d environment files:\n", len(files))
	var totalSize int64
	for _, file := range files {
		totalSize += file.Size
		if verbose {
			fmt.Printf("  • %s (%s) - %s - %s\n", 
				file.RelativePath, 
				utils.FormatSize(file.Size),
				file.ModTime.Format("2006-01-02 15:04:05"),
				file.Checksum[:8]+"...")
		} else {
			fmt.Printf("  • %s (%s)\n", file.RelativePath, utils.FormatSize(file.Size))
		}
	}
	fmt.Printf("\nTotal size: %s\n", utils.FormatSize(totalSize))

	// Dry run - exit here if requested
	if dryRun {
		fmt.Printf("\nDry run completed. Archive would be created at: %s\n", output)
		return nil
	}

	// Confirm before proceeding (unless in non-interactive mode)
	if term.IsTerminal(int(syscall.Stdin)) {
		fmt.Printf("\nProceed with packing to %s? [y/N]: ", output)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" && response != "yes" {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}


	// Prepare pack options
	packOpts := types.PackOptions{
		Files:       files,
		OutputPath:  output,
		Password:    key,
		Description: fmt.Sprintf("Environment files archive created on %s from %s", 
			time.Now().Format("2006-01-02 15:04:05"), directory),
	}

	if verbose {
		fmt.Printf("\nPacking files to %s...\n", output)
	}

	// Pack files
	startTime := time.Now()
	err = app.Archiver.Pack(packOpts)
	if err != nil {
		return fmt.Errorf("error packing files: %w", err)
	}
	duration := time.Since(startTime)

	// Success message
	fmt.Printf("✅ Successfully packed %d files to %s\n", len(files), output)
	
	if verbose {
		fmt.Printf("Operation completed in %v\n", duration)
		
		// Show archive info
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

	// Security reminder
	fmt.Println("\n🔒 Security reminder:")
	fmt.Println("   • Store your password securely")
	fmt.Println("   • Consider backing up the archive to a secure location")
	fmt.Println("   • Use 'goingenv list' to verify archive contents")

	return nil
}