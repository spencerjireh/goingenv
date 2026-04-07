package tui

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap defines keybindings available on every screen.
type GlobalKeyMap struct {
	Quit    key.Binding
	NextTab key.Binding
	PrevTab key.Binding
	Tab1    key.Binding
	Tab2    key.Binding
	Tab3    key.Binding
	Tab4    key.Binding
	Tab5    key.Binding
	Help    key.Binding
}

// GlobalKeys is the singleton global keybinding set.
var GlobalKeys = GlobalKeyMap{
	Quit:    key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
	NextTab: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
	PrevTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev tab")),
	Tab1:    key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "Pack")),
	Tab2:    key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "Unpack")),
	Tab3:    key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "List")),
	Tab4:    key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "Status")),
	Tab5:    key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "Settings")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
}

// WizardKeyMap defines keybindings common to wizard-style tabs.
type WizardKeyMap struct {
	Start   key.Binding
	Confirm key.Binding
	Cancel  key.Binding
	Back    key.Binding
	Reset   key.Binding
}

// WizardKeys is the default wizard keybinding set.
var WizardKeys = WizardKeyMap{
	Start:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "start")),
	Confirm: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	Cancel:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	Back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Reset:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reset")),
}

// NavigationKeyMap defines keybindings for list/file navigation.
type NavigationKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Back   key.Binding
}

// NavigationKeys is the default navigation keybinding set.
var NavigationKeys = NavigationKeyMap{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "up")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "down")),
	Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Back:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
}

// StatusKeyMap defines keybindings for the status tab.
type StatusKeyMap struct {
	Pack   key.Binding
	Unpack key.Binding
	Scroll key.Binding
}

// StatusKeys is the keybinding set for the status tab.
var StatusKeys = StatusKeyMap{
	Pack:   key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pack")),
	Unpack: key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "unpack")),
	Scroll: key.NewBinding(key.WithKeys("up", "down", "k", "j"), key.WithHelp("j/k", "scroll")),
}
