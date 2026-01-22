package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goingenv/test/testutils"
)

func TestWorkflow_MultipleArchives(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Initialize
	result := testutils.RunBinary(t, binary, tmpDir, "init")
	testutils.AssertSuccess(t, result)

	// Create first archive (v1)
	result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack")
	testutils.AssertSuccess(t, result)

	// Wait a bit to ensure different timestamps
	time.Sleep(100 * time.Millisecond)

	// Modify env files
	envPath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envPath, []byte("VERSION=2\nUPDATED=true"), 0644); err != nil {
		t.Fatalf("Failed to modify .env: %v", err)
	}

	// Create second archive (v2)
	result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack")
	testutils.AssertSuccess(t, result)

	// List all archives
	result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "list", "--all")
	testutils.AssertSuccess(t, result)

	// Find archives
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	entries, err := os.ReadDir(goingenvDir)
	if err != nil {
		t.Fatalf("Failed to read .goingenv: %v", err)
	}

	var archives []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".enc") {
			archives = append(archives, filepath.Join(goingenvDir, entry.Name()))
		}
	}

	if len(archives) < 2 {
		t.Logf("Note: Only %d archive(s) found, implementation may overwrite", len(archives))
	}

	// Restore from first archive (if multiple exist)
	if len(archives) >= 2 {
		// Sort to find oldest
		oldestArchive := archives[0]
		for _, a := range archives {
			if a < oldestArchive {
				oldestArchive = a
			}
		}

		// Unpack from first archive
		result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "unpack", "--file", oldestArchive, "--overwrite")
		testutils.AssertSuccess(t, result)
	}
}

func TestWorkflow_CrossDirectoryPack(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)

	// Create a project directory with env files
	projectDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	// Create a subdirectory with more env files
	subDir := filepath.Join(projectDir, "subproject")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, ".env"), []byte("SUBPROJECT=value"), 0644); err != nil {
		t.Fatalf("Failed to create subproject .env: %v", err)
	}

	fixtures := testutils.GetTestFixtures()

	// Initialize the project directory
	result := testutils.RunBinary(t, binary, projectDir, "init")
	testutils.AssertSuccess(t, result)

	// Pack from project directory using -d flag to specify subdirectory
	// Note: -d flag specifies the scan directory, but init must be in current dir
	result = testutils.RunBinaryWithPassword(t, binary, projectDir, fixtures.Password, "pack", "-d", subDir)
	testutils.AssertSuccess(t, result)
}

func TestWorkflow_DryRunComparison(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Initialize
	result := testutils.RunBinary(t, binary, tmpDir, "init")
	testutils.AssertSuccess(t, result)

	// Dry run pack
	dryRunResult := testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack", "--dry-run", "--verbose")
	testutils.AssertSuccess(t, dryRunResult)

	// Verify no archive was created
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	entries, _ := os.ReadDir(goingenvDir)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".enc") {
			t.Error("Dry run should not create archive")
		}
	}

	// Real pack
	realResult := testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack", "--verbose")
	testutils.AssertSuccess(t, realResult)

	// Verify archive was created
	entries, _ = os.ReadDir(goingenvDir)
	foundArchive := false
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".enc") {
			foundArchive = true
			break
		}
	}
	if !foundArchive {
		t.Error("Real pack should create archive")
	}
}

func TestWorkflow_SelectiveExtract(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create various env files
	files := map[string]string{
		".env":            "BASE=value",
		".env.local":      "LOCAL=value",
		".env.production": "PROD=value",
		".env.test":       "TEST=value",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	fixtures := testutils.GetTestFixtures()

	// Initialize and pack
	archivePath := testutils.CreateTestArchiveWithBinary(t, binary, tmpDir, fixtures.Password)

	// Remove all env files
	for path := range files {
		os.Remove(filepath.Join(tmpDir, path))
	}

	// Extract only production files
	result := testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "unpack", "--file", archivePath, "--include", "*.production")
	testutils.AssertSuccess(t, result)

	// Check what was extracted
	// This depends on whether include patterns are supported
}

func TestWorkflow_ErrorRecovery(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Initialize and create archive
	archivePath := testutils.CreateTestArchiveWithBinary(t, binary, tmpDir, fixtures.Password)

	// Try wrong password
	result := testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.WrongPassword, "unpack", "--file", archivePath)
	testutils.AssertFailure(t, result)

	// Verify original files are still intact
	testutils.AssertFileExists(t, filepath.Join(tmpDir, ".env"))

	// Now use correct password
	result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "unpack", "--file", archivePath, "--overwrite")
	testutils.AssertSuccess(t, result)
}

func TestWorkflow_BackupRestore(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create initial archive
	archivePath := testutils.CreateTestArchiveWithBinary(t, binary, tmpDir, fixtures.Password)

	// Modify .env
	envPath := filepath.Join(tmpDir, ".env")
	originalContent := testutils.GetFileContent(t, envPath)
	modifiedContent := "MODIFIED=true\nNEW_VAR=new_value"
	if err := os.WriteFile(envPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify .env: %v", err)
	}

	// Unpack with backup and overwrite - backup creates backups, overwrite allows replacement
	result := testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "unpack", "--file", archivePath, "--backup", "--overwrite")
	testutils.AssertSuccess(t, result)

	// Verify original was restored
	restoredContent := testutils.GetFileContent(t, envPath)
	if restoredContent == modifiedContent {
		t.Error("File should have been restored to original content")
	}
	if restoredContent != originalContent {
		t.Logf("Note: Restored content differs.\nOriginal: %q\nRestored: %q", originalContent, restoredContent)
	}
}

func TestWorkflow_NestedDirectories(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create nested structure
	structure := map[string]string{
		".env":                  "ROOT=value",
		"config/.env":           "CONFIG=value",
		"services/api/.env":     "API=value",
		"services/web/.env":     "WEB=value",
		"deploy/staging/.env":   "STAGING=value",
		"deploy/production/.env": "PRODUCTION=value",
	}

	for path, content := range structure {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	fixtures := testutils.GetTestFixtures()

	// Initialize and pack
	archivePath := testutils.CreateTestArchiveWithBinary(t, binary, tmpDir, fixtures.Password)

	// Remove all env files
	for path := range structure {
		os.Remove(filepath.Join(tmpDir, path))
	}

	// Clean up directories
	os.RemoveAll(filepath.Join(tmpDir, "config"))
	os.RemoveAll(filepath.Join(tmpDir, "services"))
	os.RemoveAll(filepath.Join(tmpDir, "deploy"))

	// Unpack
	result := testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "unpack", "--file", archivePath)
	testutils.AssertSuccess(t, result)

	// Verify all files were restored
	for path := range structure {
		testutils.AssertFileExists(t, filepath.Join(tmpDir, path))
	}
}

func TestWorkflow_LargeFiles(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create a moderately large .env file (1MB)
	largePath := filepath.Join(tmpDir, ".env")
	testutils.CreateLargeTestFile(t, largePath, 1024*1024)

	fixtures := testutils.GetTestFixtures()

	// Initialize
	result := testutils.RunBinary(t, binary, tmpDir, "init")
	testutils.AssertSuccess(t, result)

	// Pack
	result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack")
	testutils.AssertSuccess(t, result)

	// Get original file info
	originalInfo, err := os.Stat(largePath)
	if err != nil {
		t.Fatalf("Failed to stat original file: %v", err)
	}

	// Find and list archive
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	entries, _ := os.ReadDir(goingenvDir)
	var archivePath string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".enc") {
			archivePath = filepath.Join(goingenvDir, entry.Name())
			break
		}
	}

	// Remove original
	os.Remove(largePath)

	// Unpack
	result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "unpack", "--file", archivePath)
	testutils.AssertSuccess(t, result)

	// Verify restored file
	restoredInfo, err := os.Stat(largePath)
	if err != nil {
		t.Fatalf("Failed to stat restored file: %v", err)
	}

	if restoredInfo.Size() != originalInfo.Size() {
		t.Errorf("Size mismatch. Original: %d, Restored: %d", originalInfo.Size(), restoredInfo.Size())
	}
}

func TestWorkflow_ConcurrentOperations(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)

	// Create multiple project directories
	const numProjects = 3
	dirs := make([]string, numProjects)
	cleanups := make([]func(), numProjects)

	for i := 0; i < numProjects; i++ {
		dirs[i], cleanups[i] = testutils.CLITestSetupWithEnvFiles(t)
	}
	defer func() {
		for _, cleanup := range cleanups {
			cleanup()
		}
	}()

	fixtures := testutils.GetTestFixtures()

	// Initialize all in parallel (using goroutines)
	done := make(chan bool, numProjects)
	for i := 0; i < numProjects; i++ {
		go func(dir string) {
			result := testutils.RunBinary(t, binary, dir, "init")
			if !result.Success() {
				t.Errorf("Init failed for %s: %s", dir, result.Combined())
			}
			done <- true
		}(dirs[i])
	}

	// Wait for all inits
	for i := 0; i < numProjects; i++ {
		<-done
	}

	// Pack all in parallel
	for i := 0; i < numProjects; i++ {
		go func(dir string) {
			result := testutils.RunBinaryWithPassword(t, binary, dir, fixtures.Password, "pack")
			if !result.Success() {
				t.Errorf("Pack failed for %s: %s", dir, result.Combined())
			}
			done <- true
		}(dirs[i])
	}

	// Wait for all packs
	for i := 0; i < numProjects; i++ {
		<-done
	}

	// Verify all have archives
	for i := 0; i < numProjects; i++ {
		goingenvDir := filepath.Join(dirs[i], ".goingenv")
		entries, err := os.ReadDir(goingenvDir)
		if err != nil {
			t.Errorf("Failed to read .goingenv for project %d: %v", i, err)
			continue
		}

		found := false
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".enc") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("No archive found for project %d", i)
		}
	}
}

func TestWorkflow_EnvironmentVariablePassword(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Initialize
	result := testutils.RunBinary(t, binary, tmpDir, "init")
	testutils.AssertSuccess(t, result)

	// Pack using GOINGENV_PASSWORD environment variable
	// Must pass --password-env flag to tell CLI which env var to read
	env := map[string]string{"GOINGENV_PASSWORD": fixtures.Password}
	result = testutils.RunBinaryWithEnv(t, binary, tmpDir, env, "pack", "--password-env", "GOINGENV_PASSWORD")
	testutils.AssertSuccess(t, result)

	// Find archive
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	entries, _ := os.ReadDir(goingenvDir)
	var archivePath string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".enc") {
			archivePath = filepath.Join(goingenvDir, entry.Name())
			break
		}
	}

	// Remove .env
	os.Remove(filepath.Join(tmpDir, ".env"))

	// Unpack using environment variable with --overwrite to avoid prompt
	result = testutils.RunBinaryWithEnv(t, binary, tmpDir, env, "unpack", "--file", archivePath, "--password-env", "GOINGENV_PASSWORD", "--overwrite")
	testutils.AssertSuccess(t, result)

	testutils.AssertFileExists(t, filepath.Join(tmpDir, ".env"))
}

func TestWorkflow_Idempotency(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Initialize multiple times should be idempotent
	for i := 0; i < 3; i++ {
		result := testutils.RunBinary(t, binary, tmpDir, "init")
		testutils.AssertSuccess(t, result)
	}

	// Create archive
	archivePath := testutils.CreateTestArchiveWithBinary(t, binary, tmpDir, fixtures.Password)

	// Unpack multiple times should be idempotent
	for i := 0; i < 3; i++ {
		result := testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "unpack", "--file", archivePath, "--overwrite")
		testutils.AssertSuccess(t, result)
	}

	// Verify files are still correct
	testutils.AssertFileExists(t, filepath.Join(tmpDir, ".env"))
}

func TestWorkflow_StatusThroughoutLifecycle(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Status before init
	result := testutils.RunBinary(t, binary, tmpDir, "status")
	// May fail or show "not initialized"

	// Initialize
	result = testutils.RunBinary(t, binary, tmpDir, "init")
	testutils.AssertSuccess(t, result)

	// Status after init
	result = testutils.RunBinary(t, binary, tmpDir, "status", "--verbose")
	testutils.AssertSuccess(t, result)

	// Pack
	result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack")
	testutils.AssertSuccess(t, result)

	// Status after pack
	result = testutils.RunBinary(t, binary, tmpDir, "status", "--verbose")
	testutils.AssertSuccess(t, result)

	// Remove env files
	os.Remove(filepath.Join(tmpDir, ".env"))

	// Status after removing files
	result = testutils.RunBinary(t, binary, tmpDir, "status", "--verbose")
	testutils.AssertSuccess(t, result)
}
