package cli_test

import (
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goingenv/test/testutils"
)

// archivesIn lists the .enc files under .goingenv, by base name.
func archivesIn(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, ".goingenv"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".enc") {
			names = append(names, e.Name())
		}
	}
	return names
}

func TestPack_EnvNamesTheArchive(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()
	testutils.InitializeTestDir(t, dir)
	pw := testutils.GetTestFixtures().Password

	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, pw, "pack", "--env", "prod"))

	names := archivesIn(t, dir)
	if len(names) != 1 || !strings.HasPrefix(names[0], "prod-") {
		t.Fatalf("archives = %v, want one prod-<timestamp>.enc", names)
	}

	result := testutils.RunCLIWithPassword(t, dir, pw, "pack", "--env", "Bad Name")
	testutils.AssertFailure(t, result)
	result = testutils.RunCLIWithPassword(t, dir, pw, "pack", "--env", "archive")
	testutils.AssertFailure(t, result)
	testutils.AssertStderrContains(t, result, "reserved")
}

func TestPack_ManifestListsKeysNotValues(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()
	testutils.InitializeTestDir(t, dir)
	pw := testutils.GetTestFixtures().Password

	result := testutils.RunCLIWithPassword(t, dir, pw, "pack", "--env", "prod", "--manifest", "--format", "json")
	testutils.AssertSuccess(t, result)
	payload := decodeStdout(t, result)
	manifestPath, ok := payload["manifest"].(string)
	if !ok || manifestPath == "" {
		t.Fatalf("pack payload has no manifest path: %v", payload)
	}
	if payload["env"] != "prod" {
		t.Errorf("env = %v", payload["env"])
	}

	raw, err := os.ReadFile(filepath.Join(dir, manifestPath))
	if err != nil {
		t.Fatalf("manifest not written: %v", err)
	}
	text := string(raw)
	for _, key := range []string{"DATABASE_URL", "API_KEY", "NESTED_VAR", `"env": "prod"`} {
		if !strings.Contains(text, key) {
			t.Errorf("manifest lacks %q", key)
		}
	}
	for _, value := range []string{"postgres://", "test123", "mysecret", "deep_value"} {
		if strings.Contains(text, value) {
			t.Errorf("manifest leaks the value %q", value)
		}
	}
	var m struct {
		Files []struct {
			Path   string   `json:"path"`
			SHA256 string   `json:"sha256"`
			Keys   []string `json:"keys"`
		} `json:"files"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if len(m.Files) != 7 {
		t.Errorf("manifest lists %d files, want 7", len(m.Files))
	}
	for _, f := range m.Files {
		if len(f.SHA256) != 64 {
			t.Errorf("%s sha256 = %q", f.Path, f.SHA256)
		}
	}
}

func TestPack_ManifestFlagsOverrideConfig(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()
	testutils.InitializeTestDir(t, dir)
	pw := testutils.GetTestFixtures().Password
	cfg := filepath.Join(dir, ".goingenv", "config.json")
	if err := os.WriteFile(cfg, []byte(`{"manifest": true}`), 0o600); err != nil {
		t.Fatal(err)
	}

	// Config default on: a plain pack writes one.
	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, pw, "pack", "-o", "with.enc"))
	testutils.AssertFileExists(t, filepath.Join(dir, ".goingenv", "with.enc.manifest.json"))

	// --no-manifest wins over the config.
	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, pw, "pack", "-o", "without.enc", "--no-manifest"))
	testutils.AssertFileNotExists(t, filepath.Join(dir, ".goingenv", "without.enc.manifest.json"))

	// Both flags are contradictory.
	result := testutils.RunCLIWithPassword(t, dir, pw, "pack", "--manifest", "--no-manifest")
	testutils.AssertFailure(t, result)

	// status reports the effective value.
	status := testutils.RunCLI(t, dir, "status", "--format", "json")
	testutils.AssertSuccess(t, status)
	config, ok := decodeStdout(t, status)["config"].(map[string]any)
	if !ok || config["manifest"] != true {
		t.Errorf("status config.manifest = %v, want true", config["manifest"])
	}
}

func TestPasswordStdin_PackAndUnpack(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()
	testutils.InitializeTestDir(t, dir)

	result := testutils.RunCLIWithStdin(t, dir, "stdin-pw\n", nil, "pack", "--password-stdin")
	testutils.AssertSuccess(t, result)
	if len(archivesIn(t, dir)) != 1 {
		t.Fatal("pack with --password-stdin created no archive")
	}

	// Wrong password on stdin must fail; right one must unpack.
	bad := testutils.RunCLIWithStdin(t, dir, "other\n", nil, "unpack", "--password-stdin", "--target", "out-bad")
	testutils.AssertFailure(t, bad)
	good := testutils.RunCLIWithStdin(t, dir, "stdin-pw\n", nil, "unpack", "--password-stdin", "--target", "out")
	testutils.AssertSuccess(t, good)
	testutils.AssertFileExists(t, filepath.Join(dir, "out", ".env"))

	// Empty stdin is an error, not a prompt.
	empty := testutils.RunCLIWithStdin(t, dir, "", nil, "unpack", "--password-stdin", "--target", "out2")
	testutils.AssertFailure(t, empty)
	testutils.AssertStderrContains(t, empty, "stdin")
}

func TestPasswordEnvDefaults(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()
	testutils.InitializeTestDir(t, dir)

	// GOINGENV_PASSWORD is consulted without --password-env.
	result := testutils.RunCLIWithEnv(t, dir, map[string]string{"GOINGENV_PASSWORD": "default-pw"}, "pack")
	testutils.AssertSuccess(t, result)

	// GOINGENV_PASSWORD_PROD wins for --env prod, GOINGENV_PASSWORD for the rest.
	env := map[string]string{"GOINGENV_PASSWORD": "default-pw", "GOINGENV_PASSWORD_PROD": "prod-pw"}
	testutils.AssertSuccess(t, testutils.RunCLIWithEnv(t, dir, env, "pack", "--env", "prod"))
	testutils.AssertSuccess(t, testutils.RunCLIWithEnv(t, dir, map[string]string{"GOINGENV_PASSWORD": "prod-pw"}, "list", "--env", "prod"))
	testutils.AssertFailure(t, testutils.RunCLIWithEnv(t, dir, map[string]string{"GOINGENV_PASSWORD": "default-pw"}, "list", "--env", "prod"))
	testutils.AssertSuccess(t, testutils.RunCLIWithEnv(t, dir, map[string]string{"GOINGENV_PASSWORD": "default-pw"}, "list"))
}

func TestUnpack_EnvPicksItsOwnNewest(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()
	testutils.InitializeTestDir(t, dir)
	pw := testutils.GetTestFixtures().Password

	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, pw, "pack", "--env", "prod"))
	time.Sleep(1100 * time.Millisecond) // archive names have second resolution
	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, pw, "pack"))

	names := archivesIn(t, dir)
	if len(names) != 2 {
		t.Fatalf("archives = %v", names)
	}

	result := testutils.RunCLIWithPassword(t, dir, pw, "unpack", "--env", "prod", "--format", "json", "--target", "out")
	testutils.AssertSuccess(t, result)
	if got, ok := decodeStdout(t, result)["archive"].(string); !ok || !strings.Contains(got, "/prod-") {
		t.Errorf("unpack --env prod used %s", got)
	}

	result = testutils.RunCLIWithPassword(t, dir, pw, "unpack", "--format", "json", "--target", "out2")
	testutils.AssertSuccess(t, result)
	if got, ok := decodeStdout(t, result)["archive"].(string); !ok || !strings.Contains(got, "/archive-") {
		t.Errorf("unnamed unpack used %s", got)
	}

	result = testutils.RunCLIWithPassword(t, dir, pw, "unpack", "--env", "staging")
	testutils.AssertFailure(t, result)
	testutils.AssertStderrContains(t, result, "staging")

	// list --all --env narrows the listing.
	all := testutils.RunCLIWithPassword(t, dir, pw, "list", "--all", "--env", "prod")
	testutils.AssertSuccess(t, all)
	testutils.AssertOutputContains(t, all, "Archives (1)")
}

func TestLegacyArchiveIsRejectedWithRemedy(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()
	testutils.InitializeTestDir(t, dir)
	pw := testutils.GetTestFixtures().Password

	legacy := make([]byte, 200)
	if _, err := rand.Read(legacy); err != nil {
		t.Fatal(err)
	}
	if string(legacy[:4]) == "GENV" {
		legacy[0] ^= 0xFF
	}
	path := filepath.Join(dir, ".goingenv", "archive-20200101-000000.enc")
	if err := os.WriteFile(path, legacy, 0o600); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"unpack"},
		{"list", "-f", path},
		{"diff", path},
		{"run", "--", "true"},
	} {
		result := testutils.RunCLIWithPassword(t, dir, pw, args...)
		testutils.AssertFailure(t, result)
		testutils.AssertExitCode(t, result, 1)
		testutils.AssertStderrContains(t, result, "predates format v1")
		testutils.AssertStderrContains(t, result, "v1.6.0")
	}

	// In a machine format the remedy lands in the JSON error on stderr.
	result := testutils.RunCLIWithPassword(t, dir, pw, "unpack", "--format", "json")
	testutils.AssertFailure(t, result)
	testutils.AssertStdoutEmpty(t, result)
	testutils.AssertStderrContains(t, result, "v1.6.0")
}

// Without --verbose nothing is decrypted, so an unusable password source
// must not fail the listing.
func TestList_AllWithoutVerboseNeedsNoPassword(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()
	testutils.InitializeTestDir(t, dir)
	pw := testutils.GetTestFixtures().Password
	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, pw, "pack", "-o", "a.enc"))

	unsetEnv := testutils.RunCLI(t, dir, "list", "--all", "--password-env", "GOINGENV_TEST_UNSET_VAR")
	testutils.AssertSuccess(t, unsetEnv)
	testutils.AssertStdoutContains(t, unsetEnv, "a.enc")

	emptyStdin := testutils.RunCLIWithStdin(t, dir, "", nil, "list", "--all", "--password-stdin")
	testutils.AssertSuccess(t, emptyStdin)
	testutils.AssertStdoutContains(t, emptyStdin, "a.enc")

	// An ambient password is not read either, so no warning about it appears.
	ambient := testutils.RunCLIWithPassword(t, dir, pw, "list", "--all")
	testutils.AssertSuccess(t, ambient)
	if strings.Contains(ambient.Stderr, "GOINGENV_PASSWORD") {
		t.Errorf("a plain --all listing warned about a password it never used:\n%s", ambient.Stderr)
	}
}

func TestList_AllVerboseReadsStdinOnce(t *testing.T) {
	dir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()
	testutils.InitializeTestDir(t, dir)
	pw := testutils.GetTestFixtures().Password

	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, pw, "pack", "-o", "a.enc"))
	testutils.AssertSuccess(t, testutils.RunCLIWithPassword(t, dir, pw, "pack", "-o", "b.enc"))

	result := testutils.RunCLIWithStdin(t, dir, pw+"\n", nil, "list", "--all", "--verbose", "--password-stdin")
	testutils.AssertSuccess(t, result)
	if got := strings.Count(result.Stdout, "Files: 7"); got != 2 {
		t.Errorf("expected both archives described from one stdin line, got %d:\n%s", got, result.Stdout)
	}
}
