package cli_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"goingenv/test/testutils"
)

// The two tests in this file are the only coverage anywhere for:
//
//   - the default all-inclusive `\.env.*` pattern, asserted by count against
//     the built-in default config (every other pattern test supplies its own
//     explicit anchored patterns, so it would still pass if the default were
//     changed or removed); and
//   - EnvExcludePatterns having any effect at all, and more generally the fact
//     that a user-level ~/.goingenv.json changes what the CLI does.
//
// They were previously a bash target (`make test-functional`) that deleted the
// developer's real ~/.goingenv.json, wrote its own, and restored the backup
// only on the success path -- so any failing assertion left the config gone.
// Here HOME is redirected at a temp directory instead.

// envFixture is the file set the functional test used: six files the
// all-inclusive pattern must match, plus one it must not.
var envFixture = map[string]string{
	".env":             "TEST=value\n",
	".env.local":       "LOCAL=test\n",
	".env.development": "DEV=true\n",
	".env.custom":      "CUSTOM=value\n",
	".env.backup":      "BACKUP=old\n",
	".env.new_format":  "NEW=format\n",
	"regular.txt":      "IGNORED=value\n",
}

// setupEnvFixture creates the fixture files in a fresh directory that already
// has a .goingenv/ dir, and returns that directory plus an isolated HOME.
func setupEnvFixture(t *testing.T) (projectDir, homeDir string) {
	t.Helper()

	projectDir = t.TempDir()
	for name, content := range envFixture {
		path := filepath.Join(projectDir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("Failed to write fixture %s: %v", name, err)
		}
	}

	testutils.CreateTempGoingEnvDir(t, projectDir)

	return projectDir, t.TempDir()
}

// TestConfig_DefaultAllInclusivePattern checks that with no user config
// present, the built-in default (`\.env.*`) matches every .env.* file
// regardless of suffix -- including suffixes no fixture elsewhere covers, like
// .env.custom and .env.new_format -- and does not match a non-env file.
func TestConfig_DefaultAllInclusivePattern(t *testing.T) {
	projectDir, homeDir := setupEnvFixture(t)

	// No .goingenv.json in this HOME, so the binary falls back to the
	// built-in default config.
	if _, err := os.Stat(filepath.Join(homeDir, ".goingenv.json")); !os.IsNotExist(err) {
		t.Fatalf("Expected no config in the isolated home, got err=%v", err)
	}

	result := testutils.RunCLIWithEnv(t, projectDir, map[string]string{"HOME": homeDir}, "status")

	testutils.AssertSuccess(t, result)
	testutils.AssertStdoutContains(t, result, "Environment Files (6)")
	testutils.AssertOutputNotContains(t, result, "regular.txt")
}

// TestConfig_EnvExcludePatternsFromHomeConfig checks that env_exclude_patterns
// in the user-level ~/.goingenv.json actually removes a file the env_patterns
// entry matched. This is also the only test proving the CLI reads that file.
func TestConfig_EnvExcludePatternsFromHomeConfig(t *testing.T) {
	projectDir, homeDir := setupEnvFixture(t)

	// Same config the functional test wrote: all-inclusive matching, minus
	// .env.backup.
	config := `{
  "default_depth": 10,
  "env_patterns": ["\\.env.*"],
  "env_exclude_patterns": ["\\.env\\.backup$"],
  "exclude_patterns": ["node_modules/", "\\.git/"],
  "max_file_size": 10485760
}`
	configPath := filepath.Join(homeDir, ".goingenv.json")
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	result := testutils.RunCLIWithEnv(t, projectDir, map[string]string{"HOME": homeDir}, "status")

	testutils.AssertSuccess(t, result)

	// The count alone could be right for the wrong reason, so assert that the
	// excluded file specifically is gone and the others specifically remain.
	testutils.AssertStdoutContains(t, result, "Environment Files (5)")
	testutils.AssertOutputNotContains(t, result, ".env.backup")

	for _, name := range []string{".env.local", ".env.development", ".env.custom", ".env.new_format"} {
		testutils.AssertStdoutContains(t, result, name)
	}
}

// TestConfig_HomeConfigIsNotRewrittenByInit guards the fix for `goingenv init`
// saving the user-level config on every run. It used to call Save
// unconditionally, so running init in any project rewrote ~/.goingenv.json.
func TestConfig_HomeConfigIsNotRewrittenByInit(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()

	custom := `{
  "default_depth": 3,
  "env_patterns": ["\\.env$"],
  "env_exclude_patterns": [],
  "exclude_patterns": [],
  "max_file_size": 1024
}`
	configPath := filepath.Join(homeDir, ".goingenv.json")
	if err := os.WriteFile(configPath, []byte(custom), 0o600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	result := testutils.RunCLIWithEnv(t, projectDir, map[string]string{"HOME": homeDir}, "init")
	testutils.AssertSuccess(t, result)

	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config after init: %v", err)
	}
	if string(after) != custom {
		t.Errorf("init rewrote the user config.\nBefore:\n%s\n\nAfter:\n%s", custom, string(after))
	}
}

// TestConfig_InitCreatesHomeConfigWhenMissing is the other half: init should
// still write a default config the first time, when none exists.
func TestConfig_InitCreatesHomeConfigWhenMissing(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	configPath := filepath.Join(homeDir, ".goingenv.json")

	result := testutils.RunCLIWithEnv(t, projectDir, map[string]string{"HOME": homeDir}, "init")
	testutils.AssertSuccess(t, result)

	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("Expected init to create %s, got err=%v", configPath, err)
	}
}

// TestConfig_ScanDepthFromHomeConfig checks that another field of the
// user-level config -- default_depth -- reaches the scanner, so the wiring
// covered above is not specific to the exclude patterns.
func TestConfig_ScanDepthFromHomeConfig(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()

	// One .env at the root, one three levels down.
	if err := os.WriteFile(filepath.Join(projectDir, ".env"), []byte("ROOT=1\n"), 0o600); err != nil {
		t.Fatalf("Failed to write root .env: %v", err)
	}
	deep := filepath.Join(projectDir, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o750); err != nil {
		t.Fatalf("Failed to create nested dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(deep, ".env"), []byte("DEEP=1\n"), 0o600); err != nil {
		t.Fatalf("Failed to write nested .env: %v", err)
	}

	testutils.CreateTempGoingEnvDir(t, projectDir)

	writeDepth := func(depth int) {
		config := fmt.Sprintf(`{
  "default_depth": %d,
  "env_patterns": ["\\.env.*"],
  "env_exclude_patterns": [],
  "exclude_patterns": [],
  "max_file_size": 10485760
}`, depth)
		if err := os.WriteFile(filepath.Join(homeDir, ".goingenv.json"), []byte(config), 0o600); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}
	}

	writeDepth(1)
	shallow := testutils.RunCLIWithEnv(t, projectDir, map[string]string{"HOME": homeDir}, "status")
	testutils.AssertSuccess(t, shallow)
	testutils.AssertStdoutContains(t, shallow, "Environment Files (1)")

	writeDepth(10)
	full := testutils.RunCLIWithEnv(t, projectDir, map[string]string{"HOME": homeDir}, "status")
	testutils.AssertSuccess(t, full)
	testutils.AssertStdoutContains(t, full, "Environment Files (2)")
}

// TestConfig_ProjectConfigShadowsHomeConfig checks the precedence order: a
// committed .goingenv/config.json wins over the user's ~/.goingenv.json, so a
// repository can pin how it is scanned regardless of personal configuration.
//
// The home config here excludes .env.backup (the 5-file case above); the
// project config re-enables it. Getting 6 back proves the project file is the
// one being read, and that precedence is whole-file rather than per-key.
func TestConfig_ProjectConfigShadowsHomeConfig(t *testing.T) {
	projectDir, homeDir := setupEnvFixture(t)

	homeConfig := `{
  "default_depth": 10,
  "env_patterns": ["\\.env.*"],
  "env_exclude_patterns": ["\\.env\\.backup$"],
  "exclude_patterns": ["node_modules/", "\\.git/"],
  "max_file_size": 10485760
}`
	if err := os.WriteFile(filepath.Join(homeDir, ".goingenv.json"), []byte(homeConfig), 0o600); err != nil {
		t.Fatalf("Failed to write home config: %v", err)
	}

	projectConfig := `{
  "default_depth": 10,
  "env_patterns": ["\\.env.*"],
  "env_exclude_patterns": [],
  "exclude_patterns": ["node_modules/", "\\.git/"],
  "max_file_size": 10485760
}`
	projectConfigPath := filepath.Join(projectDir, ".goingenv", "config.json")
	if err := os.WriteFile(projectConfigPath, []byte(projectConfig), 0o600); err != nil {
		t.Fatalf("Failed to write project config: %v", err)
	}

	result := testutils.RunCLIWithEnv(t, projectDir, map[string]string{"HOME": homeDir}, "status")

	testutils.AssertSuccess(t, result)
	testutils.AssertStdoutContains(t, result, "Environment Files (6)")
	testutils.AssertStdoutContains(t, result, ".env.backup")
}

// TestConfig_InitWritesProjectConfigOnlyWhenAsked checks that the project
// config is opt-in, and that once written it is never silently replaced --
// it is a committed file, so --force must not discard the repository's rules.
func TestConfig_InitWritesProjectConfigOnlyWhenAsked(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	env := map[string]string{"HOME": homeDir}
	projectConfigPath := filepath.Join(projectDir, ".goingenv", "config.json")

	// Plain init must not create one.
	testutils.AssertSuccess(t, testutils.RunCLIWithEnv(t, projectDir, env, "init"))
	if _, err := os.Stat(projectConfigPath); !os.IsNotExist(err) {
		t.Fatalf("init created a project config without --project-config (err=%v)", err)
	}

	// Opting in creates it.
	testutils.AssertSuccess(t, testutils.RunCLIWithEnv(t, projectDir, env, "init", "--force", "--project-config"))
	testutils.AssertFileExists(t, projectConfigPath)

	// A subsequent init must leave a modified project config alone.
	pinned := `{
  "default_depth": 4,
  "env_patterns": ["\\.env$"],
  "env_exclude_patterns": [],
  "exclude_patterns": [],
  "max_file_size": 2048
}`
	if err := os.WriteFile(projectConfigPath, []byte(pinned), 0o600); err != nil {
		t.Fatalf("Failed to overwrite project config: %v", err)
	}

	testutils.AssertSuccess(t, testutils.RunCLIWithEnv(t, projectDir, env, "init", "--force", "--project-config"))

	after, err := os.ReadFile(projectConfigPath)
	if err != nil {
		t.Fatalf("Failed to read project config: %v", err)
	}
	if string(after) != pinned {
		t.Errorf("init --force --project-config overwrote the committed project config:\n%s", after)
	}
}
