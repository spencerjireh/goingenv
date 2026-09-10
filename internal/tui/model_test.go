package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"goingenv/pkg/types"
)

// This package had no tests at all, which meant every Bubbletea, Bubbles and
// Lipgloss upgrade was verified by nothing but the compiler. These are smoke
// tests: they drive the root model the way the runtime does and assert it
// neither panics nor renders nothing.
//
// What they deliberately do NOT do, so they survive dependency upgrades
// instead of breaking on them:
//
//   - no assertions on exact rendered strings, byte lengths or layout. Those
//     change on every Lipgloss release, which is the churn we want to absorb,
//     not detect.
//   - no golden files.
//   - no assertions that depend on ANSI escapes. CI is not a TTY, so Lipgloss
//     degrades to the Ascii profile and emits none.
//   - no filepicker coverage. It cannot be driven meaningfully headless, and
//     pretending otherwise would give false confidence -- it is exercised by
//     hand via `make tui`.

// testApp builds an App from the existing func-field mocks in pkg/types.
// Every mock returns zero values unless a Func field is set, which is all the
// render paths need.
func testApp() *types.App {
	return &types.App{
		Config: &types.Config{
			DefaultDepth: 3,
			EnvPatterns:  []string{`\.env.*`},
			MaxFileSize:  1024,
		},
		Scanner:   &types.MockScanner{},
		Archiver:  &types.MockArchiver{},
		Crypto:    &types.MockCryptor{},
		ConfigMgr: &types.MockConfigManager{},
	}
}

// newTestModel returns a model sized like a real terminal, in an empty
// working directory.
func newTestModel(t *testing.T) *Model {
	t.Helper()

	// The tab constructors call filepicker.New(), which reads the working
	// directory, and StatusTab.View calls config.IsInitialized() on it.
	t.Chdir(t.TempDir())
	return newModelHere(t)
}

// newModelHere is newTestModel without the chdir, for tests that have already
// set up a working directory the tabs need to see. The size is delivered
// through Model.Update rather than poked into the tabs, so anything relying on
// that message being forwarded is genuinely exercised.
func newModelHere(t *testing.T) *Model {
	t.Helper()

	// verbose must stay false: NewDebugLogger(true) writes a log file into the
	// real os.UserHomeDir().
	m := NewModel(testApp(), false, "v0.0.0-test")
	t.Cleanup(m.Cleanup)

	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return m
}

// xForTab finds a column that lands on the given tab, by asking the function
// under test rather than duplicating its label arithmetic here.
func xForTab(t *testing.T, tab TabID) int {
	t.Helper()
	for x := 0; x < 200; x++ {
		if tabClickIndex(x) == int(tab) {
			return x
		}
	}
	t.Fatalf("no X position maps to tab %d (%s)", tab, tabNames[tab])
	return -1
}

// TestModel_ViewRendersAllTabs guards against total layout collapse: an empty
// View, or a tab bar that stops listing its tabs.
func TestModel_ViewRendersAllTabs(t *testing.T) {
	m := newTestModel(t)

	view := m.View()
	if strings.TrimSpace(view) == "" {
		t.Fatal("View() rendered nothing")
	}

	for _, name := range tabNames {
		if !strings.Contains(view, name) {
			t.Errorf("View() does not mention tab %q\n---\n%s\n---", name, view)
		}
	}
}

// TestModel_KeyBindingsDoNotPanic feeds every key the global bindings declare.
// It takes its inputs from GlobalKeys itself, so it needs no maintenance when
// bindings change -- and it is the only guard against tea.KeyMsg.String()
// drifting (for example "esc" becoming "escape") across a Bubbletea upgrade,
// which would silently stop key.Matches from matching.
func TestModel_KeyBindingsDoNotPanic(t *testing.T) {
	bindings := []struct {
		name string
		keys []string
	}{
		{"NextTab", GlobalKeys.NextTab.Keys()},
		{"PrevTab", GlobalKeys.PrevTab.Keys()},
		{"Tab1", GlobalKeys.Tab1.Keys()},
		{"Tab2", GlobalKeys.Tab2.Keys()},
		{"Tab3", GlobalKeys.Tab3.Keys()},
		{"Tab4", GlobalKeys.Tab4.Keys()},
		{"Tab5", GlobalKeys.Tab5.Keys()},
		{"Help", GlobalKeys.Help.Keys()},
	}

	for _, b := range bindings {
		if len(b.keys) == 0 {
			t.Errorf("%s declares no keys", b.name)
			continue
		}
		for _, k := range b.keys {
			t.Run(b.name+"/"+k, func(t *testing.T) {
				m := newTestModel(t)
				m.Update(keyMsg(k))
				if m.View() == "" {
					t.Errorf("View() empty after key %q", k)
				}
			})
		}
	}
}

// TestModel_TabNavigation checks that the number keys select their tab and
// that tab/shift+tab cycle with wraparound.
func TestModel_TabNavigation(t *testing.T) {
	t.Run("number keys select", func(t *testing.T) {
		want := []TabID{TabPack, TabUnpack, TabList, TabStatus, TabSettings}
		for i, tab := range want {
			m := newTestModel(t)
			k := string(rune('1' + i))
			m.Update(keyMsg(k))
			if m.activeTab != tab {
				t.Errorf("key %q selected tab %d, want %d", k, m.activeTab, tab)
			}
		}
	})

	t.Run("tab cycles forward and wraps", func(t *testing.T) {
		m := newTestModel(t)
		start := m.activeTab
		for i := 0; i < tabCount; i++ {
			m.Update(keyMsg("tab"))
		}
		if m.activeTab != start {
			t.Errorf("after %d forward cycles activeTab=%d, want %d (wraparound)",
				tabCount, m.activeTab, start)
		}
	})

	t.Run("shift+tab cycles backward and wraps", func(t *testing.T) {
		m := newTestModel(t)
		start := m.activeTab
		m.Update(keyMsg("shift+tab"))
		if m.activeTab == start {
			t.Fatal("shift+tab did not change the active tab")
		}
		for i := 1; i < tabCount; i++ {
			m.Update(keyMsg("shift+tab"))
		}
		if m.activeTab != start {
			t.Errorf("after %d backward cycles activeTab=%d, want %d (wraparound)",
				tabCount, m.activeTab, start)
		}
	})
}

// TestModel_MouseTabClick pins the tab-bar click behaviour.
//
// The motion case is the important one. The program runs with
// tea.WithMouseCellMotion, so this handler also receives motion events
// carrying whatever button is held. In Bubbletea 0.24 MouseEventType was a
// flat enum, so a single `Type == MouseLeft` test excluded motion implicitly.
// v1 splits that into orthogonal Action and Button fields, where a drag with
// the left button held satisfies Button == MouseButtonLeft -- so a migration
// that checks only the button would silently start re-selecting a tab on every
// cell crossed during a drag. This test is what makes that visible.
func TestModel_MouseTabClick(t *testing.T) {
	t.Run("left press on the tab bar selects", func(t *testing.T) {
		m := newTestModel(t)
		m.activeTab = TabPack

		m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
			X: xForTab(t, TabList), Y: headerHeight})

		if m.activeTab != TabList {
			t.Errorf("click selected tab %d, want %d", m.activeTab, TabList)
		}
	})

	t.Run("motion over the tab bar does not select", func(t *testing.T) {
		m := newTestModel(t)
		m.activeTab = TabPack

		m.Update(tea.MouseMsg{Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft,
			X: xForTab(t, TabList), Y: headerHeight})

		if m.activeTab != TabPack {
			t.Errorf("motion selected tab %d; dragging across the tab bar must not switch tabs", m.activeTab)
		}
	})

	t.Run("press off the tab bar row does not select", func(t *testing.T) {
		m := newTestModel(t)
		m.activeTab = TabPack

		m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
			X: xForTab(t, TabList), Y: headerHeight + 5})

		if m.activeTab != TabPack {
			t.Errorf("a click off the tab bar selected tab %d", m.activeTab)
		}
	})
}

// TestModel_ViewAtExtremeSizes exercises the width-N arithmetic in the layout
// helpers. Lipgloss has changed how it treats zero and negative widths across
// releases, and several call sites compute `width - 4`.
func TestModel_ViewAtExtremeSizes(t *testing.T) {
	sizes := []struct{ w, h int }{
		{0, 0},
		{1, 1},
		{40, 10},
		{80, 24},
		{200, 60},
	}

	for _, size := range sizes {
		m := newTestModel(t)
		m.Update(tea.WindowSizeMsg{Width: size.w, Height: size.h})

		// Each tab renders differently, so visit all of them at every size.
		for tab := TabID(0); tab < tabCount; tab++ {
			m.activeTab = tab
			// A panic here fails the test; that is the assertion.
			_ = m.View()
		}
	}
}

// keyMsg builds the KeyMsg the runtime would deliver for a key name, using the
// same names the bindings in keys.go declare.
func keyMsg(k string) tea.KeyMsg {
	switch k {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
}
