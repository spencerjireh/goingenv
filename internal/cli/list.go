package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"goingenv/internal/config"
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
  goingenv list -f backup.enc -k "mypassword"
  goingenv list -f archive.enc -k "mypassword" --verbose
  goingenv list --all  # List all available archives
  goingenv list -f archive.enc -k "pass" --pattern "*.env.prod*"`,
		RunE: runListCommand,
	}

	// Add flags
	cmd.Flags().StringP("key", "k", "", "Decryption password (will prompt if not provided)")
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
	// Initialize application
	app, err := NewApp()
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	// Parse flags
	archiveFile, _ := cmd.Flags().GetString("file")
	key, _ := cmd.Flags().GetString("key")
	listAll, _ := cmd.Flags().GetBool("all")
	verbose, _ := cmd.Flags().GetBool("verbose")
	showSizes, _ := cmd.Flags().GetBool("sizes")
	showDates, _ := cmd.Flags().GetBool("dates")
	showChecksums, _ := cmd.Flags().GetBool("checksums")
	patterns, _ := cmd.Flags().GetStringSlice("pattern")
	sortBy, _ := cmd.Flags().GetString("sort")
	reverse, _ := cmd.Flags().GetBool("reverse")
	format, _ := cmd.Flags().GetString("format")
	limit, _ := cmd.Flags().GetInt("limit")

	// Handle --all flag
	if listAll {
		return listAllArchives(app, key, verbose)
	}

	// Require archive file if not listing all
	if archiveFile == "" {
		return fmt.Errorf("archive file is required. Use -f flag or --all to list all archives")
	}

	// Verify archive exists
	if _, err := os.Stat(archiveFile); os.IsNotExist(err) {
		return fmt.Errorf("archive file not found: %s", archiveFile)
	}

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

	// List archive contents
	fmt.Printf("Reading archive: %s\n", filepath.Base(archiveFile))
	archive, err := app.Archiver.List(archiveFile, key)
	if err != nil {
		return fmt.Errorf("failed to read archive (check password): %w", err)
	}

	// Display archive information
	displayListArchiveInfo(archive, archiveFile)

	// Filter files by patterns if specified
	filesToShow := archive.Files
	if len(patterns) > 0 {
		filesToShow = filterFilesByPatterns(archive.Files, patterns)
		fmt.Printf("Showing %d files matching patterns (out of %d total)\n", 
			len(filesToShow), len(archive.Files))
	}

	// Sort files
	sortFiles(filesToShow, sortBy, reverse)

	// Apply limit if specified
	if limit > 0 && len(filesToShow) > limit {
		filesToShow = filesToShow[:limit]
		fmt.Printf("Showing first %d files (use --limit 0 to show all)\n", limit)
	}

	// Display files based on format
	switch format {
	case "json":
		return displayFilesJSON(filesToShow)
	case "csv":
		return displayFilesCSV(filesToShow)
	default: // table format
		displayFilesTable(filesToShow, verbose, showSizes, showDates, showChecksums)
	}

	// Display summary
	displaySummary(archive, filesToShow)

	return nil
}

// listAllArchives lists contents of all available archives
func listAllArchives(app *types.App, key string, verbose bool) error {
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
		
		// Show basic info without requiring password
		if info, err := os.Stat(archivePath); err == nil {
			fmt.Printf("    Size: %s\n", utils.FormatSize(info.Size()))
			fmt.Printf("    Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
		}

		if verbose && key != "" {
			// Try to read archive contents
			if archive, err := app.Archiver.List(archivePath, key); err == nil {
				fmt.Printf("    Created: %s\n", archive.CreatedAt.Format("2006-01-02 15:04:05"))
				fmt.Printf("    Files: %d\n", len(archive.Files))
				fmt.Printf("    Total size: %s\n", utils.FormatSize(archive.TotalSize))
				if archive.Description != "" {
					fmt.Printf("    Description: %s\n", archive.Description)
				}
			} else {
				fmt.Printf("    Status: Cannot read (wrong password or corrupted)\n")
			}
		}
		
		fmt.Println()
	}

	if key == "" && verbose {
		fmt.Println("💡 Tip: Provide a password with -k to see detailed archive information")
	}

	return nil
}

// displayListArchiveInfo displays general archive information (renamed to avoid conflict)
func displayListArchiveInfo(archive *types.Archive, archivePath string) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("Archive Information\n")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("File: %s\n", filepath.Base(archivePath))
	fmt.Printf("Created: %s\n", archive.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Version: %s\n", archive.Version)
	if archive.Description != "" {
		fmt.Printf("Description: %s\n", archive.Description)
	}
	fmt.Printf("Total files: %d\n", len(archive.Files))
	fmt.Printf("Total size: %s\n", utils.FormatSize(archive.TotalSize))
	fmt.Println()
}

// displayFilesTable displays files in table format
func displayFilesTable(files []types.EnvFile, verbose, showSizes, showDates, showChecksums bool) {
	if len(files) == 0 {
		fmt.Println("No files to display.")
		return
	}

	fmt.Println("Files:")
	fmt.Println(strings.Repeat("-", 80))

	// Calculate column widths for better formatting
	maxNameLen := 20
	for _, file := range files {
		if len(file.RelativePath) > maxNameLen {
			maxNameLen = len(file.RelativePath)
		}
	}
	if maxNameLen > 50 {
		maxNameLen = 50
	}

	// Header
	if verbose || showSizes || showDates || showChecksums {
		fmt.Printf("%-*s", maxNameLen, "Name")
		if showSizes || verbose {
			fmt.Printf(" %10s", "Size")
		}
		if showDates || verbose {
			fmt.Printf(" %19s", "Modified")
		}
		if showChecksums || verbose {
			fmt.Printf(" %16s", "Checksum")
		}
		fmt.Println()
		fmt.Println(strings.Repeat("-", 80))
	}

	// Files
	for _, file := range files {
		name := file.RelativePath
		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		if verbose || showSizes || showDates || showChecksums {
			fmt.Printf("%-*s", maxNameLen, name)
			if showSizes || verbose {
				fmt.Printf(" %10s", utils.FormatSize(file.Size))
			}
			if showDates || verbose {
				fmt.Printf(" %19s", file.ModTime.Format("2006-01-02 15:04:05"))
			}
			if showChecksums || verbose {
				fmt.Printf(" %16s", file.Checksum[:16]+"...")
			}
			fmt.Println()
		} else {
			fmt.Printf("  • %s (%s)\n", name, utils.FormatSize(file.Size))
		}
	}
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
func displayFilesCSV(files []types.EnvFile) error {
	fmt.Println("name,path,size,modified,checksum")
	for _, file := range files {
		fmt.Printf("%s,%s,%d,%s,%s\n",
			filepath.Base(file.RelativePath),
			file.RelativePath,
			file.Size,
			file.ModTime.Format("2006-01-02 15:04:05"),
			file.Checksum)
	}
	return nil
}

// displaySummary displays summary statistics
func displaySummary(archive *types.Archive, displayedFiles []types.EnvFile) {
	if len(displayedFiles) == 0 {
		return
	}

	fmt.Println("Summary:")
	fmt.Println(strings.Repeat("-", 40))
	
	// File type statistics
	typeStats := make(map[string]int)
	var totalDisplayedSize int64
	
	for _, file := range displayedFiles {
		totalDisplayedSize += file.Size
		
		// Categorize by file extension/type
		name := filepath.Base(file.RelativePath)
		fileType := utils.CategorizeEnvFile(name)
		typeStats[fileType]++
	}

	// Display file type breakdown
	fmt.Printf("Files by type:\n")
	for fileType, count := range typeStats {
		fmt.Printf("  • %s: %d\n", fileType, count)
	}
	
	fmt.Printf("\nSize information:\n")
	fmt.Printf("  • Displayed files: %s\n", utils.FormatSize(totalDisplayedSize))
	if len(displayedFiles) < len(archive.Files) {
		fmt.Printf("  • Total archive: %s\n", utils.FormatSize(archive.TotalSize))
	}
	
	// Calculate average file size
	if len(displayedFiles) > 0 {
		avgSize := totalDisplayedSize / int64(len(displayedFiles))
		fmt.Printf("  • Average file size: %s\n", utils.FormatSize(avgSize))
	}
	
	// Time span information
	if len(displayedFiles) > 1 {
		var oldest, newest time.Time
		oldest = displayedFiles[0].ModTime
		newest = displayedFiles[0].ModTime
		
		for _, file := range displayedFiles {
			if file.ModTime.Before(oldest) {
				oldest = file.ModTime
			}
			if file.ModTime.After(newest) {
				newest = file.ModTime
			}
		}
		
		fmt.Printf("\nTime span:\n")
		fmt.Printf("  • Oldest file: %s\n", oldest.Format("2006-01-02 15:04:05"))
		fmt.Printf("  • Newest file: %s\n", newest.Format("2006-01-02 15:04:05"))
	}
	
	fmt.Println()
}

// Helper functions

// filterFilesByPatterns filters files based on glob patterns
func filterFilesByPatterns(files []types.EnvFile, patterns []string) []types.EnvFile {
	var filtered []types.EnvFile
	
	for _, file := range files {
		for _, pattern := range patterns {
			if matched, _ := filepath.Match(pattern, file.RelativePath); matched {
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