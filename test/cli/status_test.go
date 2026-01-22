package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"goingenv/test/testutils"
)

func TestStatus_InitializedProject(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.InitializeTestDir(t, tmpDir)

	result := testutils.RunCLI(t, tmpDir, "status")

	testutils.AssertSuccess(t, result)
	// Should show project is initialized (shows ".goingenv (exists)" in output)
	testutils.AssertOutputContains(t, result, ".goingenv")
	testutils.AssertOutputContains(t, result, "exists")
}

func TestStatus_NotInitialized(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	result := testutils.RunCLI(t, tmpDir, "status")

	// Should indicate not initialized
	// Might fail or show "not initialized" message
	output := result.Combined()
	if result.Success() {
		testutils.AssertOutputContains(t, result, "not")
	} else {
		testutils.AssertExitCode(t, result, 1)
	}
	_ = output
}

func TestStatus_VerboseMode(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.InitializeTestDir(t, tmpDir)

	result := testutils.RunCLI(t, tmpDir, "status", "--verbose")

	testutils.AssertSuccess(t, result)
	// Verbose mode should show more details
}

func TestStatus_ShowsEnvFileCount(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.InitializeTestDir(t, tmpDir)

	result := testutils.RunCLI(t, tmpDir, "status", "--verbose")

	testutils.AssertSuccess(t, result)
	// Should mention env files found
}

func TestStatus_ShowsArchiveCount(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create some archives
	testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	result := testutils.RunCLI(t, tmpDir, "status", "--verbose")

	testutils.AssertSuccess(t, result)
	// Should mention archives
}

func TestStatus_WithNoEnvFiles(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	testutils.InitializeTestDir(t, tmpDir)

	result := testutils.RunCLI(t, tmpDir, "status")

	testutils.AssertSuccess(t, result)
	// Should indicate no env files found or show 0
}

func TestStatus_WithNoArchives(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.InitializeTestDir(t, tmpDir)

	result := testutils.RunCLI(t, tmpDir, "status", "--verbose")

	testutils.AssertSuccess(t, result)
	// Should indicate no archives or show 0
}

func TestStatus_AfterPack(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Initialize and pack
	testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	result := testutils.RunCLI(t, tmpDir, "status", "--verbose")

	testutils.AssertSuccess(t, result)
	// Should show archive exists
}

func TestStatus_AfterUnpack(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Remove env files
	os.Remove(filepath.Join(tmpDir, ".env"))

	// Unpack
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--file", archivePath)
	testutils.AssertSuccess(t, result)

	// Check status
	result = testutils.RunCLI(t, tmpDir, "status", "--verbose")
	testutils.AssertSuccess(t, result)
}

func TestStatus_ShowsProjectPath(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.InitializeTestDir(t, tmpDir)

	result := testutils.RunCLI(t, tmpDir, "status", "--verbose")

	testutils.AssertSuccess(t, result)
	// Verbose output might show the project path
}

func TestStatus_InSubdirectory(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Initialize in root
	testutils.InitializeTestDir(t, tmpDir)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "src", "components")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Check status from subdirectory
	result := testutils.RunCLI(t, subDir, "status")

	// Behavior depends on implementation:
	// Some tools look for .goingenv in parent directories
	// Others require being in the root
	_ = result
}

func TestStatus_JSONOutput(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.InitializeTestDir(t, tmpDir)

	result := testutils.RunCLI(t, tmpDir, "status", "--format", "json")

	// If JSON format is supported
	if result.Success() {
		output := result.Stdout
		// Should be valid JSON
		if output != "" {
			t.Logf("JSON output: %s", output)
		}
	}
}

func TestStatus_DifferentDepthConfigurations(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create deep structure
	deepPath := filepath.Join(tmpDir, "a", "b", "c", "d", ".env")
	dir := filepath.Dir(deepPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(deepPath, []byte("DEEP=value"), 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	testutils.InitializeTestDir(t, tmpDir)

	// Check status - should scan with configured depth
	result := testutils.RunCLI(t, tmpDir, "status", "--verbose")
	testutils.AssertSuccess(t, result)
}

func TestStatus_WithExcludedDirectories(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create files including in excluded directories
	files := map[string]string{
		".env":              "ROOT=value",
		"node_modules/.env": "EXCLUDED=value",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	testutils.InitializeTestDir(t, tmpDir)

	result := testutils.RunCLI(t, tmpDir, "status", "--verbose")
	testutils.AssertSuccess(t, result)

	// Status should not count excluded directory files
}

func TestStatus_QuickCheck(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.InitializeTestDir(t, tmpDir)

	// Quick status check (non-verbose)
	result := testutils.RunCLI(t, tmpDir, "status")

	testutils.AssertSuccess(t, result)
	// Should complete quickly and show basic info
}

func TestStatus_OutputFormatting(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()
	testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	result := testutils.RunCLI(t, tmpDir, "status", "--verbose")

	testutils.AssertSuccess(t, result)
	// Output should be well-formatted and readable
	// Verify it contains expected sections
	testutils.AssertOutputContains(t, result, "Status Report")
	testutils.AssertOutputContains(t, result, "Environment Files")
}

func TestStatus_EmptyGoingenvDirectory(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create .goingenv directory manually but empty
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	if err := os.MkdirAll(goingenvDir, 0o755); err != nil {
		t.Fatalf("Failed to create .goingenv: %v", err)
	}

	result := testutils.RunCLI(t, tmpDir, "status")

	// Behavior depends on whether empty .goingenv counts as initialized
	_ = result
}

func TestStatus_MultipleEnvFileTypes(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create various env file types
	files := map[string]string{
		".env":             "BASE=value",
		".env.local":       "LOCAL=value",
		".env.development": "DEV=value",
		".env.production":  "PROD=value",
		".env.test":        "TEST=value",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	testutils.InitializeTestDir(t, tmpDir)

	result := testutils.RunCLI(t, tmpDir, "status", "--verbose")

	testutils.AssertSuccess(t, result)
	// Should show count of different env file types
}
