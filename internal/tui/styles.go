package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	PrimaryColor   = lipgloss.Color("#7D56F4")
	SecondaryColor = lipgloss.Color("#FAFAFA")
	ErrorColor     = lipgloss.Color("#FF5555")
	SuccessColor   = lipgloss.Color("#50FA7B")
	WarningColor   = lipgloss.Color("#F1FA8C")
	InfoColor      = lipgloss.Color("#8BE9FD")
	MutedColor     = lipgloss.Color("#6272A4")
)

// Base styles
var (
	// TitleStyle is used for screen titles
	TitleStyle = lipgloss.NewStyle().
			Background(PrimaryColor).
			Foreground(SecondaryColor).
			Padding(0, 1).
			MarginBottom(1).
			Bold(true)

	// HeaderStyle is used for section headers
	HeaderStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
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

	// ListStyle is used for bordered lists
	ListStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(1).
			MarginBottom(1)

	// CodeStyle is used for code snippets and file paths
	CodeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#282A36")).
			Foreground(lipgloss.Color("#F8F8F2")).
			Padding(0, 1).
			MarginLeft(2)

	// HelpStyle is used for help text
	HelpStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			MarginTop(1)

	// HighlightStyle is used to highlight important information
	HighlightStyle = lipgloss.NewStyle().
			Background(InfoColor).
			Foreground(lipgloss.Color("#282A36")).
			Padding(0, 1).
			Bold(true)

	// BorderStyle is used for general borders
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(PrimaryColor)

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
			PaddingLeft(1)

	// SelectedMenuItemStyle customizes selected menu items
	SelectedMenuItemStyle = lipgloss.NewStyle().
				Background(PrimaryColor).
				Foreground(SecondaryColor).
				PaddingLeft(1).
				Bold(true)
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

// Specific component styles
var (
	// PasswordInputStyle customizes password input fields
	PasswordInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(PrimaryColor).
				Padding(0, 1).
				Width(40)

	// FilePickerStyle customizes the file picker
	FilePickerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(1).
			Height(10)

	// ProgressStyle customizes progress bars
	ProgressStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(0, 1).
			Width(50)

	// StatusCardStyle is used for status information cards
	StatusCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(InfoColor).
			Padding(1).
			MarginBottom(1).
			Width(60)

	// ArchiveCardStyle is used for archive information display
	ArchiveCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(SuccessColor).
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

// DarkTheme provides a dark color scheme
var DarkTheme = Theme{
	Primary:   lipgloss.Color("#7D56F4"),
	Secondary: lipgloss.Color("#FAFAFA"),
	Error:     lipgloss.Color("#FF5555"),
	Success:   lipgloss.Color("#50FA7B"),
	Warning:   lipgloss.Color("#F1FA8C"),
	Info:      lipgloss.Color("#8BE9FD"),
	Muted:     lipgloss.Color("#6272A4"),
}

// LightTheme provides a light color scheme
var LightTheme = Theme{
	Primary:   lipgloss.Color("#5A4FCF"),
	Secondary: lipgloss.Color("#1A1A1A"),
	Error:     lipgloss.Color("#D63031"),
	Success:   lipgloss.Color("#00B894"),
	Warning:   lipgloss.Color("#FDCB6E"),
	Info:      lipgloss.Color("#74B9FF"),
	Muted:     lipgloss.Color("#636E72"),
}

// ApplyTheme applies a theme to all styles
func ApplyTheme(theme Theme) {
	TitleStyle = TitleStyle.Background(theme.Primary).Foreground(theme.Secondary)
	HeaderStyle = HeaderStyle.Foreground(theme.Primary)
	ErrorStyle = ErrorStyle.Foreground(theme.Error)
	SuccessStyle = SuccessStyle.Foreground(theme.Success)
	WarningStyle = WarningStyle.Foreground(theme.Warning)
	InfoStyle = InfoStyle.Foreground(theme.Info)
	MutedStyle = MutedStyle.Foreground(theme.Muted)
	ListStyle = ListStyle.BorderForeground(theme.Primary)
	BorderStyle = BorderStyle.BorderForeground(theme.Primary)
	ProgressBarStyle = ProgressBarStyle.Foreground(theme.Success)
	PasswordInputStyle = PasswordInputStyle.BorderForeground(theme.Primary)
	FilePickerStyle = FilePickerStyle.BorderForeground(theme.Primary)
	ProgressStyle = ProgressStyle.BorderForeground(theme.Primary)
	StatusCardStyle = StatusCardStyle.BorderForeground(theme.Info)
	ArchiveCardStyle = ArchiveCardStyle.BorderForeground(theme.Success)
}
