package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"goingenv/pkg/types"
)

// The archive picker on the List and Diff tabs is a bubbles filepicker that
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

func diffHolder(t *DiffTab) *filepickerHolder {
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

func TestDiffTab_FilePickerListsEveryArchive(t *testing.T) {
	names := []string{"alpha.enc", "bravo.enc", "charlie.enc"}
	pickerFixture(t, names...)

	m := newModelHere(t)

	tab := diffTab(t, m)
	tab.step = DiffStepSelectA
	tab.fpInitialized = true
	readDir(diffHolder(tab))

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

// The tests above cover what the picker lists. The ones below cover the rest of
// checklist items 5 and 6: where it opens, and whether choosing a file does
// anything. A picker that lists correctly but selects nothing looks identical
// from the outside until someone presses enter.

// openArchivePicker drives a tab from its idle step into archive selection the
// way a user does, through the model, and runs the directory read the picker
// asks for.
//
// The archiver mock has to report archives first: startSelection refuses to
// open the picker when GetAvailableArchives comes back empty, so without this
// the enter key silently does nothing and the test would pass against a broken
// picker.
func openArchivePicker(t *testing.T, m *Model, tab TabID, archives []string) {
	t.Helper()

	archiver, ok := m.app.Archiver.(*types.MockArchiver)
	if !ok {
		t.Fatalf("archiver is a %T, want *types.MockArchiver", m.app.Archiver)
	}
	archiver.GetAvailableArchivesFunc = func(string) ([]string, error) {
		return archives, nil
	}

	m.activeTab = tab

	// Enter moves idle -> select and returns the picker's directory read.
	cmd := send(t, m, keyMsg("enter"))
	for _, msg := range runCmd(t, cmd) {
		send(t, m, msg)
	}
}

func TestDiffTab_PickerOpensInTheArchiveDirectory(t *testing.T) {
	names := []string{"alpha.enc", "bravo.enc"}
	pickerFixture(t, names...)

	m := newModelHere(t)
	openArchivePicker(t, m, TabDiff, names)

	tab := diffTab(t, m)
	if tab.step != DiffStepSelectA {
		t.Fatalf("the tab is at step %d, want DiffStepSelectA", tab.step)
	}
	if got := tab.filepicker.CurrentDirectory; got != ".goingenv" {
		t.Errorf("the picker opened in %q, want %q", got, ".goingenv")
	}
	assertListsAll(t, tab.View(100, 30), names)
}

func TestListTab_PickerOpensInTheArchiveDirectory(t *testing.T) {
	names := []string{"alpha.enc", "bravo.enc"}
	pickerFixture(t, names...)

	m := newModelHere(t)
	openArchivePicker(t, m, TabList, names)

	tab := listTab(t, m)
	if tab.step != ListStepSelect {
		t.Fatalf("the tab is at step %d, want ListStepSelect", tab.step)
	}
	if got := tab.filepicker.CurrentDirectory; got != ".goingenv" {
		t.Errorf("the picker opened in %q, want %q", got, ".goingenv")
	}
	assertListsAll(t, tab.View(100, 30), names)
}

func TestListTab_SelectingAnArchiveAdvancesToPassword(t *testing.T) {
	names := []string{"alpha.enc", "bravo.enc", "charlie.enc"}
	pickerFixture(t, names...)

	m := newModelHere(t)
	openArchivePicker(t, m, TabList, names)

	send(t, m, keyMsg("down"), keyMsg("enter"))

	tab := listTab(t, m)
	if tab.step != ListStepPassword {
		t.Fatalf("selecting an archive left the tab at step %d, want ListStepPassword", tab.step)
	}
	if tab.selectedArchive == "" {
		t.Error("the tab advanced without recording which archive was chosen")
	}
	if !tab.InputFocused() {
		t.Error("the password field did not take focus after selection")
	}
}
