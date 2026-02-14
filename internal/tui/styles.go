package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette from DESIGN.md
var (
	PrimaryColor   = lipgloss.Color("#22d3a7")
	SecondaryColor = lipgloss.Color("#e9eaeb")
	ErrorColor     = lipgloss.Color("#ff6b6b")
	SuccessColor   = lipgloss.Color("#22d3a7")
	WarningColor   = lipgloss.Color("#ffd93d")
	InfoColor      = lipgloss.Color("#7c9cbc")
	MutedColor     = lipgloss.Color("#6b7a8f")
)

// Base styles
var (
	// TitleStyle is used for screen titles (simplified, no background)
	TitleStyle = lipgloss.NewStyle().
			MarginBottom(1)

	// HeaderStyle is used for section headers
	HeaderStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			MarginBottom(1)

	// ErrorStyle is used for error messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	// SuccessStyle is used for success messages
	SuccessStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Bold(true)

	// WarningStyle is used for warning messages
	WarningStyle = lipgloss.NewStyle().
			Foreground(WarningColor).
			Bold(true)

	// InfoStyle is used for informational messages
	InfoStyle = lipgloss.NewStyle().
			Foreground(InfoColor)

	// MutedStyle is used for less important text
	MutedStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	// DimStyle is used for debug information and very low-priority text
	DimStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Faint(true)

	// ListStyle is used for lists (borderless)
	ListStyle = lipgloss.NewStyle().
			Padding(1).
			MarginBottom(1)

	// CodeStyle is used for code snippets and file paths
	CodeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1c1f24")).
			Foreground(SecondaryColor).
			Padding(0, 1).
			MarginLeft(2)

	// HelpStyle is used for help text
	HelpStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			MarginTop(1)

	// HighlightStyle is used to highlight important information
	HighlightStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true)

	// ProgressBarStyle is used for progress indicators
	ProgressBarStyle = lipgloss.NewStyle().
				Foreground(SuccessColor)

	// FileItemStyle is used for file list items
	FileItemStyle = lipgloss.NewStyle().
			MarginLeft(2)

	// StatStyle is used for statistics display
	StatStyle = lipgloss.NewStyle().
			Foreground(InfoColor).
			MarginLeft(2)

	// MenuItemStyle customizes menu items
	MenuItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	// SelectedMenuItemStyle customizes selected menu items (chevron-based, no background)
	SelectedMenuItemStyle = lipgloss.NewStyle().
				Foreground(SecondaryColor)
)

// Layout styles for different screen sizes
var (
	// NarrowScreenStyle is used for screens narrower than 80 characters
	NarrowScreenStyle = lipgloss.NewStyle().
				Width(80).
				Padding(1)

	// WideScreenStyle is used for screens wider than 80 characters
	WideScreenStyle = lipgloss.NewStyle().
			Width(100).
			Padding(2)

	// FullWidthStyle takes up the full available width
	FullWidthStyle = lipgloss.NewStyle().
			Width(100) // This will be set dynamically
)

// Specific component styles (borderless)
var (
	// PasswordInputStyle customizes password input fields (no border)
	PasswordInputStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Width(40)

	// FilePickerStyle customizes the file picker (no border)
	FilePickerStyle = lipgloss.NewStyle().
			Padding(1).
			Height(10)

	// ProgressStyle customizes progress bars (no border)
	ProgressStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Width(50)

	// StatusCardStyle is used for status information cards (no border)
	StatusCardStyle = lipgloss.NewStyle().
			Padding(1).
			MarginBottom(1).
			Width(60)

	// ArchiveCardStyle is used for archive information display (no border)
	ArchiveCardStyle = lipgloss.NewStyle().
				Padding(1).
				MarginBottom(1)
)

// Helper functions for dynamic styling

// GetScreenStyle returns appropriate style based on screen width
func GetScreenStyle(width int) lipgloss.Style {
	if width < 80 {
		return NarrowScreenStyle.Width(width - 4)
	} else if width < 120 {
		return WideScreenStyle.Width(width - 4)
	}
	return FullWidthStyle.Width(width - 4)
}

// GetResponsiveWidth returns appropriate width based on screen size
func GetResponsiveWidth(screenWidth int, percentage float64) int {
	width := int(float64(screenWidth) * percentage)
	if width < 40 {
		return 40
	}
	if width > 120 {
		return 120
	}
	return width
}

// RenderWithIcon renders text with an icon prefix
func RenderWithIcon(icon, text string, style lipgloss.Style) string {
	return style.Render(icon + " " + text)
}

// RenderCard renders content in a card-like container
func RenderCard(title, content string, style lipgloss.Style) string {
	header := HeaderStyle.Render(title)
	body := lipgloss.NewStyle().MarginLeft(2).Render(content)
	return style.Render(header + "\n" + body)
}

// RenderKeyValue renders key-value pairs consistently
func RenderKeyValue(key, value string) string {
	keyStyle := lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true)
	return keyStyle.Render(key+":") + " " + value
}

// RenderProgressBar renders a custom progress bar
func RenderProgressBar(percentage float64, width int) string {
	if width < 10 {
		width = 10
	}

	filled := int(percentage * float64(width) / 100)
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return ProgressBarStyle.Render(bar)
}

// RenderHeader creates branded header: [●]goingenv v{version}
func RenderHeader(version string) string {
	bracketStyle := lipgloss.NewStyle().Foreground(MutedColor)
	circleStyle := lipgloss.NewStyle().Foreground(PrimaryColor)
	wordmarkStyle := lipgloss.NewStyle().Foreground(SecondaryColor)
	versionStyle := lipgloss.NewStyle().Foreground(MutedColor)

	return lipgloss.JoinHorizontal(lipgloss.Left,
		bracketStyle.Render("["),
		circleStyle.Render("●"),
		bracketStyle.Render("]"),
		wordmarkStyle.Render("goingenv"),
		versionStyle.Render(" v"+version),
	)
}

// RenderMenuItem renders menu item with optional selection chevron
func RenderMenuItem(item string, selected bool) string {
	if selected {
		chevronStyle := lipgloss.NewStyle().Foreground(PrimaryColor)
		return chevronStyle.Render("> ") + item
	}
	return "  " + item
}

// RenderFooter renders keyboard hints in muted color
func RenderFooter(hints ...string) string {
	return MutedStyle.Render(strings.Join(hints, "  "))
}

// RenderSectionHeader renders a section header
func RenderSectionHeader(title string) string {
	return MutedStyle.Render(title)
}

// Theme configuration for different modes
type Theme struct {
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Error     lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Info      lipgloss.Color
	Muted     lipgloss.Color
}

// DarkTheme provides a dark color scheme (brand-aligned)
var DarkTheme = Theme{
	Primary:   lipgloss.Color("#22d3a7"),
	Secondary: lipgloss.Color("#e9eaeb"),
	Error:     lipgloss.Color("#ff6b6b"),
	Success:   lipgloss.Color("#22d3a7"),
	Warning:   lipgloss.Color("#ffd93d"),
	Info:      lipgloss.Color("#7c9cbc"),
	Muted:     lipgloss.Color("#6b7a8f"),
}

// LightTheme provides a light color scheme
var LightTheme = Theme{
	Primary:   lipgloss.Color("#22d3a7"),
	Secondary: lipgloss.Color("#121417"),
	Error:     lipgloss.Color("#ff6b6b"),
	Success:   lipgloss.Color("#22d3a7"),
	Warning:   lipgloss.Color("#ffd93d"),
	Info:      lipgloss.Color("#7c9cbc"),
	Muted:     lipgloss.Color("#6b7a8f"),
}

// ApplyTheme applies a theme to all styles
func ApplyTheme(theme *Theme) {
	TitleStyle = TitleStyle.Foreground(theme.Secondary)
	HeaderStyle = HeaderStyle.Foreground(theme.Muted)
	ErrorStyle = ErrorStyle.Foreground(theme.Error)
	SuccessStyle = SuccessStyle.Foreground(theme.Success)
	WarningStyle = WarningStyle.Foreground(theme.Warning)
	InfoStyle = InfoStyle.Foreground(theme.Info)
	MutedStyle = MutedStyle.Foreground(theme.Muted)
	ProgressBarStyle = ProgressBarStyle.Foreground(theme.Success)
}
