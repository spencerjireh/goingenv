package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"goingenv/internal/config"
	"goingenv/internal/constants"
	"goingenv/pkg/password"
	"goingenv/pkg/types"
	"goingenv/pkg/utils"
)

// newListCommand creates the list command
func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List archive contents",
		Long: `Display the contents of an encrypted archive without extracting files.

The list command will:
- Decrypt the archive metadata using the provided password
- Display archive information (creation date, version, description)
- Show all files contained in the archive with their sizes and timestamps
- Optionally filter files by patterns or show detailed information

Examples:
  goingenv list -f backup.enc                           # Interactive password prompt
  goingenv list --password-env MY_PASSWORD --all        # List all archives with env password
  goingenv list -f archive.enc --pattern "*.env.prod*"  # Filter files by pattern`,
		RunE: runListCommand,
	}

	cmd.Flags().String("password-env", "", "Read password from environment variable")
	cmd.Flags().StringP("file", "f", "", "Archive file to list (required unless --all is used)")
	cmd.Flags().Bool("all", false, "List contents of all available archives")
	cmd.Flags().BoolP("verbose", "v", false, "Show detailed file information")
	cmd.Flags().Bool("sizes", false, "Show file sizes in detailed format")
	cmd.Flags().Bool("dates", false, "Show file modification dates")
	cmd.Flags().Bool("checksums", false, "Show file checksums")
	cmd.Flags().StringSliceP("pattern", "p", nil, "Filter files by patterns (glob-style)")
	cmd.Flags().StringP("sort", "s", "name", "Sort files by: name, size, date, type")
	cmd.Flags().Bool("reverse", false, "Reverse sort order")
	cmd.Flags().StringP("format", "", "table", "Output format: table, json, csv")
	cmd.Flags().IntP("limit", "l", 0, "Limit number of files to show (0 = no limit)")

	return cmd
}

// runListCommand executes the list command
func runListCommand(cmd *cobra.Command, args []string) error {
	out := NewOutput(appVersion)

	app, err := initApp()
	if err != nil {
		out.Header()
		out.Blank()
		out.Error(err.Error())
		return err
	}

	opts, err := parseListOpts(cmd)
	if err != nil {
		return err
	}

	passwordOpts := password.Options{PasswordEnv: opts.PassEnv}

	if opts.All {
		return listAllArchives(out, app, passwordOpts, opts.Verbose)
	}

	if opts.Archive == "" {
		out.Header()
		out.Blank()
		out.Error("Archive file is required")
		out.Hint("Use -f flag or --all to list all archives")
		return fmt.Errorf("archive file is required")
	}

	if _, statErr := os.Stat(opts.Archive); os.IsNotExist(statErr) {
		out.Header()
		out.Blank()
		out.Error(fmt.Sprintf("Archive not found: %s", opts.Archive))
		return fmt.Errorf("archive not found: %s", opts.Archive)
	}

	if validateErr := password.ValidatePasswordOptions(passwordOpts); validateErr != nil {
		return fmt.Errorf("invalid password options: %w", validateErr)
	}

	key, err := password.GetPassword(passwordOpts)
	if err != nil {
		out.Header()
		out.Blank()
		out.Error(fmt.Sprintf("Failed to get password: %v", err))
		return fmt.Errorf("failed to get password: %w", err)
	}
	defer password.ClearPassword(&key)

	out.Header()
	out.Blank()

	archive, err := app.Archiver.List(opts.Archive, key)
	if err != nil {
		out.Error("Failed to read archive (check password)")
		out.Hint("Check your password and try again")
		return fmt.Errorf("failed to read archive")
	}

	// Archive info
	out.Section(filepath.Base(opts.Archive))
	out.Indent(fmt.Sprintf("Created: %s", archive.CreatedAt.Format(constants.DateTimeFormat)))
	out.Indent(fmt.Sprintf("Version: %s", archive.Version))
	out.Blank()

	filesToShow := archive.Files
	if len(opts.Patterns) > 0 {
		filesToShow = filterFilesByPatterns(archive.Files, opts.Patterns)
		out.MutedPrint(fmt.Sprintf("  Showing %d files matching patterns (out of %d total)",
			len(filesToShow), len(archive.Files)))
		out.Blank()
	}

	sortFiles(filesToShow, opts.SortBy, opts.Reverse)

	if opts.Limit > 0 && len(filesToShow) > opts.Limit {
		filesToShow = filesToShow[:opts.Limit]
	}

	switch opts.Format {
	case "json":
		return displayFilesJSON(filesToShow)
	case "csv":
		displayFilesCSV(filesToShow)
		return nil
	default:
		displayFilesTable(out, filesToShow, opts)
	}

	// Summary
	out.Blank()
	var totalSize int64
	for _, f := range filesToShow {
		totalSize += f.Size
	}
	out.MutedPrint(fmt.Sprintf("  %d files, %s total", len(filesToShow), utils.FormatSize(totalSize)))

	return nil
}

// listAllArchives lists contents of all available archives
func listAllArchives(out *Output, app *types.App, passwordOpts password.Options, verbose bool) error { //nolint:unparam // error return kept for consistency
	archives, err := app.Archiver.GetAvailableArchives("")
	if err != nil {
		out.Header()
		out.Blank()
		out.Error(fmt.Sprintf("Failed to find archives: %v", err))
		return nil
	}

	out.Header()
	out.Blank()

	if len(archives) == 0 {
		out.Warning("No archives found")
		out.Hint(fmt.Sprintf("Archives should be in %s directory", config.GetGoingEnvDir()))
		return nil
	}

	out.Section(fmt.Sprintf("Archives (%d)", len(archives)))
	out.Blank()

	for i, archivePath := range archives {
		name := filepath.Base(archivePath)
		info, statErr := os.Stat(archivePath)
		if statErr != nil {
			continue
		}

		out.Printf("  [%d] %s\n", i+1, name)
		out.Indent(fmt.Sprintf("    Size: %s", utils.FormatSize(info.Size())))
		out.Indent(fmt.Sprintf("    Modified: %s", info.ModTime().Format(constants.DateTimeFormat)))

		if verbose && passwordOpts.PasswordEnv != "" {
			if key, keyErr := password.GetPassword(passwordOpts); keyErr == nil {
				archive, listErr := app.Archiver.List(archivePath, key)
				password.ClearPassword(&key)
				if listErr == nil {
					out.Indent(fmt.Sprintf("    Files: %d", len(archive.Files)))
					out.Indent(fmt.Sprintf("    Total size: %s", utils.FormatSize(archive.TotalSize)))
				} else {
					out.Indent("    Status: Cannot read (wrong password or corrupted)")
				}
			}
		}

		out.Blank()
	}

	if passwordOpts.PasswordEnv == "" && verbose {
		out.Hint("Provide a password with --password-env to see detailed archive information")
	}

	return nil
}

// displayFilesTable displays files in table format
func displayFilesTable(out *Output, files []types.EnvFile, opts *ListOpts) {
	if len(files) == 0 {
		out.MutedPrint("  No files to display")
		return
	}

	out.Section("Files")

	for _, file := range files {
		var parts []string
		parts = append(parts, file.RelativePath)

		if opts.Sizes || opts.Verbose {
			parts = append(parts, utils.FormatSize(file.Size))
		}
		if opts.Dates || opts.Verbose {
			parts = append(parts, file.ModTime.Format(constants.DateTimeFormat))
		}
		if opts.Checksums {
			cs := file.Checksum
			if len(cs) > 16 {
				cs = cs[:16] + "..."
			}
			parts = append(parts, cs)
		}

		out.Indent(strings.Join(parts, "    "))
	}
}

// displayFilesJSON displays files in JSON format
func displayFilesJSON(files []types.EnvFile) error {
	output := map[string]interface{}{
		"files": files,
		"count": len(files),
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

// displayFilesCSV displays files in CSV format
func displayFilesCSV(files []types.EnvFile) {
	fmt.Println("name,path,size,modified,checksum")
	for _, file := range files {
		fmt.Printf("%s,%s,%d,%s,%s\n",
			filepath.Base(file.RelativePath),
			file.RelativePath,
			file.Size,
			file.ModTime.Format(constants.DateTimeFormat),
			file.Checksum)
	}
}

// filterFilesByPatterns filters files based on glob patterns
func filterFilesByPatterns(files []types.EnvFile, patterns []string) []types.EnvFile {
	var filtered []types.EnvFile

	for _, file := range files {
		for _, pattern := range patterns {
			matched, matchErr := filepath.Match(pattern, file.RelativePath)
			if matchErr == nil && matched {
				filtered = append(filtered, file)
				break
			}
		}
	}

	return filtered
}

// sortFiles sorts files based on the specified criteria
func sortFiles(files []types.EnvFile, sortBy string, reverse bool) {
	switch sortBy {
	case "size":
		sort.Slice(files, func(i, j int) bool {
			if reverse {
				return files[i].Size > files[j].Size
			}
			return files[i].Size < files[j].Size
		})
	case "date":
		sort.Slice(files, func(i, j int) bool {
			if reverse {
				return files[i].ModTime.After(files[j].ModTime)
			}
			return files[i].ModTime.Before(files[j].ModTime)
		})
	case "type":
		sort.Slice(files, func(i, j int) bool {
			ext1 := filepath.Ext(files[i].RelativePath)
			ext2 := filepath.Ext(files[j].RelativePath)
			if reverse {
				return ext1 > ext2
			}
			return ext1 < ext2
		})
	default: // name
		sort.Slice(files, func(i, j int) bool {
			if reverse {
				return files[i].RelativePath > files[j].RelativePath
			}
			return files[i].RelativePath < files[j].RelativePath
		})
	}
}
