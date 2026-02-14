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

// runUnpackCommand executes the unpack command
func runUnpackCommand(cmd *cobra.Command, args []string) error {
	out := NewOutput(appVersion)

	app, err := initApp()
	if err != nil {
		out.Header()
		out.Blank()
		out.Error(err.Error())
		return err
	}

	opts, err := parseUnpackOpts(cmd)
	if err != nil {
		return err
	}

	archiveFile, err := selectArchive(out, app, opts)
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

	archive, err := decryptArchive(out, app, archiveFile, key)
	if err != nil {
		return fmt.Errorf("decryption failed")
	}

	filesToExtract := filterArchiveFiles(archive.Files, opts.Include, opts.Exclude)
	displayUnpackFiles(out, filesToExtract, opts.Verbose)

	if opts.DryRun {
		conflicts := checkFileConflicts(filesToExtract, opts.Target)
		out.Success(fmt.Sprintf("Dry run: would extract %d files to %s", len(filesToExtract), opts.Target))
		if len(conflicts) > 0 {
			out.Indent(fmt.Sprintf("%d existing files would be affected", len(conflicts)))
		}
		return nil
	}

	if !handleConflicts(out, filesToExtract, opts) {
		return fmt.Errorf("file conflicts detected, use --overwrite to proceed")
	}

	return executeUnpack(out, app, archiveFile, filesToExtract, opts, key)
}

// selectArchive selects the archive file to unpack
func selectArchive(out *Output, app *types.App, opts *UnpackOpts) (string, error) {
	archiveFile, err := pickArchive(app, opts.Archive)
	if err != nil {
		out.Error(err.Error())
		return "", err
	}

	if opts.Archive == "" {
		out.Action(fmt.Sprintf("Using most recent archive: %s", filepath.Base(archiveFile)))
	}

	if _, statErr := os.Stat(archiveFile); os.IsNotExist(statErr) {
		out.Error(fmt.Sprintf("Archive not found: %s", archiveFile))
		return "", statErr
	}

	return archiveFile, nil
}

// decryptArchive decrypts and reads the archive
func decryptArchive(out *Output, app *types.App, archiveFile, key string) (*types.Archive, error) {
	out.Action(fmt.Sprintf("Unpacking %s...", filepath.Base(archiveFile)))
	out.Blank()

	archive, err := app.Archiver.List(archiveFile, key)
	if err != nil {
		out.Error("Failed to decrypt archive (check password)")
		out.Hint("Check your password and try again")
		return nil, err
	}
	return archive, nil
}

// displayUnpackFiles shows the files to be extracted
func displayUnpackFiles(out *Output, files []types.EnvFile, verbose bool) {
	for i, file := range files {
		switch {
		case verbose:
			out.ListItem(fmt.Sprintf("%s (%s)", file.RelativePath, utils.FormatSize(file.Size)))
		case i < 5:
			out.ListItem(file.RelativePath)
		case i == 5:
			out.ListItem(fmt.Sprintf("... and %d more files", len(files)-5))
			return
		}
	}
	out.Blank()
}

// handleConflicts checks for and handles file conflicts
func handleConflicts(out *Output, files []types.EnvFile, opts *UnpackOpts) bool {
	conflicts := checkFileConflicts(files, opts.Target)
	if len(conflicts) == 0 || opts.Overwrite {
		return true
	}

	out.WarningList(fmt.Sprintf("%d files already exist:", len(conflicts)), conflicts, 5)
	out.Blank()
	out.Hint("Use --overwrite to overwrite")
	return false
}

// executeUnpack performs the actual unpacking operation
func executeUnpack(out *Output, app *types.App, archiveFile string, files []types.EnvFile, opts *UnpackOpts, key string) error { //nolint:unparam // error return kept for consistency
	if opts.Verbose {
		out.Action("Extracting...")
	}

	// Capture conflicts before extraction so the count is accurate
	conflicts := checkFileConflicts(files, opts.Target)

	start := time.Now()
	err := app.Archiver.Unpack(types.UnpackOptions{
		ArchivePath: archiveFile,
		Password:    key,
		TargetDir:   opts.Target,
		Overwrite:   opts.Overwrite,
		Backup:      opts.Backup,
	})
	duration := time.Since(start)

	if err != nil {
		out.Error(fmt.Sprintf("Error unpacking files: %v", err))
		return err
	}

	if opts.Verify {
		verifyUnpackedFiles(out, files, opts.Target, opts.Verbose)
	}

	displayUnpackResult(out, files, conflicts, opts, duration)
	return nil
}

// verifyUnpackedFiles verifies extracted files
func verifyUnpackedFiles(out *Output, files []types.EnvFile, targetDir string, verbose bool) {
	errs := verifyExtractedFiles(files, targetDir)
	if len(errs) > 0 {
		out.Warning("Verification warnings:")
		for _, e := range errs {
			out.Indent(e)
		}
	} else if verbose {
		out.Success("All files verified")
	}
}

// displayUnpackResult shows the unpack result
func displayUnpackResult(out *Output, files []types.EnvFile, conflicts []string, opts *UnpackOpts, duration time.Duration) {
	out.Success(fmt.Sprintf("Extracted %d files", len(files)))

	if opts.Verbose {
		out.Indent(fmt.Sprintf("Time: %v", duration.Round(time.Millisecond)))
		if len(conflicts) > 0 {
			if opts.Backup {
				out.Indent(fmt.Sprintf("Backed up %d existing files", len(conflicts)))
			} else {
				out.Indent(fmt.Sprintf("Overwrote %d existing files", len(conflicts)))
			}
		}
	}
}

// filterArchiveFiles filters files based on include/exclude patterns
func filterArchiveFiles(files []types.EnvFile, include, exclude []string) []types.EnvFile {
	if len(include) == 0 && len(exclude) == 0 {
		return files
	}
	return filterFiles(files, include, exclude)
}

// filterFiles filters files based on include/exclude patterns
func filterFiles(files []types.EnvFile, includePatterns, excludePatterns []string) []types.EnvFile {
	var filtered []types.EnvFile

	for _, file := range files {
		if len(includePatterns) > 0 && !matchesAnyPattern(file.RelativePath, includePatterns) {
			continue
		}
		if matchesAnyPattern(file.RelativePath, excludePatterns) {
			continue
		}
		filtered = append(filtered, file)
	}

	return filtered
}

// matchesAnyPattern checks if a path matches any of the given patterns
func matchesAnyPattern(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
	}
	return false
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
