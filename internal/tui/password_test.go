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
}

// TestEveryPasswordInputUsesEchoPassword covers the two tabs that cannot easily
// be driven to their password step without an archive on disk, so a future
// refactor cannot unmask one of them unnoticed.
func TestEveryPasswordInputUsesEchoPassword(t *testing.T) {
	m := newTestModel(t)

	inputs := map[string]textinput.Model{
		"Pack":   packTab(t, m).textInput,
		"Unpack": unpackTab(t, m).textInput,
		"List":   listTab(t, m).textInput,
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
