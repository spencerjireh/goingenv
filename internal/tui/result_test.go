package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/viewport"

	"goingenv/pkg/types"
)

// The result screens list every file an operation touched. These tests use
// more files than the 30-row test terminal can show at once, so they read the
// viewport content directly and separately check that the screen scrolls.
// Nothing is truncated with an "and N more" line anywhere in the TUI.

func fakeEnvFiles(n int) []types.EnvFile {
	files := make([]types.EnvFile, 0, n)
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("svc%02d/.env.packed", i)
		files = append(files, types.EnvFile{Path: name, RelativePath: name, Size: 42, ModTime: time.Now()})
	}
	return files
}

func relativePaths(files []types.EnvFile) []string {
	names := make([]string, 0, len(files))
	for _, f := range files {
		names = append(names, f.RelativePath)
	}
	return names
}

func TestPackResultListsEveryFile(t *testing.T) {
	m := newTestModel(t)
	m.activeTab = TabPack
	tab := packTab(t, m)
	tab.scannedFiles = fakeEnvFiles(40)
	tab.step = PackStepPacking

	send(t, m, PackCompleteMsg("Packed 40 files to .goingenv/test.enc"))

	if tab.step != PackStepResult {
		t.Fatalf("the Pack tab is at step %d, want PackStepResult", tab.step)
	}

	view := m.View()
	assertListsAll(t, viewportContent(&tab.viewport), relativePaths(tab.scannedFiles))
	if !strings.Contains(view, "svc00/.env.packed") {
		t.Errorf("the first packed file is not on screen\n---\n%s\n---", view)
	}
	if strings.Contains(view, "more") {
		t.Errorf("the packed list is truncated\n---\n%s\n---", view)
	}

	before := tab.viewport.YOffset
	send(t, m, keyMsg("down"))
	if tab.viewport.YOffset <= before {
		t.Error("down did not scroll the packed file list")
	}
}

func TestUnpackResultListsExtractedAndSkipped(t *testing.T) {
	m := newTestModel(t)
	m.activeTab = TabUnpack
	tab := unpackTab(t, m)
	tab.step = UnpackStepUnpacking

	extracted := relativePaths(fakeEnvFiles(35))
	skipped := []string{"keep/.env", "keep/.env.local"}
	send(t, m, UnpackCompleteMsg(types.UnpackResult{Extracted: extracted, Skipped: skipped}))

	if tab.step != UnpackStepResult {
		t.Fatalf("the Unpack tab is at step %d, want UnpackStepResult", tab.step)
	}

	m.View()
	content := viewportContent(&tab.viewport)
	assertListsAll(t, content, extracted)
	assertListsAll(t, content, skipped)
	if !strings.Contains(content, "Skipped 2 existing files") {
		t.Errorf("the skipped section is missing\n---\n%s\n---", content)
	}
	if !strings.Contains(content, "Restored 35 files") {
		t.Errorf("the restored count is missing\n---\n%s\n---", content)
	}
	if strings.Contains(content, "more") {
		t.Errorf("the restored list is truncated\n---\n%s\n---", content)
	}

	before := tab.viewport.YOffset
	send(t, m, keyMsg("down"))
	if tab.viewport.YOffset <= before {
		t.Error("down did not scroll the restored file list")
	}
}

func TestUnpackResultWithoutSkipsHasNoSkippedSection(t *testing.T) {
	m := newTestModel(t)
	m.activeTab = TabUnpack
	tab := unpackTab(t, m)
	tab.step = UnpackStepUnpacking

	send(t, m, UnpackCompleteMsg(types.UnpackResult{Extracted: []string{".env"}}))

	view := m.View()
	if strings.Contains(view, "Skipped") {
		t.Errorf("a skipped section is shown although nothing was skipped\n---\n%s\n---", view)
	}
}

func TestFormatArchiveContentsListsEveryFile(t *testing.T) {
	files := fakeEnvFiles(25)
	archive := &types.Archive{Files: files, CreatedAt: time.Now(), Version: "1"}

	got := formatArchiveContents(archive)

	assertListsAll(t, got, relativePaths(files))
	if strings.Contains(got, "more") {
		t.Errorf("the archive listing is truncated\n---\n%s\n---", got)
	}
}

// viewportContent renders the whole viewport regardless of its height, which
// is the only way to see lines below the fold without driving the scroll.
func viewportContent(vp *viewport.Model) string {
	saved := vp.Height
	vp.Height = vp.TotalLineCount() + 1
	defer func() { vp.Height = saved }()
	return vp.View()
}
