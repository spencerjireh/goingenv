package cli_test

import (
	"encoding/json"
	"strings"
	"testing"

	"goingenv/test/testutils"
)

// These tests drive the real binary, because the property being asserted is
// about process streams: in a machine format stdout carries the payload and
// nothing else, while every human message goes to stderr.
//
// That was not true before. `goingenv list --format json` printed the branded
// header, a blank line and the archive summary to stdout ahead of the JSON, so
// the one documented machine format could not be piped into jq at all.

const machinePassword = "test-password-123"

// decodeStdout fails the test with the raw bytes when stdout is not valid
// JSON, which is far easier to diagnose than an unmarshal error alone.
func decodeStdout(t *testing.T, result testutils.CLIResult) map[string]any {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal([]byte(result.Stdout), &payload); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n--- stdout ---\n%s\n--- stderr ---\n%s",
			err, result.Stdout, result.Stderr)
	}
	return payload
}

// assertCleanStdout is the core assertion: no branding on stdout.
func assertCleanStdout(t *testing.T, result testutils.CLIResult) {
	t.Helper()

	if strings.Contains(result.Stdout, "goingenv v") {
		t.Errorf("the banner leaked onto stdout, which breaks piping:\n%s", result.Stdout)
	}
}

func TestMachine_StatusJSON(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.AssertSuccess(t, testutils.RunCLI(t, dir, "init"))

	result := testutils.RunCLI(t, dir, "status", "--format", "json")
	testutils.AssertSuccess(t, result)
	assertCleanStdout(t, result)

	payload := decodeStdout(t, result)

	for _, key := range []string{"directory", "initialized", "config", "env_files", "archives", "total_size"} {
		if _, ok := payload[key]; !ok {
			t.Errorf("status payload is missing the %q field", key)
		}
	}

	if payload["initialized"] != true {
		t.Errorf("initialized = %v, want true after init", payload["initialized"])
	}

	// The config tier is the thing a script most needs: a project-local
	// .goingenv/config.json takes whole-file precedence over the user one.
	cfg, ok := payload["config"].(map[string]any)
	if !ok {
		t.Fatalf("config is %T, want an object", payload["config"])
	}
	if cfg["source"] != "user" && cfg["source"] != "project" {
		t.Errorf("config.source = %v, want \"user\" or \"project\"", cfg["source"])
	}

	// Empty collections must serialise as [] rather than null, so consumers
	// can iterate without a nil check.
	if _, ok := payload["archives"].([]any); !ok {
		t.Errorf("archives is %T, want an array even when empty", payload["archives"])
	}
}

func TestMachine_StatusPorcelain(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.AssertSuccess(t, testutils.RunCLI(t, dir, "init"))

	result := testutils.RunCLI(t, dir, "status", "--format", "porcelain")
	testutils.AssertSuccess(t, result)
	assertCleanStdout(t, result)

	lines := nonEmptyLines(result.Stdout)
	if len(lines) == 0 {
		t.Fatal("porcelain status produced no records")
	}

	for _, line := range lines {
		fields := strings.Split(line, "\t")
		// kind, path, size, modified
		if len(fields) != 4 {
			t.Errorf("record has %d tab-separated fields, want 4: %q", len(fields), line)
			continue
		}
		if fields[0] != "env" && fields[0] != "archive" {
			t.Errorf("record kind = %q, want \"env\" or \"archive\": %q", fields[0], line)
		}
	}
}

func TestMachine_ListJSONIsPipeable(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.AssertSuccess(t, testutils.RunCLI(t, dir, "init"))
	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, machinePassword, "pack"))

	archive := findArchive(t, dir)

	result := testutils.RunCLIWithPassword(t, dir, machinePassword,
		"list", "--file", archive, "--format", "json")
	testutils.AssertSuccess(t, result)
	assertCleanStdout(t, result)

	payload := decodeStdout(t, result)

	files, ok := payload["files"].([]any)
	if !ok {
		t.Fatalf("files is %T, want an array", payload["files"])
	}
	if len(files) == 0 {
		t.Error("the archive was packed with files but list reported none")
	}
	if _, ok := payload["archive"].(map[string]any); !ok {
		t.Errorf("archive is %T, want an object", payload["archive"])
	}
}

func TestMachine_PackJSON(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.AssertSuccess(t, testutils.RunCLI(t, dir, "init"))

	result := testutils.RunCLIWithPassword(t, dir, machinePassword, "pack", "--format", "json")
	testutils.AssertSuccess(t, result)
	assertCleanStdout(t, result)

	payload := decodeStdout(t, result)

	if payload["dry_run"] != false {
		t.Errorf("dry_run = %v, want false for a real pack", payload["dry_run"])
	}
	archive, ok := payload["archive"].(string)
	if !ok || archive == "" {
		t.Errorf("pack payload does not name the archive it created: %v", payload["archive"])
	}
	count, ok := payload["count"].(float64)
	if !ok || count == 0 {
		t.Errorf("pack reported no files despite the fixture having some: %v", payload["count"])
	}
}

func TestMachine_InitJSON(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	first := testutils.RunCLI(t, dir, "init", "--format", "json")
	testutils.AssertSuccess(t, first)
	assertCleanStdout(t, first)

	if decodeStdout(t, first)["created"] != true {
		t.Error("created = false on a fresh init")
	}

	// Re-running is a success, not an error. The distinction belongs in the
	// payload so a script can act on it without parsing prose.
	second := testutils.RunCLI(t, dir, "init", "--format", "json")
	testutils.AssertSuccess(t, second)
	if decodeStdout(t, second)["created"] != false {
		t.Error("created = true when the project was already initialised")
	}
}

// TestMachine_ErrorsGoToStderr pins the failure contract: stdout stays empty
// and parseable, the error is structured, and the exit code is non-zero.
func TestMachine_ErrorsGoToStderr(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	// Deliberately not initialised.
	result := testutils.RunCLI(t, dir, "status", "--format", "json")

	if result.ExitCode == 0 {
		t.Error("expected a non-zero exit code for an uninitialised project")
	}
	if strings.TrimSpace(result.Stdout) != "" {
		t.Errorf("stdout must stay empty on failure, got: %q", result.Stdout)
	}

	// The error is reported exactly once, as JSON, on stderr.
	if !strings.Contains(result.Stderr, `"error"`) {
		t.Errorf("stderr does not carry a JSON error object:\n%s", result.Stderr)
	}
	if n := strings.Count(result.Stderr, "not initialized"); n != 1 {
		t.Errorf("the error is reported %d times, want exactly 1:\n%s", n, result.Stderr)
	}
}

// TestMachine_InvalidFormatFailsLoudly guards against the worst outcome:
// silently falling back to human text, which hands a script unparseable output
// with a zero exit code.
func TestMachine_InvalidFormatFailsLoudly(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	result := testutils.RunCLI(t, dir, "status", "--format", "xml")

	if result.ExitCode == 0 {
		t.Error("an unrecognised --format exited 0")
	}
	if strings.TrimSpace(result.Stdout) != "" {
		t.Errorf("stdout must stay empty, got: %q", result.Stdout)
	}
	if !strings.Contains(result.Stderr, "unsupported format") {
		t.Errorf("stderr does not explain the rejection:\n%s", result.Stderr)
	}
}

// TestMachine_CSVIsListOnly documents the one asymmetry in the flag: csv
// predates it and only `list` ever produced it.
func TestMachine_CSVIsListOnly(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.AssertSuccess(t, testutils.RunCLI(t, dir, "init"))

	rejected := testutils.RunCLI(t, dir, "status", "--format", "csv")
	if rejected.ExitCode == 0 {
		t.Error("status accepted --format csv, which it cannot produce")
	}

	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, machinePassword, "pack"))
	archive := findArchive(t, dir)

	accepted := testutils.RunCLIWithPassword(t, dir, machinePassword,
		"list", "--file", archive, "--format", "csv")
	testutils.AssertSuccess(t, accepted)
	if !strings.HasPrefix(accepted.Stdout, "name,path,size,modified,checksum") {
		t.Errorf("csv output lost its header row:\n%s", accepted.Stdout)
	}
}

// TestMachine_TableAliasStillWorks keeps the pre-existing `--format table`
// invocations working now that the flag is global.
func TestMachine_TableAliasStillWorks(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.AssertSuccess(t, testutils.RunCLI(t, dir, "init"))
	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, machinePassword, "pack"))

	archive := findArchive(t, dir)
	result := testutils.RunCLIWithPassword(t, dir, machinePassword,
		"list", "--file", archive, "--format", "table")

	testutils.AssertSuccess(t, result)
	// table is the human format, so the banner belongs on stdout here.
	if !strings.Contains(result.Stdout, "goingenv v") {
		t.Errorf("--format table did not produce the human view:\n%s", result.Stdout)
	}
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

// findArchive returns the archive `pack` just created, via the JSON payload
// rather than by globbing the directory.
func findArchive(t *testing.T, dir string) string {
	t.Helper()

	result := testutils.RunCLI(t, dir, "status", "--format", "json")
	testutils.AssertSuccess(t, result)

	archives, ok := decodeStdout(t, result)["archives"].([]any)
	if !ok || len(archives) == 0 {
		t.Fatalf("no archives reported by status:\n%s", result.Stdout)
	}

	first, ok := archives[0].(map[string]any)
	if !ok {
		t.Fatalf("archive entry is %T, want an object", archives[0])
	}
	path, ok := first["path"].(string)
	if !ok || path == "" {
		t.Fatalf("archive entry has no usable path: %v", first["path"])
	}
	return path
}
