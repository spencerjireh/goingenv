// Package manifest writes the plaintext companion of an archive: which files
// it holds and which keys each one defines, so a pull request can show that
// STRIPE_KEY was added to .env.production without exposing its value.
//
// The manifest is meant to be committed beside the archive. It reveals file
// paths and key names, never values, and is opt-in for that reason.
package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"goingenv/pkg/envfile"
	"goingenv/pkg/types"
)

// File describes one archived env file.
type File struct {
	Path   string   `json:"path"`
	SHA256 string   `json:"sha256"`
	Keys   []string `json:"keys"`
}

// Manifest is the on-disk document.
type Manifest struct {
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
	Env         string    `json:"env"`
	Archive     string    `json:"archive"`
	Files       []File    `json:"files"`
}

// Suffix is appended to the archive path to name its manifest.
const Suffix = ".manifest.json"

// PathFor returns the manifest path for an archive.
func PathFor(archivePath string) string {
	return archivePath + Suffix
}

// Build reads every file from disk to collect its key names. The checksum
// comes from the scanner, which already hashed the file. Files are sorted by
// path so the output is stable across runs.
func Build(archivePath, env, description string, createdAt time.Time, files []types.EnvFile) (*Manifest, error) {
	m := &Manifest{
		CreatedAt:   createdAt,
		Description: description,
		Env:         env,
		Archive:     filepath.Base(archivePath),
		Files:       make([]File, 0, len(files)),
	}
	for i := range files {
		f := &files[i]
		data, err := os.ReadFile(f.Path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f.RelativePath, err)
		}
		parsed, err := envfile.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", f.RelativePath, err)
		}
		m.Files = append(m.Files, File{Path: f.RelativePath, SHA256: f.Checksum, Keys: parsed.Keys()})
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].Path < m.Files[j].Path })
	return m, nil
}

// Write serialises m to path with owner-only permissions.
func Write(path string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}
