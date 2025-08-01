package tui

import (
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"goingenv/pkg/types"
)

// Screen represents different UI screens
type Screen string

const (
	ScreenMenu           Screen = "menu"
	ScreenPackPassword   Screen = "pack_password"
	ScreenUnpackSelect   Screen = "unpack_select"
	ScreenUnpackPassword Screen = "unpack_password"
	ScreenListSelect     Screen = "list_select"
	ScreenListPassword   Screen = "list_password"
	ScreenPacking        Screen = "packing"
	ScreenUnpacking      Screen = "unpacking"
	ScreenListing        Screen = "listing"
	ScreenStatus         Screen = "status"
	ScreenSettings       Screen = "settings"
	ScreenHelp           Screen = "help"
)

// Model represents the TUI application state
type Model struct {
	// Application dependencies
	app *types.App

	// Current state
	currentScreen   Screen
	width          int
	height         int
	message        string
	error          string
	selectedArchive string

	// UI components
	menu       list.Model
	textInput  textinput.Model
	filepicker filepicker.Model
	progress   progress.Model

	// Data
	scannedFiles []types.EnvFile
}

// NewModel creates a new TUI model
func NewModel(app *types.App) *Model {
	// Initialize menu items
	items := []list.Item{
		MenuItem{
			title:       "Pack Environment Files",
			description: "Scan and encrypt environment files",
			icon:        "📦",
			action:      "pack",
		},
		MenuItem{
			title:       "Unpack Archive",
			description: "Decrypt and restore archived files",
			icon:        "📂",
			action:      "unpack",
		},
		MenuItem{
			title:       "List Archive Contents",
			description: "Browse archive contents without extracting",
			icon:        "📋",
			action:      "list",
		},
		MenuItem{
			title:       "Status",
			description: "View current directory and available archives",
			icon:        "📊",
			action:      "status",
		},
		MenuItem{
			title:       "Settings",
			description: "Configure default options",
			icon:        "⚙️",
			action:      "settings",
		},
		MenuItem{
			title:       "Help",
			description: "Command documentation and examples",
			icon:        "❓",
			action:      "help",
		},
	}

	// Create list component
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "GoingEnv - Environment File Manager"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	// Create text input component
	ti := textinput.New()
	ti.Placeholder = "Enter password..."
	ti.EchoMode = textinput.EchoPassword
	ti.CharLimit = 256

	// Create file picker component
	fp := filepicker.New()
	fp.AllowedTypes = []string{".enc"}

	// Create progress component
	prog := progress.New(progress.WithDefaultGradient())

	return &Model{
		app:           app,
		currentScreen: ScreenMenu,
		menu:         l,
		textInput:    ti,
		filepicker:   fp,
		progress:     prog,
	}
}

// Init implements tea.Model interface
func (m *Model) Init() tea.Cmd {
	return nil
}

// MenuItem represents a menu item in the TUI
type MenuItem struct {
	title       string
	description string
	icon        string
	action      string
}

// Title implements list.Item interface
func (mi MenuItem) Title() string {
	return mi.icon + " " + mi.title
}

// Description implements list.Item interface
func (mi MenuItem) Description() string {
	return mi.description
}

// FilterValue implements list.Item interface
func (mi MenuItem) FilterValue() string {
	return mi.title
}

// Message types for async operations
type (
	PackCompleteMsg    string
	UnpackCompleteMsg  string
	ListCompleteMsg    string
	ScanCompleteMsg    []types.EnvFile
	ErrorMsg           string
	ProgressMsg        float64
)

// Helper methods for state management

// SetScreen changes the current screen
func (m *Model) SetScreen(screen Screen) {
	m.currentScreen = screen
	
	// Reset state when changing screens
	m.message = ""
	m.error = ""
	
	// Focus/blur components as needed
	switch screen {
	case ScreenPackPassword, ScreenUnpackPassword, ScreenListPassword:
		m.textInput.Focus()
		m.textInput.SetValue("")
	default:
		m.textInput.Blur()
	}
}

// SetError sets an error message and returns to menu
func (m *Model) SetError(err string) {
	m.error = err
	m.message = ""
	m.SetScreen(ScreenMenu)
}

// SetMessage sets a success message
func (m *Model) SetMessage(msg string) {
	m.message = msg
	m.error = ""
}

// GetSelectedMenuItem returns the currently selected menu item
func (m *Model) GetSelectedMenuItem() MenuItem {
	if item, ok := m.menu.SelectedItem().(MenuItem); ok {
		return item
	}
	return MenuItem{}
}

// UpdateSize updates the model dimensions
func (m *Model) UpdateSize(width, height int) {
	m.width = width
	m.height = height
	m.menu.SetWidth(width)
	m.menu.SetHeight(height - 4) // Leave space for title and footer
}