package cli

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Format selects how a command reports its result.
//
// The contract, which scripts may rely on:
//
//   - text      the default. Human-readable, coloured when the terminal
//     supports it, laid out for reading. Not stable; do not parse it.
//   - json      a single JSON object on stdout. Stable field names.
//   - porcelain tab-separated records on stdout, one per line, no header
//     row, no colour, no alignment padding. Stable column order.
//     Named after git's sense of the word.
//   - csv       list only, retained from before this flag existed.
//
// In json and porcelain every human-facing message -- the banner, progress,
// hints, warnings -- goes to stderr, so stdout carries the payload and nothing
// else. That is what makes the output pipeable.
type Format string

const (
	FormatText      Format = "text"
	FormatJSON      Format = "json"
	FormatPorcelain Format = "porcelain"
	FormatCSV       Format = "csv"

	// FormatTable is the historical name for the default in `list`, kept
	// working so existing `--format table` invocations do not break.
	FormatTable Format = "table"
)

// IsMachine reports whether the format is meant to be consumed by a program
// rather than read, which is what moves human output off stdout.
func (f Format) IsMachine() bool {
	return f == FormatJSON || f == FormatPorcelain || f == FormatCSV
}

// ParseFormat validates a --format value. `allowCSV` is set by the one command
// that still accepts it.
func ParseFormat(s string, allowCSV bool) (Format, error) {
	switch Format(s) {
	case FormatText, FormatTable:
		// table is an alias, so everything downstream only sees text.
		return FormatText, nil
	case FormatJSON:
		return FormatJSON, nil
	case FormatPorcelain:
		return FormatPorcelain, nil
	case FormatCSV:
		if allowCSV {
			return FormatCSV, nil
		}
		return "", fmt.Errorf("unsupported format %q for this command (csv is only available for 'list')", s)
	default:
		valid := "text, json, porcelain"
		if allowCSV {
			valid += ", csv"
		}
		return "", fmt.Errorf("unsupported format %q: expected one of %s", s, valid)
	}
}

// EmitJSON writes a payload to stdout as indented JSON.
//
// Indented rather than compact because these payloads are small and a human
// reading `goingenv status --format json` without jq to hand is a common case.
// jq does not care either way.
func (o *Output) EmitJSON(payload any) error {
	enc := json.NewEncoder(o.stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return fmt.Errorf("failed to encode JSON output: %w", err)
	}
	return nil
}

// EmitPorcelain writes one tab-separated record per line to stdout.
//
// Fields are sanitised rather than quoted: a porcelain reader splits on tabs,
// so a tab or newline inside a field would silently corrupt the record for
// every consumer. Paths containing them are vanishingly rare and always
// pathological, so they are replaced with a space.
func (o *Output) EmitPorcelain(records [][]string) error {
	replacer := strings.NewReplacer("\t", " ", "\n", " ", "\r", " ")
	for _, rec := range records {
		cleaned := make([]string, len(rec))
		for i, f := range rec {
			cleaned[i] = replacer.Replace(f)
		}
		if _, err := fmt.Fprintln(o.stdout, strings.Join(cleaned, "\t")); err != nil {
			return fmt.Errorf("failed to write porcelain output: %w", err)
		}
	}
	return nil
}

// EmitError reports a failure on stderr in a shape matching the format, so a
// script gets something parseable rather than prose. Exit codes are unchanged
// and remain the primary signal.
func (o *Output) EmitError(err error) {
	if err == nil {
		return
	}
	if o.format == FormatJSON {
		enc := json.NewEncoder(o.stderr)
		enc.SetIndent("", "  ")
		encErr := enc.Encode(struct {
			Error string `json:"error"`
		}{Error: err.Error()})
		if encErr == nil {
			return
		}
		// Fall through to the plain form rather than swallowing the error
		// while in the middle of reporting one.
	}
	o.Error(err.Error())
}
