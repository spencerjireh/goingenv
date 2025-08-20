package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"goingenv/pkg/types"
)

// Update implements tea.Model interface
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Log the message type for debugging
	m.debugLogger.LogModelUpdate("message_received", map[string]interface{}{
		"type":   fmt.Sprintf("%T", msg),
		"screen": m.currentScreen,
	})

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.UpdateSize(msg.Width, msg.Height)
		return m, nil

	case PackCompleteMsg:
		m.debugLogger.LogMessage("pack_complete", string(msg))
		m.SetMessage(string(msg))
		m.SetScreen(ScreenMenu)
		return m, nil

	case UnpackCompleteMsg:
		m.debugLogger.LogMessage("unpack_complete", string(msg))
		m.SetMessage(string(msg))
		m.SetScreen(ScreenMenu)
		return m, nil

	case ListCompleteMsg:
		m.debugLogger.LogMessage("list_complete", string(msg))
		m.SetMessage(string(msg))
		m.SetScreen(ScreenListing)
		return m, nil

	case ScanCompleteMsg:
		m.scannedFiles = []types.EnvFile(msg)
		m.debugLogger.LogOperation("scan_complete", fmt.Sprintf("found %d files", len(m.scannedFiles)))
		if len(m.scannedFiles) == 0 {
			m.SetError("No environment files found")
		} else {
			m.SetScreen(ScreenPackPassword)
			m.textInput.Focus()
		}
		return m, nil

	case ErrorMsg:
		m.SetError(string(msg))
		return m, nil

	case ProgressMsg:
		m.debugLogger.LogProgress("operation", float64(msg))
		m.progress.SetPercent(float64(msg))
		return m, nil

	case tea.KeyMsg:
		m.debugLogger.LogKeypress(msg.String(), m.currentScreen)
		return m.handleKeyPress(msg)
	}

	// Handle component updates based on current screen
	switch m.currentScreen {
	case ScreenPackPassword, ScreenUnpackPassword, ScreenListPassword:
		m.textInput, cmd = m.textInput.Update(msg)
	case ScreenUnpackSelect, ScreenListSelect:
		m.filepicker, cmd = m.filepicker.Update(msg)
		if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
			m.selectedArchive = path
			m.debugLogger.LogFileOperation("file_selected", path, 0)
			if m.currentScreen == ScreenUnpackSelect {
				m.SetScreen(ScreenUnpackPassword)
			} else {
				m.SetScreen(ScreenListPassword)
			}
			m.textInput.Focus()
		}
	}

	return m, cmd
}

// handleKeyPress handles keyboard input based on current screen
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.currentScreen {
	case ScreenMenu:
		return m.handleMenuKeys(msg)
	case ScreenPackPassword:
		return m.handlePackPasswordKeys(msg)
	case ScreenUnpackPassword:
		return m.handleUnpackPasswordKeys(msg)
	case ScreenListPassword:
		return m.handleListPasswordKeys(msg)
	case ScreenUnpackSelect, ScreenListSelect:
		return m.handleFileSelectKeys(msg)
	default:
		return m.handleGenericKeys(msg)
	}
}

// handleMenuKeys handles keyboard input on the main menu
func (m *Model) handleMenuKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.debugLogger.LogOperation("quit", "user requested quit")
		return m, tea.Quit
	case "?":
		// Direct help access with ? key
		m.debugLogger.LogOperation("help_view", "showing help screen via ? key")
		m.SetScreen(ScreenHelp)
		return m, nil
	case "enter":
		selectedItem := m.GetSelectedMenuItem()
		m.debugLogger.LogOperation("menu_selection", fmt.Sprintf("action: %s, title: %s", selectedItem.action, selectedItem.title))
		switch selectedItem.action {
		case "pack":
			// Start scanning for files
			m.debugLogger.LogOperation("pack_start", "initiating file scan")
			return m, ScanFilesCmd(m.app)
		case "unpack":
			m.debugLogger.LogOperation("unpack_start", "showing file picker")
			m.SetScreen(ScreenUnpackSelect)
			return m, m.filepicker.Init()
		case "list":
			m.debugLogger.LogOperation("list_start", "showing file picker")
			m.SetScreen(ScreenListSelect)
			return m, m.filepicker.Init()
		case "status":
			m.debugLogger.LogOperation("status_view", "showing status screen")
			m.SetScreen(ScreenStatus)
			return m, nil
		case "settings":
			m.debugLogger.LogOperation("settings_view", "showing settings screen")
			m.SetScreen(ScreenSettings)
			return m, nil
		case "help":
			m.debugLogger.LogOperation("help_view", "showing help screen")
			m.SetScreen(ScreenHelp)
			return m, nil
		}
	default:
		// Pass all other keys (including navigation keys) to the menu component
		var cmd tea.Cmd
		m.menu, cmd = m.menu.Update(msg)
		return m, cmd
	}
	return m, nil
}

// handlePackPasswordKeys handles keyboard input during pack password entry
func (m *Model) handlePackPasswordKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.debugLogger.LogOperation("pack_cancel", "user cancelled password entry")
		m.SetScreen(ScreenMenu)
		return m, nil
	case "enter":
		password := m.textInput.Value()
		if password == "" {
			m.debugLogger.LogError("pack_password", fmt.Errorf("empty password"))
			m.SetError("Password cannot be empty")
			return m, nil
		}
		m.debugLogger.LogOperation("pack_execute", fmt.Sprintf("starting pack operation with %d files", len(m.scannedFiles)))
		m.SetScreen(ScreenPacking)
		return m, PackFilesCmd(m.app, m.scannedFiles, password)
	}
	return m, nil
}

// handleUnpackPasswordKeys handles keyboard input during unpack password entry
func (m *Model) handleUnpackPasswordKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.debugLogger.LogOperation("unpack_cancel", "user cancelled password entry")
		m.SetScreen(ScreenMenu)
		return m, nil
	case "enter":
		password := m.textInput.Value()
		if password == "" {
			m.debugLogger.LogError("unpack_password", fmt.Errorf("empty password"))
			m.SetError("Password cannot be empty")
			return m, nil
		}
		m.debugLogger.LogOperation("unpack_execute", fmt.Sprintf("starting unpack operation for %s", m.selectedArchive))
		m.SetScreen(ScreenUnpacking)
		return m, UnpackFilesCmd(m.app, password, m.selectedArchive)
	}
	return m, nil
}

// handleListPasswordKeys handles keyboard input during list password entry
func (m *Model) handleListPasswordKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.debugLogger.LogOperation("list_cancel", "user cancelled password entry")
		m.SetScreen(ScreenMenu)
		return m, nil
	case "enter":
		password := m.textInput.Value()
		if password == "" {
			m.debugLogger.LogError("list_password", fmt.Errorf("empty password"))
			m.SetError("Password cannot be empty")
			return m, nil
		}
		m.debugLogger.LogOperation("list_execute", fmt.Sprintf("starting list operation for %s", m.selectedArchive))
		return m, ListFilesCmd(m.app, password, m.selectedArchive)
	}
	return m, nil
}

// handleFileSelectKeys handles keyboard input during file selection
func (m *Model) handleFileSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.SetScreen(ScreenMenu)
		return m, nil
	}
	return m, nil
}

// handleGenericKeys handles keyboard input for generic screens
func (m *Model) handleGenericKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.SetScreen(ScreenMenu)
		return m, nil
	}
	return m, nil
}