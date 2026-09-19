package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"goingenv/internal/config"
	"goingenv/internal/diff"
	"goingenv/internal/manifest"
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
			return ErrorMsg{Tab: TabPack, Text: fmt.Sprintf("Failed to scan files: %v", err)}
		}

		if len(files) == 0 {
			return ErrorMsg{Tab: TabPack, Text: "No environment files found"}
		}

		return ScanCompleteMsg(files)
	}
}

// PackFilesCmd packs files into an encrypted archive asynchronously. env
// names the archive (<env>-<timestamp>.enc; "" for archive-<timestamp>.enc)
// and writeManifest adds the plaintext key listing beside it.
func PackFilesCmd(app *types.App, files []types.EnvFile, password, env string, writeManifest bool) tea.Cmd {
	return func() tea.Msg {
		outputPath := config.GetArchivePath(env)
		description := fmt.Sprintf("Environment files archive created on %s", time.Now().Format("2006-01-02 15:04:05"))

		packOpts := types.PackOptions{
			Files:       files,
			OutputPath:  outputPath,
			Password:    password,
			Description: description,
			Env:         env,
		}

		if err := app.Archiver.Pack(packOpts); err != nil {
			return ErrorMsg{Tab: TabPack, Text: fmt.Sprintf("Failed to pack files: %v", err)}
		}

		msg := fmt.Sprintf("Successfully packed %d files to %s", len(files), outputPath)
		if writeManifest {
			m, err := manifest.Build(outputPath, env, description, time.Now(), files)
			if err == nil {
				err = manifest.Write(manifest.PathFor(outputPath), m)
			}
			if err != nil {
				return ErrorMsg{Tab: TabPack, Text: fmt.Sprintf("Archive written, but the manifest failed: %v", err)}
			}
			msg += "\nManifest: " + manifest.PathFor(outputPath)
		}
		return PackCompleteMsg(msg)
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
			return ErrorMsg{Tab: TabUnpack, Text: fmt.Sprintf("Failed to unpack files: %v", err)}
		}

		return UnpackCompleteMsg(*result)
	}
}

// ListFilesCmd lists archive contents asynchronously
func ListFilesCmd(app *types.App, password, archivePath string) tea.Cmd {
	return func() tea.Msg {
		archive, err := app.Archiver.List(archivePath, password)
		if err != nil {
			return ErrorMsg{Tab: TabList, Text: fmt.Sprintf("Failed to read archive: %v", err)}
		}

		// Format the archive contents for display
		result := formatArchiveContents(archive)
		return ListCompleteMsg(result)
	}
}

// DiffFilesCmd compares pathA with pathB, or with the env files on disk when
// pathB is "". One password opens both archives. Values never reach the
// message: diff.Format masks them.
func DiffFilesCmd(app *types.App, password, pathA, pathB string) tea.Cmd {
	return func() tea.Msg {
		_, filesA, err := app.Archiver.ReadFiles(pathA, password)
		if err != nil {
			return ErrorMsg{Tab: TabDiff, Text: fmt.Sprintf("Failed to read %s: %v", filepath.Base(pathA), err)}
		}

		var filesB map[string][]byte
		if pathB == "" {
			filesB, err = readWorkingTree(app)
			if err != nil {
				return ErrorMsg{Tab: TabDiff, Text: fmt.Sprintf("Failed to read working tree: %v", err)}
			}
		} else {
			_, filesB, err = app.Archiver.ReadFiles(pathB, password)
			if err != nil {
				return ErrorMsg{Tab: TabDiff, Text: fmt.Sprintf("Failed to read %s: %v", filepath.Base(pathB), err)}
			}
		}

		result, err := diff.Compare(filesA, filesB)
		if err != nil {
			return ErrorMsg{Tab: TabDiff, Text: fmt.Sprintf("Failed to compare: %v", err)}
		}
		return DiffCompleteMsg(diff.Format(result))
	}
}

// readWorkingTree scans with the defaults ScanFilesCmd uses and reads every
// hit, keyed by relative path so entries line up with an archive's. An empty
// tree is a valid side of a diff, so zero files is not an error here.
func readWorkingTree(app *types.App) (map[string][]byte, error) {
	scanOpts := types.ScanOptions{RootPath: ".", MaxDepth: app.Config.DefaultDepth}
	found, err := app.Scanner.ScanFiles(&scanOpts)
	if err != nil {
		return nil, err
	}
	files := make(map[string][]byte, len(found))
	for i := range found {
		data, readErr := os.ReadFile(found[i].Path)
		if readErr != nil {
			return nil, readErr
		}
		files[found[i].RelativePath] = data
	}
	return files, nil
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
	var b strings.Builder
	b.WriteString("Archive contents\n\n")
	fmt.Fprintf(&b, "  Created: %s\n", archive.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "  Version: %s\n", archive.Version)
	if archive.Env != "" {
		fmt.Fprintf(&b, "  Environment: %s\n", archive.Env)
	}
	fmt.Fprintf(&b, "  Total Files: %d\n", len(archive.Files))
	fmt.Fprintf(&b, "  Total Size: %s\n", utils.FormatSize(archive.TotalSize))

	if archive.Description != "" {
		fmt.Fprintf(&b, "  Description: %s\n", archive.Description)
	}

	b.WriteString("\nFiles:\n")

	// Every file is listed; the tab renders this in a scrolling viewport.
	for _, file := range archive.Files {
		fmt.Fprintf(&b, "  [-] %s (%s) - %s\n",
			file.RelativePath,
			utils.FormatSize(file.Size),
			file.ModTime.Format("2006-01-02 15:04:05"))
	}

	return b.String()
}

// InitProjectCmd initializes goingenv in the current directory
func InitProjectCmd() tea.Cmd {
	return func() tea.Msg {
		// Initialize the project
		if err := config.InitializeProject(); err != nil {
			return ErrorMsg{Tab: TabPack, Text: fmt.Sprintf("Failed to initialize project: %v", err)}
		}

		return InitCompleteMsg("Initialized")
	}
}
