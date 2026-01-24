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
	view := RenderHeader(m.version)
	if m.debugLogger.IsEnabled() {
		view += " " + DimStyle.Render("[DEBUG]")
	}
	view += "\n\n"

	// Custom menu rendering with chevron selection
	items := m.menu.Items()
	for i := range items {
		if mi, ok := items[i].(MenuItem); ok {
			view += RenderMenuItem(mi.Title(), i == m.menu.Index()) + "\n"
		}
	}

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

	view += "\n\n" + RenderFooter("[up/down] navigate", "[enter] select", "[q] quit")

	return view
}

// renderPackPassword renders the pack password entry screen
func (m *Model) renderPackPassword() string {
	view := RenderHeader(m.version) + "\n\n"

	if len(m.scannedFiles) > 0 {
		view += RenderSectionHeader(fmt.Sprintf("Packing %d environment files", len(m.scannedFiles))) + "\n\n"
		for i, file := range m.scannedFiles {
			if i < 5 { // Show first 5 files
				view += fmt.Sprintf("  %s (%s)\n", file.RelativePath, utils.FormatSize(file.Size))
			} else if i == 5 {
				view += fmt.Sprintf("  ... and %d more files\n", len(m.scannedFiles)-5)
				break
			}
		}
		view += "\n"
	}

	view += "Password: " + m.textInput.View() + "\n"

	if m.error != "" {
		view += "\n" + ErrorStyle.Render("Error: "+m.error)
	}

	view += "\n\n" + RenderFooter("[enter] confirm", "[esc] cancel")

	return view
}

// renderUnpackPassword renders the unpack password entry screen
func (m *Model) renderUnpackPassword() string {
	view := RenderHeader(m.version) + "\n\n"

	view += RenderSectionHeader("Unpacking archive") + "\n\n"
	view += fmt.Sprintf("  %s\n\n", filepath.Base(m.selectedArchive))

	view += "Password: " + m.textInput.View() + "\n"

	if m.error != "" {
		view += "\n" + ErrorStyle.Render("Error: "+m.error)
	}

	view += "\n\n" + RenderFooter("[enter] confirm", "[esc] cancel")

	return view
}

// renderListPassword renders the list password entry screen
func (m *Model) renderListPassword() string {
	view := RenderHeader(m.version) + "\n\n"

	view += RenderSectionHeader("List archive contents") + "\n\n"
	view += fmt.Sprintf("  %s\n\n", filepath.Base(m.selectedArchive))

	view += "Password: " + m.textInput.View() + "\n"

	if m.error != "" {
		view += "\n" + ErrorStyle.Render("Error: "+m.error)
	}

	view += "\n\n" + RenderFooter("[enter] confirm", "[esc] cancel")

	return view
}

// renderUnpackSelect renders the unpack file selection screen
func (m *Model) renderUnpackSelect() string {
	view := RenderHeader(m.version) + "\n\n"

	view += RenderSectionHeader("Select archive to unpack") + "\n\n"
	view += m.filepicker.View() + "\n"

	view += "\n" + RenderFooter("[up/down] navigate", "[enter] select", "[esc] back")

	return view
}

// renderListSelect renders the list file selection screen
func (m *Model) renderListSelect() string {
	view := RenderHeader(m.version) + "\n\n"

	view += RenderSectionHeader("Select archive to list") + "\n\n"
	view += m.filepicker.View() + "\n"

	view += "\n" + RenderFooter("[up/down] navigate", "[enter] select", "[esc] back")

	return view
}

// renderPacking renders the packing progress screen
func (m *Model) renderPacking() string {
	view := RenderHeader(m.version) + "\n\n"

	view += RenderSectionHeader("Packing environment files...") + "\n\n"
	view += m.progress.View() + "\n\n"

	if m.message != "" {
		view += SuccessStyle.Render(m.message)
	} else {
		view += MutedStyle.Render("Encrypting and archiving...")
	}

	return view
}

// renderUnpacking renders the unpacking progress screen
func (m *Model) renderUnpacking() string {
	view := RenderHeader(m.version) + "\n\n"

	view += RenderSectionHeader("Unpacking files...") + "\n\n"
	view += m.progress.View() + "\n\n"

	if m.message != "" {
		view += SuccessStyle.Render(m.message)
	} else {
		view += MutedStyle.Render("Decrypting and extracting...")
	}

	return view
}

// renderListing renders the archive listing screen
func (m *Model) renderListing() string {
	view := RenderHeader(m.version) + "\n\n"

	view += RenderSectionHeader("Archive contents") + "\n\n"

	if m.message != "" {
		view += m.message + "\n"
	}

	view += "\n" + RenderFooter("[esc] back")

	return view
}

// renderStatus renders the status screen
func (m *Model) renderStatus() string {
	view := RenderHeader(m.version) + "\n\n"

	// Current directory
	cwd, _ := os.Getwd() //nolint:errcheck // best effort
	view += RenderSectionHeader("Directory") + "\n"
	view += fmt.Sprintf("  %s\n\n", cwd)

	// Detected environment files
	scanOpts := types.ScanOptions{
		RootPath: ".",
		MaxDepth: m.app.Config.DefaultDepth,
	}
	files, err := m.app.Scanner.ScanFiles(&scanOpts)
	if err == nil && len(files) > 0 {
		view += RenderSectionHeader(fmt.Sprintf("Environment Files (%d)", len(files))) + "\n"
		for i, file := range files {
			if i < 10 { // Show first 10 files
				view += fmt.Sprintf("  %s\n", file.RelativePath)
			} else if i == 10 {
				view += fmt.Sprintf("  ... and %d more files\n", len(files)-10)
				break
			}
		}
		view += "\n"

		// Show file statistics
		stats := scanner.GetFileStats(files)
		view += RenderSectionHeader("Statistics") + "\n"
		view += fmt.Sprintf("  Total Size: %s\n", utils.FormatSize(stats.TotalSize))
		view += "\n"
	} else if err == nil {
		view += RenderSectionHeader("Environment Files") + "\n"
		view += MutedStyle.Render("  No environment files detected") + "\n\n"
	}

	// Available archives
	archives, err := m.app.Archiver.GetAvailableArchives("")
	switch {
	case err != nil:
		view += RenderSectionHeader("Archives") + "\n"
		view += ErrorStyle.Render("  Error reading archives: "+err.Error()) + "\n"
	case len(archives) == 0:
		view += RenderSectionHeader("Archives") + "\n"
		view += MutedStyle.Render("  No archives found") + "\n"
	default:
		view += RenderSectionHeader(fmt.Sprintf("Archives (%d)", len(archives))) + "\n"
		for _, archive := range archives {
			info, statErr := os.Stat(archive)
			if statErr == nil {
				view += fmt.Sprintf("  %s    %s    %s\n",
					filepath.Base(archive),
					utils.FormatSize(info.Size()),
					utils.FormatTimeAgo(info.ModTime()))
			}
		}
	}

	view += "\n" + RenderFooter("[p] pack", "[u] unpack", "[esc] back", "[q] quit")

	return view
}

// renderSettings renders the settings screen
func (m *Model) renderSettings() string {
	view := RenderHeader(m.version) + "\n\n"

	view += RenderSectionHeader("Settings") + "\n\n"

	view += fmt.Sprintf("  Scan Depth        %d\n", m.app.Config.DefaultDepth)
	view += fmt.Sprintf("  Max File Size     %s\n\n", utils.FormatSize(m.app.Config.MaxFileSize))

	view += RenderSectionHeader("Patterns") + "\n"
	view += "  Include:\n"
	for _, pattern := range m.app.Config.EnvPatterns {
		view += fmt.Sprintf("    %s\n", pattern)
	}
	view += "  Exclude:\n"
	for _, pattern := range m.app.Config.ExcludePatterns {
		view += fmt.Sprintf("    %s\n", pattern)
	}
	view += "\n"

	view += RenderSectionHeader("Config Location") + "\n"
	view += fmt.Sprintf("  %s\n", config.GetGoingEnvDir())

	view += "\n" + RenderFooter("[esc] back")

	return view
}

// renderHelp renders the help screen
func (m *Model) renderHelp() string {
	view := RenderHeader(m.version) + "\n\n"

	view += RenderSectionHeader("Keyboard Shortcuts") + "\n\n"

	view += "  Navigation\n"
	view += MutedStyle.Render("    up/k     Move up") + "\n"
	view += MutedStyle.Render("    down/j   Move down") + "\n"
	view += MutedStyle.Render("    enter    Select") + "\n"
	view += MutedStyle.Render("    esc      Back/Cancel") + "\n"
	view += MutedStyle.Render("    q        Quit") + "\n\n"

	view += "  Quick Actions\n"
	view += MutedStyle.Render("    p        Pack files") + "\n"
	view += MutedStyle.Render("    u        Unpack archive") + "\n"
	view += MutedStyle.Render("    s        View status") + "\n\n"

	view += RenderSectionHeader("CLI Usage") + "\n\n"
	view += MutedStyle.Render("  goingenv pack -k \"password\"") + "\n"
	view += MutedStyle.Render("  goingenv unpack -k \"password\"") + "\n"
	view += MutedStyle.Render("  goingenv list -f archive.enc") + "\n"
	view += MutedStyle.Render("  goingenv status") + "\n\n"

	view += RenderSectionHeader("Security") + "\n\n"
	view += MutedStyle.Render("  AES-256-GCM encryption") + "\n"
	view += MutedStyle.Render("  PBKDF2-SHA256 (100k iterations)") + "\n"
	view += MutedStyle.Render("  SHA-256 file integrity") + "\n"

	view += "\n" + RenderFooter("[esc] back")

	return view
}
