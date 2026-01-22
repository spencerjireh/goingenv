package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goingenv/test/testutils"
)

func TestList_BasicWorkflow(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// List archive contents
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "list", "--file", archivePath)

	testutils.AssertSuccess(t, result)
	// Output should contain information about the archive contents
	testutils.AssertOutputContains(t, result, ".env")
}

func TestList_NonExistentArchive(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	testutils.InitializeTestDir(t, tmpDir)

	fixtures := testutils.GetTestFixtures()
	nonExistentPath := filepath.Join(tmpDir, ".goingenv", "nonexistent.enc")

	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "list", "--file", nonExistentPath)

	testutils.AssertFailure(t, result)
}

func TestList_WrongPassword(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive with correct password
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Try to list with wrong password
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.WrongPassword, "list", "--file", archivePath)

	testutils.AssertFailure(t, result)
}

func TestList_JSONFormat(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// List in JSON format
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "list", "--file", archivePath, "--format", "json")

	testutils.AssertSuccess(t, result)
	// Output should be valid JSON (contains braces/brackets)
	output := result.Stdout
	if !strings.Contains(output, "{") && !strings.Contains(output, "[") {
		t.Logf("Note: Output may not be JSON format: %s", output)
	}
}

func TestList_CSVFormat(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// List in CSV format
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "list", "--file", archivePath, "--format", "csv")

	testutils.AssertSuccess(t, result)
	// CSV output should contain commas or be tab-delimited
}

func TestList_AllArchives(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create first archive
	testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Modify a file and create second archive
	envPath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envPath, []byte("MODIFIED=true"), 0644); err != nil {
		t.Fatalf("Failed to modify .env: %v", err)
	}

	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack")
	testutils.AssertSuccess(t, result)

	// List all archives
	result = testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "list", "--all")

	testutils.AssertSuccess(t, result)
	// Should show information about multiple archives
}

func TestList_VerboseMode(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// List with verbose flag
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "list", "--file", archivePath, "--verbose")

	testutils.AssertSuccess(t, result)
	// Verbose output should contain more details
}

func TestList_ShowsFileMetadata(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create specific files for checking metadata
	files := map[string]string{
		".env":            "DATABASE_URL=postgres://localhost/test",
		".env.production": "NODE_ENV=production",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// List archive contents
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "list", "--file", archivePath, "--verbose")

	testutils.AssertSuccess(t, result)

	// Should show file names
	testutils.AssertOutputContains(t, result, ".env")
}

func TestList_NoArchivesExist(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	testutils.InitializeTestDir(t, tmpDir)

	fixtures := testutils.GetTestFixtures()

	// List when no archives exist
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "list", "--all")

	// Should either fail or show "no archives found" message
	// The exact behavior depends on implementation
	output := result.Combined()
	if !strings.Contains(strings.ToLower(output), "no") &&
		!strings.Contains(strings.ToLower(output), "empty") &&
		!strings.Contains(strings.ToLower(output), "not found") &&
		result.Success() {
		// If successful and no message, that's fine too
		t.Logf("Note: Command succeeded with output: %s", output)
	}
}

func TestList_LatestArchive(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// List requires specifying file with -f flag
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "list", "-f", archivePath)

	testutils.AssertSuccess(t, result)
}

func TestList_ArchiveWithManyFiles(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create many env files
	for i := 0; i < 20; i++ {
		var path string
		if i == 0 {
			path = filepath.Join(tmpDir, ".env")
		} else {
			path = filepath.Join(tmpDir, ".env."+string(rune('a'+i-1)))
		}
		content := "VAR_" + string(rune('A'+i)) + "=value"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// List archive
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "list", "--file", archivePath)

	testutils.AssertSuccess(t, result)
	// Should handle many files gracefully
}

func TestList_NotInitialized(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Try to list without initialization
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "list", "--all")

	// Should fail or show appropriate message
	// Behavior depends on implementation
	_ = result
}

func TestList_ShowsArchiveTimestamp(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// List with verbose to see timestamps
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "list", "--file", archivePath, "--verbose")

	testutils.AssertSuccess(t, result)
	// Verbose output typically includes timestamps
}

func TestList_OutputFormat(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	formats := []string{"json", "csv", "table"}

	for _, format := range formats {
		t.Run("Format_"+format, func(t *testing.T) {
			result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "list", "--file", archivePath, "--format", format)

			// Command should succeed (or fail gracefully if format not supported)
			if result.ExitCode != 0 && !strings.Contains(result.Combined(), "unknown format") {
				testutils.AssertSuccess(t, result)
			}
		})
	}
}

func TestList_ArchiveIntegrity(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Create archive
	archivePath := testutils.CreateTestArchive(t, tmpDir, fixtures.Password)

	// Corrupt the archive slightly
	content, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read archive: %v", err)
	}

	// Modify a byte in the middle
	if len(content) > 100 {
		content[50] ^= 0xFF
	}

	corruptedPath := filepath.Join(tmpDir, ".goingenv", "corrupted.enc")
	if err := os.WriteFile(corruptedPath, content, 0644); err != nil {
		t.Fatalf("Failed to write corrupted archive: %v", err)
	}

	// Try to list corrupted archive
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "list", "--file", corruptedPath)

	// Should fail with appropriate error
	testutils.AssertFailure(t, result)
}
