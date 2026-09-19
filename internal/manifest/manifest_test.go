package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goingenv/pkg/types"
)

// fixtureFiles writes three env files under dir and returns their EnvFile
// records with placeholder checksums.
func fixtureFiles(t *testing.T, dir string) []types.EnvFile {
	t.Helper()
	write := func(rel, content string) types.EnvFile {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return types.EnvFile{Path: p, RelativePath: rel, Checksum: "sha-" + rel}
	}
	return []types.EnvFile{
		write("config/.env.dev", "DEV=true\n"),
		write(".env", "SECRET=hunter2\nB=1\nSECRET=again\n"),
		write(".env.empty", "# nothing here\n"),
	}
}

func TestBuild(t *testing.T) {
	dir := t.TempDir()
	files := fixtureFiles(t, dir)

	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	m, err := Build("/x/.goingenv/prod-20260919-120000.enc", "prod", "desc", now, files)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if m.Archive != "prod-20260919-120000.enc" || m.Env != "prod" || m.Description != "desc" || !m.CreatedAt.Equal(now) {
		t.Errorf("header = %+v", m)
	}
	wantPaths := []string{".env", ".env.empty", "config/.env.dev"}
	for i, f := range m.Files {
		if f.Path != wantPaths[i] {
			t.Errorf("file %d = %s, want %s (sorted)", i, f.Path, wantPaths[i])
		}
		if f.SHA256 != "sha-"+f.Path {
			t.Errorf("%s sha = %s", f.Path, f.SHA256)
		}
	}
	if got := m.Files[0].Keys; len(got) != 2 || got[0] != "SECRET" || got[1] != "B" {
		t.Errorf(".env keys = %v, want [SECRET B]", got)
	}
}

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	m, err := Build("a.enc", "", "", time.Now(), fixtureFiles(t, dir))
	if err != nil {
		t.Fatal(err)
	}

	path := PathFor(filepath.Join(dir, "a.enc"))
	if !strings.HasSuffix(path, "a.enc.manifest.json") {
		t.Errorf("PathFor = %s", path)
	}
	if writeErr := Write(path, m); writeErr != nil {
		t.Fatalf("Write: %v", writeErr)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o, want 600", info.Mode().Perm())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "hunter2") {
		t.Error("manifest contains a value")
	}
	if !strings.Contains(string(raw), `"keys": []`) {
		t.Error("empty file should serialise keys as [] not null")
	}
	var back Manifest
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("round trip: %v", err)
	}
}

func TestBuildMissingFile(t *testing.T) {
	_, err := Build("a.enc", "", "", time.Now(), []types.EnvFile{{Path: "/nonexistent", RelativePath: ".env"}})
	if err == nil {
		t.Fatal("expected an error")
	}
}
