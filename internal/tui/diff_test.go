package tui

import (
	"errors"
	"os"
	"strings"
	"testing"

	"goingenv/pkg/types"
)

// stubArchiveContents makes ReadFiles return a single .env per archive name.
func stubArchiveContents(t *testing.T, m *Model, contents map[string]string) {
	t.Helper()
	archiver, ok := m.app.Archiver.(*types.MockArchiver)
	if !ok {
		t.Fatalf("archiver is a %T, want *types.MockArchiver", m.app.Archiver)
	}
	archiver.ReadFilesFunc = func(path, _ string) (*types.Archive, map[string][]byte, error) {
		for name, body := range contents {
			if strings.HasSuffix(path, name) {
				return &types.Archive{}, map[string][]byte{".env": []byte(body)}, nil
			}
		}
		return nil, nil, errors.New("unexpected archive " + path)
	}
}

// pickTwoArchives drives the Diff tab through selecting A, choosing "another
// archive", and selecting the next entry as B.
func pickTwoArchives(t *testing.T, m *Model) *DiffTab {
	t.Helper()
	tab := diffTab(t, m)

	send(t, m, keyMsg("enter")) // pick A
	if tab.step != DiffStepTarget || !strings.HasSuffix(tab.archiveA, "prod-1.enc") {
		t.Fatalf("after picking A: step %d, A %q", tab.step, tab.archiveA)
	}
	send(t, m, keyMsg("enter")) // another archive
	if tab.step != DiffStepSelectB {
		t.Fatalf("after choosing another archive: step %d", tab.step)
	}
	send(t, m, keyMsg("down"), keyMsg("enter")) // pick B
	if tab.step != DiffStepPassword || !strings.HasSuffix(tab.archiveB, "prod-2.enc") || tab.workingTree {
		t.Fatalf("after picking B: step %d, B %q, workingTree %v", tab.step, tab.archiveB, tab.workingTree)
	}
	if !tab.InputFocused() {
		t.Error("the password field did not take focus")
	}
	return tab
}

func TestDiffTab_FullFlow(t *testing.T) {
	names := []string{"prod-1.enc", "prod-2.enc"}
	pickerFixture(t, names...)
	m := newModelHere(t)
	stubArchiveContents(t, m, map[string]string{
		"prod-1.enc": "API_KEY=old-secret\nKEEP=1\n",
		"prod-2.enc": "API_KEY=new-secret\nKEEP=1\nNEW_KEY=fresh\n",
	})
	openArchivePicker(t, m, TabDiff, names)
	tab := pickTwoArchives(t, m)

	typeKeys(t, m, "hunter2")
	cmd := send(t, m, keyMsg("enter"))
	if tab.step != DiffStepDiffing || cmd == nil {
		t.Fatalf("after the password: step %d, cmd nil %v", tab.step, cmd == nil)
	}

	send(t, m, DiffFilesCmd(m.app, "hunter2", tab.archiveA, tab.archiveB)())
	if tab.step != DiffStepResult {
		t.Fatalf("the tab is at step %d, want DiffStepResult", tab.step)
	}
	m.View()
	report := viewportContent(&tab.viewport)
	for _, want := range []string{"API_KEY", "NEW_KEY", "prod-1.enc", "prod-2.enc"} {
		if !strings.Contains(report, want) {
			t.Errorf("report lacks %q\n---\n%s\n---", want, report)
		}
	}
	for _, leak := range []string{"old-secret", "new-secret", "fresh"} {
		if strings.Contains(report, leak) {
			t.Errorf("report leaks %q\n---\n%s\n---", leak, report)
		}
	}
	if strings.Contains(report, "KEEP") {
		t.Error("an unchanged key is listed")
	}
}

func TestDiffTab_WorkingTreeMode(t *testing.T) {
	pickerFixture(t, "prod-1.enc")
	m := newModelHere(t)
	if err := os.WriteFile(".env", []byte("API_KEY=on-disk\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m.app.Scanner.(*types.MockScanner).ScanFilesFunc = func(*types.ScanOptions) ([]types.EnvFile, error) { //nolint:errcheck // test fixture is always a mock
		return []types.EnvFile{{Path: ".env", RelativePath: ".env"}}, nil
	}
	m.app.Archiver.(*types.MockArchiver).ReadFilesFunc = func(string, string) (*types.Archive, map[string][]byte, error) { //nolint:errcheck // test fixture is always a mock
		return &types.Archive{}, map[string][]byte{".env": []byte("API_KEY=archived\n")}, nil
	}

	openArchivePicker(t, m, TabDiff, []string{"prod-1.enc"})
	tab := diffTab(t, m)
	send(t, m, keyMsg("enter"), keyMsg("w"))
	if tab.step != DiffStepPassword || !tab.workingTree || tab.archiveB != "" {
		t.Fatalf("after w: step %d, workingTree %v, B %q", tab.step, tab.workingTree, tab.archiveB)
	}

	typeKeys(t, m, "pw")
	send(t, m, keyMsg("enter"))
	send(t, m, DiffFilesCmd(m.app, "pw", tab.archiveA, tab.archiveB)())
	m.View()
	report := viewportContent(&tab.viewport)
	if !strings.Contains(report, "working tree") || !strings.Contains(report, "~ API_KEY") {
		t.Errorf("report:\n%s", report)
	}
	if strings.Contains(report, "on-disk") || strings.Contains(report, "archived") {
		t.Errorf("report leaks a value:\n%s", report)
	}
}

func TestDiffTab_SameArchiveForBIsRejected(t *testing.T) {
	pickerFixture(t, "prod-1.enc", "prod-2.enc")
	m := newModelHere(t)
	openArchivePicker(t, m, TabDiff, []string{"prod-1.enc", "prod-2.enc"})
	tab := diffTab(t, m)

	send(t, m, keyMsg("enter"), keyMsg("enter"), keyMsg("enter"))
	if tab.step != DiffStepSelectB {
		t.Fatalf("picking A again for B moved to step %d", tab.step)
	}
	if !strings.Contains(m.View(), "different archive") {
		t.Errorf("the rejection is not explained\n---\n%s\n---", m.View())
	}
}

func TestDiffTab_EscFromTargetReturnsToSelectA(t *testing.T) {
	pickerFixture(t, "prod-1.enc")
	m := newModelHere(t)
	openArchivePicker(t, m, TabDiff, []string{"prod-1.enc"})
	tab := diffTab(t, m)

	send(t, m, keyMsg("enter"), keyMsg("esc"))
	if tab.step != DiffStepSelectA {
		t.Errorf("esc from target left the tab at step %d", tab.step)
	}
}

func TestDiffErrorReachesTabAfterSwitch(t *testing.T) {
	m := newTestModel(t)
	m.activeTab = TabDiff
	tab := diffTab(t, m)
	tab.step = DiffStepDiffing

	send(t, m, keyMsg("1"))
	if m.activeTab != TabPack {
		t.Fatalf("the active tab is %d, want TabPack", m.activeTab)
	}

	cmd := send(t, m, ErrorMsg{Tab: TabDiff, Text: "Failed to read a.enc: wrong password"})
	if tab.step != DiffStepResult || tab.errorMsg == "" {
		t.Errorf("the Diff tab is at step %d with error %q", tab.step, tab.errorMsg)
	}
	if packTab(t, m).step != PackStepIdle {
		t.Error("the Pack tab reacted to a Diff error")
	}
	var toasted bool
	for _, msg := range runCmd(t, cmd) {
		if toast, ok := msg.(ToastMsg); ok && toast.IsError {
			toasted = true
		}
	}
	if !toasted {
		t.Error("the failure produced no error toast")
	}
}

func TestDiffFilesCmdReportsReadFailure(t *testing.T) {
	app := testApp()
	app.Archiver.(*types.MockArchiver).ReadFilesFunc = func(string, string) (*types.Archive, map[string][]byte, error) { //nolint:errcheck // test fixture is always a mock
		return nil, nil, errors.New("wrong password")
	}
	msg := DiffFilesCmd(app, "pw", ".goingenv/a.enc", ".goingenv/b.enc")()
	errMsg, ok := msg.(ErrorMsg)
	if !ok || errMsg.Tab != TabDiff || !strings.Contains(errMsg.Text, "a.enc") {
		t.Errorf("got %#v, want an ErrorMsg for TabDiff naming a.enc", msg)
	}
}
