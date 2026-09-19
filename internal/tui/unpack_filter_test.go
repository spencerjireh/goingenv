package tui

import (
	"reflect"
	"strings"
	"testing"
)

func TestFilterArchives(t *testing.T) {
	archives := []string{".goingenv/prod-1.enc", ".goingenv/production-1.enc", ".goingenv/staging-1.enc", ".goingenv/archive-1.enc"}
	cases := []struct {
		env  string
		want []string
	}{
		{"", archives},
		{"prod", []string{".goingenv/prod-1.enc"}},
		{"pro", nil},
		{"staging", []string{".goingenv/staging-1.enc"}},
		{"zzz", nil},
	}
	for _, tc := range cases {
		if got := filterArchives(archives, tc.env); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("filterArchives(%q) = %v, want %v", tc.env, got, tc.want)
		}
	}
}

func TestUnpackTab_ListsEveryArchive(t *testing.T) {
	names := []string{"alpha-1.enc", "bravo-1.enc", "charlie-1.enc"}
	m := newTestModel(t)
	openArchivePicker(t, m, TabUnpack, names)

	tab := unpackTab(t, m)
	if tab.step != UnpackStepSelect {
		t.Fatalf("the tab is at step %d, want UnpackStepSelect", tab.step)
	}
	if !tab.InputFocused() {
		t.Error("the filter field must own the keyboard while selecting")
	}
	m.View() // the list is a live viewport, filled on render
	assertListsAll(t, viewportContent(&tab.viewport), names)
}

func TestUnpackTab_EnvFilterNarrowsList(t *testing.T) {
	names := []string{"prod-1.enc", "production-1.enc", "staging-1.enc", "archive-1.enc"}
	m := newTestModel(t)
	openArchivePicker(t, m, TabUnpack, names)
	tab := unpackTab(t, m)

	typeKeys(t, m, "prod")
	m.View()
	view := viewportContent(&tab.viewport)
	if !strings.Contains(view, "prod-1.enc") {
		t.Errorf("prod-1.enc is missing\n---\n%s\n---", view)
	}
	for _, other := range []string{"production-1.enc", "staging-1.enc", "archive-1.enc"} {
		if strings.Contains(view, other) {
			t.Errorf("%s is listed although the filter is prod\n---\n%s\n---", other, view)
		}
	}

	send(t, m, keyMsg("backspace"))
	m.View()
	if got := tab.envInput.Value(); got != "pro" {
		t.Fatalf("filter = %q after backspace", got)
	}
	if strings.Contains(viewportContent(&tab.viewport), "prod-1.enc") {
		t.Error("a partial name must not match: the separator is part of the prefix")
	}
}

func TestUnpackTab_FilterTypingDoesNotSwitchTabs(t *testing.T) {
	m := newTestModel(t)
	openArchivePicker(t, m, TabUnpack, []string{"a-1.enc"})

	typeKeys(t, m, "3q")
	if m.activeTab != TabUnpack {
		t.Errorf("typing into the filter switched to tab %d", m.activeTab)
	}
}

func TestUnpackTab_SelectingAnArchiveAdvancesToPassword(t *testing.T) {
	names := []string{".goingenv/alpha-1.enc", ".goingenv/bravo-1.enc", ".goingenv/charlie-1.enc"}
	m := newTestModel(t)
	openArchivePicker(t, m, TabUnpack, names)

	send(t, m, keyMsg("down"), keyMsg("enter"))

	tab := unpackTab(t, m)
	if tab.step != UnpackStepPassword {
		t.Fatalf("selecting an archive left the tab at step %d, want UnpackStepPassword", tab.step)
	}
	if tab.selectedArchive != ".goingenv/bravo-1.enc" {
		t.Errorf("selected %q, want the second entry", tab.selectedArchive)
	}
	if !tab.InputFocused() {
		t.Error("the password field did not take focus after selection")
	}

	// Esc returns to the list with the filter focused again.
	send(t, m, keyMsg("esc"))
	if tab.step != UnpackStepSelect || !tab.envInput.Focused() {
		t.Errorf("esc from password: step %d, filter focused %v", tab.step, tab.envInput.Focused())
	}
}

func TestUnpackTab_EmptyFilterResultIgnoresEnter(t *testing.T) {
	m := newTestModel(t)
	openArchivePicker(t, m, TabUnpack, []string{"a-1.enc"})
	tab := unpackTab(t, m)

	typeKeys(t, m, "zzz")
	send(t, m, keyMsg("enter"))
	if tab.step != UnpackStepSelect {
		t.Errorf("enter on an empty list advanced to step %d", tab.step)
	}
	if !strings.Contains(m.View(), "No archives for environment") {
		t.Errorf("the empty result is not explained\n---\n%s\n---", m.View())
	}
}
