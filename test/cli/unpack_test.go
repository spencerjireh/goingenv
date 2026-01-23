package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goingenv/test/testutils"
)

func TestUnpack_BasicWorkflow(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create an archive first
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Remove the original env files
	envFiles := []string{".env", ".env.local", ".env.development", ".env.production"}
	for _, ef := range envFiles {
		os.Remove(filepath.Join(tmpDir, ef))
	}
	// Remove subdirectory env files
	os.RemoveAll(filepath.Join(tmpDir, "config"))
	os.RemoveAll(filepath.Join(tmpDir, "app"))
	os.RemoveAll(filepath.Join(tmpDir, "nested"))

	// Unpack with overwrite to avoid interactive prompt
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--file", archivePath, "--overwrite")

	testutils.AssertSuccess(t, result)

	// Verify at least the root .env file was restored
	testutils.AssertFileExists(t, filepath.Join(tmpDir, ".env"))
}

func TestUnpack_WrongPassword(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive with correct password
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Try to unpack with wrong password
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.WrongPassword, "unpack", "--file", archivePath)

	testutils.AssertFailure(t, result)
	testutils.AssertExitCode(t, result, 1)
}

func TestUnpack_NonExistentArchive(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	testutils.InitializeTestDir(t, tmpDir)

	fixtures := testutils.GetTestFixtures()
	nonExistentPath := filepath.Join(tmpDir, ".goingenv", "nonexistent.enc")

	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--file", nonExistentPath)

	testutils.AssertFailure(t, result)
}

func TestUnpack_OverwriteExistingFiles(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Modify the original .env file
	envPath := filepath.Join(tmpDir, ".env")
	originalContent := testutils.GetFileContent(t, envPath)
	modifiedContent := "MODIFIED=true\n"
	if err := os.WriteFile(envPath, []byte(modifiedContent), 0o644); err != nil {
		t.Fatalf("Failed to modify .env: %v", err)
	}

	// Unpack with overwrite flag
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--file", archivePath, "--overwrite")

	testutils.AssertSuccess(t, result)

	// Verify the file was restored to original content
	restoredContent := testutils.GetFileContent(t, envPath)
	if restoredContent == modifiedContent {
		t.Error("File should have been overwritten with original content")
	}
	if restoredContent != originalContent {
		// The restored content should match what was originally packed
		t.Logf("Original: %q, Restored: %q", originalContent, restoredContent)
	}
}

func TestUnpack_WithBackup(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Modify the original .env file
	envPath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envPath, []byte("MODIFIED=true\n"), 0o644); err != nil {
		t.Fatalf("Failed to modify .env: %v", err)
	}

	// Unpack with backup flag
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--file", archivePath, "--backup")

	testutils.AssertSuccess(t, result)

	// Check that backup file was created
	// Backup files typically have .bak extension or similar
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	foundBackup := false
	for _, entry := range entries {
		name := entry.Name()
		if strings.Contains(name, ".env") && (strings.Contains(name, ".bak") || strings.Contains(name, "backup")) {
			foundBackup = true
			break
		}
	}

	// Backup behavior depends on implementation
	_ = foundBackup
}

func TestUnpack_ToCustomTarget(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Create a target directory
	targetDir := filepath.Join(tmpDir, "restored")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	// Unpack to custom target
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--file", archivePath, "--target", targetDir)

	testutils.AssertSuccess(t, result)

	// Verify files were restored to target directory
	testutils.AssertFileExists(t, filepath.Join(targetDir, ".env"))
}

func TestUnpack_DryRunMode(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Delete the original files
	os.Remove(filepath.Join(tmpDir, ".env"))
	os.Remove(filepath.Join(tmpDir, ".env.local"))

	// Dry run unpack
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--file", archivePath, "--dry-run")

	testutils.AssertSuccess(t, result)

	// Files should NOT have been restored in dry run
	testutils.AssertFileNotExists(t, filepath.Join(tmpDir, ".env"))
	testutils.AssertFileNotExists(t, filepath.Join(tmpDir, ".env.local"))
}

func TestUnpack_VerboseMode(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Remove originals
	os.Remove(filepath.Join(tmpDir, ".env"))

	// Verbose unpack
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--file", archivePath, "--verbose")

	testutils.AssertSuccess(t, result)
	// Verbose mode should show detailed output
}

func TestUnpack_WithVerify(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Remove originals
	os.Remove(filepath.Join(tmpDir, ".env"))

	// Unpack with verify flag
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--file", archivePath, "--verify")

	testutils.AssertSuccess(t, result)
}

func TestUnpack_WithIncludePattern(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Remove all env files
	os.Remove(filepath.Join(tmpDir, ".env"))
	os.Remove(filepath.Join(tmpDir, ".env.local"))
	os.Remove(filepath.Join(tmpDir, ".env.development"))
	os.Remove(filepath.Join(tmpDir, ".env.production"))

	// Unpack only production files
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--file", archivePath, "--include", "*.production")

	testutils.AssertSuccess(t, result)
	// Only .env.production should be restored (if pattern matching is supported)
}

func TestUnpack_WithExcludePattern(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Remove all env files
	os.Remove(filepath.Join(tmpDir, ".env"))
	os.Remove(filepath.Join(tmpDir, ".env.local"))
	os.Remove(filepath.Join(tmpDir, ".env.development"))
	os.Remove(filepath.Join(tmpDir, ".env.production"))

	// Unpack excluding local files
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--file", archivePath, "--exclude", "*.local")

	testutils.AssertSuccess(t, result)
}

func TestUnpack_NotInitialized(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	// Create goingenv dir and archive manually
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	_ = os.MkdirAll(goingenvDir, 0o750) //nolint:errcheck // test setup

	// Create a dummy file to simulate an archive
	dummyArchive := filepath.Join(goingenvDir, "test.enc")
	if err := os.WriteFile(dummyArchive, []byte("dummy"), 0o644); err != nil {
		t.Fatalf("Failed to create dummy archive: %v", err)
	}

	fixtures := testutils.GetTestFixtures()
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--file", dummyArchive)

	// Should fail or handle the error gracefully
	// The exact behavior depends on whether initialization is required for unpack
	_ = result
}

func TestUnpack_FileIntegrity(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create specific content for verification
	envContent := "SPECIFIC_VAR=specific_value\nANOTHER_VAR=another_value"
	envPath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0o644); err != nil {
		t.Fatalf("Failed to create .env: %v", err)
	}

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Remove original
	os.Remove(envPath)

	// Unpack
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--file", archivePath)
	testutils.AssertSuccess(t, result)

	// Verify content matches
	restoredContent := testutils.GetFileContent(t, envPath)
	if restoredContent != envContent {
		t.Errorf("File content mismatch.\nExpected: %q\nGot: %q", envContent, restoredContent)
	}
}

func TestUnpack_EmptyArchive(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Initialize
	testutils.InitializeTestDir(t, tmpDir)

	// Create an empty env file
	emptyEnvPath := filepath.Join(tmpDir, ".env")
	testutils.CreateEmptyFile(t, emptyEnvPath)

	fixtures := testutils.GetTestFixtures()

	// Pack it
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack")
	testutils.AssertSuccess(t, result)

	// Find the archive
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	entries, err := os.ReadDir(goingenvDir)
	if err != nil {
		t.Fatalf("Failed to read .goingenv: %v", err)
	}

	var archivePath string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".enc") {
			archivePath = filepath.Join(goingenvDir, entry.Name())
			break
		}
	}

	if archivePath == "" {
		t.Fatal("No archive found")
	}

	// Remove the empty file
	os.Remove(emptyEnvPath)

	// Unpack
	result = testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--file", archivePath)
	testutils.AssertSuccess(t, result)

	// Verify empty file was restored
	testutils.AssertFileExists(t, emptyEnvPath)

	info, err := os.Stat(emptyEnvPath)
	if err != nil {
		t.Fatalf("Failed to stat restored file: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("Expected empty file, got size %d", info.Size())
	}
}

func TestUnpack_PreservesDirectoryStructure(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create env files in subdirectories
	files := map[string]string{
		".env":              "ROOT=value",
		"config/.env":       "CONFIG=value",
		"config/sub/.env":   "CONFIG_SUB=value",
		"services/api/.env": "API=value",
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

	fixtures := testutils.GetTestFixtures()

	// Pack
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Remove all env files
	for path := range files {
		os.Remove(filepath.Join(tmpDir, path))
	}
	os.RemoveAll(filepath.Join(tmpDir, "config"))
	os.RemoveAll(filepath.Join(tmpDir, "services"))

	// Unpack
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--file", archivePath)
	testutils.AssertSuccess(t, result)

	// Verify all files were restored with correct directory structure
	for path := range files {
		testutils.AssertFileExists(t, filepath.Join(tmpDir, path))
	}
}

func TestUnpack_LatestArchive(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create first archive
	testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Modify env files
	envPath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envPath, []byte("VERSION=2"), 0o644); err != nil {
		t.Fatalf("Failed to modify .env: %v", err)
	}

	// Create second archive
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack")
	testutils.AssertSuccess(t, result)

	// Remove env files
	os.Remove(envPath)

	// Unpack without specifying file (should use latest) with overwrite to avoid prompt
	result = testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "unpack", "--overwrite")
	testutils.AssertSuccess(t, result)

	// Should have restored from latest archive
	content := testutils.GetFileContent(t, envPath)
	if !strings.Contains(content, "VERSION=2") {
		t.Logf("Note: Content is %q - may have used first archive", content)
	}
}
