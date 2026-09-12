package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"goingenv/internal/config"
	"goingenv/pkg/types"
	"goingenv/pkg/utils"
)

// PackStep represents a step in the pack wizard.
type PackStep int

const (
	PackStepIdle     PackStep = iota // Press enter to scan
	PackStepScanning                 // Spinner active
	PackStepReview                   // Show scanned files
	PackStepPassword                 // Enter password
	PackStepConfirm                  // Re-enter password
	PackStepPacking                  // Spinner active
	PackStepResult                   // Success or error
)

var packStepNames = []string{"Scan", "Review", "Password", "Pack"}

// PackTab implements the Pack wizard tab.
type PackTab struct {
	app         *types.App
	debugLogger *DebugLogger

	step         PackStep
	scannedFiles []types.EnvFile
	textInput    textinput.Model
	confirmInput textinput.Model
	password     string // first entry, held until the confirmation matches it
	spinner      spinner.Model
	viewport     viewport.Model
	vpReady      bool
	resultMsg    string
	errorMsg     string
}

// NewPackTab creates a new PackTab.
func NewPackTab(app *types.App, debugLogger *DebugLogger) *PackTab {
	ti := textinput.New()
	ti.Placeholder = "Enter password..."
	ti.EchoMode = textinput.EchoPassword
	ti.CharLimit = 256

	ci := textinput.New()
	ci.Placeholder = "Re-enter password..."
	ci.EchoMode = textinput.EchoPassword
	ci.CharLimit = 256

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = HighlightStyle

	return &PackTab{
		app:          app,
		debugLogger:  debugLogger,
		step:         PackStepIdle,
		textInput:    ti,
		confirmInput: ci,
		spinner:      s,
	}
}

func (t *PackTab) Title() string { return "Pack" }

func (t *PackTab) InputFocused() bool {
	return t.step == PackStepPassword || t.step == PackStepConfirm
}

func (t *PackTab) ShortHelp() []key.Binding {
	switch t.step {
	case PackStepIdle:
		return []key.Binding{WizardKeys.Start}
	case PackStepScanning, PackStepPacking:
		return nil
	case PackStepReview:
		return []key.Binding{WizardKeys.Confirm, WizardKeys.Cancel}
	case PackStepPassword, PackStepConfirm:
		return []key.Binding{WizardKeys.Confirm, WizardKeys.Cancel}
	case PackStepResult:
		return []key.Binding{WizardKeys.Reset}
	}
	return nil
}

func (t *PackTab) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{WizardKeys.Start, WizardKeys.Confirm, WizardKeys.Cancel, WizardKeys.Reset},
	}
}

func (t *PackTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	switch t.step {
	case PackStepIdle:
		return t.updateIdle(msg)
	case PackStepScanning:
		return t.updateScanning(msg)
	case PackStepReview:
		return t.updateReview(msg)
	case PackStepPassword:
		return t.updatePassword(msg)
	case PackStepConfirm:
		return t.updateConfirm(msg)
	case PackStepPacking:
		return t.updatePacking(msg)
	case PackStepResult:
		return t.updateResult(msg)
	}

	return t, nil
}

// updateIdle starts a scan on Enter, initializing the project first when
// goingenv has not been set up in this directory yet.
func (t *PackTab) updateIdle(msg tea.Msg) (Tab, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok || !key.Matches(keyMsg, WizardKeys.Start) {
		return t, nil
	}

	t.step = PackStepScanning
	if !config.IsInitialized() {
		t.debugLogger.LogOperation("pack", "initializing project")
		return t, tea.Batch(t.spinner.Tick, InitProjectCmd())
	}

	t.debugLogger.LogOperation("pack", "scanning files")
	return t, tea.Batch(t.spinner.Tick, ScanFilesCmd(t.app))
}

// updateScanning waits for the scan (or the init that precedes it) to finish.
func (t *PackTab) updateScanning(msg tea.Msg) (Tab, tea.Cmd) {
	switch msg := msg.(type) {
	case ScanCompleteMsg:
		t.scannedFiles = []types.EnvFile(msg)
		t.step = PackStepReview
		t.vpReady = false
		t.debugLogger.LogOperation("pack", fmt.Sprintf("scan complete: %d files", len(t.scannedFiles)))
		return t, nil
	case InitCompleteMsg:
		// After init, scan files
		t.debugLogger.LogOperation("pack", "init complete, scanning")
		return t, ScanFilesCmd(t.app)
	case ErrorMsg:
		t.errorMsg = string(msg)
		t.step = PackStepResult
		return t, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		t.spinner, cmd = t.spinner.Update(msg)
		return t, cmd
	}

	return t, nil
}

// updateReview lets the user confirm the scanned file list or scroll through it.
func (t *PackTab) updateReview(msg tea.Msg) (Tab, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, WizardKeys.Confirm):
			t.step = PackStepPassword
			t.textInput.Reset()
			t.textInput.Focus()
			return t, textinput.Blink
		case key.Matches(keyMsg, WizardKeys.Cancel):
			t.reset()
			return t, nil
		}
	}

	if t.vpReady {
		var cmd tea.Cmd
		t.viewport, cmd = t.viewport.Update(msg)
		return t, cmd
	}

	return t, nil
}

// updatePassword collects the archive password and asks for it a second time.
func (t *PackTab) updatePassword(msg tea.Msg) (Tab, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, WizardKeys.Confirm):
			password := t.textInput.Value()
			if password == "" {
				t.errorMsg = "Password cannot be empty"
				return t, nil
			}
			t.errorMsg = ""
			t.password = password
			t.step = PackStepConfirm
			t.textInput.Blur()
			t.confirmInput.Reset()
			t.confirmInput.Focus()
			return t, textinput.Blink
		case key.Matches(keyMsg, WizardKeys.Cancel):
			t.step = PackStepReview
			t.textInput.Blur()
			return t, nil
		}
	}

	var cmd tea.Cmd
	t.textInput, cmd = t.textInput.Update(msg)
	return t, cmd
}

// updateConfirm compares the second entry to the first and kicks off the pack
// when they match. A typo here would otherwise produce an archive nobody can
// open, so a mismatch starts the entry over rather than trusting either copy.
func (t *PackTab) updateConfirm(msg tea.Msg) (Tab, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, WizardKeys.Confirm):
			password := t.password
			match := t.confirmInput.Value() == password
			t.password = ""
			t.textInput.Reset()
			t.confirmInput.Reset()
			t.confirmInput.Blur()
			if !match {
				t.errorMsg = "Passwords do not match"
				t.step = PackStepPassword
				t.textInput.Focus()
				return t, textinput.Blink
			}
			t.errorMsg = ""
			t.step = PackStepPacking
			t.debugLogger.LogOperation("pack", "packing files")
			return t, tea.Batch(t.spinner.Tick, PackFilesCmd(t.app, t.scannedFiles, password))
		case key.Matches(keyMsg, WizardKeys.Cancel):
			// Keep the first entry so it can be edited rather than retyped.
			t.password = ""
			t.step = PackStepPassword
			t.confirmInput.Reset()
			t.confirmInput.Blur()
			t.textInput.Focus()
			return t, textinput.Blink
		}
	}

	var cmd tea.Cmd
	t.confirmInput, cmd = t.confirmInput.Update(msg)
	return t, cmd
}

// updatePacking waits for the pack to finish and reports the outcome as a toast.
func (t *PackTab) updatePacking(msg tea.Msg) (Tab, tea.Cmd) {
	switch msg := msg.(type) {
	case PackCompleteMsg:
		t.resultMsg = string(msg)
		t.errorMsg = ""
		t.step = PackStepResult
		t.vpReady = false
		t.debugLogger.LogOperation("pack", "pack complete")
		return t, func() tea.Msg {
			return ToastMsg{Message: "Pack completed successfully", IsError: false}
		}
	case ErrorMsg:
		t.errorMsg = string(msg)
		t.resultMsg = ""
		t.step = PackStepResult
		return t, func() tea.Msg {
			return ToastMsg{Message: string(msg), IsError: true}
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		t.spinner, cmd = t.spinner.Update(msg)
		return t, cmd
	}

	return t, nil
}

// updateResult returns the tab to idle so another pack can be started, and
// otherwise lets the user scroll the list of packed files.
func (t *PackTab) updateResult(msg tea.Msg) (Tab, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, WizardKeys.Reset) || key.Matches(keyMsg, WizardKeys.Start) {
			t.reset()
			return t, nil
		}
	}

	if t.vpReady {
		var cmd tea.Cmd
		t.viewport, cmd = t.viewport.Update(msg)
		return t, cmd
	}

	return t, nil
}

func (t *PackTab) reset() {
	t.step = PackStepIdle
	t.scannedFiles = nil
	t.resultMsg = ""
	t.errorMsg = ""
	t.textInput.Reset()
	t.textInput.Blur()
	t.confirmInput.Reset()
	t.confirmInput.Blur()
	t.password = ""
	t.vpReady = false
}

func (t *PackTab) View(width, height int) string {
	// Reserve space for step indicator
	stepLine := renderStepIndicator(packStepNames, t.stepIndex())
	contentHeight := height - 2 // step indicator + gap

	var content string
	switch t.step {
	case PackStepIdle:
		if !config.IsInitialized() {
			content = renderEmptyState(
				"Initialize goingenv",
				"goingenv is not set up in this directory yet.",
				"Press Enter to initialize and scan for environment files.",
				width, contentHeight,
			)
		} else {
			content = renderEmptyState(
				"Pack Environment Files",
				"Scan your project for .env files and encrypt them into one archive.",
				"Press Enter to scan for environment files.",
				width, contentHeight,
			)
		}
	case PackStepScanning:
		content = fmt.Sprintf("\n  %s Scanning for environment files...", t.spinner.View())
	case PackStepReview:
		content = t.renderReview(width, contentHeight)
	case PackStepPassword, PackStepConfirm:
		content = t.renderPasswordEntry()
	case PackStepPacking:
		content = fmt.Sprintf("\n  %s Encrypting and archiving...", t.spinner.View())
	case PackStepResult:
		content = t.renderResult(width, contentHeight)
	}

	return content + "\n\n" + stepLine
}

func (t *PackTab) stepIndex() int {
	switch t.step {
	case PackStepIdle, PackStepScanning:
		return 0
	case PackStepReview:
		return 1
	case PackStepPassword, PackStepConfirm:
		return 2
	case PackStepPacking, PackStepResult:
		return 3
	}
	return 0
}

func (t *PackTab) renderReview(width, height int) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(RenderSectionHeader(fmt.Sprintf("  Found %d environment files", len(t.scannedFiles))) + "\n\n")
	for _, file := range t.scannedFiles {
		fmt.Fprintf(&b, "  %s  %s\n", file.RelativePath, MutedStyle.Render(utils.FormatSize(file.Size)))
	}
	b.WriteString("\n  " + MutedStyle.Render("Press Enter to continue, Esc to cancel"))

	return t.renderInViewport(width, height, b.String())
}

// renderInViewport shows content in the tab's viewport so a long file list
// can be scrolled rather than cut off.
func (t *PackTab) renderInViewport(width, height int, content string) string {
	if !t.vpReady {
		t.viewport = viewport.New(width, height)
		t.vpReady = true
	} else {
		t.viewport.Width = width
		t.viewport.Height = height
	}
	t.viewport.SetContent(content)
	return t.viewport.View()
}

func (t *PackTab) renderPasswordEntry() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(RenderSectionHeader(fmt.Sprintf("  Packing %d files", len(t.scannedFiles))) + "\n\n")
	b.WriteString("  Password: " + t.textInput.View() + "\n")
	if t.step == PackStepConfirm {
		b.WriteString("  Confirm:  " + t.confirmInput.View() + "\n")
	}

	if t.errorMsg != "" {
		b.WriteString("\n  " + ErrorStyle.Render(t.errorMsg))
	}

	return b.String()
}

func (t *PackTab) renderResult(width, height int) string {
	if t.errorMsg != "" {
		return renderEmptyState(
			"Pack Failed",
			t.errorMsg,
			"Press Enter or R to try again.",
			width, height,
		)
	}

	// Pack is all-or-nothing, so the scanned list is exactly what went in.
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(RenderSectionHeader("  "+t.resultMsg) + "\n\n")
	for _, file := range t.scannedFiles {
		fmt.Fprintf(&b, "  %s  %s\n", file.RelativePath, MutedStyle.Render(utils.FormatSize(file.Size)))
	}
	b.WriteString("\n  " + MutedStyle.Render("Press Enter or R to pack again"))

	return t.renderInViewport(width, height, b.String())
}
