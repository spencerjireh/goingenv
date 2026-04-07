package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// renderModalOverlay renders a modal dialog centered over dimmed content.
func (m *Model) renderModalOverlay(base string) string {
	lines := strings.Split(base, "\n")

	// Dim all background lines
	for i := range lines {
		lines[i] = DimStyle.Render(stripAnsi(lines[i]))
	}

	// Build modal content
	modalWidth := 50
	if m.width < 60 {
		modalWidth = m.width - 10
	}
	if modalWidth < 30 {
		modalWidth = 30
	}

	body := m.modal.Title + "\n\n" + m.modal.Body + "\n\n" +
		MutedStyle.Render("[y] confirm  [n/esc] cancel")
	modalBox := ModalBoxStyle.Width(modalWidth).Render(body)
	modalLines := strings.Split(modalBox, "\n")

	// Center the modal
	startY := (len(lines) - len(modalLines)) / 2
	if startY < 0 {
		startY = 0
	}

	for i, ml := range modalLines {
		y := startY + i
		if y >= 0 && y < len(lines) {
			mlWidth := lipgloss.Width(ml)
			padLeft := (m.width - mlWidth) / 2
			if padLeft < 0 {
				padLeft = 0
			}
			lines[y] = strings.Repeat(" ", padLeft) + ml
		}
	}

	return strings.Join(lines, "\n")
}

// renderToastOverlay renders toast notifications in the top-right corner.
func (m *Model) renderToastOverlay(base string) string {
	if len(m.toasts) == 0 {
		return base
	}

	lines := strings.Split(base, "\n")
	for i, toast := range m.toasts {
		lineIdx := i // top of screen
		if lineIdx >= len(lines) {
			break
		}

		var prefix string
		var style lipgloss.Style
		if toast.IsError {
			prefix = "[x] "
			style = ToastErrorStyle
		} else {
			prefix = "[+] "
			style = ToastSuccessStyle
		}
		toastText := style.Render(prefix + toast.Message)
		toastWidth := lipgloss.Width(toastText)

		// Place in top-right
		lineWidth := lipgloss.Width(lines[lineIdx])
		padLeft := m.width - toastWidth
		if padLeft < lineWidth+2 {
			padLeft = lineWidth + 2
		}
		if padLeft < 0 {
			padLeft = 0
		}

		// Overlay: pad existing line and append toast
		existingWidth := lipgloss.Width(lines[lineIdx])
		gap := padLeft - existingWidth
		if gap < 1 {
			gap = 1
		}
		lines[lineIdx] = lines[lineIdx] + strings.Repeat(" ", gap) + toastText
	}

	return strings.Join(lines, "\n")
}

// renderHelpOverlay renders the ? help overlay centered on screen.
func (m *Model) renderHelpOverlay(base string) string {
	lines := strings.Split(base, "\n")

	// Dim background
	for i := range lines {
		lines[i] = DimStyle.Render(stripAnsi(lines[i]))
	}

	// Build help content from active tab + global keys
	tabHelp := m.tabs[m.activeTab].FullHelp()

	// Add global keys as a section
	globalBindings := []key.Binding{
		GlobalKeys.NextTab,
		GlobalKeys.PrevTab,
		GlobalKeys.Help,
		GlobalKeys.Quit,
	}
	tabHelp = append(tabHelp, globalBindings)

	helpText := m.help.FullHelpView(tabHelp)

	// Wrap in a box
	helpWidth := 60
	if m.width < 70 {
		helpWidth = m.width - 10
	}
	title := HighlightStyle.Render("Keyboard Shortcuts") + "\n\n"
	dismiss := "\n" + MutedStyle.Render("Press any key to close")
	boxContent := title + helpText + dismiss
	helpBox := ModalBoxStyle.Width(helpWidth).BorderForeground(PrimaryColor).Render(boxContent)
	helpLines := strings.Split(helpBox, "\n")

	// Center vertically
	startY := (len(lines) - len(helpLines)) / 2
	if startY < 0 {
		startY = 0
	}

	for i, hl := range helpLines {
		y := startY + i
		if y >= 0 && y < len(lines) {
			hlWidth := lipgloss.Width(hl)
			padLeft := (m.width - hlWidth) / 2
			if padLeft < 0 {
				padLeft = 0
			}
			lines[y] = strings.Repeat(" ", padLeft) + hl
		}
	}

	return strings.Join(lines, "\n")
}

// stripAnsi removes ANSI escape codes from a string for dimming purposes.
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
