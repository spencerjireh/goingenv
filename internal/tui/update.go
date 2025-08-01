package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"goingenv/pkg/types"
)

// Update implements tea.Model interface
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.UpdateSize(msg.Width, msg.Height)
		return m, nil

	case PackCompleteMsg:
		m.SetMessage(string(msg))
		m.SetScreen(ScreenMenu)
		return m, nil

	case UnpackCompleteMsg:
		m.SetMessage(string(msg))
		m.SetScreen(ScreenMenu)
		return m, nil

	case ListCompleteMsg:
		m.SetMessage(string(msg))
		m.SetScreen(ScreenListing)
		return m, nil

	case ScanCompleteMsg:
		m.scannedFiles = []types.EnvFile(msg)
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
		m.progress.SetPercent(float64(msg))
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	// Handle component updates based on current screen
	switch m.currentScreen {
	case ScreenMenu:
		m.menu, cmd = m.menu.Update(msg)
	case ScreenPackPassword, ScreenUnpackPassword, ScreenListPassword:
		m.textInput, cmd = m.textInput.Update(msg)
	case ScreenUnpackSelect, ScreenListSelect:
		m.filepicker, cmd = m.filepicker.Update(msg)
		if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
			m.selectedArchive = path
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
		return m, tea.Quit
	case "enter":
		selectedItem := m.GetSelectedMenuItem()
		switch selectedItem.action {
		case "pack":
			// Start scanning for files
			return m, ScanFilesCmd(m.app)
		case "unpack":
			m.SetScreen(ScreenUnpackSelect)
			return m, m.filepicker.Init()
		case "list":
			m.SetScreen(ScreenListSelect)
			return m, m.filepicker.Init()
		case "status":
			m.SetScreen(ScreenStatus)
			return m, nil
		case "settings":
			m.SetScreen(ScreenSettings)
			return m, nil
		case "help":
			m.SetScreen(ScreenHelp)
			return m, nil
		}
	}
	return m, nil
}

// handlePackPasswordKeys handles keyboard input during pack password entry
func (m *Model) handlePackPasswordKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.SetScreen(ScreenMenu)
		return m, nil
	case "enter":
		password := m.textInput.Value()
		if password == "" {
			m.SetError("Password cannot be empty")
			return m, nil
		}
		m.SetScreen(ScreenPacking)
		return m, PackFilesCmd(m.app, m.scannedFiles, password)
	}
	return m, nil
}

// handleUnpackPasswordKeys handles keyboard input during unpack password entry
func (m *Model) handleUnpackPasswordKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.SetScreen(ScreenMenu)
		return m, nil
	case "enter":
		password := m.textInput.Value()
		if password == "" {
			m.SetError("Password cannot be empty")
			return m, nil
		}
		m.SetScreen(ScreenUnpacking)
		return m, UnpackFilesCmd(m.app, password, m.selectedArchive)
	}
	return m, nil
}

// handleListPasswordKeys handles keyboard input during list password entry
func (m *Model) handleListPasswordKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.SetScreen(ScreenMenu)
		return m, nil
	case "enter":
		password := m.textInput.Value()
		if password == "" {
			m.SetError("Password cannot be empty")
			return m, nil
		}
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