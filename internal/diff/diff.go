// Package diff compares two sets of env files by key. Values are compared
// but never reported: the output exists so a change can be reviewed without
// exposing what changed to.
package diff

import (
	"fmt"
	"sort"
	"strings"

	"goingenv/pkg/envfile"
)

// Change kinds for keys and files.
const (
	Added    = "added"
	Removed  = "removed"
	Changed  = "changed"  // keys only
	Modified = "modified" // files only: present on both sides, keys differ
)

// KeyChange is one key that differs.
type KeyChange struct {
	Key    string
	Change string
}

// FileChange is one file that differs. A file on only one side is Added or
// Removed and lists every key with the same change; a file on both sides is
// Modified and lists only the keys that differ.
type FileChange struct {
	Path   string
	Status string
	Keys   []KeyChange
}

// Result holds the files that differ, sorted by path. An empty Files means
// the two sides are equivalent.
type Result struct {
	Files []FileChange
}

// Changed reports whether anything differs.
func (r Result) Changed() bool { return len(r.Files) > 0 }

// Compare parses both sides and reports every file and key that differs.
// Sides are keyed by relative path; nil is an empty side.
func Compare(from, to map[string][]byte) (Result, error) {
	fromKV, err := parseSide(from)
	if err != nil {
		return Result{}, err
	}
	toKV, err := parseSide(to)
	if err != nil {
		return Result{}, err
	}

	var files []FileChange
	for _, path := range union(keysOf(fromKV), keysOf(toKV)) {
		a, inFrom := fromKV[path]
		b, inTo := toKV[path]
		switch {
		case inFrom && !inTo:
			files = append(files, FileChange{Path: path, Status: Removed, Keys: allKeys(a, Removed)})
		case !inFrom && inTo:
			files = append(files, FileChange{Path: path, Status: Added, Keys: allKeys(b, Added)})
		default:
			if keys := compareKeys(a, b); len(keys) > 0 {
				files = append(files, FileChange{Path: path, Status: Modified, Keys: keys})
			}
		}
	}
	return Result{Files: files}, nil
}

func parseSide(side map[string][]byte) (map[string]map[string]string, error) {
	out := make(map[string]map[string]string, len(side))
	for path, data := range side {
		f, err := envfile.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out[path] = f.Map()
	}
	return out, nil
}

func compareKeys(a, b map[string]string) []KeyChange {
	var keys []KeyChange
	for _, k := range union(keysOf(a), keysOf(b)) {
		va, inA := a[k]
		vb, inB := b[k]
		switch {
		case inA && !inB:
			keys = append(keys, KeyChange{Key: k, Change: Removed})
		case !inA && inB:
			keys = append(keys, KeyChange{Key: k, Change: Added})
		case va != vb:
			keys = append(keys, KeyChange{Key: k, Change: Changed})
		}
	}
	return keys
}

func allKeys(m map[string]string, change string) []KeyChange {
	keys := make([]KeyChange, 0, len(m))
	for _, k := range keysOf(m) {
		keys = append(keys, KeyChange{Key: k, Change: change})
	}
	return keys
}

func keysOf[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// union merges two sorted string slices without duplicates.
func union(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, s := range append(append([]string{}, a...), b...) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// Marker returns the one-character prefix used for a change in text output.
func Marker(change string) string {
	switch change {
	case Added:
		return "+"
	case Removed:
		return "-"
	default:
		return "~"
	}
}

// Format renders a result as plain text: one block per file, one line per
// key, values never included.
func Format(r Result) string {
	if !r.Changed() {
		return "No differences\n"
	}
	var b strings.Builder
	for _, f := range r.Files {
		fmt.Fprintf(&b, "%s %s (%s)\n", Marker(f.Status), f.Path, f.Status)
		for _, k := range f.Keys {
			fmt.Fprintf(&b, "    %s %s\n", Marker(k.Change), k.Key)
		}
	}
	fmt.Fprintf(&b, "%d file(s) differ\n", len(r.Files))
	return b.String()
}
