package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

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
  goingenv unpack                                         # Interactive password prompt
  goingenv unpack --password-env MY_PASSWORD             # Read from environment variable
  goingenv unpack -f backup-prod.enc --target /path/to/extract  # Specify archive and target
  goingenv unpack -f archive.enc --overwrite --backup    # Overwrite with backup`,
		RunE: runUnpackCommand,
	}

	cmd.Flags().String("password-env", "", "Read password from environment variable")
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

// showArchive displays archive info and files
func showArchive(archive *types.Archive, files []types.EnvFile, verbose bool) {
	fmt.Printf("Archive created: %s\n", archive.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Archive version: %s\n", archive.Version)
	if archive.Description != "" {
		fmt.Printf("Description: %s\n", archive.Description)
	}
	fmt.Printf("Files to extract: %d of %d total\n", len(files), len(archive.Files))

	if verbose || len(files) <= 20 {
		fmt.Println("\nFiles to extract:")
		for i, file := range files {
			if i < 20 {
				fmt.Printf("  - %s (%s) - %s\n",
					file.RelativePath,
					utils.FormatSize(file.Size),
					file.ModTime.Format("2006-01-02 15:04:05"))
			} else if i == 20 {
				fmt.Printf("  - ... and %d more files\n", len(files)-20)
				break
			}
		}
	}
}

// showConflicts displays conflicting files
func showConflicts(conflicts []string, limit int) {
	fmt.Printf("\nWarning: Found %d existing files that would be overwritten:\n", len(conflicts))
	for i, conflict := range conflicts {
		if i < limit {
			fmt.Printf("  - %s\n", conflict)
		} else if i == limit {
			fmt.Printf("  - ... and %d more files\n", len(conflicts)-limit)
			break
		}
	}
}

// showUnpackResult displays extraction result
func showUnpackResult(files []types.EnvFile, conflicts []string, duration time.Duration, opts *UnpackOpts) {
	fmt.Printf("Successfully extracted %d files\n", len(files))

	if opts.Verbose {
		fmt.Printf("Operation completed in %v\n", duration)
	}

	if len(conflicts) > 0 {
		if opts.Backup {
			fmt.Printf("Created backups for %d existing files\n", len(conflicts))
		} else {
			fmt.Printf("Overwrote %d existing files\n", len(conflicts))
		}
	}

	fmt.Println("\nNext steps:")
	fmt.Println("   - Review extracted files for correctness")
	fmt.Println("   - Update any file permissions if needed")
	if len(conflicts) > 0 && opts.Backup {
		fmt.Println("   - Remove .backup files once you've verified the extraction")
	}
}

// filterArchiveFiles filters files based on include/exclude patterns
func filterArchiveFiles(files []types.EnvFile, include, exclude []string) []types.EnvFile {
	if len(include) == 0 && len(exclude) == 0 {
		return files
	}
	return filterFiles(files, include, exclude)
}

// doUnpack performs the actual unpacking
func doUnpack(app *types.App, opts types.UnpackOptions) (time.Duration, error) {
	start := time.Now()
	err := app.Archiver.Unpack(opts)
	return time.Since(start), err
}

// verifyFiles verifies extracted files and displays results
func verifyFiles(files []types.EnvFile, targetDir string, verbose bool) {
	fmt.Printf("Verifying extracted files...\n")
	errs := verifyExtractedFiles(files, targetDir)
	if len(errs) > 0 {
		fmt.Printf("Verification warnings:\n")
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
	} else if verbose {
		fmt.Printf("All files verified successfully\n")
	}
}

// showVerboseInfo displays verbose info before unpacking
func showVerboseInfo(archiveFile string, opts *UnpackOpts) {
	if !opts.Verbose {
		return
	}
	fmt.Printf("Archive: %s\n", archiveFile)
	fmt.Printf("Target directory: %s\n", opts.Target)
	fmt.Printf("Overwrite mode: %v\n", opts.Overwrite)
	fmt.Printf("Backup mode: %v\n", opts.Backup)
	fmt.Println()
}

// handleConflictsPrompt handles conflict resolution with user
func handleConflictsPrompt(conflicts []string, dryRun bool) (proceed, overwrite bool) {
	if len(conflicts) == 0 {
		return true, false
	}
	showConflicts(conflicts, 10)

	if dryRun {
		return true, false
	}

	fmt.Printf("\nUse --overwrite to replace existing files, or --backup to create backups.\n")
	if !confirm("Continue anyway?") {
		fmt.Println("Operation cancelled.")
		return false, false
	}
	return true, true
}

// showDryRunResult displays dry run summary
func showDryRunResult(fileCount int, target string, conflicts []string) {
	fmt.Printf("\nDry run completed. %d files would be extracted to %s\n", fileCount, target)
	if len(conflicts) > 0 {
		fmt.Printf("%d existing files would be affected\n", len(conflicts))
	}
}

// runUnpackCommand executes the unpack command
func runUnpackCommand(cmd *cobra.Command, args []string) error {
	app, err := initApp()
	if err != nil {
		return err
	}

	opts, err := parseUnpackOpts(cmd)
	if err != nil {
		return err
	}

	archiveFile, err := pickArchive(app, opts.Archive)
	if err != nil {
		return err
	}
	if opts.Archive == "" {
		fmt.Printf("Using most recent archive: %s\n", filepath.Base(archiveFile))
	}

	if _, statErr := os.Stat(archiveFile); os.IsNotExist(statErr) {
		return fmt.Errorf("archive file not found: %s", archiveFile)
	}

	key, cleanup, err := getPass(opts.PassEnv)
	if err != nil {
		return err
	}
	defer cleanup()

	showVerboseInfo(archiveFile, opts)

	fmt.Printf("Reading archive: %s\n", filepath.Base(archiveFile))
	archive, err := app.Archiver.List(archiveFile, key)
	if err != nil {
		return fmt.Errorf("failed to read archive (check password): %w", err)
	}

	filesToExtract := filterArchiveFiles(archive.Files, opts.Include, opts.Exclude)
	showArchive(archive, filesToExtract, opts.Verbose)

	conflicts := checkFileConflicts(filesToExtract, opts.Target)
	if !opts.Overwrite {
		proceed, overwrite := handleConflictsPrompt(conflicts, opts.DryRun)
		if !proceed {
			return nil
		}
		opts.Overwrite = overwrite
	}

	if opts.DryRun {
		showDryRunResult(len(filesToExtract), opts.Target, conflicts)
		return nil
	}

	if opts.Verbose {
		fmt.Printf("\nExtracting files to %s...\n", opts.Target)
	}

	duration, err := doUnpack(app, types.UnpackOptions{
		ArchivePath: archiveFile,
		Password:    key,
		TargetDir:   opts.Target,
		Overwrite:   opts.Overwrite,
		Backup:      opts.Backup,
	})
	if err != nil {
		return fmt.Errorf("error unpacking files: %w", err)
	}

	if opts.Verify {
		verifyFiles(filesToExtract, opts.Target, opts.Verbose)
	}

	showUnpackResult(filesToExtract, conflicts, duration, opts)

	return nil
}

// filterFiles filters files based on include/exclude patterns
func filterFiles(files []types.EnvFile, includePatterns, excludePatterns []string) []types.EnvFile {
	var filtered []types.EnvFile

	for _, file := range files {
		if len(includePatterns) > 0 {
			included := false
			for _, pattern := range includePatterns {
				matched, matchErr := filepath.Match(pattern, file.RelativePath)
				if matchErr == nil && matched {
					included = true
					break
				}
			}
			if !included {
				continue
			}
		}

		excluded := false
		for _, pattern := range excludePatterns {
			matched, matchErr := filepath.Match(pattern, file.RelativePath)
			if matchErr == nil && matched {
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

		info, err := os.Stat(targetPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: file not found after extraction", file.RelativePath))
			continue
		}

		if info.Size() != file.Size {
			errors = append(errors, fmt.Sprintf("%s: size mismatch (expected %d, got %d)",
				file.RelativePath, file.Size, info.Size()))
			continue
		}

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
