package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"goingenv/pkg/types"
)

// UnpackStep represents a step in the unpack wizard.
type UnpackStep int

const (
	UnpackStepIdle      UnpackStep = iota // Empty state or select archive
	UnpackStepSelect                      // Filtered archive list
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
	archives        []string        // every .enc, refreshed when selection starts
	envInput        textinput.Model // environment filter, live while selecting
	cursor          int             // index into visibleArchives()
	textInput       textinput.Model
	spinner         spinner.Model
	viewport        viewport.Model
	vpReady         bool
	result          types.UnpackResult
	errorMsg        string
}

// NewUnpackTab creates a new UnpackTab.
func NewUnpackTab(app *types.App, debugLogger *DebugLogger) *UnpackTab {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = HighlightStyle

	return &UnpackTab{
		app:         app,
		debugLogger: debugLogger,
		step:        UnpackStepIdle,
		textInput:   newPasswordInput("Enter password..."),
		envInput:    newTextInput("filter by environment"),
		spinner:     s,
	}
}

func (t *UnpackTab) Title() string { return "Unpack" }

func (t *UnpackTab) InputFocused() bool {
	return t.step == UnpackStepSelect || t.step == UnpackStepPassword
}

func (t *UnpackTab) ShortHelp() []key.Binding {
	switch t.step {
	case UnpackStepIdle:
		return []key.Binding{WizardKeys.Start}
	case UnpackStepSelect:
		return []key.Binding{FilterListKeys.Up, FilterListKeys.Down, FilterListKeys.Select, FilterListKeys.Cancel}
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
		{FilterListKeys.Up, FilterListKeys.Down, FilterListKeys.Select},
	}
}

func (t *UnpackTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	switch t.step {
	case UnpackStepIdle:
		return t.updateIdle(msg)
	case UnpackStepSelect:
		return t.updateSelect(msg)
	case UnpackStepPassword:
		return t.updatePassword(msg)
	case UnpackStepUnpacking:
		return t.updateUnpacking(msg)
	case UnpackStepResult:
		return t.updateResult(msg)
	}

	return t, nil
}

// updateIdle moves to archive selection on Enter.
func (t *UnpackTab) updateIdle(msg tea.Msg) (Tab, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, WizardKeys.Start) {
			return t.startSelection()
		}
	}

	return t, nil
}

// updateSelect drives the filtered archive list. Arrow keys move the cursor,
// Enter picks, Esc cancels; every other key types into the environment
// filter, which is why the list does not use j/k.
func (t *UnpackTab) updateSelect(msg tea.Msg) (Tab, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, FilterListKeys.Cancel):
			t.reset()
			return t, nil
		case key.Matches(keyMsg, FilterListKeys.Up):
			if t.cursor > 0 {
				t.cursor--
			}
			return t, nil
		case key.Matches(keyMsg, FilterListKeys.Down):
			if t.cursor < len(t.visibleArchives())-1 {
				t.cursor++
			}
			return t, nil
		case key.Matches(keyMsg, FilterListKeys.Select):
			visible := t.visibleArchives()
			if len(visible) == 0 {
				return t, nil
			}
			t.selectedArchive = visible[t.cursor]
			t.step = UnpackStepPassword
			t.envInput.Blur()
			t.textInput.Reset()
			t.textInput.Focus()
			t.debugLogger.LogOperation("unpack", fmt.Sprintf("selected: %s", t.selectedArchive))
			return t, textinput.Blink
		}
	}

	var cmd tea.Cmd
	t.envInput, cmd = t.envInput.Update(msg)
	if n := len(t.visibleArchives()); t.cursor >= n {
		t.cursor = max(n-1, 0)
	}
	return t, cmd
}

// filterArchives keeps the archives whose file name starts with env + "-".
// An empty env keeps everything. Prefix matching is exact on the separator,
// so "prod" does not claim "production-*".
func filterArchives(archives []string, env string) []string {
	if env == "" {
		return archives
	}
	prefix := env + "-"
	var out []string
	for _, p := range archives {
		if strings.HasPrefix(filepath.Base(p), prefix) {
			out = append(out, p)
		}
	}
	return out
}

func (t *UnpackTab) visibleArchives() []string {
	return filterArchives(t.archives, t.envInput.Value())
}

// updatePassword collects the archive password and kicks off the unpack.
func (t *UnpackTab) updatePassword(msg tea.Msg) (Tab, tea.Cmd) {
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
			t.envInput.Focus()
			return t, textinput.Blink
		}
	}

	var cmd tea.Cmd
	t.textInput, cmd = t.textInput.Update(msg)
	return t, cmd
}

// updateUnpacking waits for the unpack to finish and reports it as a toast.
func (t *UnpackTab) updateUnpacking(msg tea.Msg) (Tab, tea.Cmd) {
	switch msg := msg.(type) {
	case UnpackCompleteMsg:
		t.result = types.UnpackResult(msg)
		t.errorMsg = ""
		t.step = UnpackStepResult
		t.vpReady = false
		t.debugLogger.LogOperation("unpack", "complete")
		toast := fmt.Sprintf("Unpacked %d files", len(t.result.Extracted))
		if n := len(t.result.Skipped); n > 0 {
			toast += fmt.Sprintf(", skipped %d", n)
		}
		return t, func() tea.Msg {
			return ToastMsg{Message: toast, IsError: false}
		}
	case ErrorMsg:
		t.errorMsg = msg.Text
		t.result = types.UnpackResult{}
		t.step = UnpackStepResult
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

// updateResult returns the tab to idle so another unpack can be started, and
// otherwise lets the user scroll the list of restored files.
func (t *UnpackTab) updateResult(msg tea.Msg) (Tab, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, WizardKeys.Reset) || key.Matches(keyMsg, WizardKeys.Start) {
			t.reset()
			return t, nil
		}
	}

	return t, scrollViewport(&t.viewport, t.vpReady, msg)
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
	t.cursor = 0
	t.envInput.Reset()
	t.envInput.Focus()
	t.vpReady = false
	return t, textinput.Blink
}

func (t *UnpackTab) reset() {
	t.step = UnpackStepIdle
	t.selectedArchive = ""
	t.result = types.UnpackResult{}
	t.errorMsg = ""
	t.textInput.Reset()
	t.textInput.Blur()
	t.envInput.Reset()
	t.envInput.Blur()
	t.cursor = 0
	t.vpReady = false
}

func (t *UnpackTab) View(width, height int) string {
	stepLine := renderStepIndicator(unpackStepNames, t.stepIndex())
	contentHeight := height - 2

	var content string
	switch t.step {
	case UnpackStepIdle:
		archives, err := t.app.Archiver.GetAvailableArchives("")
		if err != nil {
			// Rendering cannot surface an error; show the empty state instead.
			archives = nil
		}
		if len(archives) == 0 {
			content = renderEmptyState(
				"No Archives Found",
				"There is nothing to unpack yet.",
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
		content = t.renderSelect(width, contentHeight)
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

// renderSelect shows the filter field above the archive list. The list is a
// live viewport that follows the cursor, so it never truncates.
func (t *UnpackTab) renderSelect(width, height int) string {
	header := "\n" + RenderSectionHeader("  Select archive to unpack") + "\n\n" +
		"  Environment: " + t.envInput.View() + "  " + MutedStyle.Render("(blank shows all)") + "\n\n"
	listHeight := height - strings.Count(header, "\n")

	visible := t.visibleArchives()
	var b strings.Builder
	if len(visible) == 0 {
		b.WriteString("  " + MutedStyle.Render(fmt.Sprintf("No archives for environment %q", t.envInput.Value())) + "\n")
	}
	for i, p := range visible {
		marker := "  "
		if i == t.cursor {
			marker = HighlightStyle.Render("> ")
		}
		b.WriteString("  " + marker + filepath.Base(p) + "\n")
	}

	ensureViewport(&t.viewport, &t.vpReady, width, listHeight)
	t.viewport.SetContent(b.String())
	if t.cursor < t.viewport.YOffset {
		t.viewport.SetYOffset(t.cursor)
	} else if t.cursor >= t.viewport.YOffset+t.viewport.Height {
		t.viewport.SetYOffset(t.cursor - t.viewport.Height + 1)
	}
	return header + t.viewport.View()
}

func (t *UnpackTab) renderPasswordEntry() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(RenderSectionHeader("  Unpacking archive") + "\n\n")
	fmt.Fprintf(&b, "  %s\n\n", t.selectedArchive)
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

	// The remedy for skipped files and the restart hint sit under the list on
	// fixed lines: with a long list they would otherwise be below the fold.
	footer := "  " + MutedStyle.Render("Press Enter or R to unpack again")
	if len(t.result.Skipped) > 0 {
		footer = "  " + MutedStyle.Render("Existing files are never overwritten by the TUI; remove them or run goingenv unpack --overwrite") + "\n" + footer
	}

	return renderScrollable(&t.viewport, &t.vpReady, width, height, footer, t.renderFileLists)
}

// renderFileLists lists what was restored and, when anything was left alone
// because it already existed, lists that too rather than counting it as
// restored. The TUI never overwrites.
func (t *UnpackTab) renderFileLists() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(RenderSectionHeader(fmt.Sprintf("  Restored %d files to the current directory", len(t.result.Extracted))) + "\n\n")
	for _, path := range t.result.Extracted {
		fmt.Fprintf(&b, "  %s\n", path)
	}

	if len(t.result.Skipped) > 0 {
		b.WriteString("\n")
		b.WriteString(RenderSectionHeader(fmt.Sprintf("  Skipped %d existing files", len(t.result.Skipped))) + "\n\n")
		for _, path := range t.result.Skipped {
			fmt.Fprintf(&b, "  %s\n", path)
		}
	}

	return b.String()
}
