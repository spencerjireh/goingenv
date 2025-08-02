package integration

import (
	"os"
	"path/filepath"
	"testing"

	"goingenv/internal/archive"
	"goingenv/internal/config"
	"goingenv/internal/crypto"
	"goingenv/internal/scanner"
	"goingenv/pkg/types"
	"goingenv/test/testutils"
)

func TestFullWorkflow(t *testing.T) {
	// Setup
	tmpDir := testutils.CreateTempEnvFiles(t)
	defer os.RemoveAll(tmpDir)

	// Initialize services
	cfg := testutils.CreateTestConfig()
	cryptoService := crypto.NewService()
	scannerService := scanner.NewService(cfg)
	archiverService := archive.NewService(cryptoService)

	// Test scanning
	t.Run("Scan Files", func(t *testing.T) {
		scanOpts := types.ScanOptions{
			RootPath: tmpDir,
			MaxDepth: cfg.DefaultDepth,
		}

		files, err := scannerService.ScanFiles(scanOpts)
		testutils.AssertNoError(t, err)

		if len(files) == 0 {
			t.Fatal("No files found during scan")
		}

		t.Logf("Found %d environment files", len(files))

		// Verify expected files are found
		expectedFiles := []string{".env", ".env.local", ".env.development", ".env.production"}
		foundFiles := make(map[string]bool)
		for _, file := range files {
			foundFiles[filepath.Base(file.RelativePath)] = true
		}

		for _, expected := range expectedFiles {
			if !foundFiles[expected] {
				t.Errorf("Expected file %s not found in scan results", expected)
			}
		}

		// Store files for next test
		t.Cleanup(func() {
			// Cleanup will be handled by defer
		})
	})

	// Get files for archiving tests
	scanOpts := types.ScanOptions{
		RootPath: tmpDir,
		MaxDepth: cfg.DefaultDepth,
	}
	files, err := scannerService.ScanFiles(scanOpts)
	testutils.AssertNoError(t, err)

	// Test packing
	archivePath := filepath.Join(tmpDir, "test-archive.enc")
	password := "test-password-123"

	t.Run("Pack Files", func(t *testing.T) {
		packOpts := types.PackOptions{
			Files:       files,
			OutputPath:  archivePath,
			Password:    password,
			Description: "Integration test archive",
		}

		err := archiverService.Pack(packOpts)
		testutils.AssertNoError(t, err)

		testutils.AssertFileExists(t, archivePath)

		// Verify archive file is not empty
		stat, err := os.Stat(archivePath)
		testutils.AssertNoError(t, err)
		if stat.Size() == 0 {
			t.Error("Archive file is empty")
		}

		t.Logf("Created archive: %s (%d bytes)", archivePath, stat.Size())
	})

	// Test listing
	t.Run("List Archive Contents", func(t *testing.T) {
		archive, err := archiverService.List(archivePath, password)
		testutils.AssertNoError(t, err)

		if len(archive.Files) != len(files) {
			t.Errorf("Archive contains %d files, expected %d", len(archive.Files), len(files))
		}

		if archive.Description != "Integration test archive" {
			t.Errorf("Archive description mismatch: got %s", archive.Description)
		}

		// Verify all original files are in archive
		archiveFiles := make(map[string]bool)
		for _, file := range archive.Files {
			archiveFiles[file.RelativePath] = true
		}

		for _, original := range files {
			if !archiveFiles[original.RelativePath] {
				t.Errorf("Original file %s not found in archive", original.RelativePath)
			}
		}

		t.Logf("Archive contains %d files with total size %d bytes", archive.FileCount, archive.TotalSize)
	})

	// Test unpacking
	t.Run("Unpack Archive", func(t *testing.T) {
		unpackDir := filepath.Join(tmpDir, "unpacked")
		err := os.MkdirAll(unpackDir, 0755)
		testutils.AssertNoError(t, err)

		unpackOpts := types.UnpackOptions{
			ArchivePath: archivePath,
			Password:    password,
			TargetDir:   unpackDir,
			Overwrite:   true,
			Backup:      false,
		}

		err = archiverService.Unpack(unpackOpts)
		testutils.AssertNoError(t, err)

		// Verify unpacked files
		for _, file := range files {
			unpackedPath := filepath.Join(unpackDir, file.RelativePath)
			testutils.AssertFileExists(t, unpackedPath)

			originalPath := file.Path
			if !testutils.CompareFiles(t, originalPath, unpackedPath) {
				t.Errorf("Unpacked file %s doesn't match original", file.RelativePath)
			}
		}

		t.Logf("Successfully unpacked %d files to %s", len(files), unpackDir)
	})

	// Test list available archives
	t.Run("List Available Archives", func(t *testing.T) {
		archives, err := archiverService.GetAvailableArchives(tmpDir)
		testutils.AssertNoError(t, err)

		if len(archives) == 0 {
			t.Error("No archives found")
		}

		found := false
		for _, archive := range archives {
			if filepath.Base(archive) == "test-archive.enc" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Created archive not found in available archives list")
		}

		t.Logf("Found %d available archives", len(archives))
	})
}

func TestErrorHandling(t *testing.T) {
	cfg := testutils.CreateTestConfig()
	cryptoService := crypto.NewService()
	archiverService := archive.NewService(cryptoService)

	t.Run("Pack with Invalid Path", func(t *testing.T) {
		packOpts := types.PackOptions{
			Files:      []types.EnvFile{},
			OutputPath: "/invalid/path/archive.enc",
			Password:   "test",
		}

		err := archiverService.Pack(packOpts)
		if err == nil {
			t.Error("Expected error when packing to invalid path, got nil")
		}

		if archiveErr, ok := err.(*types.ArchiveError); !ok {
			t.Errorf("Expected ArchiveError, got %T", err)
		} else if archiveErr.Operation != "pack" {
			t.Errorf("Expected operation 'pack', got %s", archiveErr.Operation)
		}
	})

	t.Run("Unpack with Wrong Password", func(t *testing.T) {
		tmpDir := testutils.CreateTempEnvFiles(t)
		defer os.RemoveAll(tmpDir)

		// Create valid archive first
		scannerService := scanner.NewService(cfg)
		files, err := scannerService.ScanFiles(types.ScanOptions{
			RootPath: tmpDir,
			MaxDepth: 2,
		})
		testutils.AssertNoError(t, err)

		archivePath := filepath.Join(tmpDir, "test.enc")
		correctPassword := "correct-password"

		packOpts := types.PackOptions{
			Files:       files,
			OutputPath:  archivePath,
			Password:    correctPassword,
			Description: "Test archive for wrong password test",
		}

		err = archiverService.Pack(packOpts)
		testutils.AssertNoError(t, err)

		// Try to unpack with wrong password
		unpackOpts := types.UnpackOptions{
			ArchivePath: archivePath,
			Password:    "wrong-password",
			TargetDir:   tmpDir,
		}

		err = archiverService.Unpack(unpackOpts)
		if err == nil {
			t.Error("Expected error when unpacking with wrong password, got nil")
		}

		if archiveErr, ok := err.(*types.ArchiveError); !ok {
			t.Errorf("Expected ArchiveError, got %T", err)
		} else if archiveErr.Operation != "unpack" {
			t.Errorf("Expected operation 'unpack', got %s", archiveErr.Operation)
		}
	})

	t.Run("List Non-existent Archive", func(t *testing.T) {
		_, err := archiverService.List("/path/to/nonexistent.enc", "password")
		if err == nil {
			t.Error("Expected error when listing non-existent archive, got nil")
		}
	})

	t.Run("Scan Invalid Directory", func(t *testing.T) {
		scannerService := scanner.NewService(cfg)
		_, err := scannerService.ScanFiles(types.ScanOptions{
			RootPath: "/nonexistent/directory",
			MaxDepth: 2,
		})

		if err == nil {
			t.Error("Expected error when scanning non-existent directory, got nil")
		}

		if scanErr, ok := err.(*types.ScanError); !ok {
			t.Errorf("Expected ScanError, got %T", err)
		} else if scanErr.Operation != "scan" {
			t.Errorf("Expected operation 'scan', got %s", scanErr.Operation)
		}
	})
}

func TestConfigIntegration(t *testing.T) {
	tmpDir := testutils.CreateTempDir(t, "config-test-*")
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	configManager := config.NewManager(configPath)

	t.Run("Save and Load Configuration", func(t *testing.T) {
		// Create test config
		testConfig := &types.Config{
			DefaultDepth: 5,
			EnvPatterns: []string{
				`\.env$`,
				`\.env\.custom$`,
			},
			ExcludePatterns: []string{
				`node_modules/`,
				`\.git/`,
				`custom_exclude/`,
			},
			MaxFileSize: 5 * 1024 * 1024, // 5MB
		}

		// Save config
		err := configManager.Save(testConfig)
		testutils.AssertNoError(t, err)

		testutils.AssertFileExists(t, configPath)

		// Load config
		loadedConfig, err := configManager.Load()
		testutils.AssertNoError(t, err)

		// Verify loaded config matches saved config
		if loadedConfig.DefaultDepth != testConfig.DefaultDepth {
			t.Errorf("DefaultDepth mismatch: got %d, want %d", loadedConfig.DefaultDepth, testConfig.DefaultDepth)
		}

		if len(loadedConfig.EnvPatterns) != len(testConfig.EnvPatterns) {
			t.Errorf("EnvPatterns length mismatch: got %d, want %d", len(loadedConfig.EnvPatterns), len(testConfig.EnvPatterns))
		}

		if len(loadedConfig.ExcludePatterns) != len(testConfig.ExcludePatterns) {
			t.Errorf("ExcludePatterns length mismatch: got %d, want %d", len(loadedConfig.ExcludePatterns), len(testConfig.ExcludePatterns))
		}

		if loadedConfig.MaxFileSize != testConfig.MaxFileSize {
			t.Errorf("MaxFileSize mismatch: got %d, want %d", loadedConfig.MaxFileSize, testConfig.MaxFileSize)
		}
	})

	t.Run("Default Configuration", func(t *testing.T) {
		defaultConfig := configManager.GetDefault()

		if defaultConfig == nil {
			t.Fatal("Default config should not be nil")
		}

		if defaultConfig.DefaultDepth == 0 {
			t.Error("Default depth should be greater than 0")
		}

		if len(defaultConfig.EnvPatterns) == 0 {
			t.Error("Default config should have environment patterns")
		}

		if defaultConfig.MaxFileSize == 0 {
			t.Error("Default max file size should be greater than 0")
		}
	})

	t.Run("Configuration Validation", func(t *testing.T) {
		validConfig := &types.Config{
			DefaultDepth:    3,
			EnvPatterns:     []string{`\.env$`},
			ExcludePatterns: []string{`node_modules/`},
			MaxFileSize:     1024 * 1024,
		}

		err := configManager.Validate(validConfig)
		testutils.AssertNoError(t, err)

		// Test invalid config
		invalidConfig := &types.Config{
			DefaultDepth:    -1, // Invalid depth
			EnvPatterns:     []string{},
			ExcludePatterns: []string{},
			MaxFileSize:     0, // Invalid size
		}

		err = configManager.Validate(invalidConfig)
		if err == nil {
			t.Error("Expected validation error for invalid config, got nil")
		}
	})
}

func TestLargeFileHandling(t *testing.T) {
	tmpDir := testutils.CreateTempDir(t, "large-file-test-*")
	defer os.RemoveAll(tmpDir)

	// Create a configuration with smaller max file size for testing
	cfg := &types.Config{
		DefaultDepth: 2,
		EnvPatterns:  []string{`\.env$`},
		ExcludePatterns: []string{},
		MaxFileSize: 1024, // 1KB limit
	}

	scannerService := scanner.NewService(cfg)

	t.Run("Skip Large Files During Scan", func(t *testing.T) {
		// Create small valid file
		smallFile := filepath.Join(tmpDir, ".env")
		testutils.WriteTestFile(t, smallFile, "SMALL_VAR=value")

		// Create large file that exceeds limit
		largeFile := filepath.Join(tmpDir, "large.env")
		testutils.CreateLargeTestFile(t, largeFile, 2048) // 2KB, exceeds 1KB limit

		files, err := scannerService.ScanFiles(types.ScanOptions{
			RootPath: tmpDir,
			MaxDepth: 2,
		})

		testutils.AssertNoError(t, err)

		// Should only find the small file
		if len(files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(files))
		}

		if len(files) > 0 && files[0].RelativePath != ".env" {
			t.Errorf("Expected .env file, got %s", files[0].RelativePath)
		}
	})

	t.Run("Validate Large File Rejection", func(t *testing.T) {
		largeFile := filepath.Join(tmpDir, "large.env")
		testutils.CreateLargeTestFile(t, largeFile, 2048)

		err := scannerService.ValidateFile(largeFile)
		if err == nil {
			t.Error("Expected error for large file validation, got nil")
		}

		if scanErr, ok := err.(*types.ScanError); !ok {
			t.Errorf("Expected ScanError, got %T", err)
		} else {
			testutils.AssertStringContains(t, scanErr.Error(), "too large")
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	tmpDir := testutils.CreateTempEnvFiles(t)
	defer os.RemoveAll(tmpDir)

	cfg := testutils.CreateTestConfig()
	cryptoService := crypto.NewService()
	scannerService := scanner.NewService(cfg)
	archiverService := archive.NewService(cryptoService)

	// Get files for testing
	files, err := scannerService.ScanFiles(types.ScanOptions{
		RootPath: tmpDir,
		MaxDepth: 2,
	})
	testutils.AssertNoError(t, err)

	t.Run("Concurrent Archive Creation", func(t *testing.T) {
		const numGoroutines = 5
		done := make(chan error, numGoroutines)

		// Create archives concurrently
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				archivePath := filepath.Join(tmpDir, "concurrent-test-"+string(rune(id))+".enc")
				packOpts := types.PackOptions{
					Files:       files,
					OutputPath:  archivePath,
					Password:    "concurrent-test-password",
					Description: "Concurrent test archive",
				}

				err := archiverService.Pack(packOpts)
				done <- err
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			select {
			case err := <-done:
				testutils.AssertNoError(t, err)
			}
		}

		// Verify all archives were created
		for i := 0; i < numGoroutines; i++ {
			archivePath := filepath.Join(tmpDir, "concurrent-test-"+string(rune(i))+".enc")
			testutils.AssertFileExists(t, archivePath)
		}
	})
}

func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	tmpDir := testutils.CreateTempDir(t, "memory-test-*")
	defer os.RemoveAll(tmpDir)

	// Create many small files to test memory efficiency
	const numFiles = 100
	for i := 0; i < numFiles; i++ {
		filename := filepath.Join(tmpDir, "file"+string(rune(i))+".env")
		content := "VAR" + string(rune(i)) + "=value" + string(rune(i))
		testutils.WriteTestFile(t, filename, content)
	}

	cfg := testutils.CreateTestConfig()
	scannerService := scanner.NewService(cfg)

	t.Run("Memory Efficient Scanning", func(t *testing.T) {
		files, err := scannerService.ScanFiles(types.ScanOptions{
			RootPath: tmpDir,
			MaxDepth: 2,
		})

		testutils.AssertNoError(t, err)

		if len(files) != numFiles {
			t.Errorf("Expected %d files, got %d", numFiles, len(files))
		}

		// Verify file data is populated correctly
		for _, file := range files {
			if file.Path == "" || file.RelativePath == "" {
				t.Error("File path information missing")
			}
			if file.Size == 0 {
				t.Error("File size should be greater than 0")
			}
			if file.Checksum == "" {
				t.Error("File checksum missing")
			}
		}
	})
}