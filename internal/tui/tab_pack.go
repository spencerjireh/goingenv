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

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = HighlightStyle

	return &PackTab{
		app:         app,
		debugLogger: debugLogger,
		step:        PackStepIdle,
		textInput:   ti,
		spinner:     s,
	}
}

func (t *PackTab) Title() string { return "Pack" }

func (t *PackTab) InputFocused() bool {
	return t.step == PackStepPassword
}

func (t *PackTab) ShortHelp() []key.Binding {
	switch t.step {
	case PackStepIdle:
		return []key.Binding{WizardKeys.Start}
	case PackStepScanning, PackStepPacking:
		return nil
	case PackStepReview:
		return []key.Binding{WizardKeys.Confirm, WizardKeys.Cancel}
	case PackStepPassword:
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
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(keyMsg, WizardKeys.Start) {
				if !config.IsInitialized() {
					// Initialize first
					t.step = PackStepScanning
					t.debugLogger.LogOperation("pack", "initializing project")
					return t, tea.Batch(t.spinner.Tick, InitProjectCmd())
				}
				t.step = PackStepScanning
				t.debugLogger.LogOperation("pack", "scanning files")
				return t, tea.Batch(t.spinner.Tick, ScanFilesCmd(t.app))
			}
		}

	case PackStepScanning:
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

	case PackStepReview:
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

	case PackStepPassword:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch {
			case key.Matches(keyMsg, WizardKeys.Confirm):
				password := t.textInput.Value()
				if password == "" {
					t.errorMsg = "Password cannot be empty"
					return t, nil
				}
				t.errorMsg = ""
				t.step = PackStepPacking
				t.debugLogger.LogOperation("pack", "packing files")
				return t, tea.Batch(t.spinner.Tick, PackFilesCmd(t.app, t.scannedFiles, password))
			case key.Matches(keyMsg, WizardKeys.Cancel):
				t.step = PackStepReview
				t.textInput.Blur()
				return t, nil
			}
		}
		var cmd tea.Cmd
		t.textInput, cmd = t.textInput.Update(msg)
		return t, cmd

	case PackStepPacking:
		switch msg := msg.(type) {
		case PackCompleteMsg:
			t.resultMsg = string(msg)
			t.errorMsg = ""
			t.step = PackStepResult
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

	case PackStepResult:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(keyMsg, WizardKeys.Reset) || key.Matches(keyMsg, WizardKeys.Start) {
				t.reset()
				return t, nil
			}
		}
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
				"Scan your project for .env files and encrypt them into a secure archive.",
				"Press Enter to scan for environment files.",
				width, contentHeight,
			)
		}
	case PackStepScanning:
		content = fmt.Sprintf("\n  %s Scanning for environment files...", t.spinner.View())
	case PackStepReview:
		content = t.renderReview(width, contentHeight)
	case PackStepPassword:
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
	case PackStepPassword:
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
		b.WriteString(fmt.Sprintf("  %s  %s\n", file.RelativePath, MutedStyle.Render(utils.FormatSize(file.Size))))
	}
	b.WriteString("\n  " + MutedStyle.Render("Press Enter to continue, Esc to cancel"))

	if !t.vpReady {
		t.viewport = viewport.New(width, height)
		t.vpReady = true
	} else {
		t.viewport.Width = width
		t.viewport.Height = height
	}
	t.viewport.SetContent(b.String())
	return t.viewport.View()
}

func (t *PackTab) renderPasswordEntry() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(RenderSectionHeader(fmt.Sprintf("  Packing %d files", len(t.scannedFiles))) + "\n\n")
	b.WriteString("  Password: " + t.textInput.View() + "\n")

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
	return renderEmptyState(
		"Pack Successful",
		t.resultMsg,
		"Press Enter or R to pack again.",
		width, height,
	)
}
