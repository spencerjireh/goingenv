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

		files, err := app.Scanner.ScanFiles(&scanOpts)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Failed to scan files: %v", err))
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
			return ErrorMsg(fmt.Sprintf("Failed to pack files: %v", err))
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
		result, err := app.Archiver.Unpack(unpackOpts)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Failed to unpack files: %v", err))
		}

		return UnpackCompleteMsg(*result)
	}
}

// ListFilesCmd lists archive contents asynchronously
func ListFilesCmd(app *types.App, password, archivePath string) tea.Cmd {
	return func() tea.Msg {
		archive, err := app.Archiver.List(archivePath, password)
		if err != nil {
			return ErrorMsg(fmt.Sprintf("Failed to read archive: %v", err))
		}

		// Format the archive contents for display
		result := formatArchiveContents(archive)
		return ListCompleteMsg(result)
	}
}

// Additional message types for new commands
type InitCompleteMsg string

// ShowModalMsg triggers a confirmation modal overlay.
type ShowModalMsg struct {
	Title     string
	Body      string
	OnConfirm tea.Cmd
}

// ToastMsg triggers a toast notification.
type ToastMsg struct {
	Message string
	IsError bool
}

// ToastExpiredMsg signals that a toast should be removed.
type ToastExpiredMsg struct {
	ID int
}

// SwitchTabMsg requests switching to a specific tab from within a tab.
type SwitchTabMsg struct {
	Tab TabID
}

// Helper function to format archive contents for display
func formatArchiveContents(archive *types.Archive) string {
	result := "Archive contents\n\n"
	result += fmt.Sprintf("  Created: %s\n", archive.CreatedAt.Format("2006-01-02 15:04:05"))
	result += fmt.Sprintf("  Version: %s\n", archive.Version)
	result += fmt.Sprintf("  Total Files: %d\n", len(archive.Files))
	result += fmt.Sprintf("  Total Size: %s\n", utils.FormatSize(archive.TotalSize))

	if archive.Description != "" {
		result += fmt.Sprintf("  Description: %s\n", archive.Description)
	}

	result += "\nFiles:\n"

	// Every file is listed; the tab renders this in a scrolling viewport.
	for _, file := range archive.Files {
		result += fmt.Sprintf("  [-] %s (%s) - %s\n",
			file.RelativePath,
			utils.FormatSize(file.Size),
			file.ModTime.Format("2006-01-02 15:04:05"))
	}

	return result
}

// InitProjectCmd initializes goingenv in the current directory
func InitProjectCmd() tea.Cmd {
	return func() tea.Msg {
		// Initialize the project
		if err := config.InitializeProject(); err != nil {
			return ErrorMsg(fmt.Sprintf("Failed to initialize project: %v", err))
		}

		return InitCompleteMsg("Initialized")
	}
}
