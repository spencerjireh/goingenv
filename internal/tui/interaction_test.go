package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// These tests cover the parts of the manual TUI checklist in
// docs/development.md that are reachable without a terminal: the mouse-drag
// guard, wheel handling, the help overlay, the confirmation modal, toasts and
// quit.
//
// They follow the rules model_test.go sets out -- plain substrings, no golden
// files, no layout arithmetic -- so they survive a lipgloss or bubbletea
// upgrade instead of breaking on one.

// send applies messages in order and returns the command from the last one.
// Model.Update mutates through a pointer receiver, so the returned tea.Model is
// the same m and can be ignored.
func send(t *testing.T, m *Model, msgs ...tea.Msg) tea.Cmd {
	t.Helper()

	var cmd tea.Cmd
	for _, msg := range msgs {
		_, cmd = m.Update(msg)
	}
	return cmd
}

// runCmd executes a command and returns every message it produced, flattening
// one level of tea.Batch. Commands are the only way to observe intent that
// leaves no mark on the model -- tea.Quit is a function value and cannot be
// compared, but the tea.QuitMsg it returns can.
//
// It deliberately does not recurse into timer commands: spinner.Tick and the
// tea.Tick behind a toast block for their full duration when invoked.
func runCmd(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()

	if cmd == nil {
		return nil
	}

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}

	var out []tea.Msg
	for _, c := range batch {
		if c != nil {
			out = append(out, c())
		}
	}
	return out
}

// containsMsg reports whether msgs holds a message of the same type as want.
func containsMsg(msgs []tea.Msg, want tea.Msg) bool {
	for _, m := range msgs {
		if m == want {
			return true
		}
	}
	return false
}

// TestModel_DragAcrossTabBarKeepsTheSelection is the checklist item that only a
// human could confirm until now: press the left button on one tab, then drag
// the pointer the whole width of the tab bar with the button still held. The
// active tab must not follow the pointer.
//
// TestModel_MouseTabClick already covers a single motion event. This covers the
// real gesture, which is a press followed by a run of them -- the shape that
// would make a broken guard switch tabs several times on the way across.
func TestModel_DragAcrossTabBarKeepsTheSelection(t *testing.T) {
	m := newTestModel(t)

	start := xForTab(t, TabPack)
	end := xForTab(t, TabSettings)

	send(t, m, tea.MouseMsg{
		Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
		X: start, Y: headerHeight,
	})
	if m.activeTab != TabPack {
		t.Fatalf("the initial press selected tab %d, want %d", m.activeTab, TabPack)
	}

	for x := start; x <= end; x++ {
		send(t, m, tea.MouseMsg{
			Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft,
			X: x, Y: headerHeight,
		})
		if m.activeTab != TabPack {
			t.Fatalf("dragging over column %d switched to tab %d; a held-button drag must not select",
				x, m.activeTab)
		}
	}

	// Releasing over a different tab must not select it either. Only a press
	// does.
	send(t, m, tea.MouseMsg{
		Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft,
		X: end, Y: headerHeight,
	})
	if m.activeTab != TabPack {
		t.Errorf("releasing over another tab selected %d, want %d", m.activeTab, TabPack)
	}
}

// TestModel_WheelDoesNotSelectTabs covers the wheel half of the same guard.
//
// Wheel events arrive as presses carrying a wheel button, so the button test is
// what excludes them. Note that the wheel currently does nothing anywhere in
// the TUI: handleMouseMsg never forwards to the tabs, so no viewport scrolls
// with it. If that is ever wired up, this test should fail and be rewritten
// rather than deleted -- it is here so the change is deliberate.
func TestModel_WheelDoesNotSelectTabs(t *testing.T) {
	buttons := []tea.MouseButton{tea.MouseButtonWheelUp, tea.MouseButtonWheelDown}

	for _, button := range buttons {
		m := newTestModel(t)
		m.activeTab = TabPack

		send(t, m, tea.MouseMsg{
			Action: tea.MouseActionPress, Button: button,
			X: xForTab(t, TabList), Y: headerHeight,
		})

		if m.activeTab != TabPack {
			t.Errorf("a wheel event selected tab %d; the wheel must not switch tabs", m.activeTab)
		}
	}
}

// TestModel_HelpOverlay covers `?`: it opens, any key closes it, and it is
// suppressed while a text input has focus so a password can contain '?'.
func TestModel_HelpOverlay(t *testing.T) {
	const heading = "Keyboard Shortcuts"

	t.Run("toggles open and closed", func(t *testing.T) {
		m := newTestModel(t)

		if strings.Contains(m.View(), heading) {
			t.Fatal("the help overlay is visible before ? was pressed")
		}

		send(t, m, keyMsg("?"))
		if !strings.Contains(m.View(), heading) {
			t.Errorf("? did not open the help overlay\n---\n%s\n---", m.View())
		}

		send(t, m, keyMsg("?"))
		if strings.Contains(m.View(), heading) {
			t.Error("a second ? did not close the help overlay")
		}
	})

	t.Run("any key closes it", func(t *testing.T) {
		for _, k := range []string{"1", "j", "enter", "esc"} {
			m := newTestModel(t)
			send(t, m, keyMsg("?"), keyMsg(k))

			if strings.Contains(m.View(), heading) {
				t.Errorf("the help overlay survived key %q", k)
			}
		}
	})

	t.Run("suppressed while an input is focused", func(t *testing.T) {
		m := newTestModel(t)
		focusPackPassword(t, m)

		send(t, m, keyMsg("?"))

		if strings.Contains(m.View(), heading) {
			t.Error("? opened the help overlay while the password field had focus")
		}
	})
}

// TestModel_ConfirmModal covers y/n/esc and the modal's priority over every
// other key.
//
// Nothing in internal/tui emits ShowModalMsg today, so the message is injected
// here the way a caller would send it. The modal machinery is reachable and
// correct; what is missing is a caller. See docs/development.md.
func TestModel_ConfirmModal(t *testing.T) {
	// A sentinel the confirm path must hand back untouched.
	type confirmedMsg struct{}
	confirm := func() tea.Msg { return confirmedMsg{} }

	show := ShowModalMsg{
		Title:     "Overwrite files?",
		Body:      "Existing files will be replaced.",
		OnConfirm: confirm,
	}

	t.Run("renders over the page", func(t *testing.T) {
		m := newTestModel(t)
		send(t, m, show)

		view := m.View()
		if !strings.Contains(view, "Overwrite files?") {
			t.Errorf("the modal title is not rendered\n---\n%s\n---", view)
		}
		if !strings.Contains(view, "[y] confirm") {
			t.Errorf("the modal does not show its key hints\n---\n%s\n---", view)
		}
	})

	t.Run("y runs the confirm command", func(t *testing.T) {
		m := newTestModel(t)
		send(t, m, show)

		cmd := send(t, m, keyMsg("y"))
		if m.modal != nil {
			t.Error("y left the modal open")
		}
		if !containsMsg(runCmd(t, cmd), confirmedMsg{}) {
			t.Error("y did not run the OnConfirm command")
		}
	})

	for _, k := range []string{"n", "esc"} {
		t.Run(k+" dismisses without confirming", func(t *testing.T) {
			m := newTestModel(t)
			send(t, m, show)

			cmd := send(t, m, keyMsg(k))
			if m.modal != nil {
				t.Errorf("%q left the modal open", k)
			}
			if containsMsg(runCmd(t, cmd), confirmedMsg{}) {
				t.Errorf("%q ran OnConfirm; only y may confirm", k)
			}
		})
	}

	t.Run("swallows keys meant for the page", func(t *testing.T) {
		m := newTestModel(t)
		m.activeTab = TabPack
		send(t, m, show)

		send(t, m, keyMsg("3"), keyMsg("tab"))

		if m.activeTab != TabPack {
			t.Errorf("a key reached the tab bar through the modal: activeTab=%d", m.activeTab)
		}
		if m.modal == nil {
			t.Error("an unrelated key dismissed the modal")
		}
	})
}

// TestModel_Toast covers the appear-and-expire cycle.
//
// The expiry message is synthesised from the toast's own ID rather than by
// running the command Update returns: that command is a three-second tea.Tick
// and invoking it would block the test for its full duration. The ID is read
// back from the model because toastCounter is a package-level variable, so IDs
// carry over between tests in this package.
func TestModel_Toast(t *testing.T) {
	cases := []struct {
		name    string
		isError bool
		prefix  string
	}{
		{"success", false, "[+] "},
		{"error", true, "[x] "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel(t)

			send(t, m, ToastMsg{Message: "archive written", IsError: tc.isError})

			if len(m.toasts) != 1 {
				t.Fatalf("the model holds %d toasts, want 1", len(m.toasts))
			}
			if view := m.View(); !strings.Contains(view, tc.prefix+"archive written") {
				t.Errorf("the toast is not rendered\n---\n%s\n---", view)
			}

			send(t, m, ToastExpiredMsg{ID: m.toasts[0].ID})

			if len(m.toasts) != 0 {
				t.Fatalf("the toast survived its expiry message: %+v", m.toasts)
			}
			if strings.Contains(m.View(), "archive written") {
				t.Error("the toast is still rendered after expiring")
			}
		})
	}

	t.Run("an unknown id removes nothing", func(t *testing.T) {
		m := newTestModel(t)
		send(t, m, ToastMsg{Message: "archive written"})

		send(t, m, ToastExpiredMsg{ID: m.toasts[0].ID + 1000})

		if len(m.toasts) != 1 {
			t.Errorf("a stale expiry removed the wrong toast: %+v", m.toasts)
		}
	})
}

// TestModel_Quit covers the half of checklist item 12 that does not need a
// terminal. Whether the alt-screen is restored is terminal state, not model
// state, and stays a manual check.
//
// `q` is worth pinning because it is a bare string comparison in handleKeyMsg
// rather than a key.Binding, so the binding-driven tests never reach it.
func TestModel_Quit(t *testing.T) {
	for _, k := range []string{"q", "ctrl+c"} {
		t.Run(k+" quits", func(t *testing.T) {
			m := newTestModel(t)

			cmd := send(t, m, keyMsg(k))
			if !containsMsg(runCmd(t, cmd), tea.QuitMsg{}) {
				t.Errorf("%q did not quit", k)
			}
		})
	}

	t.Run("q types into a focused password field", func(t *testing.T) {
		m := newTestModel(t)
		focusPackPassword(t, m)

		cmd := send(t, m, keyMsg("q"))
		if containsMsg(runCmd(t, cmd), tea.QuitMsg{}) {
			t.Fatal("q quit while the password field had focus")
		}

		tab := packTab(t, m)
		if got := tab.textInput.Value(); got != "q" {
			t.Errorf("the password field holds %q, want %q", got, "q")
		}
	})

	t.Run("ctrl+c quits even from a focused field", func(t *testing.T) {
		m := newTestModel(t)
		focusPackPassword(t, m)

		cmd := send(t, m, keyMsg("ctrl+c"))
		if !containsMsg(runCmd(t, cmd), tea.QuitMsg{}) {
			t.Error("ctrl+c did not quit from the password step")
		}
	})
}

// packTab returns the Pack tab, failing the test if it is not the type it
// should be.
func packTab(t *testing.T, m *Model) *PackTab {
	t.Helper()

	tab, ok := m.tabs[TabPack].(*PackTab)
	if !ok {
		t.Fatalf("tab %d is a %T, want *PackTab", TabPack, m.tabs[TabPack])
	}
	return tab
}

// focusPackPassword drives the Pack tab to its password step, which is the
// state in which InputFocused() is true and the global keys are suppressed.
//
// The scan is short-circuited with a ScanCompleteMsg rather than run, so the
// helper does not depend on what the scanner mock returns.
func focusPackPassword(t *testing.T, m *Model) {
	t.Helper()

	m.activeTab = TabPack
	tab := packTab(t, m)
	tab.step = PackStepReview

	send(t, m, keyMsg("enter"))

	if tab.step != PackStepPassword {
		t.Fatalf("the Pack tab is at step %d, want PackStepPassword", tab.step)
	}
	if !m.tabs[TabPack].InputFocused() {
		t.Fatal("the Pack tab does not report its input as focused")
	}
}
