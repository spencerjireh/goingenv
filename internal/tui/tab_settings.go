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
	if t.ready {
		var cmd tea.Cmd
		t.viewport, cmd = t.viewport.Update(msg)
		return t, cmd
	}
	return t, nil
}

func (t *SettingsTab) View(width, height int) string {
	content := t.buildContent()

	if !t.ready {
		t.viewport = viewport.New(width, height)
		t.ready = true
	} else {
		t.viewport.Width = width
		t.viewport.Height = height
	}
	t.viewport.SetContent(content)
	return t.viewport.View()
}

func (t *SettingsTab) buildContent() string {
	var b strings.Builder

	b.WriteString(RenderSectionHeader("Settings") + "\n\n")

	b.WriteString(fmt.Sprintf("  Scan Depth        %d\n", t.app.Config.DefaultDepth))
	b.WriteString(fmt.Sprintf("  Max File Size     %s\n\n", utils.FormatSize(t.app.Config.MaxFileSize)))

	b.WriteString(RenderSectionHeader("Patterns") + "\n")
	b.WriteString("  Include:\n")
	for _, pattern := range t.app.Config.EnvPatterns {
		b.WriteString(fmt.Sprintf("    %s\n", pattern))
	}
	b.WriteString("  Exclude:\n")
	for _, pattern := range t.app.Config.ExcludePatterns {
		b.WriteString(fmt.Sprintf("    %s\n", pattern))
	}

	b.WriteString("\n" + RenderSectionHeader("Config Location") + "\n")
	b.WriteString(fmt.Sprintf("  %s\n", config.GetGoingEnvDir()))

	return b.String()
}
