package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"goingenv/internal/config"
	"goingenv/pkg/types"
	"goingenv/pkg/utils"
)

// SettingsTab displays configuration settings.
type SettingsTab struct {
	app         *types.App
	debugLogger *DebugLogger
	viewport    viewport.Model
	ready       bool
}

// NewSettingsTab creates a new SettingsTab.
func NewSettingsTab(app *types.App, debugLogger *DebugLogger) *SettingsTab {
	return &SettingsTab{
		app:         app,
		debugLogger: debugLogger,
	}
}

func (t *SettingsTab) Title() string { return "Settings" }

func (t *SettingsTab) InputFocused() bool { return false }

func (t *SettingsTab) ShortHelp() []key.Binding {
	return []key.Binding{
		NavigationKeys.Up,
		NavigationKeys.Down,
	}
}

func (t *SettingsTab) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{NavigationKeys.Up, NavigationKeys.Down},
	}
}

func (t *SettingsTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	return t, scrollViewport(&t.viewport, t.ready, msg)
}

func (t *SettingsTab) View(width, height int) string {
	ensureViewport(&t.viewport, &t.ready, width, height)
	t.viewport.SetContent(t.buildContent())
	return t.viewport.View()
}

func (t *SettingsTab) buildContent() string {
	var b strings.Builder

	b.WriteString(RenderSectionHeader("Settings") + "\n\n")

	fmt.Fprintf(&b, "  Scan depth        %d\n", t.app.Config.DefaultDepth)
	fmt.Fprintf(&b, "  Max file size     %s\n", utils.FormatSize(t.app.Config.MaxFileSize))
	fmt.Fprintf(&b, "  Manifest          %t\n\n", t.app.Config.Manifest)

	b.WriteString(RenderSectionHeader("Patterns") + "\n")
	b.WriteString("  Include:\n")
	for _, pattern := range t.app.Config.EnvPatterns {
		fmt.Fprintf(&b, "    %s\n", pattern)
	}
	b.WriteString("  Exclude:\n")
	for _, pattern := range t.app.Config.ExcludePatterns {
		fmt.Fprintf(&b, "    %s\n", pattern)
	}

	b.WriteString("\n" + RenderSectionHeader("Config location") + "\n")
	fmt.Fprintf(&b, "  %s\n", config.ResolveConfigPath())

	return b.String()
}
