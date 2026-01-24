package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Color palette from DESIGN.md
var (
	brandColor   = lipgloss.Color("#22d3a7")
	successColor = lipgloss.Color("#22d3a7")
	warningColor = lipgloss.Color("#ffd93d")
	errorColor   = lipgloss.Color("#ff6b6b")
	infoColor    = lipgloss.Color("#7c9cbc")
	mutedColor   = lipgloss.Color("#6b7a8f")
)

// Styles for colored output
var (
	brandStyle    = lipgloss.NewStyle().Foreground(brandColor)
	successStyle  = lipgloss.NewStyle().Foreground(successColor)
	warningStyle  = lipgloss.NewStyle().Foreground(warningColor)
	errorStyleCLI = lipgloss.NewStyle().Foreground(errorColor)
	infoStyle     = lipgloss.NewStyle().Foreground(infoColor)
	mutedStyle    = lipgloss.NewStyle().Foreground(mutedColor)
)

// Output handles CLI output with TTY-aware coloring
type Output struct {
	stdout    io.Writer
	stderr    io.Writer
	useColors bool
	version   string
}

// NewOutput creates a new Output instance with TTY detection
func NewOutput(version string) *Output {
	output := termenv.NewOutput(os.Stdout)
	useColors := output.Profile != termenv.Ascii

	return &Output{
		stdout:    os.Stdout,
		stderr:    os.Stderr,
		useColors: useColors,
		version:   version,
	}
}

// NewOutputWithWriter creates an Output with custom writers (for testing)
func NewOutputWithWriter(stdout, stderr io.Writer, useColors bool, version string) *Output {
	return &Output{
		stdout:    stdout,
		stderr:    stderr,
		useColors: useColors,
		version:   version,
	}
}

// Header prints the branded header: [*] goingenv v{version}
func (o *Output) Header() {
	if o.useColors {
		fmt.Fprintf(o.stdout, "%s goingenv v%s\n",
			brandStyle.Render("[*]"),
			o.version)
	} else {
		fmt.Fprintf(o.stdout, "[*] goingenv v%s\n", o.version)
	}
}

// Success prints a success message: [+] message
func (o *Output) Success(msg string) {
	if o.useColors {
		fmt.Fprintf(o.stdout, "%s %s\n", successStyle.Render("[+]"), msg)
	} else {
		fmt.Fprintf(o.stdout, "[+] %s\n", msg)
	}
}

// Warning prints a warning message: [!] message
func (o *Output) Warning(msg string) {
	if o.useColors {
		fmt.Fprintf(o.stdout, "%s %s\n", warningStyle.Render("[!]"), msg)
	} else {
		fmt.Fprintf(o.stdout, "[!] %s\n", msg)
	}
}

// Error prints an error message to stderr: [x] message
func (o *Output) Error(msg string) {
	if o.useColors {
		fmt.Fprintf(o.stderr, "%s %s\n", errorStyleCLI.Render("[x]"), msg)
	} else {
		fmt.Fprintf(o.stderr, "[x] %s\n", msg)
	}
}

// Action prints an action in progress: [>] message
func (o *Output) Action(msg string) {
	if o.useColors {
		fmt.Fprintf(o.stdout, "%s %s\n", mutedStyle.Render("[>]"), msg)
	} else {
		fmt.Fprintf(o.stdout, "[>] %s\n", msg)
	}
}

// Hint prints a hint or tip: [?] message
func (o *Output) Hint(msg string) {
	if o.useColors {
		fmt.Fprintf(o.stdout, "%s %s\n", infoStyle.Render("[?]"), msg)
	} else {
		fmt.Fprintf(o.stdout, "[?] %s\n", msg)
	}
}

// ListItem prints a list item: [-] message
func (o *Output) ListItem(msg string) {
	if o.useColors {
		fmt.Fprintf(o.stdout, "%s %s\n", mutedStyle.Render("[-]"), msg)
	} else {
		fmt.Fprintf(o.stdout, "[-] %s\n", msg)
	}
}

// Skipped prints a skipped item: [~] message
func (o *Output) Skipped(msg string) {
	if o.useColors {
		fmt.Fprintf(o.stdout, "%s %s\n", mutedStyle.Render("[~]"), msg)
	} else {
		fmt.Fprintf(o.stdout, "[~] %s\n", msg)
	}
}

// Section prints a section header in muted color
func (o *Output) Section(title string) {
	if o.useColors {
		fmt.Fprintf(o.stdout, "%s\n", mutedStyle.Render(title))
	} else {
		fmt.Fprintf(o.stdout, "%s\n", title)
	}
}

// Indent prints content with 2-space indentation
func (o *Output) Indent(msg string) {
	fmt.Fprintf(o.stdout, "  %s\n", msg)
}

// IndentMultiple prints multiple lines with indentation
func (o *Output) IndentMultiple(lines ...string) {
	for _, line := range lines {
		fmt.Fprintf(o.stdout, "  %s\n", line)
	}
}

// Blank prints an empty line
func (o *Output) Blank() {
	fmt.Fprintln(o.stdout)
}

// Print prints a plain message without prefix
func (o *Output) Print(msg string) {
	fmt.Fprintln(o.stdout, msg)
}

// Printf prints a formatted message without prefix
func (o *Output) Printf(format string, args ...interface{}) {
	fmt.Fprintf(o.stdout, format, args...)
}

// MutedPrint prints text in muted color
func (o *Output) MutedPrint(msg string) {
	if o.useColors {
		fmt.Fprintln(o.stdout, mutedStyle.Render(msg))
	} else {
		fmt.Fprintln(o.stdout, msg)
	}
}

// SuccessHighlight prints a success message with highlighted text
func (o *Output) SuccessHighlight(prefix, highlight string) {
	if o.useColors {
		fmt.Fprintf(o.stdout, "%s %s %s\n",
			successStyle.Render("[+]"),
			prefix,
			successStyle.Render(highlight))
	} else {
		fmt.Fprintf(o.stdout, "[+] %s %s\n", prefix, highlight)
	}
}

// WarningList prints a warning followed by a list of items
func (o *Output) WarningList(msg string, items []string, limit int) {
	o.Warning(msg)
	for i, item := range items {
		if limit > 0 && i >= limit {
			o.Indent(fmt.Sprintf("... and %d more", len(items)-limit))
			break
		}
		o.Indent(item)
	}
}

// FormatKeyValue formats a key-value pair
func (o *Output) FormatKeyValue(key, value string) string {
	return fmt.Sprintf("%s: %s", key, value)
}

// Table prints items in a simple table format with indentation
func (o *Output) Table(rows [][]string) {
	if len(rows) == 0 {
		return
	}

	// Calculate column widths
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	widths := make([]int, maxCols)
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print rows
	for _, row := range rows {
		var parts []string
		for i, cell := range row {
			if i < len(widths) {
				parts = append(parts, fmt.Sprintf("%-*s", widths[i], cell))
			}
		}
		o.Indent(strings.Join(parts, "  "))
	}
}

// Global output instance (set during command execution)
var globalOutput *Output

// SetGlobalOutput sets the global output instance
func SetGlobalOutput(out *Output) {
	globalOutput = out
}

// GetGlobalOutput returns the global output instance
func GetGlobalOutput() *Output {
	return globalOutput
}
