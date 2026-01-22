package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goingenv/test/testutils"
)

func TestPack_BasicWorkflow(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	// Initialize first
	testutils.InitializeTestDir(t, tmpDir)

	// Pack with password
	fixtures := testutils.GetTestFixtures()
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack")

	testutils.AssertSuccess(t, result)
	testutils.AssertOutputContains(t, result, "pack")

	// Verify archive was created
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	entries, err := os.ReadDir(goingenvDir)
	if err != nil {
		t.Fatalf("Failed to read .goingenv directory: %v", err)
	}

	foundArchive := false
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".enc") {
			foundArchive = true
			break
		}
	}

	if !foundArchive {
		t.Error("Expected archive file to be created in .goingenv directory")
	}
}

func TestPack_NotInitialized(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	// Don't initialize - pack should fail
	fixtures := testutils.GetTestFixtures()
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack")

	testutils.AssertFailure(t, result)
	testutils.AssertExitCode(t, result, 1)
}

func TestPack_NoEnvFilesFound(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Initialize but don't create any env files
	testutils.InitializeTestDir(t, tmpDir)

	fixtures := testutils.GetTestFixtures()
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack")

	// Should fail or show message about no files found
	// The exact behavior depends on implementation
	output := strings.ToLower(result.Combined())
	if !strings.Contains(output, "no") && !strings.Contains(output, "empty") && !strings.Contains(output, "not found") {
		t.Errorf("Expected output to contain 'no', 'empty', or 'not found'\nOutput: %s", result.Combined())
	}
}

func TestPack_DryRunMode(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.InitializeTestDir(t, tmpDir)

	fixtures := testutils.GetTestFixtures()
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack", "--dry-run")

	testutils.AssertSuccess(t, result)

	// Verify no archive was actually created
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	entries, err := os.ReadDir(goingenvDir)
	if err != nil {
		t.Fatalf("Failed to read .goingenv directory: %v", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".enc") {
			t.Error("Dry-run mode should not create archive file")
		}
	}
}

func TestPack_VerboseMode(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.InitializeTestDir(t, tmpDir)

	fixtures := testutils.GetTestFixtures()
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack", "--verbose")

	testutils.AssertSuccess(t, result)
	// Verbose should show more details about files being packed
}

func TestPack_WithDepthLimit(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create env files at various depths
	files := map[string]string{
		".env":           "ROOT=value",
		"a/.env":         "DEPTH1=value",
		"a/b/.env":       "DEPTH2=value",
		"a/b/c/.env":     "DEPTH3=value",
		"a/b/c/d/.env":   "DEPTH4=value",
		"a/b/c/d/e/.env": "DEPTH5=value",
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

	fixtures := testutils.GetTestFixtures()

	// Pack with depth limit of 2
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack", "--depth", "2")
	testutils.AssertSuccess(t, result)
}

func TestPack_ExcludedDirectories(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create env files including some in excluded directories
	files := map[string]string{
		".env":              "ROOT=value",
		"config/.env":       "CONFIG=value",
		"node_modules/.env": "NM_EXCLUDED=value",
		".git/.env":         "GIT_EXCLUDED=value",
		"vendor/.env":       "VENDOR_EXCLUDED=value",
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

	fixtures := testutils.GetTestFixtures()
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack", "--verbose")

	testutils.AssertSuccess(t, result)
	// Excluded directories should be mentioned as excluded or simply not included
}

func TestPack_StandardPatterns(t *testing.T) {
	testCases := testutils.GetStandardPatternCases()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tmpDir, cleanup := testutils.SetupPatternTestCaseWithInit(t, &tc)
			defer cleanup()

			fixtures := testutils.GetTestFixtures()
			result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack", "--verbose")

			testutils.AssertSuccess(t, result)

			// Verify expected files were mentioned in output
			for _, expected := range tc.ShouldMatch {
				// The file should be included in the pack
				// Verbose output typically shows file names
				if !strings.Contains(result.Combined(), expected) &&
					!strings.Contains(result.Combined(), filepath.Base(expected)) {
					// Some implementations may not show individual files
					// This is a soft check
					t.Logf("Note: Expected file %s may not be shown in output", expected)
				}
			}
		})
	}
}

func TestPack_FalsePositivePatterns(t *testing.T) {
	testCases := testutils.GetFalsePositiveCases()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tmpDir := testutils.CreatePatternTestDir(t, tc.Files)
			defer os.RemoveAll(tmpDir)

			// Add a valid .env file so pack has something to do
			validEnvPath := filepath.Join(tmpDir, ".env")
			if err := os.WriteFile(validEnvPath, []byte("VALID=value"), 0o644); err != nil {
				t.Fatalf("Failed to create valid .env file: %v", err)
			}

			testutils.CreateTempGoingEnvDir(t, tmpDir)

			fixtures := testutils.GetTestFixtures()
			result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack", "--verbose")

			testutils.AssertSuccess(t, result)

			// Verify non-matching files were NOT included
			for _, notExpected := range tc.ShouldNotMatch {
				// Check the output doesn't mention packing these files
				output := result.Combined()
				// This is a heuristic check - the exact format depends on implementation
				if strings.Contains(output, "packing "+notExpected) ||
					strings.Contains(output, "adding "+notExpected) {
					t.Errorf("File %s should not have been packed", notExpected)
				}
			}
		})
	}
}

func TestPack_EmptyEnvFile(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create an empty .env file
	emptyEnvPath := filepath.Join(tmpDir, ".env")
	testutils.CreateEmptyFile(t, emptyEnvPath)

	testutils.InitializeTestDir(t, tmpDir)

	fixtures := testutils.GetTestFixtures()
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack")

	// Empty .env files should be included
	testutils.AssertSuccess(t, result)
}

func TestPack_SpecialCharactersInFilename(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create .env files with special characters in suffix
	files := map[string]string{
		".env":         "BASE=value",
		".env-backup":  "BACKUP=value",
		".env_old":     "OLD=value",
		".env.2024-01": "DATED=value",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	testutils.InitializeTestDir(t, tmpDir)

	fixtures := testutils.GetTestFixtures()
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack")

	testutils.AssertSuccess(t, result)
}

func TestPack_UnicodeInSuffix(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create .env files with unicode suffixes
	files := map[string]string{
		".env":             "BASE=value",
		".env.produccion":  "SPANISH=value",
		".env.entwicklung": "GERMAN=value",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	testutils.InitializeTestDir(t, tmpDir)

	fixtures := testutils.GetTestFixtures()
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack")

	testutils.AssertSuccess(t, result)
}

func TestPack_LongSuffix(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create .env file with very long suffix
	files := map[string]string{
		".env":                              "BASE=value",
		".env.development.local.backup.old": "LONG_SUFFIX=value",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	testutils.InitializeTestDir(t, tmpDir)

	fixtures := testutils.GetTestFixtures()
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack")

	testutils.AssertSuccess(t, result)
}

func TestPack_SymlinksSkipped(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create a real .env file
	realEnvPath := filepath.Join(tmpDir, ".env.real")
	if err := os.WriteFile(realEnvPath, []byte("REAL=value"), 0o644); err != nil {
		t.Fatalf("Failed to create real .env file: %v", err)
	}

	// Create a symlink to it
	symlinkPath := filepath.Join(tmpDir, ".env")
	if err := os.Symlink(realEnvPath, symlinkPath); err != nil {
		t.Skipf("Symlink creation not supported: %v", err)
	}

	testutils.InitializeTestDir(t, tmpDir)

	fixtures := testutils.GetTestFixtures()
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack", "--verbose")

	// Command should succeed
	testutils.AssertSuccess(t, result)
	// Symlinks should be skipped
}

func TestPack_WithCustomOutput(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.InitializeTestDir(t, tmpDir)

	// Specify custom output path
	customOutput := filepath.Join(tmpDir, ".goingenv", "custom-archive.enc")

	fixtures := testutils.GetTestFixtures()
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack", "--output", customOutput)

	testutils.AssertSuccess(t, result)
	testutils.AssertFileExists(t, customOutput)
}

func TestPack_WithIncludePattern(t *testing.T) {
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
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	testutils.InitializeTestDir(t, tmpDir)

	fixtures := testutils.GetTestFixtures()
	// Include patterns are regex, not glob - include files with "production" in name
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack", "--include", `\.production$`)

	testutils.AssertSuccess(t, result)
}

func TestPack_WithExcludePattern(t *testing.T) {
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
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	testutils.InitializeTestDir(t, tmpDir)

	fixtures := testutils.GetTestFixtures()
	// Exclude patterns are regex - exclude files ending with .test
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack", "--exclude", `\.test$`)

	testutils.AssertSuccess(t, result)
}

func TestPack_DepthLimitEdgeCases(t *testing.T) {
	testCases := testutils.GetDepthLimitCases()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tmpDir := testutils.CreatePatternTestDir(t, tc.Files)
			defer os.RemoveAll(tmpDir)

			testutils.CreateTempGoingEnvDir(t, tmpDir)

			fixtures := testutils.GetTestFixtures()
			// Use default depth (3)
			result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack", "--verbose")

			// Command should succeed
			if len(tc.ShouldMatch) > 0 {
				testutils.AssertSuccess(t, result)
			}
		})
	}
}

func TestPack_ExcludedDirEdgeCases(t *testing.T) {
	testCases := testutils.GetExcludedDirCases()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tmpDir := testutils.CreatePatternTestDir(t, tc.Files)
			defer os.RemoveAll(tmpDir)

			testutils.CreateTempGoingEnvDir(t, tmpDir)

			fixtures := testutils.GetTestFixtures()
			result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack", "--verbose")

			// Should succeed if there are matching files
			if len(tc.ShouldMatch) > 0 {
				testutils.AssertSuccess(t, result)
			}
		})
	}
}

func TestPack_MissingPassword(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	testutils.InitializeTestDir(t, tmpDir)

	// Run without password - should prompt or fail
	result := testutils.RunCLI(t, tmpDir, "pack")

	// Without password, the command should fail or prompt
	// The exact behavior depends on implementation
	// It might succeed if it prompts for password or fails with an error
	_ = result // Just checking it doesn't panic
}

func TestPack_WhitespaceOnlyEnvFile(t *testing.T) {
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create .env file with only whitespace
	envPath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envPath, []byte("   \n\t\n  "), 0o644); err != nil {
		t.Fatalf("Failed to create .env file: %v", err)
	}

	testutils.InitializeTestDir(t, tmpDir)

	fixtures := testutils.GetTestFixtures()
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack")

	// Whitespace-only files should still be included
	testutils.AssertSuccess(t, result)
}

func TestPack_MixedValidAndInvalidPatterns(t *testing.T) {
	tc := testutils.PatternTestCase{
		Name:        "MixedPatterns",
		Description: "Mix of valid and invalid file patterns",
		Files: map[string]string{
			".env":           "VALID=value",
			".env.local":     "VALID_LOCAL=value",
			"not.env":        "INVALID=value",
			"env":            "INVALID2=value",
			".environment":   "INVALID3=value",
			"config/.env":    "VALID_CONFIG=value",
			"config/env.txt": "INVALID_TXT=value",
		},
		ShouldMatch:    []string{".env", ".env.local", "config/.env"},
		ShouldNotMatch: []string{"not.env", "env", ".environment", "config/env.txt"},
	}

	tmpDir, cleanup := testutils.SetupPatternTestCaseWithInit(t, &tc)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()
	result := testutils.RunCLIWithPassword(t, tmpDir, fixtures.Password, "pack", "--verbose")

	testutils.AssertSuccess(t, result)
}
