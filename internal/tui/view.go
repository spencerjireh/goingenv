package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"goingenv/internal/config"
	"goingenv/internal/scanner"
	"goingenv/pkg/types"
	"goingenv/pkg/utils"
)

// View implements tea.Model interface
func (m *Model) View() string {
	switch m.currentScreen {
	case ScreenMenu:
		return m.renderMenu()
	case ScreenPackPassword:
		return m.renderPackPassword()
	case ScreenUnpackPassword:
		return m.renderUnpackPassword()
	case ScreenListPassword:
		return m.renderListPassword()
	case ScreenUnpackSelect:
		return m.renderUnpackSelect()
	case ScreenListSelect:
		return m.renderListSelect()
	case ScreenPacking:
		return m.renderPacking()
	case ScreenUnpacking:
		return m.renderUnpacking()
	case ScreenListing:
		return m.renderListing()
	case ScreenStatus:
		return m.renderStatus()
	case ScreenSettings:
		return m.renderSettings()
	case ScreenHelp:
		return m.renderHelp()
	default:
		return "Unknown screen"
	}
}

// renderMenu renders the main menu screen
func (m *Model) renderMenu() string {
	title := "goingenv v1.0.0"
	if m.debugLogger.IsEnabled() {
		title += " [DEBUG MODE]"
	}
	view := TitleStyle.Render(title) + "\n"
	view += m.menu.View()

	if m.error != "" {
		view += "\n" + ErrorStyle.Render("Error: "+m.error)
	}

	if m.message != "" {
		view += "\n" + SuccessStyle.Render(m.message)
	}

	// Show debug info at bottom if verbose mode is enabled
	if m.debugLogger.IsEnabled() {
		view += "\n\n" + DimStyle.Render("DEBUG: Logging to "+m.debugLogger.GetLogPath())
		view += "\n" + DimStyle.Render("Current screen: "+string(m.currentScreen))
		view += "\n" + DimStyle.Render("Window size: "+fmt.Sprintf("%dx%d", m.width, m.height))
	}

	return view
}

// renderPackPassword renders the pack password entry screen
func (m *Model) renderPackPassword() string {
	view := TitleStyle.Render("Pack Environment Files") + "\n\n"

	if len(m.scannedFiles) > 0 {
		view += HeaderStyle.Render(fmt.Sprintf("Found %d environment files:", len(m.scannedFiles))) + "\n"
		for i, file := range m.scannedFiles {
			if i < 5 { // Show first 5 files
				view += fmt.Sprintf("  • %s (%s)\n", file.RelativePath, utils.FormatSize(file.Size))
			} else if i == 5 {
				view += fmt.Sprintf("  • ... and %d more files\n", len(m.scannedFiles)-5)
				break
			}
		}
		view += "\n"
	}

	view += HeaderStyle.Render("Enter encryption password:") + "\n"
	view += m.textInput.View() + "\n\n"
	view += "Press Enter to continue, Esc to go back\n"

	if m.error != "" {
		view += ErrorStyle.Render("Error: " + m.error)
	}

	return view
}

// renderUnpackPassword renders the unpack password entry screen
func (m *Model) renderUnpackPassword() string {
	view := TitleStyle.Render("Unpack Archive") + "\n\n"
	view += HeaderStyle.Render("Selected archive:") + "\n"
	view += fmt.Sprintf("  %s\n\n", filepath.Base(m.selectedArchive))
	view += HeaderStyle.Render("Enter decryption password:") + "\n"
	view += m.textInput.View() + "\n\n"
	view += "Press Enter to continue, Esc to go back\n"

	if m.error != "" {
		view += ErrorStyle.Render("Error: " + m.error)
	}

	return view
}

// renderListPassword renders the list password entry screen
func (m *Model) renderListPassword() string {
	view := TitleStyle.Render("List Archive Contents") + "\n\n"
	view += HeaderStyle.Render("Selected archive:") + "\n"
	view += fmt.Sprintf("  %s\n\n", filepath.Base(m.selectedArchive))
	view += HeaderStyle.Render("Enter decryption password:") + "\n"
	view += m.textInput.View() + "\n\n"
	view += "Press Enter to continue, Esc to go back\n"

	if m.error != "" {
		view += ErrorStyle.Render("Error: " + m.error)
	}

	return view
}

// renderUnpackSelect renders the unpack file selection screen
func (m *Model) renderUnpackSelect() string {
	view := TitleStyle.Render("Select Archive to Unpack") + "\n\n"
	view += m.filepicker.View() + "\n"
	view += "Select a .enc file, Esc to go back"
	return view
}

// renderListSelect renders the list file selection screen
func (m *Model) renderListSelect() string {
	view := TitleStyle.Render("Select Archive to List") + "\n\n"
	view += m.filepicker.View() + "\n"
	view += "Select a .enc file, Esc to go back"
	return view
}

// renderPacking renders the packing progress screen
func (m *Model) renderPacking() string {
	view := TitleStyle.Render("Packing Files...") + "\n\n"
	view += m.progress.View() + "\n\n"

	if m.message != "" {
		view += SuccessStyle.Render(m.message)
	} else {
		view += "Encrypting and archiving environment files..."
	}

	return view
}

// renderUnpacking renders the unpacking progress screen
func (m *Model) renderUnpacking() string {
	view := TitleStyle.Render("Unpacking Files...") + "\n\n"
	view += m.progress.View() + "\n\n"

	if m.message != "" {
		view += SuccessStyle.Render(m.message)
	} else {
		view += "Decrypting and extracting files..."
	}

	return view
}

// renderListing renders the archive listing screen
func (m *Model) renderListing() string {
	view := TitleStyle.Render("Archive Contents") + "\n\n"

	if m.message != "" {
		view += m.message + "\n\n"
	}

	view += "Press Esc to go back"
	return view
}

// renderStatus renders the status screen
func (m *Model) renderStatus() string {
	view := TitleStyle.Render("Status") + "\n\n"

	// Current directory
	cwd, _ := os.Getwd()
	view += HeaderStyle.Render("Current Directory:") + "\n"
	view += fmt.Sprintf("  %s\n\n", cwd)

	// Available archives
	archives, err := m.app.Archiver.GetAvailableArchives("")
	if err != nil {
		view += HeaderStyle.Render("Available Archives:") + "\n"
		view += ErrorStyle.Render("Error reading archives: "+err.Error()) + "\n\n"
	} else if len(archives) == 0 {
		view += HeaderStyle.Render("Available Archives:") + "\n"
		view += "  No archives found in .goingenv folder\n\n"
	} else {
		view += HeaderStyle.Render(fmt.Sprintf("Available Archives (%d):", len(archives))) + "\n"
		for _, archive := range archives {
			info, err := os.Stat(archive)
			if err == nil {
				view += fmt.Sprintf("  • %s (%s) - %s\n",
					filepath.Base(archive),
					utils.FormatSize(info.Size()),
					info.ModTime().Format("2006-01-02 15:04:05"))
			}
		}
		view += "\n"
	}

	// Detected environment files
	scanOpts := types.ScanOptions{
		RootPath: ".",
		MaxDepth: m.app.Config.DefaultDepth,
	}
	files, err := m.app.Scanner.ScanFiles(scanOpts)
	if err == nil && len(files) > 0 {
		view += HeaderStyle.Render(fmt.Sprintf("Detected Environment Files (%d):", len(files))) + "\n"
		for i, file := range files {
			if i < 10 { // Show first 10 files
				view += fmt.Sprintf("  • %s (%s)\n", file.RelativePath, utils.FormatSize(file.Size))
			} else if i == 10 {
				view += fmt.Sprintf("  • ... and %d more files\n", len(files)-10)
				break
			}
		}
		view += "\n"

		// Show file statistics
		stats := scanner.GetFileStats(files)
		view += HeaderStyle.Render("Statistics:") + "\n"
		view += fmt.Sprintf("  • Total Size: %s\n", utils.FormatSize(stats["total_size"].(int64)))
		view += fmt.Sprintf("  • Average Size: %s\n", utils.FormatSize(stats["average_size"].(int64)))
	}

	view += "Press Esc to go back"
	return view
}

// renderSettings renders the settings screen
func (m *Model) renderSettings() string {
	view := TitleStyle.Render("Settings") + "\n\n"

	view += HeaderStyle.Render("Default Scan Depth:") + "\n"
	view += fmt.Sprintf("  %d directories deep\n\n", m.app.Config.DefaultDepth)

	view += HeaderStyle.Render("Environment File Patterns:") + "\n"
	for _, pattern := range m.app.Config.EnvPatterns {
		view += fmt.Sprintf("  • %s\n", pattern)
	}
	view += "\n"

	view += HeaderStyle.Render("Exclude Patterns:") + "\n"
	for _, pattern := range m.app.Config.ExcludePatterns {
		view += fmt.Sprintf("  • %s\n", pattern)
	}
	view += "\n"

	view += HeaderStyle.Render("Configuration:") + "\n"
	view += fmt.Sprintf("  • Max File Size: %s\n", utils.FormatSize(m.app.Config.MaxFileSize))
	view += fmt.Sprintf("  • goingenv Directory: %s\n", config.GetGoingEnvDir())
	view += "\n"

	view += "Press Esc to go back"
	return view
}

// renderHelp renders the help screen
func (m *Model) renderHelp() string {
	view := TitleStyle.Render("Help & Documentation") + "\n\n"

	view += HeaderStyle.Render("Interactive Mode Navigation:") + "\n"
	view += "  • Use arrow keys or j/k to navigate menus\n"
	view += "  • Press Enter to select options\n"
	view += "  • Press Esc to go back to previous screen\n"
	view += "  • Press q or Ctrl+C to quit the application\n\n"

	view += HeaderStyle.Render("Command Line Usage:") + "\n"
	view += "  • goingenv pack -k \"password\" [-d /path] [-o name.enc]\n"
	view += "  • goingenv unpack -k \"password\" [-f archive.enc] [--overwrite]\n"
	view += "  • goingenv list -f archive.enc -k \"password\"\n"
	view += "  • goingenv status\n\n"

	view += HeaderStyle.Render("Environment File Patterns:") + "\n"
	view += "  • .env (main environment file)\n"
	view += "  • .env.local (local overrides)\n"
	view += "  • .env.development (development settings)\n"
	view += "  • .env.production (production settings)\n"
	view += "  • .env.staging (staging settings)\n"
	view += "  • .env.test (test settings)\n\n"

	view += HeaderStyle.Render("Security Features:") + "\n"
	view += "  • AES-256-GCM encryption\n"
	view += "  • PBKDF2 key derivation (100,000 iterations)\n"
	view += "  • SHA-256 file integrity checksums\n"
	view += "  • Secure random salt and nonce generation\n\n"

	view += HeaderStyle.Render("Tips:") + "\n"
	view += "  • Use strong, unique passwords for each archive\n"
	view += "  • Archives are stored in .goingenv/ directory\n"
	view += "  • .goingenv/ is automatically added to .gitignore\n"
	view += "  • Use 'status' command to see detected files\n\n"

	view += "Press Esc to go back"
	return view
}
