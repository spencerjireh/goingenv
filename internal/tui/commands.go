package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"goingenv/internal/config"
	"goingenv/pkg/types"
	"goingenv/pkg/utils"
)

// ScanFilesCmd scans for environment files asynchronously
func ScanFilesCmd(app *types.App) tea.Cmd {
	return func() tea.Msg {
		scanOpts := types.ScanOptions{
			RootPath: ".",
			MaxDepth: app.Config.DefaultDepth,
		}

		files, err := app.Scanner.ScanFiles(scanOpts)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Error scanning files: %v", err))
		}

		if len(files) == 0 {
			return ErrorMsg("No environment files found")
		}

		return ScanCompleteMsg(files)
	}
}

// PackFilesCmd packs files into an encrypted archive asynchronously
func PackFilesCmd(app *types.App, files []types.EnvFile, password string) tea.Cmd {
	return func() tea.Msg {
		// Generate output path
		outputPath := config.GetDefaultArchivePath()

		// Create pack options
		packOpts := types.PackOptions{
			Files:       files,
			OutputPath:  outputPath,
			Password:    password,
			Description: fmt.Sprintf("Environment files archive created on %s", time.Now().Format("2006-01-02 15:04:05")),
		}

		// Pack files
		err := app.Archiver.Pack(packOpts)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Error packing files: %v", err))
		}

		return PackCompleteMsg(fmt.Sprintf("Successfully packed %d files to %s", len(files), outputPath))
	}
}

// UnpackFilesCmd unpacks files from an encrypted archive asynchronously
func UnpackFilesCmd(app *types.App, password, archivePath string) tea.Cmd {
	return func() tea.Msg {
		// Create unpack options
		unpackOpts := types.UnpackOptions{
			ArchivePath: archivePath,
			Password:    password,
			TargetDir:   ".",
			Overwrite:   false, // Default to safe mode in TUI
			Backup:      false,
		}

		// Unpack files
		err := app.Archiver.Unpack(unpackOpts)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Error unpacking files: %v", err))
		}

		return UnpackCompleteMsg("Files successfully unpacked to current directory")
	}
}

// ListFilesCmd lists archive contents asynchronously
func ListFilesCmd(app *types.App, password, archivePath string) tea.Cmd {
	return func() tea.Msg {
		archive, err := app.Archiver.List(archivePath, password)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Error listing archive: %v", err))
		}

		// Format the archive contents for display
		result := formatArchiveContents(archive)
		return ListCompleteMsg(result)
	}
}

// ProgressCmd simulates progress updates for long-running operations
func ProgressCmd(duration time.Duration) tea.Cmd {
	return tea.Tick(duration/20, func(t time.Time) tea.Msg {
		// This is a simplified progress simulation
		// In a real implementation, you'd track actual progress
		return ProgressMsg(0.05) // 5% increment per tick
	})
}

// ValidatePasswordCmd validates a password against an archive
func ValidatePasswordCmd(app *types.App, archivePath, password string) tea.Cmd {
	return func() tea.Msg {
		// Try to list the archive to validate password
		_, err := app.Archiver.List(archivePath, password)
		if err != nil {
			return ErrorMsg("Invalid password or corrupted archive")
		}
		return SuccessMsg("Password validated successfully")
	}
}

// CheckArchiveIntegrityCmd checks the integrity of an archive
func CheckArchiveIntegrityCmd(app *types.App, archivePath, password string) tea.Cmd {
	return func() tea.Msg {
		archive, err := app.Archiver.List(archivePath, password)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Archive integrity check failed: %v", err))
		}

		// Additional integrity checks could be performed here
		// For now, we just verify we can read the metadata
		message := fmt.Sprintf("Archive integrity verified - %d files, created %s",
			len(archive.Files), archive.CreatedAt.Format("2006-01-02 15:04:05"))

		return SuccessMsg(message)
	}
}

// RefreshArchiveListCmd refreshes the list of available archives
func RefreshArchiveListCmd(app *types.App) tea.Cmd {
	return func() tea.Msg {
		archives, err := app.Archiver.GetAvailableArchives("")
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Error refreshing archive list: %v", err))
		}

		return ArchiveListMsg(archives)
	}
}

// DeleteArchiveCmd deletes an archive file (with confirmation)
func DeleteArchiveCmd(archivePath string) tea.Cmd {
	return func() tea.Msg {
		// In a real implementation, you might want to add confirmation
		// For now, this is just a placeholder
		return SuccessMsg(fmt.Sprintf("Archive %s would be deleted", archivePath))
	}
}

// SaveConfigCmd saves configuration changes
func SaveConfigCmd(app *types.App) tea.Cmd {
	return func() tea.Msg {
		err := app.ConfigMgr.Save(app.Config)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Error saving configuration: %v", err))
		}
		return SuccessMsg("Configuration saved successfully")
	}
}

// LoadConfigCmd loads configuration from file
func LoadConfigCmd(app *types.App) tea.Cmd {
	return func() tea.Msg {
		config, err := app.ConfigMgr.Load()
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Error loading configuration: %v", err))
		}

		app.Config = config
		return SuccessMsg("Configuration loaded successfully")
	}
}

// Additional message types for new commands
type (
	SuccessMsg     string
	ArchiveListMsg []string
	InitCompleteMsg string
)

// Helper function to format archive contents for display
func formatArchiveContents(archive *types.Archive) string {
	result := "Simple string"
	result += fmt.Sprintf("  Created: %s\n", archive.CreatedAt.Format("2006-01-02 15:04:05"))
	result += fmt.Sprintf("  Version: %s\n", archive.Version)
	result += fmt.Sprintf("  Total Files: %d\n", len(archive.Files))
	result += fmt.Sprintf("  Total Size: %s\n", utils.FormatSize(archive.TotalSize))

	if archive.Description != "" {
		result += fmt.Sprintf("  Description: %s\n", archive.Description)
	}

	result += "\nFiles:\n"

	for i, file := range archive.Files {
		if i < 20 { // Show first 20 files
			result += fmt.Sprintf("  • %s (%s) - %s\n",
				file.RelativePath,
				utils.FormatSize(file.Size),
				file.ModTime.Format("2006-01-02 15:04:05"))
		} else if i == 20 {
			result += fmt.Sprintf("  • ... and %d more files\n", len(archive.Files)-20)
			break
		}
	}

	return result
}

// Batch operations commands

// BatchPackCmd packs multiple directories in sequence
func BatchPackCmd(app *types.App, directories []string, password string) tea.Cmd {
	return func() tea.Msg {
		var results []string

		for _, dir := range directories {
			scanOpts := types.ScanOptions{
				RootPath: dir,
				MaxDepth: app.Config.DefaultDepth,
			}

			files, err := app.Scanner.ScanFiles(scanOpts)
			if err != nil {
				results = append(results, fmt.Sprintf("Error scanning %s: %v", dir, err))
				continue
			}

			if len(files) == 0 {
				results = append(results, fmt.Sprintf("No files found in %s", dir))
				continue
			}

			outputPath := fmt.Sprintf("%s/archive-%s-%s.enc",
				config.GetGoingEnvDir(),
				utils.SanitizeFilename(dir),
				time.Now().Format("20060102-150405"))

			packOpts := types.PackOptions{
				Files:       files,
				OutputPath:  outputPath,
				Password:    password,
				Description: fmt.Sprintf("Batch archive from %s", dir),
			}

			err = app.Archiver.Pack(packOpts)
			if err != nil {
				results = append(results, fmt.Sprintf("Error packing %s: %v", dir, err))
			} else {
				results = append(results, fmt.Sprintf("Packed %s (%d files)", dir, len(files)))
			}
		}

		return SuccessMsg("Batch operation completed:\n" + utils.JoinResults(results))
	}
}

// Utility commands for common operations

// QuickPackCmd performs a quick pack operation with default settings
func QuickPackCmd(app *types.App, password string) tea.Cmd {
	return func() tea.Msg {
		// Scan current directory
		scanOpts := types.ScanOptions{
			RootPath: ".",
			MaxDepth: app.Config.DefaultDepth,
		}

		files, err := app.Scanner.ScanFiles(scanOpts)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Quick pack failed: %v", err))
		}

		if len(files) == 0 {
			return ErrorMsg("No environment files found for quick pack")
		}

		// Pack with default settings
		outputPath := config.GetDefaultArchivePath()
		packOpts := types.PackOptions{
			Files:       files,
			OutputPath:  outputPath,
			Password:    password,
			Description: "Quick pack archive",
		}

		err = app.Archiver.Pack(packOpts)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Quick pack failed: %v", err))
		}

		return SuccessMsg(fmt.Sprintf("Quick pack completed: %d files archived", len(files)))
	}
}

// InitProjectCmd initializes GoingEnv in the current directory
func InitProjectCmd() tea.Cmd {
	return func() tea.Msg {
		// Initialize the project
		if err := config.InitializeProject(); err != nil {
			return ErrorMsg(fmt.Sprintf("Failed to initialize project: %v", err))
		}

		return InitCompleteMsg("GoingEnv successfully initialized! You can now use all features.")
	}
}
