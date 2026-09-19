package tui

import (
	"fmt"
	"path/filepath"
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

// DiffStep represents a step in the diff wizard.
type DiffStep int

const (
	DiffStepIdle     DiffStep = iota // Empty state; Enter opens the picker for A
	DiffStepSelectA                  // File picker: first archive
	DiffStepTarget                   // Compare with another archive, or the working tree
	DiffStepSelectB                  // File picker: second archive
	DiffStepPassword                 // One password, used for both archives
	DiffStepDiffing                  // Spinner active
	DiffStepResult                   // Masked report or error
)

var diffStepNames = []string{"Archive A", "Archive B", "Password", "Diff"}

// DiffTab compares two archives, or an archive and the env files on disk, by
// key. Values never reach the screen.
type DiffTab struct {
	app         *types.App
	debugLogger *DebugLogger

	step          DiffStep
	archiveA      string
	archiveB      string // "" with workingTree set means compare against disk
	workingTree   bool
	textInput     textinput.Model
	filepicker    filepicker.Model // one instance, reused for A and B
	fpInitialized bool
	spinner       spinner.Model
	viewport      viewport.Model
	vpReady       bool
	resultMsg     string
	errorMsg      string
}

// NewDiffTab creates a new DiffTab.
func NewDiffTab(app *types.App, debugLogger *DebugLogger) *DiffTab {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = HighlightStyle

	fp := filepicker.New()
	fp.AllowedTypes = []string{".enc"}
	fp.CurrentDirectory = config.GetGoingEnvDir()

	return &DiffTab{
		app:         app,
		debugLogger: debugLogger,
		step:        DiffStepIdle,
		textInput:   newPasswordInput("Enter password..."),
		filepicker:  fp,
		spinner:     s,
	}
}

func (t *DiffTab) Title() string { return "Diff" }

func (t *DiffTab) InputFocused() bool {
	return t.step == DiffStepPassword
}

func (t *DiffTab) ShortHelp() []key.Binding {
	switch t.step {
	case DiffStepIdle:
		return []key.Binding{WizardKeys.Start}
	case DiffStepSelectA, DiffStepSelectB:
		return []key.Binding{NavigationKeys.Up, NavigationKeys.Down, NavigationKeys.Select, WizardKeys.Cancel}
	case DiffStepTarget:
		return []key.Binding{DiffKeys.Archive, DiffKeys.WorkingTree, WizardKeys.Back}
	case DiffStepPassword:
		return []key.Binding{WizardKeys.Confirm, WizardKeys.Cancel}
	case DiffStepDiffing:
		return nil
	case DiffStepResult:
		return []key.Binding{WizardKeys.Reset, WizardKeys.Back}
	}
	return nil
}

func (t *DiffTab) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{WizardKeys.Start, WizardKeys.Confirm, WizardKeys.Cancel, WizardKeys.Reset},
		{NavigationKeys.Up, NavigationKeys.Down, NavigationKeys.Select},
		{DiffKeys.Archive, DiffKeys.WorkingTree},
	}
}

func (t *DiffTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	// The picker sizes itself from the window size; see ListTab.Update.
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		var cmd tea.Cmd
		t.filepicker, cmd = t.filepicker.Update(sizeMsg)
		return t, cmd
	}

	switch t.step {
	case DiffStepIdle:
		return t.updateIdle(msg)
	case DiffStepSelectA:
		return t.updateSelectA(msg)
	case DiffStepTarget:
		return t.updateTarget(msg)
	case DiffStepSelectB:
		return t.updateSelectB(msg)
	case DiffStepPassword:
		return t.updatePassword(msg)
	case DiffStepDiffing:
		return t.updateDiffing(msg)
	case DiffStepResult:
		return t.updateResult(msg)
	}

	return t, nil
}

// updateIdle opens the picker for archive A on Enter.
func (t *DiffTab) updateIdle(msg tea.Msg) (Tab, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, WizardKeys.Start) {
			return t.startSelection()
		}
	}
	return t, nil
}

// updateSelectA drives the picker until archive A is chosen.
func (t *DiffTab) updateSelectA(msg tea.Msg) (Tab, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, WizardKeys.Cancel) {
			t.reset()
			return t, nil
		}
	}

	var cmd tea.Cmd
	t.filepicker, cmd = t.filepicker.Update(msg)

	if didSelect, path := t.filepicker.DidSelectFile(msg); didSelect {
		t.archiveA = path
		t.step = DiffStepTarget
		t.debugLogger.LogOperation("diff", fmt.Sprintf("selected A: %s", path))
		return t, nil
	}
	return t, cmd
}

// updateTarget asks what to compare A with.
func (t *DiffTab) updateTarget(msg tea.Msg) (Tab, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return t, nil
	}
	switch {
	case key.Matches(keyMsg, DiffKeys.Archive):
		t.step = DiffStepSelectB
		t.errorMsg = ""
		return t, nil
	case key.Matches(keyMsg, DiffKeys.WorkingTree):
		t.workingTree = true
		t.archiveB = ""
		return t.focusPassword()
	case key.Matches(keyMsg, WizardKeys.Back):
		t.step = DiffStepSelectA
		return t, nil
	}
	return t, nil
}

// updateSelectB drives the picker until archive B is chosen. Picking A again
// is refused in place: a diff of an archive with itself says nothing.
func (t *DiffTab) updateSelectB(msg tea.Msg) (Tab, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, WizardKeys.Cancel) {
			t.step = DiffStepTarget
			t.errorMsg = ""
			return t, nil
		}
	}

	var cmd tea.Cmd
	t.filepicker, cmd = t.filepicker.Update(msg)

	if didSelect, path := t.filepicker.DidSelectFile(msg); didSelect {
		if path == t.archiveA {
			t.errorMsg = "Pick a different archive for B"
			return t, cmd
		}
		t.archiveB = path
		t.workingTree = false
		t.debugLogger.LogOperation("diff", fmt.Sprintf("selected B: %s", path))
		return t.focusPassword()
	}
	return t, cmd
}

func (t *DiffTab) focusPassword() (Tab, tea.Cmd) {
	t.step = DiffStepPassword
	t.errorMsg = ""
	t.textInput.Reset()
	t.textInput.Focus()
	return t, textinput.Blink
}

// updatePassword collects the password and kicks off the comparison.
func (t *DiffTab) updatePassword(msg tea.Msg) (Tab, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, WizardKeys.Confirm):
			password := t.textInput.Value()
			if password == "" {
				t.errorMsg = "Password cannot be empty"
				return t, nil
			}
			t.errorMsg = ""
			t.step = DiffStepDiffing
			t.textInput.Reset()
			t.debugLogger.LogOperation("diff", "comparing")
			return t, tea.Batch(t.spinner.Tick, DiffFilesCmd(t.app, password, t.archiveA, t.archiveB))
		case key.Matches(keyMsg, WizardKeys.Cancel):
			t.step = DiffStepTarget
			t.textInput.Blur()
			return t, nil
		}
	}

	var cmd tea.Cmd
	t.textInput, cmd = t.textInput.Update(msg)
	return t, cmd
}

// updateDiffing waits for the comparison to finish.
func (t *DiffTab) updateDiffing(msg tea.Msg) (Tab, tea.Cmd) {
	switch msg := msg.(type) {
	case DiffCompleteMsg:
		t.resultMsg = string(msg)
		t.errorMsg = ""
		t.step = DiffStepResult
		t.vpReady = false
		t.debugLogger.LogOperation("diff", "complete")
		return t, nil
	case ErrorMsg:
		t.errorMsg = msg.Text
		t.resultMsg = ""
		t.step = DiffStepResult
		return t, func() tea.Msg {
			return ToastMsg{Message: msg.Text, IsError: true}
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		t.spinner, cmd = t.spinner.Update(msg)
		return t, cmd
	}
	return t, nil
}

// updateResult lets the user scroll the report or start over.
func (t *DiffTab) updateResult(msg tea.Msg) (Tab, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, WizardKeys.Reset),
			key.Matches(keyMsg, WizardKeys.Start),
			key.Matches(keyMsg, WizardKeys.Back):
			t.reset()
			return t, nil
		}
	}
	return t, scrollViewport(&t.viewport, t.vpReady, msg)
}

func (t *DiffTab) startSelection() (Tab, tea.Cmd) {
	archives, err := t.app.Archiver.GetAvailableArchives("")
	if err != nil || len(archives) == 0 {
		return t, nil
	}
	t.step = DiffStepSelectA

	if !t.fpInitialized {
		t.fpInitialized = true
		return t, t.filepicker.Init()
	}
	return t, nil
}

func (t *DiffTab) reset() {
	t.step = DiffStepIdle
	t.archiveA = ""
	t.archiveB = ""
	t.workingTree = false
	t.resultMsg = ""
	t.errorMsg = ""
	t.textInput.Reset()
	t.textInput.Blur()
	t.vpReady = false
}

func (t *DiffTab) labelA() string { return filepath.Base(t.archiveA) }

func (t *DiffTab) labelB() string {
	if t.workingTree {
		return "working tree"
	}
	return filepath.Base(t.archiveB)
}

func (t *DiffTab) View(width, height int) string {
	stepLine := renderStepIndicator(diffStepNames, t.stepIndex())
	contentHeight := height - 2

	var content string
	switch t.step {
	case DiffStepIdle:
		content = t.renderIdle(width, contentHeight)
	case DiffStepSelectA:
		content = "\n" + RenderSectionHeader("  Select archive A") + "\n\n" + t.filepicker.View()
	case DiffStepTarget:
		content = t.renderTarget()
	case DiffStepSelectB:
		// The picker fills the rest of the screen, so the message goes above it.
		content = "\n" + RenderSectionHeader("  Select archive B") + "\n"
		if t.errorMsg != "" {
			content += "  " + ErrorStyle.Render(t.errorMsg)
		}
		content += "\n" + t.filepicker.View()
	case DiffStepPassword:
		content = t.renderPasswordEntry()
	case DiffStepDiffing:
		content = fmt.Sprintf("\n  %s Comparing...", t.spinner.View())
	case DiffStepResult:
		content = t.renderResult(width, contentHeight)
	}

	return content + "\n\n" + stepLine
}

func (t *DiffTab) renderIdle(width, height int) string {
	archives, err := t.app.Archiver.GetAvailableArchives("")
	if err != nil {
		archives = nil
	}
	if len(archives) == 0 {
		return renderEmptyState(
			"No Archives Found",
			"There is nothing to compare yet.",
			"Switch to the Pack tab to create an archive.",
			width, height,
		)
	}
	return renderEmptyState(
		"Diff Archives",
		"See which keys differ between two archives, or between an archive and the env files on disk. Values are never shown.",
		"Press Enter to select the first archive.",
		width, height,
	)
}

func (t *DiffTab) stepIndex() int {
	switch t.step {
	case DiffStepIdle, DiffStepSelectA:
		return 0
	case DiffStepTarget, DiffStepSelectB:
		return 1
	case DiffStepPassword:
		return 2
	case DiffStepDiffing, DiffStepResult:
		return 3
	}
	return 0
}

func (t *DiffTab) renderTarget() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(RenderSectionHeader("  Compare "+t.labelA()+" with") + "\n\n")
	b.WriteString("  [enter] another archive\n")
	b.WriteString("  [w]     the working tree (env files on disk)\n")
	b.WriteString("  [esc]   back\n")
	return b.String()
}

func (t *DiffTab) renderPasswordEntry() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(RenderSectionHeader("  Diff "+t.labelA()+" -> "+t.labelB()) + "\n\n")
	b.WriteString("  Password: " + t.textInput.View() + "\n")
	if t.errorMsg != "" {
		b.WriteString("\n  " + ErrorStyle.Render(t.errorMsg))
	}
	return b.String()
}

func (t *DiffTab) renderResult(width, height int) string {
	if t.errorMsg != "" {
		return renderEmptyState(
			"Diff Failed",
			t.errorMsg,
			"Press Enter or R to try again.",
			width, height,
		)
	}

	return renderScrollable(&t.viewport, &t.vpReady, width, height, "", func() string {
		return "\n" + RenderSectionHeader("  "+t.labelA()+" -> "+t.labelB()) + "\n\n" + t.resultMsg
	})
}
