package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

const (
	headerHeight = 1
	tabBarHeight = 1
	gapHeight    = 1
	footerHeight = 1
	chromeHeight = headerHeight + tabBarHeight + gapHeight + footerHeight
)

// ContentHeight returns the available height for tab content.
func ContentHeight(totalHeight int) int {
	h := totalHeight - chromeHeight
	if h < 1 {
		h = 1
	}
	return h
}

// tabNames maps TabID to display names.
var tabNames = [5]string{"Pack", "Unpack", "List", "Status", "Settings"}

// renderTabBar renders the tab bar with active tab indicator.
func renderTabBar(active TabID, width int) string {
	var tabs []string
	for i, name := range tabNames {
		label := fmt.Sprintf(" %d:%s ", i+1, name)
		if TabID(i) == active {
			tabs = append(tabs, ActiveTabStyle.Render(label))
		} else {
			tabs = append(tabs, InactiveTabStyle.Render(label))
		}
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)
	// Pad to full width with the tab bar border
	if lipgloss.Width(bar) < width {
		padding := strings.Repeat(" ", width-lipgloss.Width(bar))
		bar += TabGapStyle.Render(padding)
	}
	return bar
}

// renderShortHelp renders the footer with context-sensitive key hints.
func renderShortHelp(tabBindings []key.Binding) string {
	// Combine tab-specific and global bindings
	all := make([]key.Binding, 0, len(tabBindings)+3)
	all = append(all, tabBindings...)
	all = append(all, GlobalKeys.NextTab, GlobalKeys.Help, GlobalKeys.Quit)

	var parts []string
	for _, b := range all {
		h := b.Help()
		if h.Key != "" {
			parts = append(parts, fmt.Sprintf("[%s] %s", h.Key, h.Desc))
		}
	}
	return MutedStyle.Render(strings.Join(parts, "  "))
}

// padContent ensures content fills the available height by adding blank lines.
func padContent(content string, targetHeight int) string {
	lines := strings.Split(content, "\n")
	for len(lines) < targetHeight {
		lines = append(lines, "")
	}
	// Truncate if too long (viewport should handle this, but safety net)
	if len(lines) > targetHeight {
		lines = lines[:targetHeight]
	}
	return strings.Join(lines, "\n")
}

// tabClickIndex returns which tab was clicked based on X position, or -1.
func tabClickIndex(x int) int {
	offset := 0
	for i, name := range tabNames {
		label := fmt.Sprintf(" %d:%s ", i+1, name)
		w := lipgloss.Width(label)
		if x >= offset && x < offset+w {
			return i
		}
		offset += w
	}
	return -1
}

// renderEmptyState renders a centered empty state message.
func renderEmptyState(title, description, hint string, width, height int) string {
	titleLine := HighlightStyle.Render(title)
	descLine := MutedStyle.Render(description)
	hintLine := InfoStyle.Render(hint)

	content := titleLine + "\n\n" + descLine + "\n\n" + hintLine
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

// renderStepIndicator renders wizard step progress.
func renderStepIndicator(steps []string, current int) string {
	var parts []string
	for i, name := range steps {
		if i == current {
			parts = append(parts, StepActiveStyle.Render(fmt.Sprintf("[*] %s", name)))
		} else if i < current {
			parts = append(parts, StepDoneStyle.Render(fmt.Sprintf("[+] %s", name)))
		} else {
			parts = append(parts, StepInactiveStyle.Render(fmt.Sprintf("[o] %s", name)))
		}
	}
	return strings.Join(parts, "  ")
}
