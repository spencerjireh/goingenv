package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"goingenv/pkg/types"
)

// focusPackOptions drives the Pack tab from review to the options step.
func focusPackOptions(t *testing.T, m *Model) *PackTab {
	t.Helper()
	m.activeTab = TabPack
	tab := packTab(t, m)
	tab.step = PackStepReview
	send(t, m, keyMsg("enter"))
	if tab.step != PackStepOptions {
		t.Fatalf("the Pack tab is at step %d, want PackStepOptions", tab.step)
	}
	return tab
}

// The environment field owns every printable key while it has focus:
// digits must not switch tabs, q must not quit, ? must not open help.
func TestPackOptionsEnvFieldDoesNotSwitchTabs(t *testing.T) {
	m := newTestModel(t)
	tab := focusPackOptions(t, m)

	typeKeys(t, m, "123q?")

	if m.activeTab != TabPack {
		t.Errorf("typing into the env field switched to tab %d", m.activeTab)
	}
	if m.helpVisible {
		t.Error("? opened the help overlay while the env field had focus")
	}
	if got := tab.envInput.Value(); got != "123q?" {
		t.Errorf("env field holds %q, want the typed text", got)
	}
}

func TestPackOptionsRejectsInvalidEnvName(t *testing.T) {
	m := newTestModel(t)
	tab := focusPackOptions(t, m)

	typeKeys(t, m, "Prod")
	send(t, m, keyMsg("enter"))
	if tab.step != PackStepOptions {
		t.Fatalf("an invalid name advanced to step %d", tab.step)
	}
	if !strings.Contains(m.View(), "invalid environment name") {
		t.Errorf("the reason is not shown\n---\n%s\n---", m.View())
	}

	tab.envInput.SetValue("prod")
	send(t, m, keyMsg("enter"))
	if tab.step != PackStepPassword {
		t.Errorf("a valid name left the tab at step %d, want PackStepPassword", tab.step)
	}
}

func TestPackManifestDefaultsFromConfigAndToggles(t *testing.T) {
	for _, def := range []bool{false, true} {
		m := newTestModel(t)
		m.app.Config.Manifest = def
		m.tabs[TabPack] = NewPackTab(m.app, m.debugLogger)
		tab := focusPackOptions(t, m)

		if tab.manifest != def {
			t.Errorf("config %v: toggle starts at %v", def, tab.manifest)
		}
		if !strings.Contains(m.View(), "Manifest") {
			t.Errorf("the options step does not show the manifest toggle\n---\n%s\n---", m.View())
		}
		send(t, m, keyMsg(" "))
		if tab.manifest == def {
			t.Error("space did not flip the toggle")
		}
		send(t, m, keyMsg(" "))
		if tab.manifest != def {
			t.Error("a second space did not flip it back")
		}

		tab.manifest = !def
		tab.reset()
		if tab.manifest != def {
			t.Errorf("reset restored %v, want the config default %v", tab.manifest, def)
		}
	}
}

func TestPackFilesCmdUsesEnvAndManifest(t *testing.T) {
	t.Chdir(t.TempDir())
	app := testApp()
	var got types.PackOptions
	app.Archiver.(*types.MockArchiver).PackFunc = func(opts types.PackOptions) error { //nolint:errcheck // test fixture is always a mock
		got = opts
		return nil
	}
	files := []types.EnvFile{{Path: ".env", RelativePath: ".env"}}

	msg := PackFilesCmd(app, files, "pw", "prod", false)()
	if _, ok := msg.(PackCompleteMsg); !ok {
		t.Fatalf("got %T, want PackCompleteMsg", msg)
	}
	base := filepath.Base(got.OutputPath)
	if !strings.HasPrefix(base, "prod-") || !strings.HasSuffix(base, ".enc") {
		t.Errorf("output path %q, want prod-<ts>.enc", got.OutputPath)
	}
	if got.Env != "prod" {
		t.Errorf("Env = %q", got.Env)
	}

	PackFilesCmd(app, files, "pw", "", false)()
	if base := filepath.Base(got.OutputPath); !strings.HasPrefix(base, "archive-") {
		t.Errorf("unnamed output path %q, want archive-<ts>.enc", got.OutputPath)
	}

	// With the manifest on, a missing source file is reported rather than
	// silently producing an archive without one.
	msg = PackFilesCmd(app, files, "pw", "", true)()
	if errMsg, ok := msg.(ErrorMsg); !ok || errMsg.Tab != TabPack || !strings.Contains(errMsg.Text, "manifest") {
		t.Errorf("got %#v, want an ErrorMsg about the manifest", msg)
	}
}

func TestPackOptionsFlowStartsPackWithEnv(t *testing.T) {
	m := newTestModel(t)
	var got types.PackOptions
	m.app.Archiver.(*types.MockArchiver).PackFunc = func(opts types.PackOptions) error { //nolint:errcheck // test fixture is always a mock
		got = opts
		return nil
	}
	tab := focusPackOptions(t, m)

	typeKeys(t, m, "prod")
	send(t, m, keyMsg("enter"))
	typeKeys(t, m, "hunter2")
	send(t, m, keyMsg("enter"))
	typeKeys(t, m, "hunter2")
	cmd := send(t, m, keyMsg("enter"))
	if tab.step != PackStepPacking {
		t.Fatalf("the tab is at step %d, want PackStepPacking", tab.step)
	}

	// The batch also carries the spinner tick, which sleeps briefly.
	for _, msg := range runCmd(t, cmd) {
		if _, ok := msg.(PackCompleteMsg); ok {
			send(t, m, msg)
		}
	}
	if got.Env != "prod" || !strings.HasPrefix(filepath.Base(got.OutputPath), "prod-") {
		t.Errorf("the archiver received %+v, want env prod", got)
	}
	if tab.step != PackStepResult {
		t.Errorf("the tab is at step %d, want PackStepResult", tab.step)
	}
}
