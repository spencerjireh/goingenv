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

// newUnpackCommand creates the unpack command
func newUnpackCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unpack",
		Short: "Unpack and decrypt archived files",
		Long: `Decrypt and extract files from an encrypted archive.

The unpack command will:
- Decrypt the specified archive using the provided password
- Verify file integrity using stored checksums
- Extract files to the specified directory (default: current directory)
- Optionally create backups of existing files before overwriting

Examples:
  goingenv unpack -k "mypassword"
  goingenv unpack -f backup-prod.enc -k "mypassword"
  goingenv unpack -k "mypassword" --target /path/to/extract
  goingenv unpack -f archive.enc -k "mypassword" --overwrite --backup`,
		RunE: runUnpackCommand,
	}

	// Add flags
	cmd.Flags().StringP("key", "k", "", "Decryption password (will prompt if not provided)")
	cmd.Flags().StringP("file", "f", "", "Archive file to unpack (default: most recent)")
	cmd.Flags().StringP("target", "t", "", "Target directory for extraction (default: current directory)")
	cmd.Flags().Bool("overwrite", false, "Overwrite existing files without prompting")
	cmd.Flags().Bool("backup", false, "Create backups of existing files before overwriting")
	cmd.Flags().Bool("verify", true, "Verify file checksums after extraction")
	cmd.Flags().BoolP("verbose", "v", false, "Show detailed information during unpacking")
	cmd.Flags().BoolP("dry-run", "", false, "Show what would be extracted without actually doing it")
	cmd.Flags().StringSliceP("include", "i", nil, "Only extract files matching these patterns")
	cmd.Flags().StringSliceP("exclude", "e", nil, "Skip files matching these patterns")

	return cmd
}

// runUnpackCommand executes the unpack command
func runUnpackCommand(cmd *cobra.Command, args []string) error {
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
	archiveFile, _ := cmd.Flags().GetString("file")
	if archiveFile == "" {
		// Find the most recent archive
		archives, err := app.Archiver.GetAvailableArchives("")
		if err != nil {
			return fmt.Errorf("failed to find archives: %w", err)
		}
		if len(archives) == 0 {
			return fmt.Errorf("no archives found in %s directory. Use -f flag to specify an archive", config.GetGoingEnvDir())
		}
		archiveFile = archives[len(archives)-1] // Use the last one (most recent)
		fmt.Printf("Using most recent archive: %s\n", filepath.Base(archiveFile))
	}

	// Verify archive exists
	if _, err := os.Stat(archiveFile); os.IsNotExist(err) {
		return fmt.Errorf("archive file not found: %s", archiveFile)
	}

	key, _ := cmd.Flags().GetString("key")
	targetDir, _ := cmd.Flags().GetString("target")
	if targetDir == "" {
		targetDir = "."
	}

	overwrite, _ := cmd.Flags().GetBool("overwrite")
	backup, _ := cmd.Flags().GetBool("backup")
	verify, _ := cmd.Flags().GetBool("verify")
	verbose, _ := cmd.Flags().GetBool("verbose")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	includePatterns, _ := cmd.Flags().GetStringSlice("include")
	excludePatterns, _ := cmd.Flags().GetStringSlice("exclude")

	// Prompt for password if not provided
	if key == "" {
		fmt.Print("Enter decryption password: ")
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

	if verbose {
		fmt.Printf("Archive: %s\n", archiveFile)
		fmt.Printf("Target directory: %s\n", targetDir)
		fmt.Printf("Overwrite mode: %v\n", overwrite)
		fmt.Printf("Backup mode: %v\n", backup)
		fmt.Println()
	}

	// First, list the archive contents to show what will be extracted
	fmt.Printf("Reading archive: %s\n", filepath.Base(archiveFile))
	archive, err := app.Archiver.List(archiveFile, key)
	if err != nil {
		return fmt.Errorf("failed to read archive (check password): %w", err)
	}

	// Filter files if patterns are specified
	filesToExtract := archive.Files
	if len(includePatterns) > 0 || len(excludePatterns) > 0 {
		filesToExtract = filterFiles(archive.Files, includePatterns, excludePatterns)
	}

	// Display archive information
	fmt.Printf("Archive created: %s\n", archive.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Archive version: %s\n", archive.Version)
	if archive.Description != "" {
		fmt.Printf("Description: %s\n", archive.Description)
	}
	fmt.Printf("Files to extract: %d of %d total\n", len(filesToExtract), len(archive.Files))

	if verbose || len(filesToExtract) <= 20 {
		fmt.Println("\nFiles to extract:")
		for i, file := range filesToExtract {
			if i < 20 {
				fmt.Printf("  • %s (%s) - %s\n",
					file.RelativePath,
					utils.FormatSize(file.Size),
					file.ModTime.Format("2006-01-02 15:04:05"))
			} else if i == 20 {
				fmt.Printf("  • ... and %d more files\n", len(filesToExtract)-20)
				break
			}
		}
	}

	// Check for conflicts with existing files
	conflicts := checkFileConflicts(filesToExtract, targetDir)
	if len(conflicts) > 0 && !overwrite {
		fmt.Printf("\n⚠️  Found %d existing files that would be overwritten:\n", len(conflicts))
		for i, conflict := range conflicts {
			if i < 10 {
				fmt.Printf("  • %s\n", conflict)
			} else if i == 10 {
				fmt.Printf("  • ... and %d more files\n", len(conflicts)-10)
				break
			}
		}

		if !dryRun {
			fmt.Printf("\nUse --overwrite to replace existing files, or --backup to create backups.\n")
			fmt.Printf("Continue anyway? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" && response != "yes" {
				fmt.Println("Operation cancelled.")
				return nil
			}
			overwrite = true // User confirmed
		}
	}

	// Dry run - exit here if requested
	if dryRun {
		fmt.Printf("\nDry run completed. %d files would be extracted to %s\n", 
			len(filesToExtract), targetDir)
		if len(conflicts) > 0 {
			fmt.Printf("%d existing files would be affected\n", len(conflicts))
		}
		return nil
	}

	// Prepare unpack options
	unpackOpts := types.UnpackOptions{
		ArchivePath: archiveFile,
		Password:    key,
		TargetDir:   targetDir,
		Overwrite:   overwrite,
		Backup:      backup,
	}

	if verbose {
		fmt.Printf("\nExtracting files to %s...\n", targetDir)
	}

	// Unpack files
	startTime := time.Now()
	err = app.Archiver.Unpack(unpackOpts)
	if err != nil {
		return fmt.Errorf("error unpacking files: %w", err)
	}
	duration := time.Since(startTime)

	// Verify extracted files if requested
	if verify {
		fmt.Printf("Verifying extracted files...\n")
		verifyErrors := verifyExtractedFiles(filesToExtract, targetDir)
		if len(verifyErrors) > 0 {
			fmt.Printf("⚠️  Verification warnings:\n")
			for _, verifyErr := range verifyErrors {
				fmt.Printf("  • %s\n", verifyErr)
			}
		} else if verbose {
			fmt.Printf("✅ All files verified successfully\n")
		}
	}

	// Success message
	fmt.Printf("✅ Successfully extracted %d files from %s\n", 
		len(filesToExtract), filepath.Base(archiveFile))
	
	if verbose {
		fmt.Printf("Operation completed in %v\n", duration)
	}

	// Show summary of what was done
	if len(conflicts) > 0 {
		if backup {
			fmt.Printf("📋 Created backups for %d existing files\n", len(conflicts))
		} else {
			fmt.Printf("📝 Overwrote %d existing files\n", len(conflicts))
		}
	}

	// Helpful next steps
	fmt.Println("\n💡 Next steps:")
	fmt.Println("   • Review extracted files for correctness")
	fmt.Println("   • Update any file permissions if needed")
	if len(conflicts) > 0 && backup {
		fmt.Println("   • Remove .backup files once you've verified the extraction")
	}

	return nil
}

// Helper functions

// filterFiles filters files based on include/exclude patterns
func filterFiles(files []types.EnvFile, includePatterns, excludePatterns []string) []types.EnvFile {
	var filtered []types.EnvFile
	
	for _, file := range files {
		// Check include patterns
		if len(includePatterns) > 0 {
			included := false
			for _, pattern := range includePatterns {
				if matched, _ := filepath.Match(pattern, file.RelativePath); matched {
					included = true
					break
				}
			}
			if !included {
				continue
			}
		}
		
		// Check exclude patterns
		excluded := false
		for _, pattern := range excludePatterns {
			if matched, _ := filepath.Match(pattern, file.RelativePath); matched {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}
		
		filtered = append(filtered, file)
	}
	
	return filtered
}

// checkFileConflicts checks for existing files that would be overwritten
func checkFileConflicts(files []types.EnvFile, targetDir string) []string {
	var conflicts []string
	
	for _, file := range files {
		targetPath := filepath.Join(targetDir, file.RelativePath)
		if _, err := os.Stat(targetPath); err == nil {
			conflicts = append(conflicts, file.RelativePath)
		}
	}
	
	return conflicts
}

// verifyExtractedFiles verifies that extracted files match their expected checksums
func verifyExtractedFiles(files []types.EnvFile, targetDir string) []string {
	var errors []string
	
	for _, file := range files {
		targetPath := filepath.Join(targetDir, file.RelativePath)
		
		// Check if file exists
		info, err := os.Stat(targetPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: file not found after extraction", file.RelativePath))
			continue
		}
		
		// Check file size
		if info.Size() != file.Size {
			errors = append(errors, fmt.Sprintf("%s: size mismatch (expected %d, got %d)", 
				file.RelativePath, file.Size, info.Size()))
			continue
		}
		
		// Calculate and verify checksum
		actualChecksum, err := utils.CalculateFileChecksum(targetPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to calculate checksum: %v", 
				file.RelativePath, err))
			continue
		}
		
		if actualChecksum != file.Checksum {
			errors = append(errors, fmt.Sprintf("%s: checksum mismatch", file.RelativePath))
		}
	}
	
	return errors
}