package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Color palette from docs/design.md
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

// Output handles CLI output with TTY-aware coloring.
//
// Two writers, deliberately:
//
//	humanW  every decorative and progress message
//	stdout  machine-readable payloads ONLY
//
// In the default text format humanW is stdout, so output looks as it always
// has. In a machine format (--format json|porcelain) humanW becomes stderr,
// which is what makes `goingenv ... | jq` work: the payload is then the only
// thing on stdout. Before this split the branded header and the archive
// summary were printed to stdout ahead of the JSON, so the advertised
// --format json had never been pipeable.
type Output struct {
	humanW    io.Writer
	stdout    io.Writer
	stderr    io.Writer
	useColors bool
	version   string
	format    Format
}

// NewOutput creates a new Output instance with TTY detection, in the default
// text format.
func NewOutput(version string) *Output {
	return NewOutputFormat(version, FormatText)
}

// NewOutputFormat creates an Output for the given format. Anything other than
// FormatText redirects human output to stderr and disables colour, leaving
// stdout clean for the payload.
func NewOutputFormat(version string, format Format) *Output {
	output := termenv.NewOutput(os.Stdout)
	useColors := output.Profile != termenv.Ascii

	o := &Output{
		humanW:    os.Stdout,
		stdout:    os.Stdout,
		stderr:    os.Stderr,
		useColors: useColors,
		version:   version,
		format:    format,
	}
	if format.IsMachine() {
		o.humanW = os.Stderr
		// Colour on the human stream would still be correct, but a machine
		// format is overwhelmingly consumed by scripts and logs where the
		// escapes are noise.
		o.useColors = false
	}
	return o
}

// NewOutputWithWriter creates an Output with custom writers (for testing)
func NewOutputWithWriter(stdout, stderr io.Writer, useColors bool, version string) *Output {
	return &Output{
		humanW:    stdout,
		stdout:    stdout,
		stderr:    stderr,
		useColors: useColors,
		version:   version,
		format:    FormatText,
	}
}

// NewOutputWithWriterFormat is NewOutputWithWriter for a specific format,
// applying the same stdout/stderr split the real constructor does.
func NewOutputWithWriterFormat(stdout, stderr io.Writer, useColors bool, version string, format Format) *Output {
	o := NewOutputWithWriter(stdout, stderr, useColors, version)
	o.format = format
	if format.IsMachine() {
		o.humanW = stderr
		o.useColors = false
	}
	return o
}

// Format returns the output format this Output was built for.
func (o *Output) Format() Format { return o.format }

// Header prints the branded header: [●] goingenv v{version}
//
// Release builds set main.Version from the git tag, which already carries a
// "v", so prefixing one unconditionally printed "goingenv vv1.4.0".
func (o *Output) Header() {
	version := "v" + strings.TrimPrefix(o.version, "v")
	if o.useColors {
		fmt.Fprintf(o.humanW, "%s goingenv %s\n",
			brandStyle.Render("[●]"),
			version)
	} else {
		fmt.Fprintf(o.humanW, "[*] goingenv %s\n", version)
	}
}

// Success prints a success message: [+] message
func (o *Output) Success(msg string) {
	if o.useColors {
		fmt.Fprintf(o.humanW, "%s %s\n", successStyle.Render("[+]"), msg)
	} else {
		fmt.Fprintf(o.humanW, "[+] %s\n", msg)
	}
}

// Warning prints a warning message: [!] message
func (o *Output) Warning(msg string) {
	if o.useColors {
		fmt.Fprintf(o.humanW, "%s %s\n", warningStyle.Render("[!]"), msg)
	} else {
		fmt.Fprintf(o.humanW, "[!] %s\n", msg)
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
		fmt.Fprintf(o.humanW, "%s %s\n", mutedStyle.Render("[>]"), msg)
	} else {
		fmt.Fprintf(o.humanW, "[>] %s\n", msg)
	}
}

// Hint prints a hint or tip: [?] message
func (o *Output) Hint(msg string) {
	if o.useColors {
		fmt.Fprintf(o.humanW, "%s %s\n", infoStyle.Render("[?]"), msg)
	} else {
		fmt.Fprintf(o.humanW, "[?] %s\n", msg)
	}
}

// ListItem prints a list item: [-] message
func (o *Output) ListItem(msg string) {
	if o.useColors {
		fmt.Fprintf(o.humanW, "%s %s\n", mutedStyle.Render("[-]"), msg)
	} else {
		fmt.Fprintf(o.humanW, "[-] %s\n", msg)
	}
}

// Skipped prints a skipped item: [~] message
func (o *Output) Skipped(msg string) {
	if o.useColors {
		fmt.Fprintf(o.humanW, "%s %s\n", mutedStyle.Render("[~]"), msg)
	} else {
		fmt.Fprintf(o.humanW, "[~] %s\n", msg)
	}
}

// Section prints a section header in muted color
func (o *Output) Section(title string) {
	if o.useColors {
		fmt.Fprintf(o.humanW, "%s\n", mutedStyle.Render(title))
	} else {
		fmt.Fprintf(o.humanW, "%s\n", title)
	}
}

// Indent prints content with 2-space indentation
func (o *Output) Indent(msg string) {
	fmt.Fprintf(o.humanW, "  %s\n", msg)
}

// IndentMultiple prints multiple lines with indentation
func (o *Output) IndentMultiple(lines ...string) {
	for _, line := range lines {
		fmt.Fprintf(o.humanW, "  %s\n", line)
	}
}

// Blank prints an empty line
func (o *Output) Blank() {
	fmt.Fprintln(o.humanW)
}

// Print prints a plain message without prefix
func (o *Output) Print(msg string) {
	fmt.Fprintln(o.humanW, msg)
}

// Printf prints a formatted message without prefix
func (o *Output) Printf(format string, args ...interface{}) {
	fmt.Fprintf(o.humanW, format, args...)
}

// MutedPrint prints text in muted color
func (o *Output) MutedPrint(msg string) {
	if o.useColors {
		fmt.Fprintln(o.humanW, mutedStyle.Render(msg))
	} else {
		fmt.Fprintln(o.humanW, msg)
	}
}

// SuccessHighlight prints a success message with highlighted text
func (o *Output) SuccessHighlight(prefix, highlight string) {
	if o.useColors {
		fmt.Fprintf(o.humanW, "%s %s %s\n",
			successStyle.Render("[+]"),
			prefix,
			successStyle.Render(highlight))
	} else {
		fmt.Fprintf(o.humanW, "[+] %s %s\n", prefix, highlight)
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
