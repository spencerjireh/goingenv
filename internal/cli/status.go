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
	"goingenv/internal/constants"
	"goingenv/internal/scanner"
	"goingenv/pkg/types"
	"goingenv/pkg/utils"
)

// sizeStats holds file size distribution
type sizeStats struct {
	Small  int
	Medium int
	Large  int
}

// ageStats holds file age distribution
type ageStats struct {
	Recent int
	Old    int
}

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

	cmd.Flags().BoolP("verbose", "v", false, "Show detailed information")
	cmd.Flags().StringP("directory", "d", "", "Directory to analyze (default: current directory)")
	cmd.Flags().Bool("archives", true, "Show archive information")
	cmd.Flags().Bool("files", true, "Show detected files")
	cmd.Flags().Bool("config", false, "Show detailed configuration")
	cmd.Flags().Bool("stats", false, "Show statistics and analysis")
	cmd.Flags().Bool("recommendations", false, "Show recommendations and tips")

	return cmd
}

// analyzeSizes returns size distribution (pure function)
func analyzeSizes(files []types.EnvFile) sizeStats {
	var s sizeStats
	for _, file := range files {
		switch {
		case file.Size < constants.SmallFileThreshold:
			s.Small++
		case file.Size < constants.MediumFileThreshold:
			s.Medium++
		default:
			s.Large++
		}
	}
	return s
}

// analyzeAges returns age distribution (pure function)
func analyzeAges(files []types.EnvFile) ageStats {
	var a ageStats
	now := time.Now()
	for _, file := range files {
		if now.Sub(file.ModTime) < constants.RecentFileAge {
			a.Recent++
		} else {
			a.Old++
		}
	}
	return a
}

// showAnalysis displays file and archive analysis
func showAnalysis(files []types.EnvFile, archives []string, verbose bool) {
	if len(files) > 0 {
		fmt.Printf("File analysis:\n")

		sizes := analyzeSizes(files)
		fmt.Printf("  Size distribution: %d small (<1KB), %d medium (1-10KB), %d large (>10KB)\n",
			sizes.Small, sizes.Medium, sizes.Large)

		ages := analyzeAges(files)
		fmt.Printf("  Age distribution: %d recent (<30 days), %d older (>30 days)\n",
			ages.Recent, ages.Old)
	}

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

		if len(files) > 0 {
			var totalFileSize int64
			for _, file := range files {
				totalFileSize += file.Size
			}

			if totalFileSize > 0 {
				avgCompressionRatio := float64(totalArchiveSize) / float64(totalFileSize) * 100
				fmt.Printf("  Estimated compression: %.1f%% of original size\n", avgCompressionRatio)
			}
		}
	}

	if verbose {
		fmt.Printf("\nPerformance:\n")
		fmt.Printf("  Last scan took: <1s (estimated)\n")
		if len(archives) > 0 {
			fmt.Printf("  Encryption overhead: ~%d%% of file size\n", 10)
		}
	}
}

// applyDefaults sets default sections if none specified
func applyDefaults(opts *StatusOpts) {
	if opts.ShowArchives || opts.ShowFiles || opts.ShowConfig || opts.ShowStats || opts.ShowRecommend {
		return
	}
	opts.ShowArchives = true
	opts.ShowFiles = true
	if opts.Verbose {
		opts.ShowConfig = true
		opts.ShowStats = true
		opts.ShowRecommend = true
	}
}

// showSections displays the requested status sections
func showSections(app *types.App, opts *StatusOpts) {
	if opts.ShowArchives {
		if err := displayArchiveInfo(app, opts.Verbose); err != nil {
			fmt.Printf("Warning: Could not read archive information: %v\n", err)
		}
	}

	if opts.ShowFiles {
		if err := displayDetectedFiles(app, opts.Directory, opts.Verbose); err != nil {
			fmt.Printf("Warning: Could not scan files: %v\n", err)
		}
	}

	if opts.ShowConfig {
		displayConfigInfo(app, opts.Verbose)
	}

	if opts.ShowStats {
		if err := displayStatsAndAnalysis(app, opts.Directory, opts.Verbose); err != nil {
			fmt.Printf("Warning: Could not generate statistics: %v\n", err)
		}
	}

	if opts.ShowRecommend {
		displayRecommendations(app, opts.Directory)
	}
}

// runStatusCommand executes the status command
func runStatusCommand(cmd *cobra.Command, args []string) error {
	app, err := initApp()
	if err != nil {
		return err
	}

	opts, err := parseStatusOpts(cmd)
	if err != nil {
		return err
	}

	applyDefaults(opts)

	fmt.Printf("goingenv Status Report\n")
	fmt.Printf("Generated: %s\n", time.Now().Format(constants.DateTimeFormat))
	fmt.Println(strings.Repeat("=", 60))

	displaySystemInfo(opts.Directory, opts.Verbose)
	showSections(app, opts)

	return nil
}

// displaySystemInfo shows system and directory information
func displaySystemInfo(directory string, verbose bool) {
	fmt.Println("\nSystem Information")
	fmt.Println(strings.Repeat("-", 40))

	cwd, _ := os.Getwd() //nolint:errcheck // best effort
	fmt.Printf("Current directory: %s\n", cwd)

	if directory != "." {
		absDir, _ := filepath.Abs(directory) //nolint:errcheck // best effort
		fmt.Printf("Target directory: %s\n", absDir)
	}

	if verbose {
		fmt.Printf("Operating system: %s %s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("Go version: %s\n", runtime.Version())

		if stat, err := os.Stat(cwd); err == nil {
			fmt.Printf("Directory permissions: %v\n", stat.Mode())
		}
	}

	goingenvDir := config.GetGoingEnvDir()
	if _, err := os.Stat(goingenvDir); err == nil {
		fmt.Printf("goingenv directory: %s (exists)\n", goingenvDir)
	} else {
		fmt.Printf("goingenv directory: %s (not created)\n", goingenvDir)
	}
}

// displayArchiveInfo shows information about available archives
func displayArchiveInfo(app *types.App, verbose bool) error {
	fmt.Println("\nArchive Information")
	fmt.Println(strings.Repeat("-", 40))

	archives, err := app.Archiver.GetAvailableArchives("")
	if err != nil {
		return err
	}

	if len(archives) == 0 {
		fmt.Println("No archives found in .goingenv directory")
		fmt.Println("Tip: Use 'goingenv pack' to create your first archive")
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

		fmt.Printf("  - %s\n", filepath.Base(archivePath))
		if verbose {
			fmt.Printf("    Size: %s\n", utils.FormatSize(info.Size()))
			fmt.Printf("    Modified: %s\n", info.ModTime().Format(constants.DateTimeFormat))
		} else {
			fmt.Printf("    %s - %s\n", utils.FormatSize(info.Size()), info.ModTime().Format(constants.DateTimeFormat))
		}
	}

	fmt.Printf("\nArchive summary:\n")
	fmt.Printf("  Total size: %s\n", utils.FormatSize(totalSize))
	if len(archives) > 1 {
		fmt.Printf("  Date range: %s to %s\n",
			oldestDate.Format(constants.DateFormat),
			newestDate.Format(constants.DateFormat))
	}
	fmt.Printf("  Average size: %s\n", utils.FormatSize(totalSize/int64(len(archives))))

	return nil
}

// displayDetectedFiles shows environment files found in the directory
func displayDetectedFiles(app *types.App, directory string, verbose bool) error {
	fmt.Println("\nDetected Environment Files")
	fmt.Println(strings.Repeat("-", 40))

	scanOpts := types.ScanOptions{
		RootPath: directory,
		MaxDepth: app.Config.DefaultDepth,
	}

	files, err := app.Scanner.ScanFiles(&scanOpts)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		fmt.Println("No environment files detected")
		fmt.Println("Tip: Make sure you're in a directory with .env files")
		return nil
	}

	fmt.Printf("Found %d environment file(s):\n", len(files))

	filesByType := make(map[string][]types.EnvFile)
	var totalSize int64

	for _, file := range files {
		totalSize += file.Size
		name := filepath.Base(file.RelativePath)
		fileType := utils.CategorizeEnvFile(name)
		filesByType[fileType] = append(filesByType[fileType], file)
	}

	categories := []string{"Main", "Local", "Development", "Production", "Staging", "Test", "Other"}
	for _, category := range categories {
		if categoryFiles, exists := filesByType[category]; exists {
			fmt.Printf("\n  %s Environment Files:\n", category)
			for _, file := range categoryFiles {
				if verbose {
					fmt.Printf("    - %s (%s) - %s - %s\n",
						file.RelativePath,
						utils.FormatSize(file.Size),
						file.ModTime.Format(constants.DateTimeFormat),
						file.Checksum[:8]+"...")
				} else {
					fmt.Printf("    - %s (%s)\n", file.RelativePath, utils.FormatSize(file.Size))
				}
			}
		}
	}

	stats := scanner.GetFileStats(files)
	fmt.Printf("\nFile statistics:\n")
	fmt.Printf("  Total size: %s\n", utils.FormatSize(totalSize))
	fmt.Printf("  Average size: %s\n", utils.FormatSize(stats.AverageSize))

	if verbose && len(stats.FilesByPattern) > 0 {
		fmt.Printf("  Files by pattern:\n")
		for pattern, count := range stats.FilesByPattern {
			fmt.Printf("    - %s: %d\n", pattern, count)
		}
	}

	return nil
}

// displayConfigInfo shows configuration settings
func displayConfigInfo(app *types.App, verbose bool) {
	fmt.Println("\nConfiguration")
	fmt.Println(strings.Repeat("-", 40))

	cfg := app.Config

	fmt.Printf("Scan depth: %d directories\n", cfg.DefaultDepth)
	fmt.Printf("Max file size: %s\n", utils.FormatSize(cfg.MaxFileSize))

	fmt.Printf("\nFile patterns (%d):\n", len(cfg.EnvPatterns))
	for i, pattern := range cfg.EnvPatterns {
		if verbose || i < 5 {
			fmt.Printf("  - %s\n", pattern)
		} else if i == 5 {
			fmt.Printf("  - ... and %d more patterns\n", len(cfg.EnvPatterns)-5)
			break
		}
	}

	if verbose {
		fmt.Printf("\nExclude patterns (%d):\n", len(cfg.ExcludePatterns))
		for _, pattern := range cfg.ExcludePatterns {
			fmt.Printf("  - %s\n", pattern)
		}
	}
}

// displayStatsAndAnalysis shows statistics and analysis
func displayStatsAndAnalysis(app *types.App, directory string, verbose bool) error {
	fmt.Println("\nStatistics & Analysis")
	fmt.Println(strings.Repeat("-", 40))

	scanOpts := types.ScanOptions{
		RootPath: directory,
		MaxDepth: app.Config.DefaultDepth,
	}
	files, err := app.Scanner.ScanFiles(&scanOpts)
	if err != nil {
		return err
	}

	archives, err := app.Archiver.GetAvailableArchives("")
	if err != nil {
		return err
	}

	showAnalysis(files, archives, verbose)

	return nil
}

// displayRecommendations shows recommendations and tips
func displayRecommendations(app *types.App, directory string) {
	fmt.Println("\nRecommendations")
	fmt.Println(strings.Repeat("-", 40))

	scanOpts := types.ScanOptions{
		RootPath: directory,
		MaxDepth: app.Config.DefaultDepth,
	}
	files, _ := app.Scanner.ScanFiles(&scanOpts)         //nolint:errcheck // best effort for recommendations
	archives, _ := app.Archiver.GetAvailableArchives("") //nolint:errcheck // best effort for recommendations

	var recommendations []string

	if len(files) == 0 {
		recommendations = append(recommendations,
			"No environment files detected. Make sure you're in the right directory.")
	} else if len(files) > 10 {
		recommendations = append(recommendations,
			"Many environment files detected. Consider using exclude patterns for better performance.")
	}

	if len(archives) == 0 {
		recommendations = append(recommendations,
			"No archives found. Create your first backup with 'goingenv pack'.")
	} else if len(archives) > 20 {
		recommendations = append(recommendations,
			"Many archives found. Consider cleaning up old archives to save space.")
	}

	if len(files) > 0 {
		recommendations = append(recommendations,
			"Use strong, unique passwords for each archive.",
			"Verify archive contents regularly with 'goingenv list'.",
			"Share encrypted archives via git - they are safe to commit.")
	}

	if app.Config.DefaultDepth > 5 {
		recommendations = append(recommendations,
			"Consider reducing scan depth for better performance in large projects.")
	}

	if len(recommendations) == 0 {
		fmt.Println("Everything looks good! No specific recommendations at this time.")
	} else {
		for i, rec := range recommendations {
			fmt.Printf("%d. %s\n", i+1, rec)
		}
	}

	fmt.Printf("\nTips:\n")
	fmt.Println("  - Use 'goingenv pack --dry-run' to preview what will be archived")
	fmt.Println("  - Run 'goingenv status --verbose' for detailed information")
	fmt.Println("  - Check 'goingenv help' for all available commands")
}
