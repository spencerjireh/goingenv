package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"goingenv/internal/config"
	"goingenv/internal/scanner"
	"goingenv/pkg/types"
	"goingenv/pkg/utils"
)

// newStatusCommand creates the status command
func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current status and available archives",
		Long: `Display comprehensive status information about the current environment.

The status command shows:
- Current directory and system information
- Available archives in .goingenv directory
- Detected environment files in current directory
- Configuration settings and file patterns
- Statistics and recommendations

Examples:
  goingenv status
  goingenv status --verbose
  goingenv status --directory /path/to/project`,
		RunE: runStatusCommand,
	}

	// Add flags
	cmd.Flags().BoolP("verbose", "v", false, "Show detailed information")
	cmd.Flags().StringP("directory", "d", "", "Directory to analyze (default: current directory)")
	cmd.Flags().Bool("archives", true, "Show archive information")
	cmd.Flags().Bool("files", true, "Show detected files")
	cmd.Flags().Bool("config", false, "Show detailed configuration")
	cmd.Flags().Bool("stats", false, "Show statistics and analysis")
	cmd.Flags().Bool("recommendations", false, "Show recommendations and tips")

	return cmd
}

// runStatusCommand executes the status command
func runStatusCommand(cmd *cobra.Command, args []string) error {
	// Initialize application
	app, err := NewApp()
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	// Parse flags
	verbose, _ := cmd.Flags().GetBool("verbose")
	directory, _ := cmd.Flags().GetString("directory")
	if directory == "" {
		directory = "."
	}
	
	showArchives, _ := cmd.Flags().GetBool("archives")
	showFiles, _ := cmd.Flags().GetBool("files")
	showConfig, _ := cmd.Flags().GetBool("config")
	showStats, _ := cmd.Flags().GetBool("stats")
	showRecommendations, _ := cmd.Flags().GetBool("recommendations")

	// Show all sections if none specifically requested
	if !showArchives && !showFiles && !showConfig && !showStats && !showRecommendations {
		showArchives = true
		showFiles = true
		if verbose {
			showConfig = true
			showStats = true
			showRecommendations = true
		}
	}

	fmt.Printf("GoingEnv Status Report\n")
	fmt.Printf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println(strings.Repeat("=", 60))

	// System Information
	displaySystemInfo(directory, verbose)

	// Archive Information
	if showArchives {
		err := displayArchiveInfo(app, verbose)
		if err != nil {
			fmt.Printf("Warning: Could not read archive information: %v\n", err)
		}
	}

	// Detected Files
	if showFiles {
		err := displayDetectedFiles(app, directory, verbose)
		if err != nil {
			fmt.Printf("Warning: Could not scan files: %v\n", err)
		}
	}

	// Configuration
	if showConfig {
		displayConfigInfo(app, verbose)
	}

	// Statistics and Analysis
	if showStats {
		err := displayStatsAndAnalysis(app, directory, verbose)
		if err != nil {
			fmt.Printf("Warning: Could not generate statistics: %v\n", err)
		}
	}

	// Recommendations
	if showRecommendations {
		err := displayRecommendations(app, directory)
		if err != nil {
			fmt.Printf("Warning: Could not generate recommendations: %v\n", err)
		}
	}

	return nil
}

// displaySystemInfo shows system and directory information
func displaySystemInfo(directory string, verbose bool) {
	fmt.Println("\n📍 System Information")
	fmt.Println(strings.Repeat("-", 40))
	
	// Current directory
	cwd, _ := os.Getwd()
	fmt.Printf("Current directory: %s\n", cwd)
	
	if directory != "." {
		absDir, _ := filepath.Abs(directory)
		fmt.Printf("Target directory: %s\n", absDir)
	}
	
	// System info
	if verbose {
		fmt.Printf("Operating system: %s %s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("Go version: %s\n", runtime.Version())
		
		// Disk space (if available)
		if stat, err := os.Stat(cwd); err == nil {
			fmt.Printf("Directory permissions: %v\n", stat.Mode())
		}
	}
	
	// GoingEnv directory
	goingenvDir := config.GetGoingEnvDir()
	if _, err := os.Stat(goingenvDir); err == nil {
		fmt.Printf("GoingEnv directory: %s (exists)\n", goingenvDir)
	} else {
		fmt.Printf("GoingEnv directory: %s (not created)\n", goingenvDir)
	}
}

// displayArchiveInfo shows information about available archives
func displayArchiveInfo(app *types.App, verbose bool) error {
	fmt.Println("\n📦 Archive Information")
	fmt.Println(strings.Repeat("-", 40))
	
	archives, err := app.Archiver.GetAvailableArchives("")
	if err != nil {
		return err
	}
	
	if len(archives) == 0 {
		fmt.Println("No archives found in .goingenv directory")
		fmt.Println("💡 Use 'goingenv pack' to create your first archive")
		return nil
	}
	
	fmt.Printf("Found %d archive(s):\n", len(archives))
	
	var totalSize int64
	var oldestDate, newestDate time.Time
	
	for i, archivePath := range archives {
		info, err := os.Stat(archivePath)
		if err != nil {
			continue
		}
		
		totalSize += info.Size()
		
		if i == 0 {
			oldestDate = info.ModTime()
			newestDate = info.ModTime()
		} else {
			if info.ModTime().Before(oldestDate) {
				oldestDate = info.ModTime()
			}
			if info.ModTime().After(newestDate) {
				newestDate = info.ModTime()
			}
		}
		
		fmt.Printf("  • %s\n", filepath.Base(archivePath))
		if verbose {
			fmt.Printf("    Size: %s\n", utils.FormatSize(info.Size()))
			fmt.Printf("    Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("    %s - %s\n", utils.FormatSize(info.Size()), info.ModTime().Format("2006-01-02 15:04:05"))
		}
	}
	
	// Summary
	fmt.Printf("\nArchive summary:\n")
	fmt.Printf("  Total size: %s\n", utils.FormatSize(totalSize))
	if len(archives) > 1 {
		fmt.Printf("  Date range: %s to %s\n", 
			oldestDate.Format("2006-01-02"), 
			newestDate.Format("2006-01-02"))
	}
	fmt.Printf("  Average size: %s\n", utils.FormatSize(totalSize/int64(len(archives))))
	
	return nil
}

// displayDetectedFiles shows environment files found in the directory
func displayDetectedFiles(app *types.App, directory string, verbose bool) error {
	fmt.Println("\n🔍 Detected Environment Files")
	fmt.Println(strings.Repeat("-", 40))
	
	// Scan for files
	scanOpts := types.ScanOptions{
		RootPath: directory,
		MaxDepth: app.Config.DefaultDepth,
	}
	
	files, err := app.Scanner.ScanFiles(scanOpts)
	if err != nil {
		return err
	}
	
	if len(files) == 0 {
		fmt.Println("No environment files detected")
		fmt.Println("💡 Make sure you're in a directory with .env files")
		return nil
	}
	
	fmt.Printf("Found %d environment file(s):\n", len(files))
	
	// Group files by type
	filesByType := make(map[string][]types.EnvFile)
	var totalSize int64
	
	for _, file := range files {
		totalSize += file.Size
		
		name := filepath.Base(file.RelativePath)
		fileType := utils.CategorizeEnvFile(name)
		filesByType[fileType] = append(filesByType[fileType], file)
	}
	
	// Display by category
	categories := []string{"Main", "Local", "Development", "Production", "Staging", "Test", "Other"}
	for _, category := range categories {
		if categoryFiles, exists := filesByType[category]; exists {
			fmt.Printf("\n  %s Environment Files:\n", category)
			for _, file := range categoryFiles {
				if verbose {
					fmt.Printf("    • %s (%s) - %s - %s\n",
						file.RelativePath,
						utils.FormatSize(file.Size),
						file.ModTime.Format("2006-01-02 15:04:05"),
						file.Checksum[:8]+"...")
				} else {
					fmt.Printf("    • %s (%s)\n", file.RelativePath, utils.FormatSize(file.Size))
				}
			}
		}
	}
	
	// File statistics
	stats := scanner.GetFileStats(files)
	fmt.Printf("\nFile statistics:\n")
	fmt.Printf("  Total size: %s\n", utils.FormatSize(totalSize))
	fmt.Printf("  Average size: %s\n", utils.FormatSize(stats["average_size"].(int64)))
	
	if verbose {
		fmt.Printf("  Files by pattern:\n")
		for pattern, count := range stats["files_by_pattern"].(map[string]int) {
			fmt.Printf("    • %s: %d\n", pattern, count)
		}
	}
	
	return nil
}

// displayConfigInfo shows configuration settings
func displayConfigInfo(app *types.App, verbose bool) {
	fmt.Println("\n⚙️ Configuration")
	fmt.Println(strings.Repeat("-", 40))
	
	config := app.Config
	
	fmt.Printf("Scan depth: %d directories\n", config.DefaultDepth)
	fmt.Printf("Max file size: %s\n", utils.FormatSize(config.MaxFileSize))
	
	fmt.Printf("\nFile patterns (%d):\n", len(config.EnvPatterns))
	for i, pattern := range config.EnvPatterns {
		if verbose || i < 5 {
			fmt.Printf("  • %s\n", pattern)
		} else if i == 5 {
			fmt.Printf("  • ... and %d more patterns\n", len(config.EnvPatterns)-5)
			break
		}
	}
	
	if verbose {
		fmt.Printf("\nExclude patterns (%d):\n", len(config.ExcludePatterns))
		for _, pattern := range config.ExcludePatterns {
			fmt.Printf("  • %s\n", pattern)
		}
	}
}

// displayStatsAndAnalysis shows statistics and analysis
func displayStatsAndAnalysis(app *types.App, directory string, verbose bool) error {
	fmt.Println("\n📊 Statistics & Analysis")
	fmt.Println(strings.Repeat("-", 40))
	
	// Get files and archives
	scanOpts := types.ScanOptions{
		RootPath: directory,
		MaxDepth: app.Config.DefaultDepth,
	}
	files, err := app.Scanner.ScanFiles(scanOpts)
	if err != nil {
		return err
	}
	
	archives, err := app.Archiver.GetAvailableArchives("")
	if err != nil {
		return err
	}
	
	// File analysis
	if len(files) > 0 {
		fmt.Printf("File analysis:\n")
		
		// Size distribution
		var small, medium, large int
		for _, file := range files {
			if file.Size < 1024 {
				small++
			} else if file.Size < 10*1024 {
				medium++
			} else {
				large++
			}
		}
		
		fmt.Printf("  Size distribution: %d small (<1KB), %d medium (1-10KB), %d large (>10KB)\n", 
			small, medium, large)
		
		// Age analysis
		now := time.Now()
		var recent, old int
		for _, file := range files {
			age := now.Sub(file.ModTime)
			if age < 30*24*time.Hour { // 30 days
				recent++
			} else {
				old++
			}
		}
		
		fmt.Printf("  Age distribution: %d recent (<30 days), %d older (>30 days)\n", recent, old)
	}
	
	// Archive analysis
	if len(archives) > 0 {
		fmt.Printf("\nArchive analysis:\n")
		
		var totalArchiveSize int64
		for _, archivePath := range archives {
			if info, err := os.Stat(archivePath); err == nil {
				totalArchiveSize += info.Size()
			}
		}
		
		fmt.Printf("  Storage used: %s across %d archives\n", 
			utils.FormatSize(totalArchiveSize), len(archives))
		
		// Estimate compression ratio
		if len(files) > 0 {
			var totalFileSize int64
			for _, file := range files {
				totalFileSize += file.Size
			}
			
			if totalFileSize > 0 && len(archives) > 0 {
				avgCompressionRatio := float64(totalArchiveSize) / float64(totalFileSize) * 100
				fmt.Printf("  Estimated compression: %.1f%% of original size\n", avgCompressionRatio)
			}
		}
	}
	
	// Performance metrics
	if verbose {
		fmt.Printf("\nPerformance:\n")
		fmt.Printf("  Last scan took: <1s (estimated)\n")
		if len(archives) > 0 {
			fmt.Printf("  Encryption overhead: ~%d%% of file size\n", 10) // Rough estimate
		}
	}
	
	return nil
}

// displayRecommendations shows recommendations and tips
func displayRecommendations(app *types.App, directory string) error {
	fmt.Println("\n💡 Recommendations")
	fmt.Println(strings.Repeat("-", 40))
	
	// Get current state
	scanOpts := types.ScanOptions{
		RootPath: directory,
		MaxDepth: app.Config.DefaultDepth,
	}
	files, _ := app.Scanner.ScanFiles(scanOpts)
	archives, _ := app.Archiver.GetAvailableArchives("")
	
	recommendations := []string{}
	
	// File-based recommendations
	if len(files) == 0 {
		recommendations = append(recommendations, 
			"No environment files detected. Make sure you're in the right directory.")
	} else if len(files) > 10 {
		recommendations = append(recommendations, 
			"Many environment files detected. Consider using exclude patterns for better performance.")
	}
	
	// Archive-based recommendations
	if len(archives) == 0 {
		recommendations = append(recommendations, 
			"No archives found. Create your first backup with 'goingenv pack'.")
	} else if len(archives) > 20 {
		recommendations = append(recommendations, 
			"Many archives found. Consider cleaning up old archives to save space.")
	}
	
	// Security recommendations
	if len(files) > 0 {
		recommendations = append(recommendations, 
			"Ensure .goingenv/ is in your .gitignore to avoid committing encrypted archives.")
		recommendations = append(recommendations, 
			"Use strong, unique passwords for each archive.")
		recommendations = append(recommendations, 
			"Verify archive contents regularly with 'goingenv list'.")
	}
	
	// Performance recommendations
	if app.Config.DefaultDepth > 5 {
		recommendations = append(recommendations, 
			"Consider reducing scan depth for better performance in large projects.")
	}
	
	// Display recommendations
	if len(recommendations) == 0 {
		fmt.Println("✅ Everything looks good! No specific recommendations at this time.")
	} else {
		for i, rec := range recommendations {
			fmt.Printf("%d. %s\n", i+1, rec)
		}
	}
	
	// General tips
	fmt.Printf("\n📖 Tips:\n")
	fmt.Println("  • Use 'goingenv pack --dry-run' to preview what will be archived")
	fmt.Println("  • Run 'goingenv status --verbose' for detailed information")
	fmt.Println("  • Check 'goingenv help' for all available commands")
	
	return nil
}