package testutils

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings" // Used for string operations in assertions
	"sync"
	"testing"
	"time"
)

// CLIResult holds the result of a CLI command execution
type CLIResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Err      error
}

// Combined returns stdout and stderr combined
func (r CLIResult) Combined() string {
	return r.Stdout + r.Stderr
}

// Success returns true if the command exited with code 0
func (r CLIResult) Success() bool {
	return r.ExitCode == 0
}

var (
	// binaryPath caches the compiled binary path
	binaryPath     string
	binaryPathOnce sync.Once
	binaryBuildErr error
)

// RunCLI executes the goingenv CLI with the given arguments
// Uses a compiled binary for proper working directory support
func RunCLI(t *testing.T, workDir string, args ...string) CLIResult {
	t.Helper()
	binary := BuildBinary(t)
	return runBinaryCommand(t, binary, workDir, nil, args...)
}

// RunCLIWithEnv executes the goingenv CLI with custom environment variables
func RunCLIWithEnv(t *testing.T, workDir string, env map[string]string, args ...string) CLIResult {
	t.Helper()
	binary := BuildBinary(t)
	return runBinaryCommand(t, binary, workDir, env, args...)
}

// RunCLIWithPassword executes the goingenv CLI with GOINGENV_PASSWORD set
// It automatically adds --password-env GOINGENV_PASSWORD for pack/unpack/list commands
func RunCLIWithPassword(t *testing.T, workDir, password string, args ...string) CLIResult {
	t.Helper()
	env := map[string]string{"GOINGENV_PASSWORD": password}
	binary := BuildBinary(t)

	// Check if this is a command that needs password
	// Add --password-env GOINGENV_PASSWORD if not already specified
	if len(args) > 0 {
		cmd := args[0]
		needsPassword := cmd == "pack" || cmd == "unpack" || cmd == "list"

		if needsPassword {
			hasPasswordEnv := false
			for _, arg := range args {
				if arg == "--password-env" || strings.HasPrefix(arg, "--password-env=") {
					hasPasswordEnv = true
					break
				}
			}
			if !hasPasswordEnv {
				args = append(args, "--password-env", "GOINGENV_PASSWORD")
			}
		}
	}

	return runBinaryCommand(t, binary, workDir, env, args...)
}

// BuildBinary compiles the goingenv binary for E2E tests
// The binary is cached and reused across tests
func BuildBinary(t *testing.T) string {
	t.Helper()

	binaryPathOnce.Do(func() {
		// Get project root (go up from test directory)
		projectRoot, err := getProjectRoot()
		if err != nil {
			binaryBuildErr = err
			return
		}

		// Create temp directory for binary
		tmpDir, err := os.MkdirTemp("", "goingenv-binary-*")
		if err != nil {
			binaryBuildErr = err
			return
		}

		binaryPath = filepath.Join(tmpDir, "goingenv")

		// Build the binary
		cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/goingenv")
		cmd.Dir = projectRoot

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			binaryBuildErr = &BuildError{
				Err:    err,
				Stderr: stderr.String(),
			}
			return
		}
	})

	if binaryBuildErr != nil {
		t.Fatalf("Failed to build binary: %v", binaryBuildErr)
	}

	return binaryPath
}

// BuildError represents a binary build failure
type BuildError struct {
	Err    error
	Stderr string
}

func (e *BuildError) Error() string {
	return e.Err.Error() + ": " + e.Stderr
}

// RunBinary executes the compiled binary with the given arguments
func RunBinary(t *testing.T, binaryPath, workDir string, args ...string) CLIResult {
	t.Helper()
	return runBinaryCommand(t, binaryPath, workDir, nil, args...)
}

// RunBinaryWithEnv executes the compiled binary with custom environment variables
func RunBinaryWithEnv(t *testing.T, binaryPath, workDir string, env map[string]string, args ...string) CLIResult {
	t.Helper()
	return runBinaryCommand(t, binaryPath, workDir, env, args...)
}

// RunBinaryWithPassword executes the compiled binary with GOINGENV_PASSWORD set
// It automatically adds --password-env GOINGENV_PASSWORD for pack/unpack/list commands
func RunBinaryWithPassword(t *testing.T, binaryPath, workDir, password string, args ...string) CLIResult {
	t.Helper()
	env := map[string]string{"GOINGENV_PASSWORD": password}

	// Check if this is a command that needs password
	// Add --password-env GOINGENV_PASSWORD if not already specified
	if len(args) > 0 {
		cmd := args[0]
		needsPassword := cmd == "pack" || cmd == "unpack" || cmd == "list"

		if needsPassword {
			hasPasswordEnv := false
			for _, arg := range args {
				if arg == "--password-env" || strings.HasPrefix(arg, "--password-env=") {
					hasPasswordEnv = true
					break
				}
			}
			if !hasPasswordEnv {
				args = append(args, "--password-env", "GOINGENV_PASSWORD")
			}
		}
	}

	return runBinaryCommand(t, binaryPath, workDir, env, args...)
}

// runBinaryCommand runs the compiled binary with the provided arguments
func runBinaryCommand(t *testing.T, binaryPath, workDir string, env map[string]string, args ...string) CLIResult {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}

	// Set up environment
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	result := CLIResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
		Err:      err,
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	} else if err != nil {
		result.ExitCode = -1
	}

	return result
}

// getProjectRoot returns the root directory of the goingenv project
func getProjectRoot() (string, error) {
	// Start from current working directory and look for go.mod
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding go.mod
			break
		}
		dir = parent
	}

	// Fallback: try to find it relative to the test file
	// This handles cases where tests are run from various directories
	return filepath.Abs("../..")
}

// Assertion Helpers

// AssertExitCode checks that the CLI result has the expected exit code
func AssertExitCode(t *testing.T, result CLIResult, expected int) {
	t.Helper()
	if result.ExitCode != expected {
		t.Errorf("Expected exit code %d, got %d\nStdout: %s\nStderr: %s",
			expected, result.ExitCode, result.Stdout, result.Stderr)
	}
}

// AssertSuccess checks that the CLI command succeeded (exit code 0)
func AssertSuccess(t *testing.T, result CLIResult) {
	t.Helper()
	if !result.Success() {
		t.Errorf("Expected command to succeed, but got exit code %d\nStdout: %s\nStderr: %s",
			result.ExitCode, result.Stdout, result.Stderr)
	}
}

// AssertFailure checks that the CLI command failed (exit code != 0)
func AssertFailure(t *testing.T, result CLIResult) {
	t.Helper()
	if result.Success() {
		t.Errorf("Expected command to fail, but it succeeded\nStdout: %s\nStderr: %s",
			result.Stdout, result.Stderr)
	}
}

// AssertOutputContains checks that stdout or stderr contains the expected string
func AssertOutputContains(t *testing.T, result CLIResult, expected string) {
	t.Helper()
	combined := result.Combined()
	if !strings.Contains(combined, expected) {
		t.Errorf("Expected output to contain %q\nStdout: %s\nStderr: %s",
			expected, result.Stdout, result.Stderr)
	}
}

// AssertOutputNotContains checks that stdout and stderr do not contain the string
func AssertOutputNotContains(t *testing.T, result CLIResult, notExpected string) {
	t.Helper()
	combined := result.Combined()
	if strings.Contains(combined, notExpected) {
		t.Errorf("Expected output to NOT contain %q\nStdout: %s\nStderr: %s",
			notExpected, result.Stdout, result.Stderr)
	}
}

// AssertStdoutContains checks that stdout contains the expected string
func AssertStdoutContains(t *testing.T, result CLIResult, expected string) {
	t.Helper()
	if !strings.Contains(result.Stdout, expected) {
		t.Errorf("Expected stdout to contain %q\nStdout: %s",
			expected, result.Stdout)
	}
}

// AssertStderrContains checks that stderr contains the expected string
func AssertStderrContains(t *testing.T, result CLIResult, expected string) {
	t.Helper()
	if !strings.Contains(result.Stderr, expected) {
		t.Errorf("Expected stderr to contain %q\nStderr: %s",
			expected, result.Stderr)
	}
}

// AssertStdoutEmpty checks that stdout is empty
func AssertStdoutEmpty(t *testing.T, result CLIResult) {
	t.Helper()
	if strings.TrimSpace(result.Stdout) != "" {
		t.Errorf("Expected stdout to be empty, got: %s", result.Stdout)
	}
}

// AssertStderrEmpty checks that stderr is empty
func AssertStderrEmpty(t *testing.T, result CLIResult) {
	t.Helper()
	if strings.TrimSpace(result.Stderr) != "" {
		t.Errorf("Expected stderr to be empty, got: %s", result.Stderr)
	}
}

// CLITestSetup sets up a temporary directory for CLI testing
// Returns the temp dir path and a cleanup function
func CLITestSetup(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "goingenv-cli-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// CLITestSetupWithEnvFiles sets up a temp directory with sample .env files
func CLITestSetupWithEnvFiles(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir := CreateTempEnvFiles(t)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// InitializeTestDir initializes goingenv in the given directory
func InitializeTestDir(t *testing.T, dir string) {
	t.Helper()

	result := RunCLI(t, dir, "init")
	if !result.Success() {
		t.Fatalf("Failed to initialize test directory: %s", result.Combined())
	}
}

// InitializeTestDirWithBinary initializes goingenv using the compiled binary
func InitializeTestDirWithBinary(t *testing.T, binaryPath, dir string) {
	t.Helper()

	result := RunBinary(t, binaryPath, dir, "init")
	if !result.Success() {
		t.Fatalf("Failed to initialize test directory: %s", result.Combined())
	}
}

// CleanupBinary removes the cached binary (call in TestMain cleanup)
func CleanupBinary() {
	if binaryPath != "" {
		os.RemoveAll(filepath.Dir(binaryPath))
	}
}

// SkipIfShort skips the test if running in short mode
func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}
}

// CreateTestArchive creates a test archive in the given directory
// Returns the archive path
func CreateTestArchive(t *testing.T, dir, password string) string {
	t.Helper()

	// First initialize
	InitializeTestDir(t, dir)

	// Then pack
	result := RunCLIWithPassword(t, dir, password, "pack")
	if !result.Success() {
		t.Fatalf("Failed to create test archive: %s", result.Combined())
	}

	// Find the created archive
	goingenvDir := filepath.Join(dir, ".goingenv")
	entries, err := os.ReadDir(goingenvDir)
	if err != nil {
		t.Fatalf("Failed to read .goingenv directory: %v", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".enc") {
			return filepath.Join(goingenvDir, entry.Name())
		}
	}

	t.Fatalf("No archive found after pack command")
	return ""
}

// CreateTestArchiveWithBinary creates a test archive using the compiled binary
func CreateTestArchiveWithBinary(t *testing.T, binaryPath, dir, password string) string {
	t.Helper()

	// First initialize
	InitializeTestDirWithBinary(t, binaryPath, dir)

	// Then pack
	result := RunBinaryWithPassword(t, binaryPath, dir, password, "pack")
	if !result.Success() {
		t.Fatalf("Failed to create test archive: %s", result.Combined())
	}

	// Find the created archive
	goingenvDir := filepath.Join(dir, ".goingenv")
	entries, err := os.ReadDir(goingenvDir)
	if err != nil {
		t.Fatalf("Failed to read .goingenv directory: %v", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".enc") {
			return filepath.Join(goingenvDir, entry.Name())
		}
	}

	t.Fatalf("No archive found after pack command")
	return ""
}
