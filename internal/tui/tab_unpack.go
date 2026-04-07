package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"goingenv/internal/config"
	"goingenv/pkg/types"
)

// UnpackStep represents a step in the unpack wizard.
type UnpackStep int

const (
	UnpackStepIdle      UnpackStep = iota // Empty state or select archive
	UnpackStepSelect                      // File picker
	UnpackStepPassword                    // Enter password
	UnpackStepUnpacking                   // Spinner active
	UnpackStepResult                      // Success or error
)

var unpackStepNames = []string{"Select", "Password", "Unpack"}

// UnpackTab implements the Unpack wizard tab.
type UnpackTab struct {
	app         *types.App
	debugLogger *DebugLogger

	step            UnpackStep
	selectedArchive string
	archives        []string
	textInput       textinput.Model
	filepicker      filepicker.Model
	spinner         spinner.Model
	resultMsg       string
	errorMsg        string
	fpInitialized   bool
}

// NewUnpackTab creates a new UnpackTab.
func NewUnpackTab(app *types.App, debugLogger *DebugLogger) *UnpackTab {
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

	return &UnpackTab{
		app:         app,
		debugLogger: debugLogger,
		step:        UnpackStepIdle,
		textInput:   ti,
		filepicker:  fp,
		spinner:     s,
	}
}

func (t *UnpackTab) Title() string { return "Unpack" }

func (t *UnpackTab) InputFocused() bool {
	return t.step == UnpackStepPassword
}

func (t *UnpackTab) ShortHelp() []key.Binding {
	switch t.step {
	case UnpackStepIdle:
		return []key.Binding{WizardKeys.Start}
	case UnpackStepSelect:
		return []key.Binding{NavigationKeys.Up, NavigationKeys.Down, NavigationKeys.Select, WizardKeys.Cancel}
	case UnpackStepPassword:
		return []key.Binding{WizardKeys.Confirm, WizardKeys.Cancel}
	case UnpackStepUnpacking:
		return nil
	case UnpackStepResult:
		return []key.Binding{WizardKeys.Reset}
	}
	return nil
}

func (t *UnpackTab) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{WizardKeys.Start, WizardKeys.Confirm, WizardKeys.Cancel, WizardKeys.Reset},
		{NavigationKeys.Up, NavigationKeys.Down, NavigationKeys.Select},
	}
}

func (t *UnpackTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	switch t.step {
	case UnpackStepIdle:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(keyMsg, WizardKeys.Start) {
				return t.startSelection()
			}
		}

	case UnpackStepSelect:
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
			t.step = UnpackStepPassword
			t.textInput.Reset()
			t.textInput.Focus()
			t.debugLogger.LogOperation("unpack", fmt.Sprintf("selected: %s", path))
			return t, textinput.Blink
		}

		return t, cmd

	case UnpackStepPassword:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch {
			case key.Matches(keyMsg, WizardKeys.Confirm):
				password := t.textInput.Value()
				if password == "" {
					t.errorMsg = "Password cannot be empty"
					return t, nil
				}
				t.errorMsg = ""
				t.step = UnpackStepUnpacking
				t.debugLogger.LogOperation("unpack", "unpacking")
				return t, tea.Batch(t.spinner.Tick, UnpackFilesCmd(t.app, password, t.selectedArchive))
			case key.Matches(keyMsg, WizardKeys.Cancel):
				t.step = UnpackStepSelect
				t.textInput.Blur()
				return t, nil
			}
		}
		var cmd tea.Cmd
		t.textInput, cmd = t.textInput.Update(msg)
		return t, cmd

	case UnpackStepUnpacking:
		switch msg := msg.(type) {
		case UnpackCompleteMsg:
			t.resultMsg = string(msg)
			t.errorMsg = ""
			t.step = UnpackStepResult
			t.debugLogger.LogOperation("unpack", "complete")
			return t, func() tea.Msg {
				return ToastMsg{Message: "Unpack completed successfully", IsError: false}
			}
		case ErrorMsg:
			t.errorMsg = string(msg)
			t.resultMsg = ""
			t.step = UnpackStepResult
			return t, func() tea.Msg {
				return ToastMsg{Message: string(msg), IsError: true}
			}
		case spinner.TickMsg:
			var cmd tea.Cmd
			t.spinner, cmd = t.spinner.Update(msg)
			return t, cmd
		}

	case UnpackStepResult:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(keyMsg, WizardKeys.Reset) || key.Matches(keyMsg, WizardKeys.Start) {
				t.reset()
				return t, nil
			}
		}
	}

	return t, nil
}

func (t *UnpackTab) startSelection() (Tab, tea.Cmd) {
	// Refresh archives list
	archives, err := t.app.Archiver.GetAvailableArchives("")
	if err != nil || len(archives) == 0 {
		// No archives: stay on idle with error
		return t, nil
	}
	t.archives = archives
	t.step = UnpackStepSelect

	if !t.fpInitialized {
		t.fpInitialized = true
		return t, t.filepicker.Init()
	}
	return t, nil
}

func (t *UnpackTab) reset() {
	t.step = UnpackStepIdle
	t.selectedArchive = ""
	t.resultMsg = ""
	t.errorMsg = ""
	t.textInput.Reset()
	t.textInput.Blur()
}

func (t *UnpackTab) View(width, height int) string {
	stepLine := renderStepIndicator(unpackStepNames, t.stepIndex())
	contentHeight := height - 2

	var content string
	switch t.step {
	case UnpackStepIdle:
		archives, _ := t.app.Archiver.GetAvailableArchives("")
		if len(archives) == 0 {
			content = renderEmptyState(
				"No Archives Found",
				"Pack your environment files first, then come back here to unpack.",
				"Switch to the Pack tab to create an archive.",
				width, contentHeight,
			)
		} else {
			content = renderEmptyState(
				"Unpack Archive",
				fmt.Sprintf("Decrypt and restore environment files from %d available archive(s).", len(archives)),
				"Press Enter to select an archive.",
				width, contentHeight,
			)
		}
	case UnpackStepSelect:
		content = "\n" + RenderSectionHeader("  Select archive to unpack") + "\n\n"
		content += t.filepicker.View()
	case UnpackStepPassword:
		content = t.renderPasswordEntry()
	case UnpackStepUnpacking:
		content = fmt.Sprintf("\n  %s Decrypting and extracting...", t.spinner.View())
	case UnpackStepResult:
		content = t.renderResult(width, contentHeight)
	}

	return content + "\n\n" + stepLine
}

func (t *UnpackTab) stepIndex() int {
	switch t.step {
	case UnpackStepIdle, UnpackStepSelect:
		return 0
	case UnpackStepPassword:
		return 1
	case UnpackStepUnpacking, UnpackStepResult:
		return 2
	}
	return 0
}

func (t *UnpackTab) renderPasswordEntry() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(RenderSectionHeader("  Unpacking archive") + "\n\n")
	b.WriteString(fmt.Sprintf("  %s\n\n", t.selectedArchive))
	b.WriteString("  Password: " + t.textInput.View() + "\n")

	if t.errorMsg != "" {
		b.WriteString("\n  " + ErrorStyle.Render(t.errorMsg))
	}

	return b.String()
}

func (t *UnpackTab) renderResult(width, height int) string {
	if t.errorMsg != "" {
		return renderEmptyState(
			"Unpack Failed",
			t.errorMsg,
			"Press Enter or R to try again.",
			width, height,
		)
	}
	return renderEmptyState(
		"Unpack Successful",
		t.resultMsg,
		"Press Enter or R to unpack again.",
		width, height,
	)
}
