package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goingenv/test/testutils"
)

func TestInit_FreshDirectory(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	result := testutils.RunCLI(t, tmpDir, "init")

	testutils.AssertSuccess(t, result)
	testutils.AssertOutputContains(t, result, "initialized")

	// Verify .goingenv directory was created
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	testutils.AssertDirExists(t, goingenvDir)

	// Verify .gitignore was created
	gitignorePath := filepath.Join(goingenvDir, ".gitignore")
	testutils.AssertFileExists(t, gitignorePath)
}

func TestInit_AlreadyInitialized(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Initialize first time
	result1 := testutils.RunCLI(t, tmpDir, "init")
	testutils.AssertSuccess(t, result1)

	// Initialize second time - should be idempotent
	result2 := testutils.RunCLI(t, tmpDir, "init")
	testutils.AssertSuccess(t, result2)
	// May contain "already initialized" message or just succeed silently

	// Verify .goingenv still exists
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	testutils.AssertDirExists(t, goingenvDir)
}

func TestInit_WithForceFlag(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Initialize first time
	result1 := testutils.RunCLI(t, tmpDir, "init")
	testutils.AssertSuccess(t, result1)

	// Create a marker file to verify reinitialization
	markerPath := filepath.Join(tmpDir, ".goingenv", "marker.txt")
	if err := os.WriteFile(markerPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create marker file: %v", err)
	}

	// Reinitialize with --force
	result2 := testutils.RunCLI(t, tmpDir, "init", "--force")
	testutils.AssertSuccess(t, result2)

	// Verify .goingenv still exists
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	testutils.AssertDirExists(t, goingenvDir)
}

func TestInit_CreatesGitignore(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	result := testutils.RunCLI(t, tmpDir, "init")
	testutils.AssertSuccess(t, result)

	gitignorePath := filepath.Join(tmpDir, ".goingenv", ".gitignore")
	content := testutils.GetFileContent(t, gitignorePath)

	// Verify .gitignore exists and has some content
	// The gitignore should NOT have a line that ignores *.enc files for safe transfer
	// (*.enc may appear in comments explaining this)
	if len(content) == 0 {
		t.Error("Expected .gitignore to have content")
	}

	// Check that *.enc is not being ignored (should not have "*.enc" as an ignore rule)
	// A line starting with *.enc (not a comment) would ignore enc files
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "*.enc" {
			t.Error("Expected .gitignore to NOT ignore *.enc files")
		}
	}
}

func TestInit_OutputMessages(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	result := testutils.RunCLI(t, tmpDir, "init")

	testutils.AssertSuccess(t, result)
	// Verify informative output
	testutils.AssertOutputContains(t, result, ".goingenv")
}

func TestInit_DirectoryPermissions(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	result := testutils.RunCLI(t, tmpDir, "init")
	testutils.AssertSuccess(t, result)

	// Verify directory has correct permissions
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	info, err := os.Stat(goingenvDir)
	if err != nil {
		t.Fatalf("Failed to stat .goingenv directory: %v", err)
	}

	if !info.IsDir() {
		t.Errorf("Expected .goingenv to be a directory")
	}

	// Check permissions (should be readable/writable by owner)
	mode := info.Mode()
	if mode.Perm()&0700 != 0700 {
		t.Errorf("Expected .goingenv to have at least 0700 permissions, got %o", mode.Perm())
	}
}

func TestInit_InSubdirectory(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	result := testutils.RunCLI(t, subDir, "init")
	testutils.AssertSuccess(t, result)

	// Verify .goingenv was created in the subdirectory
	goingenvDir := filepath.Join(subDir, ".goingenv")
	testutils.AssertDirExists(t, goingenvDir)

	// Verify it was NOT created in parent
	parentGoingenvDir := filepath.Join(tmpDir, ".goingenv")
	testutils.AssertFileNotExists(t, parentGoingenvDir)
}

func TestInit_VerboseMode(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	result := testutils.RunCLI(t, tmpDir, "init", "--verbose")
	testutils.AssertSuccess(t, result)

	// Verbose mode should provide more detailed output
	// Just verify it completes successfully
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	testutils.AssertDirExists(t, goingenvDir)
}
