package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// The archive picker on the Unpack and List tabs is a bubbles filepicker that
// sizes itself from a tea.WindowSizeMsg (AutoHeight). The root model used to
// consume window-size messages without forwarding them to tabs, so the picker's
// height stayed 0 and it listed at most one archive however many existed -- on
// bubbles 0.16.1 it listed none at all.
//
// These tests drive the real Model, not a bare tab, so the assertion covers the
// routing as well as the picker.

// pickerFixture creates a .goingenv directory holding the named archives and
// chdirs into its parent, which is where both tabs look.
func pickerFixture(t *testing.T, names ...string) {
	t.Helper()

	dir := t.TempDir()
	t.Chdir(dir)

	ge := filepath.Join(dir, ".goingenv")
	if err := os.MkdirAll(ge, 0o750); err != nil {
		t.Fatalf("failed to create .goingenv: %v", err)
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(ge, n), []byte("archive"), 0o600); err != nil {
			t.Fatalf("failed to write %s: %v", n, err)
		}
	}
}

// readDir runs the command filepicker.Init returns, which is what populates
// the picker from disk.
func readDir(fp *filepickerHolder) {
	if cmd := fp.init(); cmd != nil {
		fp.update(cmd())
	}
}

// filepickerHolder abstracts over the two tabs, which hold their picker in an
// unexported field of the same type.
type filepickerHolder struct {
	init   func() tea.Cmd
	update func(tea.Msg)
}

func unpackHolder(t *UnpackTab) *filepickerHolder {
	return &filepickerHolder{
		init:   func() tea.Cmd { return t.filepicker.Init() },
		update: func(m tea.Msg) { t.filepicker, _ = t.filepicker.Update(m) },
	}
}

func listHolder(t *ListTab) *filepickerHolder {
	return &filepickerHolder{
		init:   func() tea.Cmd { return t.filepicker.Init() },
		update: func(m tea.Msg) { t.filepicker, _ = t.filepicker.Update(m) },
	}
}

func assertListsAll(t *testing.T, view string, names []string) {
	t.Helper()
	for _, n := range names {
		if !strings.Contains(view, n) {
			t.Errorf("archive %q is not listed by the picker\n---\n%s\n---", n, view)
		}
	}
}

func TestUnpackTab_FilePickerListsEveryArchive(t *testing.T) {
	names := []string{"alpha.enc", "bravo.enc", "charlie.enc"}
	pickerFixture(t, names...)

	// newModelHere sends the WindowSizeMsg through Model.Update, which is the
	// path that has to forward it to the tabs.
	m := newModelHere(t)

	tab, ok := m.tabs[TabUnpack].(*UnpackTab)
	if !ok {
		t.Fatalf("tab %d is a %T, want *UnpackTab", TabUnpack, m.tabs[TabUnpack])
	}
	tab.step = UnpackStepSelect
	tab.fpInitialized = true
	readDir(unpackHolder(tab))

	assertListsAll(t, tab.View(100, 30), names)
}

func TestListTab_FilePickerListsEveryArchive(t *testing.T) {
	names := []string{"alpha.enc", "bravo.enc", "charlie.enc"}
	pickerFixture(t, names...)

	m := newModelHere(t)

	tab, ok := m.tabs[TabList].(*ListTab)
	if !ok {
		t.Fatalf("tab %d is a %T, want *ListTab", TabList, m.tabs[TabList])
	}
	tab.step = ListStepSelect
	tab.fpInitialized = true
	readDir(listHolder(tab))

	assertListsAll(t, tab.View(100, 30), names)
}
