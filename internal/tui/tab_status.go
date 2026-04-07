package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"goingenv/internal/config"
	"goingenv/internal/scanner"
	"goingenv/pkg/types"
	"goingenv/pkg/utils"
)

// StatusTab displays project status with a two-column layout.
type StatusTab struct {
	app         *types.App
	debugLogger *DebugLogger
	viewport    viewport.Model
	ready       bool
}

// NewStatusTab creates a new StatusTab.
func NewStatusTab(app *types.App, debugLogger *DebugLogger) *StatusTab {
	return &StatusTab{
		app:         app,
		debugLogger: debugLogger,
	}
}

func (t *StatusTab) Title() string { return "Status" }

func (t *StatusTab) InputFocused() bool { return false }

func (t *StatusTab) ShortHelp() []key.Binding {
	return []key.Binding{
		StatusKeys.Scroll,
		StatusKeys.Pack,
		StatusKeys.Unpack,
	}
}

func (t *StatusTab) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{StatusKeys.Scroll, StatusKeys.Pack, StatusKeys.Unpack},
	}
}

func (t *StatusTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, StatusKeys.Pack):
			return t, func() tea.Msg { return SwitchTabMsg{Tab: TabPack} }
		case key.Matches(msg, StatusKeys.Unpack):
			return t, func() tea.Msg { return SwitchTabMsg{Tab: TabUnpack} }
		}
	}

	if t.ready {
		var cmd tea.Cmd
		t.viewport, cmd = t.viewport.Update(msg)
		return t, cmd
	}
	return t, nil
}

func (t *StatusTab) View(width, height int) string {
	if !config.IsInitialized() {
		return renderEmptyState(
			"Not Initialized",
			"goingenv is not set up in this directory.",
			"Switch to the Pack tab and initialize first.",
			width, height,
		)
	}

	// Build content for the two-column layout
	leftContent := t.buildLeftColumn(width)
	rightContent := t.buildRightColumn(width)

	// Two-column split
	halfWidth := (width - 3) / 2 // 3 for divider + padding
	if halfWidth < 20 {
		// Narrow terminal: stack vertically
		content := leftContent + "\n\n" + rightContent
		t.ensureViewport(width, height, content)
		return t.viewport.View()
	}

	leftStyle := lipgloss.NewStyle().Width(halfWidth)
	rightStyle := lipgloss.NewStyle().Width(halfWidth)

	left := leftStyle.Render(leftContent)
	right := rightStyle.Render(rightContent)

	combined := lipgloss.JoinHorizontal(lipgloss.Top, left, "  "+SplitDividerStyle.Render("│")+"  ", right)

	t.ensureViewport(width, height, combined)
	return t.viewport.View()
}

func (t *StatusTab) ensureViewport(width, height int, content string) {
	if !t.ready {
		t.viewport = viewport.New(width, height)
		t.ready = true
	} else {
		t.viewport.Width = width
		t.viewport.Height = height
	}
	t.viewport.SetContent(content)
}

func (t *StatusTab) buildLeftColumn(width int) string {
	var b strings.Builder

	// Directory
	cwd, _ := os.Getwd() //nolint:errcheck
	b.WriteString(RenderSectionHeader("Directory") + "\n")
	b.WriteString(fmt.Sprintf("  %s\n\n", cwd))

	// Environment files
	scanOpts := types.ScanOptions{
		RootPath: ".",
		MaxDepth: t.app.Config.DefaultDepth,
	}
	files, err := t.app.Scanner.ScanFiles(&scanOpts)
	if err == nil && len(files) > 0 {
		b.WriteString(RenderSectionHeader(fmt.Sprintf("Environment Files (%d)", len(files))) + "\n")
		for i, file := range files {
			if i < 15 {
				b.WriteString(fmt.Sprintf("  %s\n", file.RelativePath))
			} else if i == 15 {
				b.WriteString(fmt.Sprintf("  ... and %d more\n", len(files)-15))
				break
			}
		}

		stats := scanner.GetFileStats(files)
		b.WriteString(fmt.Sprintf("\n  Total: %s\n", utils.FormatSize(stats.TotalSize)))
	} else {
		b.WriteString(RenderSectionHeader("Environment Files") + "\n")
		b.WriteString(MutedStyle.Render("  No environment files detected") + "\n")
	}

	return b.String()
}

func (t *StatusTab) buildRightColumn(width int) string {
	var b strings.Builder

	// Archives
	archives, err := t.app.Archiver.GetAvailableArchives("")
	switch {
	case err != nil:
		b.WriteString(RenderSectionHeader("Archives") + "\n")
		b.WriteString(ErrorStyle.Render("  Error: "+err.Error()) + "\n")
	case len(archives) == 0:
		b.WriteString(RenderSectionHeader("Archives") + "\n")
		b.WriteString(MutedStyle.Render("  No archives found") + "\n")
	default:
		b.WriteString(RenderSectionHeader(fmt.Sprintf("Archives (%d)", len(archives))) + "\n")
		for _, archive := range archives {
			info, statErr := os.Stat(archive)
			if statErr == nil {
				b.WriteString(fmt.Sprintf("  %s  %s  %s\n",
					filepath.Base(archive),
					utils.FormatSize(info.Size()),
					utils.FormatTimeAgo(info.ModTime())))
			}
		}
	}

	// Config info
	b.WriteString("\n" + RenderSectionHeader("Configuration") + "\n")
	b.WriteString(fmt.Sprintf("  Scan depth:    %d\n", t.app.Config.DefaultDepth))
	b.WriteString(fmt.Sprintf("  Max file size: %s\n", utils.FormatSize(t.app.Config.MaxFileSize)))
	b.WriteString(fmt.Sprintf("  Config:        %s\n", config.GetGoingEnvDir()))

	return b.String()
}
