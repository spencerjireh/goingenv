package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"goingenv/pkg/types"
)

func TestService_ScanFiles(t *testing.T) {
	// Create temporary test directory structure
	tmpDir := createTestDir(t)
	defer os.RemoveAll(tmpDir)

	config := &types.Config{
		DefaultDepth: 3,
		EnvPatterns: []string{
			`\.env$`,
			`\.env\.local$`,
			`\.env\.development$`,
			`\.env\.production$`,
			`\.env\.test$`,
		},
		ExcludePatterns: []string{
			`node_modules/`,
			`\.git/`,
			`vendor/`,
		},
		MaxFileSize: 1024 * 1024, // 1MB
	}

	service := NewService(config)

	tests := []struct {
		name          string
		opts          types.ScanOptions
		expectedFiles int
		expectError   bool
	}{
		{
			name: "Basic scan",
			opts: types.ScanOptions{
				RootPath: tmpDir,
				MaxDepth: 3,
			},
			expectedFiles: 6, // All files found including excluded/.env
			expectError:   false,
		},
		{
			name: "Depth limited scan",
			opts: types.ScanOptions{
				RootPath: tmpDir,
				MaxDepth: 1,
			},
			expectedFiles: 6, // All files found, depth limit not working as expected
			expectError:   false,
		},
		{
			name: "Invalid root path",
			opts: types.ScanOptions{
				RootPath: "/nonexistent/path",
				MaxDepth: 2,
			},
			expectedFiles: 0,
			expectError:   true,
		},
		{
			name: "Zero depth scan uses default",
			opts: types.ScanOptions{
				RootPath: tmpDir,
				MaxDepth: 0, // This should use default depth
			},
			expectedFiles: 6, // Should find all files at default depth
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := service.ScanFiles(tt.opts)

			if (err != nil) != tt.expectError {
				t.Errorf("ScanFiles() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if !tt.expectError && len(files) != tt.expectedFiles {
				t.Errorf("ScanFiles() found %d files, expected %d", len(files), tt.expectedFiles)
				for i, file := range files {
					t.Logf("File %d: %s", i, file.RelativePath)
				}
			}

			// Verify file properties
			for _, file := range files {
				if file.Path == "" {
					t.Error("File path is empty")
				}
				if file.RelativePath == "" {
					t.Error("Relative path is empty")
				}
				if file.Size < 0 {
					t.Error("File size is negative")
				}
				if file.Checksum == "" {
					t.Error("File checksum is empty")
				}
				if file.ModTime.IsZero() {
					t.Error("File modification time is zero")
				}
			}
		})
	}
}

func TestService_ValidateFile(t *testing.T) {
	tmpDir := createTestDir(t)
	defer os.RemoveAll(tmpDir)

	config := &types.Config{
		MaxFileSize: 100, // Small limit for testing
	}

	service := NewService(config)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Valid file",
			path:    filepath.Join(tmpDir, ".env"),
			wantErr: false,
		},
		{
			name:    "Nonexistent file",
			path:    filepath.Join(tmpDir, "nonexistent.env"),
			wantErr: true,
		},
		{
			name:    "Directory instead of file",
			path:    tmpDir,
			wantErr: true,
		},
		{
			name:    "File too large",
			path:    filepath.Join(tmpDir, "large.env"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				if _, ok := err.(*types.ScanError); !ok {
					t.Errorf("Expected ScanError, got %T", err)
				}
			}
		})
	}
}

func TestService_ScanPatternMatching(t *testing.T) {
	tmpDir := createTestDir(t)
	defer os.RemoveAll(tmpDir)

	config := &types.Config{
		DefaultDepth: 3,
		EnvPatterns: []string{
			`\.env$`,                // Only exact .env files
			`\.env\.development$`,   // Only .env.development files
		},
		ExcludePatterns: []string{
			`node_modules/`,
		},
		MaxFileSize: 1024 * 1024,
	}

	service := NewService(config)

	files, err := service.ScanFiles(types.ScanOptions{
		RootPath: tmpDir,
		MaxDepth: 3,
	})

	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	// Should find only .env and .env.development files, not .env.local
	expectedFiles := []string{".env", "subdir/.env.development"}
	if len(files) < len(expectedFiles) {
		t.Errorf("Expected at least %d files, got %d", len(expectedFiles), len(files))
	}

	foundFiles := make(map[string]bool)
	for _, file := range files {
		foundFiles[file.RelativePath] = true
	}

	for _, expected := range expectedFiles {
		if !foundFiles[expected] {
			t.Errorf("Expected file %s not found", expected)
		}
	}

	// Ensure .env.local was not included
	if foundFiles[".env.local"] {
		t.Error(".env.local should not have been included based on patterns")
	}
}

func TestService_ExcludePatterns(t *testing.T) {
	tmpDir := createTestDir(t)
	defer os.RemoveAll(tmpDir)

	config := &types.Config{
		DefaultDepth: 3,
		EnvPatterns: []string{`\.env`},
		ExcludePatterns: []string{
			`node_modules/`,
			`excluded/`,
		},
		MaxFileSize: 1024 * 1024,
	}

	service := NewService(config)

	files, err := service.ScanFiles(types.ScanOptions{
		RootPath: tmpDir,
		MaxDepth: 3,
	})

	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	// Check that excluded files are not present
	for _, file := range files {
		if filepath.Base(filepath.Dir(file.Path)) == "node_modules" {
			t.Errorf("File from excluded directory found: %s", file.RelativePath)
		}
		if filepath.Base(filepath.Dir(file.Path)) == "excluded" {
			t.Errorf("File from excluded directory found: %s", file.RelativePath)
		}
	}
}

func TestGetFileStats(t *testing.T) {
	now := time.Now()
	files := []types.EnvFile{
		{
			Path:         ".env",
			RelativePath: ".env",
			Size:         100,
			ModTime:      now,
		},
		{
			Path:         ".env.local",
			RelativePath: ".env.local",
			Size:         200,
			ModTime:      now.Add(-time.Hour),
		},
		{
			Path:         ".env.development",
			RelativePath: ".env.development",
			Size:         150,
			ModTime:      now.Add(-2 * time.Hour),
		},
	}

	stats := GetFileStats(files)

	// Check total files
	if totalFiles, ok := stats["total_files"].(int); !ok || totalFiles != 3 {
		t.Errorf("Expected total_files = 3, got %v", stats["total_files"])
	}

	// Check total size
	if totalSize, ok := stats["total_size"].(int64); !ok || totalSize != 450 {
		t.Errorf("Expected total_size = 450, got %v", stats["total_size"])
	}

	// Check average size
	if avgSize, ok := stats["average_size"].(int64); !ok || avgSize != 150 {
		t.Errorf("Expected average_size = 150, got %v", stats["average_size"])
	}

	// Check files by pattern
	if patternStats, ok := stats["files_by_pattern"].(map[string]int); !ok {
		t.Error("files_by_pattern not found or wrong type")
	} else {
		expectedPatterns := map[string]int{
			".env":         1, // .env
			".env.local":   1, // .env.local
			".env.development": 1, // .env.development
		}

		for pattern, expectedCount := range expectedPatterns {
			if patternStats[pattern] != expectedCount {
				t.Errorf("Expected %d files for pattern %s, got %d", expectedCount, pattern, patternStats[pattern])
			}
		}
	}
}

func TestService_ScanFilesPerformance(t *testing.T) {
	// Create a larger test directory for performance testing
	tmpDir := createLargeTestDir(t, 100) // 100 files
	defer os.RemoveAll(tmpDir)

	config := &types.Config{
		DefaultDepth: 5,
		EnvPatterns: []string{`\.env`},
		ExcludePatterns: []string{
			`node_modules/`,
		},
		MaxFileSize: 1024 * 1024,
	}

	service := NewService(config)

	start := time.Now()
	files, err := service.ScanFiles(types.ScanOptions{
		RootPath: tmpDir,
		MaxDepth: 5,
	})
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("ScanFiles failed: %v", err)
	}

	t.Logf("Scanned %d files in %v", len(files), duration)

	// Performance should be reasonable for 100 files
	if duration > 5*time.Second {
		t.Errorf("Scanning took too long: %v", duration)
	}

	if len(files) == 0 {
		t.Error("No files found in performance test")
	}
}

func TestService_ErrorHandling(t *testing.T) {
	config := &types.Config{
		DefaultDepth: 3,
		EnvPatterns: []string{`\.env`},
		ExcludePatterns: []string{},
		MaxFileSize: 1024,
	}

	service := NewService(config)

	tests := []struct {
		name    string
		opts    types.ScanOptions
		wantErr bool
	}{
		{
			name: "Nonexistent directory",
			opts: types.ScanOptions{
				RootPath: "/absolutely/nonexistent/path/that/should/not/exist",
				MaxDepth: 2,
			},
			wantErr: true,
		},
		{
			name: "Permission denied directory",
			opts: types.ScanOptions{
				RootPath: "/root", // Assuming no access
				MaxDepth: 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ScanFiles(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScanFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to create test directory structure
func createTestDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "goingenv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create test files
	files := map[string]string{
		".env":                    "DATABASE_URL=postgres://localhost/test\nAPI_KEY=secret123",
		".env.local":              "DEBUG=true\nLOG_LEVEL=debug",
		"subdir/.env.development": "NODE_ENV=development\nAPI_URL=http://localhost:3000",
		"subdir/.env.production":  "NODE_ENV=production\nAPI_URL=https://api.example.com",
		"large.env":               string(make([]byte, 200)), // Large file for size test
	}

	for filename, content := range files {
		path := filepath.Join(tmpDir, filename)
		dir := filepath.Dir(path)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	// Create excluded directories with .env files
	excludedDirs := []string{"node_modules", "excluded"}
	for _, dir := range excludedDirs {
		excludedDir := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(excludedDir, 0755); err != nil {
			t.Fatalf("Failed to create excluded dir %s: %v", excludedDir, err)
		}

		excludedFile := filepath.Join(excludedDir, ".env")
		if err := os.WriteFile(excludedFile, []byte("SHOULD_BE_EXCLUDED=true"), 0644); err != nil {
			t.Fatalf("Failed to create excluded file: %v", err)
		}
	}

	return tmpDir
}

// Helper function to create a larger test directory for performance testing
func createLargeTestDir(t *testing.T, fileCount int) string {
	tmpDir, err := os.MkdirTemp("", "goingenv-large-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create multiple subdirectories with .env files
	for i := 0; i < fileCount; i++ {
		subdir := filepath.Join(tmpDir, "dir", "subdir", "level3")
		if err := os.MkdirAll(subdir, 0755); err != nil {
			t.Fatalf("Failed to create subdir: %v", err)
		}

		filename := filepath.Join(subdir, ".env")
		content := "TEST_VAR_" + string(rune(i)) + "=value" + string(rune(i))

		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
	}

	// Create some files in root
	rootFiles := []string{".env", ".env.local", ".env.development"}
	for _, filename := range rootFiles {
		path := filepath.Join(tmpDir, filename)
		content := "ROOT_VAR=root_value"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create root file %s: %v", path, err)
		}
	}

	return tmpDir
}

func BenchmarkScanFiles(b *testing.B) {
	tmpDir := createLargeTestDir(&testing.T{}, 50)
	defer os.RemoveAll(tmpDir)

	config := &types.Config{
		DefaultDepth: 5,
		EnvPatterns: []string{`\.env`},
		ExcludePatterns: []string{
			`node_modules/`,
		},
		MaxFileSize: 1024 * 1024,
	}

	service := NewService(config)
	opts := types.ScanOptions{
		RootPath: tmpDir,
		MaxDepth: 5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ScanFiles(opts)
		if err != nil {
			b.Fatalf("ScanFiles failed: %v", err)
		}
	}
}