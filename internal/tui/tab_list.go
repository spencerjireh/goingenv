package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"goingenv/internal/config"
	"goingenv/pkg/types"
)

// ListStep represents a step in the list wizard.
type ListStep int

const (
	ListStepIdle     ListStep = iota // Empty state
	ListStepSelect                   // File picker
	ListStepPassword                 // Enter password
	ListStepListing                  // Spinner active
	ListStepResult                   // Show contents or error
)

var listStepNames = []string{"Select", "Password", "View"}

// ListTab implements the List wizard tab.
type ListTab struct {
	app         *types.App
	debugLogger *DebugLogger

	step            ListStep
	selectedArchive string
	textInput       textinput.Model
	filepicker      filepicker.Model
	spinner         spinner.Model
	viewport        viewport.Model
	vpReady         bool
	resultMsg       string
	errorMsg        string
	fpInitialized   bool
}

// NewListTab creates a new ListTab.
func NewListTab(app *types.App, debugLogger *DebugLogger) *ListTab {
	ti := textinput.New()
	ti.Placeholder = "Enter password..."
	ti.EchoMode = textinput.EchoPassword
	ti.CharLimit = 256

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = HighlightStyle

	fp := filepicker.New()
	fp.AllowedTypes = []string{".enc"}
	fp.CurrentDirectory = config.GetGoingEnvDir()

	return &ListTab{
		app:         app,
		debugLogger: debugLogger,
		step:        ListStepIdle,
		textInput:   ti,
		filepicker:  fp,
		spinner:     s,
	}
}

func (t *ListTab) Title() string { return "List" }

func (t *ListTab) InputFocused() bool {
	return t.step == ListStepPassword
}

func (t *ListTab) ShortHelp() []key.Binding {
	switch t.step {
	case ListStepIdle:
		return []key.Binding{WizardKeys.Start}
	case ListStepSelect:
		return []key.Binding{NavigationKeys.Up, NavigationKeys.Down, NavigationKeys.Select, WizardKeys.Cancel}
	case ListStepPassword:
		return []key.Binding{WizardKeys.Confirm, WizardKeys.Cancel}
	case ListStepListing:
		return nil
	case ListStepResult:
		return []key.Binding{WizardKeys.Reset, WizardKeys.Back}
	}
	return nil
}

func (t *ListTab) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{WizardKeys.Start, WizardKeys.Confirm, WizardKeys.Cancel, WizardKeys.Reset},
		{NavigationKeys.Up, NavigationKeys.Down, NavigationKeys.Select},
	}
}

func (t *ListTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	switch t.step {
	case ListStepIdle:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(keyMsg, WizardKeys.Start) {
				return t.startSelection()
			}
		}

	case ListStepSelect:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(keyMsg, WizardKeys.Cancel) {
				t.reset()
				return t, nil
			}
		}

		var cmd tea.Cmd
		t.filepicker, cmd = t.filepicker.Update(msg)

		if didSelect, path := t.filepicker.DidSelectFile(msg); didSelect {
			t.selectedArchive = path
			t.step = ListStepPassword
			t.textInput.Reset()
			t.textInput.Focus()
			t.debugLogger.LogOperation("list", fmt.Sprintf("selected: %s", path))
			return t, textinput.Blink
		}

		return t, cmd

	case ListStepPassword:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch {
			case key.Matches(keyMsg, WizardKeys.Confirm):
				password := t.textInput.Value()
				if password == "" {
					t.errorMsg = "Password cannot be empty"
					return t, nil
				}
				t.errorMsg = ""
				t.step = ListStepListing
				t.debugLogger.LogOperation("list", "listing archive")
				return t, tea.Batch(t.spinner.Tick, ListFilesCmd(t.app, password, t.selectedArchive))
			case key.Matches(keyMsg, WizardKeys.Cancel):
				t.step = ListStepSelect
				t.textInput.Blur()
				return t, nil
			}
		}
		var cmd tea.Cmd
		t.textInput, cmd = t.textInput.Update(msg)
		return t, cmd

	case ListStepListing:
		switch msg := msg.(type) {
		case ListCompleteMsg:
			t.resultMsg = string(msg)
			t.errorMsg = ""
			t.step = ListStepResult
			t.vpReady = false
			t.debugLogger.LogOperation("list", "listing complete")
			return t, nil
		case ErrorMsg:
			t.errorMsg = string(msg)
			t.resultMsg = ""
			t.step = ListStepResult
			return t, func() tea.Msg {
				return ToastMsg{Message: string(msg), IsError: true}
			}
		case spinner.TickMsg:
			var cmd tea.Cmd
			t.spinner, cmd = t.spinner.Update(msg)
			return t, cmd
		}

	case ListStepResult:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch {
			case key.Matches(keyMsg, WizardKeys.Reset), key.Matches(keyMsg, WizardKeys.Start):
				t.reset()
				return t, nil
			case key.Matches(keyMsg, WizardKeys.Back):
				t.reset()
				return t, nil
			}
		}
		if t.vpReady {
			var cmd tea.Cmd
			t.viewport, cmd = t.viewport.Update(msg)
			return t, cmd
		}
	}

	return t, nil
}

func (t *ListTab) startSelection() (Tab, tea.Cmd) {
	archives, err := t.app.Archiver.GetAvailableArchives("")
	if err != nil || len(archives) == 0 {
		return t, nil
	}
	t.step = ListStepSelect

	if !t.fpInitialized {
		t.fpInitialized = true
		return t, t.filepicker.Init()
	}
	return t, nil
}

func (t *ListTab) reset() {
	t.step = ListStepIdle
	t.selectedArchive = ""
	t.resultMsg = ""
	t.errorMsg = ""
	t.textInput.Reset()
	t.textInput.Blur()
	t.vpReady = false
}

func (t *ListTab) View(width, height int) string {
	stepLine := renderStepIndicator(listStepNames, t.stepIndex())
	contentHeight := height - 2

	var content string
	switch t.step {
	case ListStepIdle:
		archives, _ := t.app.Archiver.GetAvailableArchives("")
		if len(archives) == 0 {
			content = renderEmptyState(
				"No Archives Found",
				"Pack your environment files first, then come back here to browse contents.",
				"Switch to the Pack tab to create an archive.",
				width, contentHeight,
			)
		} else {
			content = renderEmptyState(
				"List Archive Contents",
				fmt.Sprintf("Browse the contents of %d available archive(s) without extracting.", len(archives)),
				"Press Enter to select an archive.",
				width, contentHeight,
			)
		}
	case ListStepSelect:
		content = "\n" + RenderSectionHeader("  Select archive to list") + "\n\n"
		content += t.filepicker.View()
	case ListStepPassword:
		content = t.renderPasswordEntry()
	case ListStepListing:
		content = fmt.Sprintf("\n  %s Reading archive contents...", t.spinner.View())
	case ListStepResult:
		content = t.renderResult(width, contentHeight)
	}

	return content + "\n\n" + stepLine
}

func (t *ListTab) stepIndex() int {
	switch t.step {
	case ListStepIdle, ListStepSelect:
		return 0
	case ListStepPassword:
		return 1
	case ListStepListing, ListStepResult:
		return 2
	}
	return 0
}

func (t *ListTab) renderPasswordEntry() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(RenderSectionHeader("  List archive contents") + "\n\n")
	b.WriteString(fmt.Sprintf("  %s\n\n", t.selectedArchive))
	b.WriteString("  Password: " + t.textInput.View() + "\n")

	if t.errorMsg != "" {
		b.WriteString("\n  " + ErrorStyle.Render(t.errorMsg))
	}

	return b.String()
}

func (t *ListTab) renderResult(width, height int) string {
	if t.errorMsg != "" {
		return renderEmptyState(
			"List Failed",
			t.errorMsg,
			"Press Enter or R to try again.",
			width, height,
		)
	}

	if !t.vpReady {
		t.viewport = viewport.New(width, height)
		t.vpReady = true
	} else {
		t.viewport.Width = width
		t.viewport.Height = height
	}
	t.viewport.SetContent("\n" + RenderSectionHeader("  Archive Contents") + "\n\n" + t.resultMsg)
	return t.viewport.View()
}
