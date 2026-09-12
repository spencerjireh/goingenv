package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

// Checklist item 7: the password fields must not echo what is typed.
//
// Three tabs each own an independently constructed textinput, so masking is
// asserted on all three. Cursor blink is not asserted: it is a timer the model
// only forwards, and pinning it would test bubbles rather than this code.

// TestPasswordFieldsAreMasked types a password into the Pack tab and checks it
// does not appear in the rendered view.
//
// This is the assertion that matters. Reading EchoMode back proves the field
// was configured; rendering proves nothing downstream undoes it.
func TestPasswordFieldsAreMasked(t *testing.T) {
	const secret = "correct horse battery staple"

	m := newTestModel(t)
	focusPackPassword(t, m)

	for _, r := range secret {
		send(t, m, keyMsg(string(r)))
	}

	tab := packTab(t, m)
	if got := tab.textInput.Value(); got != secret {
		t.Fatalf("the field holds %q, want %q -- the typing did not land", got, secret)
	}

	view := m.View()
	if strings.Contains(view, secret) {
		t.Errorf("the password is echoed in the view\n---\n%s\n---", view)
	}
	// A short password could coincidentally appear; a distinctive word from
	// the middle of this one could not.
	if strings.Contains(view, "battery") {
		t.Errorf("part of the password is echoed in the view\n---\n%s\n---", view)
	}
	if !strings.Contains(view, "Password:") {
		t.Errorf("the password prompt is missing entirely\n---\n%s\n---", view)
	}

	// The confirmation field is a second, independently constructed input.
	send(t, m, keyMsg("enter"))
	for _, r := range secret {
		send(t, m, keyMsg(string(r)))
	}

	if got := tab.confirmInput.Value(); got != secret {
		t.Fatalf("the confirm field holds %q, want %q -- the typing did not land", got, secret)
	}

	view = m.View()
	if strings.Contains(view, "battery") {
		t.Errorf("the password is echoed in the confirm view\n---\n%s\n---", view)
	}
	if !strings.Contains(view, "Confirm:") {
		t.Errorf("the confirm prompt is missing entirely\n---\n%s\n---", view)
	}
}

// TestEveryPasswordInputUsesEchoPassword covers the two tabs that cannot easily
// be driven to their password step without an archive on disk, so a future
// refactor cannot unmask one of them unnoticed.
func TestEveryPasswordInputUsesEchoPassword(t *testing.T) {
	m := newTestModel(t)

	inputs := map[string]textinput.Model{
		"Pack":         packTab(t, m).textInput,
		"Pack confirm": packTab(t, m).confirmInput,
		"Unpack":       unpackTab(t, m).textInput,
		"List":         listTab(t, m).textInput,
	}

	for name, ti := range inputs {
		if ti.EchoMode != textinput.EchoPassword {
			t.Errorf("the %s password field uses EchoMode %v, want EchoPassword", name, ti.EchoMode)
		}
	}
}

// TestEmptyPasswordIsRejected pins the guard that stops an empty password
// starting a pack, which would otherwise encrypt with no secret at all.
func TestEmptyPasswordIsRejected(t *testing.T) {
	m := newTestModel(t)
	focusPackPassword(t, m)

	send(t, m, keyMsg("enter"))

	tab := packTab(t, m)
	if tab.step != PackStepPassword {
		t.Errorf("an empty password advanced to step %d; it must not", tab.step)
	}
	if !strings.Contains(m.View(), "Password cannot be empty") {
		t.Errorf("no reason was shown for refusing the empty password\n---\n%s\n---", m.View())
	}
}

// TestMatchingConfirmationStartsPack drives the two entries through to the
// pack kickoff. send does not run the returned command, so nothing is packed.
func TestMatchingConfirmationStartsPack(t *testing.T) {
	m := newTestModel(t)
	focusPackPassword(t, m)
	tab := packTab(t, m)

	typeKeys(t, m, "hunter2")
	send(t, m, keyMsg("enter"))

	if tab.step != PackStepConfirm {
		t.Fatalf("after the first entry the tab is at step %d, want PackStepConfirm", tab.step)
	}
	if !tab.InputFocused() {
		t.Error("InputFocused() is false on the confirm step; global keys would steal the typing")
	}

	typeKeys(t, m, "hunter2")
	cmd := send(t, m, keyMsg("enter"))

	if tab.step != PackStepPacking {
		t.Errorf("after a matching entry the tab is at step %d, want PackStepPacking", tab.step)
	}
	if cmd == nil {
		t.Error("no command was returned; the pack was never started")
	}
	if tab.password != "" || tab.textInput.Value() != "" || tab.confirmInput.Value() != "" {
		t.Error("the password is still held by the tab after the pack started")
	}
}

// TestMismatchedConfirmationReturnsToPassword pins the whole point of the
// second entry: a typo must not reach the archiver.
func TestMismatchedConfirmationReturnsToPassword(t *testing.T) {
	m := newTestModel(t)
	focusPackPassword(t, m)
	tab := packTab(t, m)

	typeKeys(t, m, "abc")
	send(t, m, keyMsg("enter"))
	typeKeys(t, m, "abd")
	send(t, m, keyMsg("enter"))

	if tab.step != PackStepPassword {
		t.Errorf("a mismatch left the tab at step %d, want PackStepPassword", tab.step)
	}
	if !strings.Contains(m.View(), "Passwords do not match") {
		t.Errorf("no reason was shown for refusing the mismatch\n---\n%s\n---", m.View())
	}
	if got := tab.textInput.Value(); got != "" {
		t.Errorf("the first field still holds %q; a mismatch must start over", got)
	}
	if got := tab.confirmInput.Value(); got != "" {
		t.Errorf("the confirm field still holds %q; a mismatch must start over", got)
	}
	if !tab.textInput.Focused() {
		t.Error("the first field is not focused after a mismatch")
	}
}

// TestEscOnConfirmReturnsToPassword checks that backing out of the confirm
// step keeps the first entry, so a user who noticed a typo can edit it.
func TestEscOnConfirmReturnsToPassword(t *testing.T) {
	m := newTestModel(t)
	focusPackPassword(t, m)
	tab := packTab(t, m)

	typeKeys(t, m, "abc")
	send(t, m, keyMsg("enter"))
	send(t, m, keyMsg("esc"))

	if tab.step != PackStepPassword {
		t.Errorf("esc left the tab at step %d, want PackStepPassword", tab.step)
	}
	if got := tab.textInput.Value(); got != "abc" {
		t.Errorf("the first field holds %q after esc, want the original entry", got)
	}
	if !tab.textInput.Focused() {
		t.Error("the first field is not focused after esc")
	}
}

func typeKeys(t *testing.T, m *Model, text string) {
	t.Helper()
	for _, r := range text {
		send(t, m, keyMsg(string(r)))
	}
}

func unpackTab(t *testing.T, m *Model) *UnpackTab {
	t.Helper()

	tab, ok := m.tabs[TabUnpack].(*UnpackTab)
	if !ok {
		t.Fatalf("tab %d is a %T, want *UnpackTab", TabUnpack, m.tabs[TabUnpack])
	}
	return tab
}

func listTab(t *testing.T, m *Model) *ListTab {
	t.Helper()

	tab, ok := m.tabs[TabList].(*ListTab)
	if !ok {
		t.Fatalf("tab %d is a %T, want *ListTab", TabList, m.tabs[TabList])
	}
	return tab
}
