package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goingenv/pkg/types"
)

func TestFilterFiles(t *testing.T) {
	files := []types.EnvFile{
		{RelativePath: ".env"},
		{RelativePath: ".env.local"},
		{RelativePath: ".env.production"},
		{RelativePath: ".env.development"},
		{RelativePath: "config/.env"},
	}

	tests := []struct {
		name            string
		includePatterns []string
		excludePatterns []string
		expectedCount   int
	}{
		{
			name:            "No filters",
			includePatterns: nil,
			excludePatterns: nil,
			expectedCount:   5,
		},
		{
			name:            "Include .env.local only",
			includePatterns: []string{".env.local"},
			excludePatterns: nil,
			expectedCount:   1,
		},
		{
			name:            "Include multiple patterns",
			includePatterns: []string{".env", ".env.local"},
			excludePatterns: nil,
			expectedCount:   2,
		},
		{
			name:            "Exclude production",
			includePatterns: nil,
			excludePatterns: []string{".env.production"},
			expectedCount:   4,
		},
		{
			name:            "Include and exclude combined",
			includePatterns: []string{".env*"},
			excludePatterns: []string{".env.production"},
			expectedCount:   3, // .env, .env.local, .env.development (not .env.production, not config/.env)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterFiles(files, tt.includePatterns, tt.excludePatterns)
			if len(result) != tt.expectedCount {
				t.Errorf("filterFiles() returned %d files, expected %d", len(result), tt.expectedCount)
			}
		})
	}
}

func TestFilterFilesByPatterns(t *testing.T) {
	files := []types.EnvFile{
		{RelativePath: ".env"},
		{RelativePath: ".env.local"},
		{RelativePath: ".env.production"},
		{RelativePath: "config/.env"},
		{RelativePath: "api/.env.test"},
	}

	tests := []struct {
		name          string
		patterns      []string
		expectedCount int
	}{
		{
			name:          "Match exact file",
			patterns:      []string{".env"},
			expectedCount: 1,
		},
		{
			name:          "Match wildcard pattern",
			patterns:      []string{".env.*"},
			expectedCount: 2, // .env.local, .env.production (glob doesn't match path separators)
		},
		{
			name:          "Multiple patterns",
			patterns:      []string{".env", ".env.local"},
			expectedCount: 2,
		},
		{
			name:          "No matches",
			patterns:      []string{"nonexistent"},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterFilesByPatterns(files, tt.patterns)
			if len(result) != tt.expectedCount {
				t.Errorf("filterFilesByPatterns() returned %d files, expected %d", len(result), tt.expectedCount)
			}
		})
	}
}

func TestSortFiles(t *testing.T) {
	files := []types.EnvFile{
		{RelativePath: "b.env", Size: 100, ModTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{RelativePath: "a.env", Size: 200, ModTime: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)},
		{RelativePath: "c.env", Size: 50, ModTime: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
	}

	tests := []struct {
		name        string
		sortBy      string
		reverse     bool
		firstResult string
	}{
		{
			name:        "Sort by name ascending",
			sortBy:      "name",
			reverse:     false,
			firstResult: "a.env",
		},
		{
			name:        "Sort by name descending",
			sortBy:      "name",
			reverse:     true,
			firstResult: "c.env",
		},
		{
			name:        "Sort by size ascending",
			sortBy:      "size",
			reverse:     false,
			firstResult: "c.env", // 50 bytes
		},
		{
			name:        "Sort by size descending",
			sortBy:      "size",
			reverse:     true,
			firstResult: "a.env", // 200 bytes
		},
		{
			name:        "Sort by date ascending",
			sortBy:      "date",
			reverse:     false,
			firstResult: "b.env", // Jan 1
		},
		{
			name:        "Sort by date descending",
			sortBy:      "date",
			reverse:     true,
			firstResult: "a.env", // Jan 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid mutating the original
			filesCopy := make([]types.EnvFile, len(files))
			copy(filesCopy, files)

			sortFiles(filesCopy, tt.sortBy, tt.reverse)

			if filesCopy[0].RelativePath != tt.firstResult {
				t.Errorf("sortFiles() first result = %s, expected %s", filesCopy[0].RelativePath, tt.firstResult)
			}
		})
	}
}

func TestCheckFileConflicts(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-cli-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some existing files
	existingFiles := []string{".env", ".env.local"}
	for _, f := range existingFiles {
		if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0o600); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	files := []types.EnvFile{
		{RelativePath: ".env"},
		{RelativePath: ".env.local"},
		{RelativePath: ".env.production"}, // This one doesn't exist
	}

	conflicts := checkFileConflicts(files, tmpDir)

	if len(conflicts) != 2 {
		t.Errorf("checkFileConflicts() returned %d conflicts, expected 2", len(conflicts))
	}

	// Verify specific conflicts
	hasEnv := false
	hasLocal := false
	for _, c := range conflicts {
		if c == ".env" {
			hasEnv = true
		}
		if c == ".env.local" {
			hasLocal = true
		}
	}

	if !hasEnv || !hasLocal {
		t.Errorf("Expected conflicts for .env and .env.local")
	}
}

func TestNewRootCommand(t *testing.T) {
	cmd := NewRootCommand("test-version")

	if cmd == nil {
		t.Fatal("NewRootCommand() returned nil")
	}

	if cmd.Use != "goingenv" {
		t.Errorf("Root command Use = %s, want goingenv", cmd.Use)
	}

	if cmd.Version != "test-version" {
		t.Errorf("Root command Version = %s, want test-version", cmd.Version)
	}

	// Check that subcommands are registered
	subcommands := []string{"init", "pack", "unpack", "list", "status"}
	for _, name := range subcommands {
		found := false
		for _, subcmd := range cmd.Commands() {
			// Check if command name starts with expected name (e.g., "status" or "status [directory]")
			if subcmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand %s not found", name)
		}
	}
}

func TestNewInitCommand(t *testing.T) {
	cmd := newInitCommand()

	if cmd == nil {
		t.Fatal("newInitCommand() returned nil")
	}

	if cmd.Use != "init" {
		t.Errorf("Init command Use = %s, want init", cmd.Use)
	}

	// Check for force flag
	forceFlag := cmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Error("Init command missing --force flag")
	}
}

func TestNewPackCommand(t *testing.T) {
	cmd := newPackCommand()

	if cmd == nil {
		t.Fatal("newPackCommand() returned nil")
	}

	if cmd.Use != "pack" {
		t.Errorf("Pack command Use = %s, want pack", cmd.Use)
	}

	// Check for required flags
	expectedFlags := []string{"password-env", "directory", "output", "depth", "include", "exclude", "env-exclude", "dry-run", "verbose"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Pack command missing --%s flag", flag)
		}
	}
}

func TestNewUnpackCommand(t *testing.T) {
	cmd := newUnpackCommand()

	if cmd == nil {
		t.Fatal("newUnpackCommand() returned nil")
	}

	if cmd.Use != "unpack" {
		t.Errorf("Unpack command Use = %s, want unpack", cmd.Use)
	}

	// Check for required flags
	expectedFlags := []string{"password-env", "file", "target", "overwrite", "backup", "verify", "verbose", "dry-run", "include", "exclude"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Unpack command missing --%s flag", flag)
		}
	}
}

func TestNewListCommand(t *testing.T) {
	cmd := newListCommand()

	if cmd == nil {
		t.Fatal("newListCommand() returned nil")
	}

	if cmd.Use != "list" {
		t.Errorf("List command Use = %s, want list", cmd.Use)
	}

	// Check for required flags. --format is deliberately absent here: it is a
	// persistent flag on the root command now, so that every command offers
	// the same contract. Declaring it locally as well would shadow the
	// persistent one.
	expectedFlags := []string{"password-env", "file", "all", "verbose", "sizes", "dates", "checksums", "pattern", "sort", "reverse", "limit"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("List command missing --%s flag", flag)
		}
	}

	if cmd.Flags().Lookup("format") != nil {
		t.Error("List declares a local --format flag, which would shadow the persistent one")
	}
}

// TestRootCommand_FormatIsPersistent pins --format to the root command, where
// every subcommand inherits it.
func TestRootCommand_FormatIsPersistent(t *testing.T) {
	root := NewRootCommand("test")

	flag := root.PersistentFlags().Lookup("format")
	if flag == nil {
		t.Fatal("root command is missing the persistent --format flag")
	}
	if flag.DefValue != string(FormatText) {
		t.Errorf("--format default = %q, want %q", flag.DefValue, FormatText)
	}

	for _, name := range []string{"init", "pack", "unpack", "list", "status"} {
		var found bool
		for _, sub := range root.Commands() {
			if sub.Name() == name {
				found = true
				if sub.InheritedFlags().Lookup("format") == nil {
					t.Errorf("%s does not inherit --format", name)
				}
			}
		}
		if !found {
			t.Errorf("root command is missing the %s subcommand", name)
		}
	}
}

func TestNewStatusCommand(t *testing.T) {
	cmd := newStatusCommand()

	if cmd == nil {
		t.Fatal("newStatusCommand() returned nil")
	}

	if cmd.Name() != "status" {
		t.Errorf("Status command Name = %s, want status", cmd.Name())
	}

	// Check for required flags
	expectedFlags := []string{"verbose"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Status command missing --%s flag", flag)
		}
	}
}

func TestNewApp(t *testing.T) {
	// Save and change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "goingenv-cli-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir) //nolint:errcheck // cleanup in defer
		_ = os.RemoveAll(tmpDir)
	}()

	if chdirErr := os.Chdir(tmpDir); chdirErr != nil {
		t.Fatalf("Failed to change directory: %v", chdirErr)
	}

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	if app == nil {
		t.Fatal("NewApp() returned nil app")
	}

	if app.Config == nil {
		t.Error("NewApp() returned app with nil Config")
	}

	if app.Scanner == nil {
		t.Error("NewApp() returned app with nil Scanner")
	}

	if app.Archiver == nil {
		t.Error("NewApp() returned app with nil Archiver")
	}

	if app.Crypto == nil {
		t.Error("NewApp() returned app with nil Crypto")
	}

	if app.ConfigMgr == nil {
		t.Error("NewApp() returned app with nil ConfigMgr")
	}
}

func TestDisplayFilesJSON(t *testing.T) {
	files := []types.EnvFile{
		{
			RelativePath: ".env",
			Size:         100,
			ModTime:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Checksum:     "abc123",
		},
	}

	// The payload now goes to an injectable writer, so this can assert on the
	// contract rather than just checking for a panic.
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriterFormat(&stdout, &stderr, false, "test", FormatJSON)

	payload := ListPayload{
		Archive: ListArchiveInfo{Name: "a.enc", Path: ".goingenv/a.enc"},
		Files:   toFileInfos(files),
		Count:   len(files),
	}
	if err := out.EmitJSON(payload); err != nil {
		t.Fatalf("EmitJSON() error = %v", err)
	}

	// Round-trips as JSON: the whole point of the format.
	var decoded ListPayload
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("payload is not valid JSON: %v\ngot: %q", err, stdout.String())
	}
	if decoded.Count != 1 || len(decoded.Files) != 1 || decoded.Files[0].Path != ".env" {
		t.Errorf("decoded payload does not match input: %+v", decoded)
	}
	if stderr.Len() != 0 {
		t.Errorf("JSON payload wrote to stderr: %q", stderr.String())
	}
}

func TestDisplayFilesCSV(t *testing.T) {
	files := []types.EnvFile{
		{
			RelativePath: ".env",
			Size:         100,
			ModTime:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Checksum:     "abc123",
		},
	}

	// CSV goes to the payload stream, so it can now be asserted on rather
	// than merely checked for not panicking.
	var stdout, stderr bytes.Buffer
	out := NewOutputWithWriterFormat(&stdout, &stderr, false, "test", FormatCSV)

	displayFilesCSV(out, files)

	got := stdout.String()
	wantHeader := "name,path,size,modified,checksum"
	if !strings.HasPrefix(got, wantHeader) {
		t.Errorf("CSV output does not start with the header row.\ngot: %q", got)
	}
	if !strings.Contains(got, ".env") || !strings.Contains(got, "100") || !strings.Contains(got, "abc123") {
		t.Errorf("CSV output is missing file fields.\ngot: %q", got)
	}
	if stderr.Len() != 0 {
		t.Errorf("CSV output wrote to stderr: %q", stderr.String())
	}
}

func TestBuildScanOpts_EnvExcludeAppendsConfigPatterns(t *testing.T) {
	cfg := &types.Config{
		DefaultDepth:       7,
		EnvPatterns:        []string{`\.env.*`},
		EnvExcludePatterns: []string{`\.env\.example$`},
		ExcludePatterns:    []string{`node_modules/`},
		MaxFileSize:        1024,
	}
	p := &PackOpts{Dir: ".", EnvExclude: []string{`\.env\.backup$`}, Exclude: []string{`dist/`}}

	opts := buildScanOpts(p, cfg)

	// A flag must never be able to drop a configured exclusion: both narrow
	// the scan, so the result is the union.
	wantEnvExclude := map[string]bool{`\.env\.backup$`: false, `\.env\.example$`: false}
	for _, got := range opts.EnvExcludePatterns {
		if _, ok := wantEnvExclude[got]; !ok {
			t.Errorf("unexpected EnvExcludePattern %q", got)
			continue
		}
		if wantEnvExclude[got] {
			t.Errorf("EnvExcludePattern %q appears more than once", got)
		}
		wantEnvExclude[got] = true
	}
	for pattern, seen := range wantEnvExclude {
		if !seen {
			t.Errorf("EnvExcludePatterns missing %q", pattern)
		}
	}

	if len(opts.ExcludePatterns) != 2 {
		t.Errorf("ExcludePatterns = %v, want the flag and config patterns exactly once each", opts.ExcludePatterns)
	}
}

func TestBuildScanOpts_DoesNotAliasConfigSlices(t *testing.T) {
	cfg := &types.Config{
		DefaultDepth:       7,
		EnvPatterns:        []string{`\.env.*`},
		EnvExcludePatterns: []string{`\.env\.example$`},
		ExcludePatterns:    []string{`node_modules/`},
		MaxFileSize:        1024,
	}

	// No flags at all is the case that used to alias the config's own slices,
	// so appending to the ScanOptions could write into the loaded config.
	opts := buildScanOpts(&PackOpts{Dir: "."}, cfg)

	opts.ExcludePatterns = append(opts.ExcludePatterns, "scratch/")
	opts.EnvExcludePatterns = append(opts.EnvExcludePatterns, "scratch")

	if len(cfg.ExcludePatterns) != 1 {
		t.Errorf("cfg.ExcludePatterns = %v, want it untouched by appends to ScanOptions", cfg.ExcludePatterns)
	}
	if len(cfg.EnvExcludePatterns) != 1 {
		t.Errorf("cfg.EnvExcludePatterns = %v, want it untouched by appends to ScanOptions", cfg.EnvExcludePatterns)
	}
}

func TestBuildScanOpts_FallsBackToConfigDepthAndPatterns(t *testing.T) {
	cfg := &types.Config{
		DefaultDepth:    7,
		EnvPatterns:     []string{`\.env.*`},
		ExcludePatterns: []string{`node_modules/`},
		MaxFileSize:     1024,
	}

	opts := buildScanOpts(&PackOpts{Dir: "."}, cfg)

	if opts.MaxDepth != 7 {
		t.Errorf("MaxDepth = %d, want the config's 7", opts.MaxDepth)
	}
	if len(opts.Patterns) != 1 || opts.Patterns[0] != `\.env.*` {
		t.Errorf("Patterns = %v, want the config's patterns", opts.Patterns)
	}
}
