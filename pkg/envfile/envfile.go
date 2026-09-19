// Package envfile parses dotenv-style files.
//
// The grammar is deliberately small and does no expansion:
//
//   - blank lines and lines starting with # are ignored
//   - a leading "export " is stripped
//   - the key is everything before the first "=" and must match
//     ^[A-Za-z_][A-Za-z0-9_]*$; a line without "=" or with another shape is
//     skipped rather than rejected, since env-file patterns also match files
//     such as .env.example that may hold prose
//   - a double-quoted value may span lines and understands \n \r \t \" \\
//   - a single-quoted value is literal and may span lines
//   - an unquoted value ends at the line, minus a trailing " #comment"
//   - $VAR and ${VAR} are never expanded
//
// The only error is an unterminated quote.
package envfile

import (
	"fmt"
	"regexp"
	"strings"
)

// Entry is one KEY=value assignment.
type Entry struct {
	Key   string
	Value string
	Line  int // 1-based line the assignment starts on
}

// File is a parsed env file. Entries keep file order and duplicates.
type File struct {
	Entries []Entry
}

var keyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Parse parses data. The only error is an unterminated quoted value.
func Parse(data []byte) (*File, error) {
	f := &File{}
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		lineNo := i + 1
		line := strings.TrimSpace(strings.TrimSuffix(lines[i], "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)

		key, rest, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || !keyRe.MatchString(key) {
			continue
		}
		rest = strings.TrimSpace(rest)

		value, consumed, err := parseValue(rest, lines[i+1:], lineNo)
		if err != nil {
			return nil, err
		}
		i += consumed
		f.Entries = append(f.Entries, Entry{Key: key, Value: value, Line: lineNo})
	}
	return f, nil
}

// parseValue parses the text after "=". following holds the lines after the
// current one, for quoted values that continue; consumed reports how many of
// them were used.
func parseValue(rest string, following []string, lineNo int) (value string, consumed int, err error) {
	if rest == "" {
		return "", 0, nil
	}
	switch rest[0] {
	case '"', '\'':
		return parseQuoted(rest, following, lineNo)
	default:
		return stripInlineComment(rest), 0, nil
	}
}

func parseQuoted(rest string, following []string, lineNo int) (value string, consumed int, err error) {
	quote := rest[0]
	body := rest[1:]
	// Gather lines until the closing quote, honouring backslash escapes in
	// double quotes only.
	for {
		if end := closingQuote(body, quote); end >= 0 {
			raw := body[:end]
			if quote == '"' {
				raw = unescape(raw)
			}
			return raw, consumed, nil
		}
		if consumed >= len(following) {
			return "", 0, fmt.Errorf("line %d: unterminated quoted value", lineNo)
		}
		body += "\n" + strings.TrimSuffix(following[consumed], "\r")
		consumed++
	}
}

// closingQuote returns the index of the first unescaped quote in s, or -1.
func closingQuote(s string, quote byte) int {
	for i := 0; i < len(s); i++ {
		switch {
		case quote == '"' && s[i] == '\\':
			i++ // skip the escaped character
		case s[i] == quote:
			return i
		}
	}
	return -1
}

func unescape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case '"', '\\':
			b.WriteByte(s[i])
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// stripInlineComment removes " #..." from an unquoted value and trims it.
func stripInlineComment(s string) string {
	if i := strings.Index(s, " #"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// Keys returns the distinct keys in first-occurrence order. Never nil.
func (f *File) Keys() []string {
	keys := make([]string, 0, len(f.Entries))
	seen := make(map[string]bool, len(f.Entries))
	for _, e := range f.Entries {
		if !seen[e.Key] {
			seen[e.Key] = true
			keys = append(keys, e.Key)
		}
	}
	return keys
}

// Map returns the entries as a map; a later duplicate wins.
func (f *File) Map() map[string]string {
	m := make(map[string]string, len(f.Entries))
	for _, e := range f.Entries {
		m[e.Key] = e.Value
	}
	return m
}
