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

// tableOpts holds table display options
type tableOpts struct {
	Sizes     bool
	Dates     bool
	Checksums bool
	Verbose   bool
}

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

// maxWidth calculates max name width for table (pure function)
func maxWidth(files []types.EnvFile, minWidth, maxWidth int) int {
	width := minWidth
	for _, file := range files {
		if len(file.RelativePath) > width {
			width = len(file.RelativePath)
		}
	}
	if width > maxWidth {
		width = maxWidth
	}
	return width
}

// fmtRow formats a single file row (pure function)
func fmtRow(f *types.EnvFile, width int, o tableOpts) string {
	name := f.RelativePath
	if len(name) > width {
		name = name[:width-3] + "..."
	}

	if !o.Verbose && !o.Sizes && !o.Dates && !o.Checksums {
		return fmt.Sprintf("  - %s (%s)", name, utils.FormatSize(f.Size))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%-*s", width, name)
	if o.Sizes || o.Verbose {
		fmt.Fprintf(&b, " %10s", utils.FormatSize(f.Size))
	}
	if o.Dates || o.Verbose {
		fmt.Fprintf(&b, " %19s", f.ModTime.Format(constants.DateTimeFormat))
	}
	if o.Checksums || o.Verbose {
		fmt.Fprintf(&b, " %16s", f.Checksum[:16]+"...")
	}
	return b.String()
}

// printTableHeader prints table header
func printTableHeader(width int, o tableOpts) {
	if !o.Verbose && !o.Sizes && !o.Dates && !o.Checksums {
		return
	}

	fmt.Printf("%-*s", width, "Name")
	if o.Sizes || o.Verbose {
		fmt.Printf(" %10s", "Size")
	}
	if o.Dates || o.Verbose {
		fmt.Printf(" %19s", "Modified")
	}
	if o.Checksums || o.Verbose {
		fmt.Printf(" %16s", "Checksum")
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 80))
}

// printTable prints files in table format
func printTable(files []types.EnvFile, o tableOpts) {
	if len(files) == 0 {
		fmt.Println("No files to display.")
		return
	}

	fmt.Println("Files:")
	fmt.Println(strings.Repeat("-", 80))

	width := maxWidth(files, 20, 50)
	printTableHeader(width, o)

	for i := range files {
		fmt.Println(fmtRow(&files[i], width, o))
	}
	fmt.Println()
}

// runListCommand executes the list command
func runListCommand(cmd *cobra.Command, args []string) error {
	app, err := initApp()
	if err != nil {
		return err
	}

	opts, err := parseListOpts(cmd)
	if err != nil {
		return err
	}

	passwordOpts := password.Options{PasswordEnv: opts.PassEnv}

	if opts.All {
		return listAllArchives(app, passwordOpts, opts.Verbose)
	}

	if opts.Archive == "" {
		return fmt.Errorf("archive file is required. Use -f flag or --all to list all archives")
	}

	if _, statErr := os.Stat(opts.Archive); os.IsNotExist(statErr) {
		return fmt.Errorf("archive file not found: %s", opts.Archive)
	}

	if validateErr := password.ValidatePasswordOptions(passwordOpts); validateErr != nil {
		return fmt.Errorf("invalid password options: %w", validateErr)
	}

	key, err := password.GetPassword(passwordOpts)
	if err != nil {
		return fmt.Errorf("failed to get password: %w", err)
	}
	defer password.ClearPassword(&key)

	fmt.Printf("Reading archive: %s\n", filepath.Base(opts.Archive))
	archive, err := app.Archiver.List(opts.Archive, key)
	if err != nil {
		return fmt.Errorf("failed to read archive (check password): %w", err)
	}

	displayListArchiveInfo(archive, opts.Archive)

	filesToShow := archive.Files
	if len(opts.Patterns) > 0 {
		filesToShow = filterFilesByPatterns(archive.Files, opts.Patterns)
		fmt.Printf("Showing %d files matching patterns (out of %d total)\n",
			len(filesToShow), len(archive.Files))
	}

	sortFiles(filesToShow, opts.SortBy, opts.Reverse)

	if opts.Limit > 0 && len(filesToShow) > opts.Limit {
		filesToShow = filesToShow[:opts.Limit]
		fmt.Printf("Showing first %d files (use --limit 0 to show all)\n", opts.Limit)
	}

	switch opts.Format {
	case "json":
		return displayFilesJSON(filesToShow)
	case "csv":
		displayFilesCSV(filesToShow)
		return nil
	default:
		printTable(filesToShow, tableOpts{
			Sizes:     opts.Sizes,
			Dates:     opts.Dates,
			Checksums: opts.Checksums,
			Verbose:   opts.Verbose,
		})
	}

	displaySummary(archive, filesToShow)

	return nil
}

// listAllArchives lists contents of all available archives
func listAllArchives(app *types.App, passwordOpts password.Options, verbose bool) error {
	archives, err := app.Archiver.GetAvailableArchives("")
	if err != nil {
		return fmt.Errorf("failed to find archives: %w", err)
	}

	if len(archives) == 0 {
		fmt.Printf("No archives found in %s directory\n", config.GetGoingEnvDir())
		return nil
	}

	fmt.Printf("Found %d archive(s):\n\n", len(archives))

	for i, archivePath := range archives {
		fmt.Printf("[%d] %s\n", i+1, filepath.Base(archivePath))

		if info, err := os.Stat(archivePath); err == nil {
			fmt.Printf("    Size: %s\n", utils.FormatSize(info.Size()))
			fmt.Printf("    Modified: %s\n", info.ModTime().Format(constants.DateTimeFormat))
		}

		if verbose && passwordOpts.PasswordEnv != "" {
			if key, keyErr := password.GetPassword(passwordOpts); keyErr == nil {
				archive, listErr := app.Archiver.List(archivePath, key)
				password.ClearPassword(&key)
				if listErr == nil {
					fmt.Printf("    Created: %s\n", archive.CreatedAt.Format(constants.DateTimeFormat))
					fmt.Printf("    Files: %d\n", len(archive.Files))
					fmt.Printf("    Total size: %s\n", utils.FormatSize(archive.TotalSize))
					if archive.Description != "" {
						fmt.Printf("    Description: %s\n", archive.Description)
					}
				} else {
					fmt.Printf("    Status: Cannot read (wrong password or corrupted)\n")
				}
			} else {
				fmt.Printf("    Status: Cannot read (password error)\n")
			}
		}

		fmt.Println()
	}

	if passwordOpts.PasswordEnv == "" && verbose {
		fmt.Println("Tip: Provide a password with --password-env to see detailed archive information")
	}

	return nil
}

// displayListArchiveInfo displays general archive information
func displayListArchiveInfo(archive *types.Archive, archivePath string) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("Archive Information\n")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("File: %s\n", filepath.Base(archivePath))
	fmt.Printf("Created: %s\n", archive.CreatedAt.Format(constants.DateTimeFormat))
	fmt.Printf("Version: %s\n", archive.Version)
	if archive.Description != "" {
		fmt.Printf("Description: %s\n", archive.Description)
	}
	fmt.Printf("Total files: %d\n", len(archive.Files))
	fmt.Printf("Total size: %s\n", utils.FormatSize(archive.TotalSize))
	fmt.Println()
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

// displaySummary displays summary statistics
func displaySummary(archive *types.Archive, displayedFiles []types.EnvFile) {
	if len(displayedFiles) == 0 {
		return
	}

	fmt.Println("Summary:")
	fmt.Println(strings.Repeat("-", 40))

	typeStats := make(map[string]int)
	var totalDisplayedSize int64

	for _, file := range displayedFiles {
		totalDisplayedSize += file.Size
		name := filepath.Base(file.RelativePath)
		fileType := utils.CategorizeEnvFile(name)
		typeStats[fileType]++
	}

	fmt.Printf("Files by type:\n")
	for fileType, count := range typeStats {
		fmt.Printf("  - %s: %d\n", fileType, count)
	}

	fmt.Printf("\nSize information:\n")
	fmt.Printf("  - Displayed files: %s\n", utils.FormatSize(totalDisplayedSize))
	if len(displayedFiles) < len(archive.Files) {
		fmt.Printf("  - Total archive: %s\n", utils.FormatSize(archive.TotalSize))
	}

	if len(displayedFiles) > 0 {
		avgSize := totalDisplayedSize / int64(len(displayedFiles))
		fmt.Printf("  - Average file size: %s\n", utils.FormatSize(avgSize))
	}

	if len(displayedFiles) > 1 {
		oldest, newest := displayedFiles[0].ModTime, displayedFiles[0].ModTime
		for _, file := range displayedFiles {
			if file.ModTime.Before(oldest) {
				oldest = file.ModTime
			}
			if file.ModTime.After(newest) {
				newest = file.ModTime
			}
		}

		fmt.Printf("\nTime span:\n")
		fmt.Printf("  - Oldest file: %s\n", oldest.Format(constants.DateTimeFormat))
		fmt.Printf("  - Newest file: %s\n", newest.Format(constants.DateTimeFormat))
	}

	fmt.Println()
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
