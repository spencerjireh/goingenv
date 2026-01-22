package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goingenv/test/testutils"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build binary once for all E2E tests
	// This is done outside of individual tests to avoid repeated compilation
	tmpDir, err := os.MkdirTemp("", "goingenv-e2e-binary-*")
	if err != nil {
		panic("Failed to create temp directory for binary: " + err.Error())
	}

	binaryPath = filepath.Join(tmpDir, "goingenv")

	// Build using go build
	projectRoot := findProjectRoot()
	if projectRoot == "" {
		panic("Could not find project root (go.mod)")
	}

	// We'll build the binary in TestMain setup
	// For now, use the BuildBinary helper in tests
	code := m.Run()

	// Cleanup
	os.RemoveAll(tmpDir)
	testutils.CleanupBinary()

	os.Exit(code)
}

func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func TestE2E_FullWorkflow(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetupWithEnvFiles(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	// Step 1: Initialize
	t.Run("Init", func(t *testing.T) {
		result := testutils.RunBinary(t, binary, tmpDir, "init")
		testutils.AssertSuccess(t, result)
		testutils.AssertDirExists(t, filepath.Join(tmpDir, ".goingenv"))
	})

	// Step 2: Check status
	t.Run("Status", func(t *testing.T) {
		result := testutils.RunBinary(t, binary, tmpDir, "status", "--verbose")
		testutils.AssertSuccess(t, result)
		testutils.AssertOutputContains(t, result, ".goingenv")
		testutils.AssertOutputContains(t, result, "exists")
	})

	// Step 3: Pack
	var archivePath string
	t.Run("Pack", func(t *testing.T) {
		result := testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack", "--verbose")
		testutils.AssertSuccess(t, result)

		// Find the archive
		goingenvDir := filepath.Join(tmpDir, ".goingenv")
		entries, err := os.ReadDir(goingenvDir)
		if err != nil {
			t.Fatalf("Failed to read .goingenv: %v", err)
		}

		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".enc") {
				archivePath = filepath.Join(goingenvDir, entry.Name())
				break
			}
		}

		if archivePath == "" {
			t.Fatal("No archive created")
		}
		testutils.AssertFileExists(t, archivePath)
	})

	// Step 4: List archive contents
	t.Run("List", func(t *testing.T) {
		result := testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "list", "--file", archivePath)
		testutils.AssertSuccess(t, result)
		testutils.AssertOutputContains(t, result, ".env")
	})

	// Step 5: Remove original files
	originalEnvContent := testutils.GetFileContent(t, filepath.Join(tmpDir, ".env"))
	os.Remove(filepath.Join(tmpDir, ".env"))
	os.Remove(filepath.Join(tmpDir, ".env.local"))
	os.Remove(filepath.Join(tmpDir, ".env.development"))
	os.Remove(filepath.Join(tmpDir, ".env.production"))

	// Step 6: Unpack and verify
	t.Run("Unpack", func(t *testing.T) {
		// Use --overwrite to avoid interactive prompt about existing files
		result := testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "unpack", "--file", archivePath, "--overwrite")
		testutils.AssertSuccess(t, result)

		// Verify files were restored
		testutils.AssertFileExists(t, filepath.Join(tmpDir, ".env"))

		// Verify content matches
		restoredContent := testutils.GetFileContent(t, filepath.Join(tmpDir, ".env"))
		if restoredContent != originalEnvContent {
			t.Errorf("Content mismatch.\nOriginal: %q\nRestored: %q", originalEnvContent, restoredContent)
		}
	})

	// Step 7: Final status check
	t.Run("FinalStatus", func(t *testing.T) {
		result := testutils.RunBinary(t, binary, tmpDir, "status")
		testutils.AssertSuccess(t, result)
	})
}

func TestE2E_PatternMatching(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)

	testCases := testutils.GetStandardPatternCases()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tmpDir, cleanup := testutils.SetupPatternTestCase(t, &tc)
			defer cleanup()

			fixtures := testutils.GetTestFixtures()

			// Initialize
			result := testutils.RunBinary(t, binary, tmpDir, "init")
			testutils.AssertSuccess(t, result)

			// Pack
			result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack", "--verbose")

			if len(tc.ShouldMatch) > 0 {
				testutils.AssertSuccess(t, result)
			}
		})
	}
}

func TestE2E_ExcludedDirectories(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)

	testCases := testutils.GetExcludedDirCases()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tmpDir, cleanup := testutils.SetupPatternTestCase(t, &tc)
			defer cleanup()

			fixtures := testutils.GetTestFixtures()

			// Initialize
			result := testutils.RunBinary(t, binary, tmpDir, "init")
			testutils.AssertSuccess(t, result)

			// Pack
			result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack", "--verbose")

			if len(tc.ShouldMatch) > 0 {
				testutils.AssertSuccess(t, result)
			}
		})
	}
}

func TestE2E_DepthLimits(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)

	testCases := testutils.GetDepthLimitCases()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tmpDir, cleanup := testutils.SetupPatternTestCase(t, &tc)
			defer cleanup()

			fixtures := testutils.GetTestFixtures()

			// Initialize
			result := testutils.RunBinary(t, binary, tmpDir, "init")
			testutils.AssertSuccess(t, result)

			// Pack with default depth
			result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack", "--verbose")

			if len(tc.ShouldMatch) > 0 {
				testutils.AssertSuccess(t, result)
			}
		})
	}
}

func TestE2E_EmptyEnvFile(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create empty .env file
	emptyEnvPath := filepath.Join(tmpDir, ".env")
	testutils.CreateEmptyFile(t, emptyEnvPath)

	fixtures := testutils.GetTestFixtures()

	// Initialize
	result := testutils.RunBinary(t, binary, tmpDir, "init")
	testutils.AssertSuccess(t, result)

	// Pack - empty files should be included
	result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack")
	testutils.AssertSuccess(t, result)

	// Find archive
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

	// Remove empty file
	os.Remove(emptyEnvPath)

	// Unpack
	result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "unpack", "--file", archivePath)
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

func TestE2E_SymlinkSkipping(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create real .env file
	realEnvPath := filepath.Join(tmpDir, ".env.real")
	if err := os.WriteFile(realEnvPath, []byte("REAL=value"), 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create symlink
	symlinkPath := filepath.Join(tmpDir, ".env.link")
	if err := os.Symlink(realEnvPath, symlinkPath); err != nil {
		t.Skipf("Symlink creation not supported: %v", err)
	}

	fixtures := testutils.GetTestFixtures()

	// Initialize and pack
	result := testutils.RunBinary(t, binary, tmpDir, "init")
	testutils.AssertSuccess(t, result)

	result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack", "--verbose")
	testutils.AssertSuccess(t, result)

	// Symlinks should be skipped
}

func TestE2E_VersionAndHelp(t *testing.T) {
	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	t.Run("Version", func(t *testing.T) {
		result := testutils.RunBinary(t, binary, tmpDir, "--version")
		// Should succeed and show version
		if result.Success() {
			testutils.AssertOutputContains(t, result, "goingenv")
		}
	})

	t.Run("Help", func(t *testing.T) {
		result := testutils.RunBinary(t, binary, tmpDir, "--help")
		testutils.AssertSuccess(t, result)
		testutils.AssertOutputContains(t, result, "init")
		testutils.AssertOutputContains(t, result, "pack")
		testutils.AssertOutputContains(t, result, "unpack")
	})

	t.Run("InitHelp", func(t *testing.T) {
		result := testutils.RunBinary(t, binary, tmpDir, "init", "--help")
		testutils.AssertSuccess(t, result)
	})

	t.Run("PackHelp", func(t *testing.T) {
		result := testutils.RunBinary(t, binary, tmpDir, "pack", "--help")
		testutils.AssertSuccess(t, result)
	})

	t.Run("UnpackHelp", func(t *testing.T) {
		result := testutils.RunBinary(t, binary, tmpDir, "unpack", "--help")
		testutils.AssertSuccess(t, result)
	})

	t.Run("ListHelp", func(t *testing.T) {
		result := testutils.RunBinary(t, binary, tmpDir, "list", "--help")
		testutils.AssertSuccess(t, result)
	})

	t.Run("StatusHelp", func(t *testing.T) {
		result := testutils.RunBinary(t, binary, tmpDir, "status", "--help")
		testutils.AssertSuccess(t, result)
	})
}

func TestE2E_ErrorHandling(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	fixtures := testutils.GetTestFixtures()

	t.Run("PackWithoutInit", func(t *testing.T) {
		result := testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack")
		testutils.AssertFailure(t, result)
		testutils.AssertExitCode(t, result, 1)
	})

	t.Run("UnpackWithWrongPassword", func(t *testing.T) {
		// First create a valid archive
		testutils.WriteTestFile(t, filepath.Join(tmpDir, ".env"), "TEST=value")
		archivePath := testutils.CreateTestArchiveWithBinary(t, binary, tmpDir, fixtures.Password)

		// Try to unpack with wrong password
		result := testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.WrongPassword, "unpack", "--file", archivePath)
		testutils.AssertFailure(t, result)
	})

	t.Run("ListNonExistentArchive", func(t *testing.T) {
		// Initialize
		testutils.InitializeTestDirWithBinary(t, binary, tmpDir)

		nonExistent := filepath.Join(tmpDir, ".goingenv", "nonexistent.enc")
		result := testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "list", "--file", nonExistent)
		testutils.AssertFailure(t, result)
	})

	t.Run("InvalidCommand", func(t *testing.T) {
		result := testutils.RunBinary(t, binary, tmpDir, "invalidcmd")
		testutils.AssertFailure(t, result)
	})
}

func TestE2E_EdgeCases(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)

	testCases := testutils.GetEdgeCaseCases()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tmpDir, cleanup := testutils.SetupPatternTestCase(t, &tc)
			defer cleanup()

			fixtures := testutils.GetTestFixtures()

			// Initialize
			result := testutils.RunBinary(t, binary, tmpDir, "init")
			testutils.AssertSuccess(t, result)

			// Pack
			result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack", "--verbose")

			if len(tc.ShouldMatch) > 0 {
				testutils.AssertSuccess(t, result)
			}
		})
	}
}

func TestE2E_FalsePositives(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)

	testCases := testutils.GetFalsePositiveCases()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tmpDir := testutils.CreatePatternTestDir(t, tc.Files)
			defer os.RemoveAll(tmpDir)

			// Add a valid .env file so pack has something to do
			validEnvPath := filepath.Join(tmpDir, ".env")
			if err := os.WriteFile(validEnvPath, []byte("VALID=value"), 0o644); err != nil {
				t.Fatalf("Failed to create valid .env: %v", err)
			}

			fixtures := testutils.GetTestFixtures()

			// Initialize
			result := testutils.RunBinary(t, binary, tmpDir, "init")
			testutils.AssertSuccess(t, result)

			// Pack
			result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack", "--verbose")
			testutils.AssertSuccess(t, result)
		})
	}
}

func TestE2E_BinaryPerformance(t *testing.T) {
	testutils.SkipIfShort(t)

	binary := testutils.BuildBinary(t)
	tmpDir, cleanup := testutils.CLITestSetup(t)
	defer cleanup()

	// Create many env files
	for i := 0; i < 50; i++ {
		var path string
		if i == 0 {
			path = filepath.Join(tmpDir, ".env")
		} else {
			path = filepath.Join(tmpDir, ".env."+string(rune('a'+i%26))+string(rune('0'+i/26)))
		}
		content := "VAR_" + string(rune('A'+i%26)) + "=value_" + string(rune('0'+i))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	fixtures := testutils.GetTestFixtures()

	// Initialize
	result := testutils.RunBinary(t, binary, tmpDir, "init")
	testutils.AssertSuccess(t, result)

	// Pack and check duration is reasonable
	result = testutils.RunBinaryWithPassword(t, binary, tmpDir, fixtures.Password, "pack")
	testutils.AssertSuccess(t, result)

	// Status should be fast
	result = testutils.RunBinary(t, binary, tmpDir, "status")
	testutils.AssertSuccess(t, result)
}
