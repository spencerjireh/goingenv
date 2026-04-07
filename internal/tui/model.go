package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"goingenv/pkg/types"
)

// TabID identifies each tab in the tab bar.
type TabID int

const (
	TabPack TabID = iota
	TabUnpack
	TabList
	TabStatus
	TabSettings
	tabCount = 5
)

// Tab is the interface every tab must implement.
type Tab interface {
	Update(msg tea.Msg) (Tab, tea.Cmd)
	View(width, height int) string
	ShortHelp() []key.Binding
	FullHelp() [][]key.Binding
	Title() string
	// InputFocused returns true when a text input is active (disables global number keys).
	InputFocused() bool
}

// Message types used by the old screen-based model (kept for commands.go compatibility).
type (
	PackCompleteMsg   string
	UnpackCompleteMsg string
	ListCompleteMsg   string
	ScanCompleteMsg   []types.EnvFile
	ErrorMsg          string
	ProgressMsg       float64
)

// Model is the root Bubbletea model for the full-screen TUI.
type Model struct {
	app         *types.App
	debugLogger *DebugLogger
	version     string

	width  int
	height int

	// Tab system
	activeTab TabID
	tabs      [tabCount]Tab

	// Overlays
	modal       *ModalState
	toasts      []Toast
	helpVisible bool
	help        help.Model
}

// ModalState holds the state for a confirmation modal overlay.
type ModalState struct {
	Title     string
	Body      string
	OnConfirm tea.Cmd
}

// Toast holds the state for a toast notification.
type Toast struct {
	ID        int
	Message   string
	IsError   bool
	ExpiresAt time.Time
}

var toastCounter int

// NewModel creates a new root TUI model.
func NewModel(app *types.App, verbose bool, version string) *Model {
	debugLogger := NewDebugLogger(verbose)

	h := help.New()

	m := &Model{
		app:         app,
		debugLogger: debugLogger,
		version:     version,
		activeTab:   TabPack,
		help:        h,
	}

	// Initialize all tabs
	m.tabs[TabPack] = NewPackTab(app, debugLogger)
	m.tabs[TabUnpack] = NewUnpackTab(app, debugLogger)
	m.tabs[TabList] = NewListTab(app, debugLogger)
	m.tabs[TabStatus] = NewStatusTab(app, debugLogger)
	m.tabs[TabSettings] = NewSettingsTab(app, debugLogger)

	debugLogger.Log("TUI Model initialized (tabbed layout) with verbose logging: %v", verbose)
	if verbose {
		debugLogger.Log("Debug log file: %s", debugLogger.GetLogPath())
	}

	return m
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.debugLogger.Log("Window resized: %dx%d", msg.Width, msg.Height)
		return m, nil

	case ToastExpiredMsg:
		m.removeToast(msg.ID)
		return m, nil

	case ShowModalMsg:
		m.modal = &ModalState{
			Title:     msg.Title,
			Body:      msg.Body,
			OnConfirm: msg.OnConfirm,
		}
		return m, nil

	case ToastMsg:
		return m, m.addToast(msg.Message, msg.IsError)

	case SwitchTabMsg:
		if msg.Tab >= 0 && msg.Tab < tabCount {
			m.activeTab = msg.Tab
		}
		return m, nil

	// Route async completion messages to their owning tab, not the active tab.
	case ScanCompleteMsg, PackCompleteMsg:
		updated, cmd := m.tabs[TabPack].Update(msg)
		m.tabs[TabPack] = updated
		return m, cmd

	case UnpackCompleteMsg:
		updated, cmd := m.tabs[TabUnpack].Update(msg)
		m.tabs[TabUnpack] = updated
		return m, cmd

	case ListCompleteMsg:
		updated, cmd := m.tabs[TabList].Update(msg)
		m.tabs[TabList] = updated
		return m, cmd

	// ErrorMsg: route to the active tab (the tab that initiated the operation).
	case ErrorMsg:
		updated, cmd := m.tabs[m.activeTab].Update(msg)
		m.tabs[m.activeTab] = updated
		return m, cmd

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.MouseMsg:
		return m.handleMouseMsg(msg)

	// Spinner ticks route to all tabs that might have spinners.
	case spinner.TickMsg:
		var cmds []tea.Cmd
		for i := range m.tabs {
			updated, cmd := m.tabs[i].Update(msg)
			m.tabs[i] = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}

	// Default: dispatch to active tab
	updated, cmd := m.tabs[m.activeTab].Update(msg)
	m.tabs[m.activeTab] = updated
	return m, cmd
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Modal takes priority over everything
	if m.modal != nil {
		return m.updateModal(msg)
	}

	// Ctrl+C always quits
	if key.Matches(msg, GlobalKeys.Quit) {
		return m, tea.Quit
	}

	inputFocused := m.tabs[m.activeTab].InputFocused()

	// Help toggle (disabled when text input focused)
	if !inputFocused && key.Matches(msg, GlobalKeys.Help) {
		m.helpVisible = !m.helpVisible
		return m, nil
	}

	// When help overlay is visible, any key closes it
	if m.helpVisible {
		m.helpVisible = false
		return m, nil
	}

	// Tab switching (disabled when text input focused)
	if !inputFocused {
		switch {
		case key.Matches(msg, GlobalKeys.NextTab):
			m.activeTab = (m.activeTab + 1) % tabCount
			m.debugLogger.Log("Tab switched to: %s", tabNames[m.activeTab])
			return m, nil
		case key.Matches(msg, GlobalKeys.PrevTab):
			m.activeTab = (m.activeTab + tabCount - 1) % tabCount
			m.debugLogger.Log("Tab switched to: %s", tabNames[m.activeTab])
			return m, nil
		case key.Matches(msg, GlobalKeys.Tab1):
			m.activeTab = TabPack
			return m, nil
		case key.Matches(msg, GlobalKeys.Tab2):
			m.activeTab = TabUnpack
			return m, nil
		case key.Matches(msg, GlobalKeys.Tab3):
			m.activeTab = TabList
			return m, nil
		case key.Matches(msg, GlobalKeys.Tab4):
			m.activeTab = TabStatus
			return m, nil
		case key.Matches(msg, GlobalKeys.Tab5):
			m.activeTab = TabSettings
			return m, nil
		}

		// q quits only when no input is focused
		if msg.String() == "q" {
			return m, tea.Quit
		}
	}

	// Dispatch to active tab
	updated, cmd := m.tabs[m.activeTab].Update(msg)
	m.tabs[m.activeTab] = updated
	return m, cmd
}

func (m *Model) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Check if click is on the tab bar row (row 1, after header)
	if msg.Type == tea.MouseLeft && msg.Y == headerHeight {
		idx := tabClickIndex(msg.X)
		if idx >= 0 && idx < tabCount {
			m.activeTab = TabID(idx)
			m.debugLogger.Log("Tab clicked: %s", tabNames[m.activeTab])
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) updateModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		cmd := m.modal.OnConfirm
		m.modal = nil
		return m, cmd
	case "n", "N", "esc":
		m.modal = nil
		return m, nil
	}
	return m, nil
}

func (m *Model) addToast(message string, isError bool) tea.Cmd {
	toastCounter++
	id := toastCounter
	m.toasts = append(m.toasts, Toast{
		ID:        id,
		Message:   message,
		IsError:   isError,
		ExpiresAt: time.Now().Add(3 * time.Second),
	})
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return ToastExpiredMsg{ID: id}
	})
}

func (m *Model) removeToast(id int) {
	for i, t := range m.toasts {
		if t.ID == id {
			m.toasts = append(m.toasts[:i], m.toasts[i+1:]...)
			return
		}
	}
}

// View implements tea.Model.
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	contentH := ContentHeight(m.height)

	header := RenderHeader(m.version)
	tabBar := renderTabBar(m.activeTab, m.width)
	content := m.tabs[m.activeTab].View(m.width, contentH)
	content = padContent(content, contentH)
	footer := renderShortHelp(m.tabs[m.activeTab].ShortHelp())

	// Compose vertical layout
	base := header + "\n" + tabBar + "\n" + content + "\n" + footer

	// Layer overlays
	if m.helpVisible {
		base = m.renderHelpOverlay(base)
	}
	if m.modal != nil {
		base = m.renderModalOverlay(base)
	}
	if len(m.toasts) > 0 {
		base = m.renderToastOverlay(base)
	}

	return base
}

// Cleanup closes the debug logger.
func (m *Model) Cleanup() {
	if m.debugLogger != nil {
		m.debugLogger.Close()
	}
}
