package testutils

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"goingenv/pkg/types"
)

// CreateTempEnvFiles creates a temporary directory with sample .env files
func CreateTempEnvFiles(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "goingenv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	files := map[string]string{
		".env":              "DATABASE_URL=postgres://localhost/test\nAPI_KEY=test123\nSECRET_KEY=mysecret",
		".env.local":        "DEBUG=true\nLOG_LEVEL=debug\nLOCAL_OVERRIDE=true",
		".env.development":  "NODE_ENV=development\nAPI_URL=http://localhost:3000\nDB_HOST=localhost",
		".env.production":   "NODE_ENV=production\nAPI_URL=https://api.example.com\nDB_HOST=prod.example.com",
		"config/.env.test":  "TEST_DB=memory\nTEST_TIMEOUT=30s\nTEST_MODE=unit",
		"app/.env.staging":  "NODE_ENV=staging\nAPI_URL=https://staging.example.com\nDEBUG=false",
		"nested/deep/.env":  "NESTED_VAR=deep_value\nDEEP_CONFIG=true",
	}

	for filename, content := range files {
		path := filepath.Join(tmpDir, filename)
		dir := filepath.Dir(path)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	// Create some non-env files to test filtering
	nonEnvFiles := map[string]string{
		"package.json":     `{"name": "test", "version": "1.0.0"}`,
		"README.md":        "# Test Project",
		"config/app.yaml":  "database:\n  host: localhost",
	}

	for filename, content := range nonEnvFiles {
		path := filepath.Join(tmpDir, filename)
		dir := filepath.Dir(path)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	// Create excluded directories
	excludedDirs := []string{"node_modules", ".git", "vendor"}
	for _, dir := range excludedDirs {
		excludedDir := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(excludedDir, 0755); err != nil {
			t.Fatalf("Failed to create excluded directory %s: %v", excludedDir, err)
		}

		// Add .env file in excluded directory (should be ignored)
		excludedEnv := filepath.Join(excludedDir, ".env")
		if err := os.WriteFile(excludedEnv, []byte("EXCLUDED=true"), 0644); err != nil {
			t.Fatalf("Failed to create excluded env file: %v", err)
		}
	}

	return tmpDir
}

// CreateTestConfig returns a test configuration
func CreateTestConfig() *types.Config {
	return &types.Config{
		DefaultDepth: 3,
		EnvPatterns: []string{
			`\.env$`,
			`\.env\.local$`,
			`\.env\.development$`,
			`\.env\.production$`,
			`\.env\.test$`,
			`\.env\.staging$`,
		},
		ExcludePatterns: []string{
			`node_modules/`,
			`\.git/`,
			`vendor/`,
			`\.DS_Store`,
			`Thumbs\.db`,
		},
		MaxFileSize: 10 * 1024 * 1024, // 10MB
	}
}

// CreateTestEnvFile creates a test EnvFile struct
func CreateTestEnvFile(path, relativePath string, size int64) types.EnvFile {
	return types.EnvFile{
		Path:         path,
		RelativePath: relativePath,
		Size:         size,
		ModTime:      time.Now(),
		Checksum:     "test-checksum-" + relativePath,
	}
}

// CreateTestEnvFiles creates a slice of test EnvFile structs
func CreateTestEnvFiles(count int) []types.EnvFile {
	files := make([]types.EnvFile, count)
	for i := 0; i < count; i++ {
		files[i] = CreateTestEnvFile(
			filepath.Join("/test", "path", "file"+string(rune(i))+".env"),
			"file"+string(rune(i))+".env",
			int64((i+1)*100),
		)
	}
	return files
}

// AssertFileExists checks if a file exists and fails test if not
func AssertFileExists(t *testing.T, path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist, but it doesn't", path)
	}
}

// AssertFileNotExists checks if a file doesn't exist and fails test if it does
func AssertFileNotExists(t *testing.T, path string) {
	if _, err := os.Stat(path); err == nil {
		t.Errorf("Expected file %s to not exist, but it does", path)
	}
}

// AssertDirExists checks if a directory exists and fails test if not
func AssertDirExists(t *testing.T, path string) {
	if stat, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected directory %s to exist, but it doesn't", path)
	} else if err == nil && !stat.IsDir() {
		t.Errorf("Expected %s to be a directory, but it's a file", path)
	}
}

// CompareFiles compares two files and returns true if they are identical
func CompareFiles(t *testing.T, path1, path2 string) bool {
	content1, err := os.ReadFile(path1)
	if err != nil {
		t.Errorf("Failed to read file %s: %v", path1, err)
		return false
	}

	content2, err := os.ReadFile(path2)
	if err != nil {
		t.Errorf("Failed to read file %s: %v", path2, err)
		return false
	}

	return string(content1) == string(content2)
}

// GetFileContent reads and returns the content of a file
func GetFileContent(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return string(content)
}

// WriteTestFile creates a test file with specified content
func WriteTestFile(t *testing.T, path, content string) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

// CreateTempFile creates a temporary file with specified content
func CreateTempFile(t *testing.T, pattern, content string) string {
	tmpFile, err := os.CreateTemp("", pattern)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	return tmpFile.Name()
}

// CreateTempDir creates a temporary directory
func CreateTempDir(t *testing.T, pattern string) string {
	tmpDir, err := os.MkdirTemp("", pattern)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	return tmpDir
}

// CleanupTempFiles removes temporary files and directories created during tests
func CleanupTempFiles(t *testing.T, paths ...string) {
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			t.Logf("Warning: Failed to cleanup temp path %s: %v", path, err)
		}
	}
}

// AssertStringContains checks if a string contains a substring
func AssertStringContains(t *testing.T, str, substr string) {
	if !contains(str, substr) {
		t.Errorf("String %q does not contain %q", str, substr)
	}
}

// AssertStringNotContains checks if a string does not contain a substring
func AssertStringNotContains(t *testing.T, str, substr string) {
	if contains(str, substr) {
		t.Errorf("String %q should not contain %q", str, substr)
	}
}

// AssertSliceContains checks if a slice contains a specific item
func AssertSliceContains(t *testing.T, slice []string, item string) {
	for _, s := range slice {
		if s == item {
			return
		}
	}
	t.Errorf("Slice %v does not contain %q", slice, item)
}

// AssertSliceNotContains checks if a slice does not contain a specific item
func AssertSliceNotContains(t *testing.T, slice []string, item string) {
	for _, s := range slice {
		if s == item {
			t.Errorf("Slice %v should not contain %q", slice, item)
			return
		}
	}
}

// AssertError checks if an error occurred when one was expected
func AssertError(t *testing.T, err error, expectedErr bool) {
	if (err != nil) != expectedErr {
		if expectedErr {
			t.Error("Expected an error but got nil")
		} else {
			t.Errorf("Expected no error but got: %v", err)
		}
	}
}

// AssertNoError checks if no error occurred
func AssertNoError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
}

// CreateValidArchiveOptions creates valid PackOptions for testing
func CreateValidArchiveOptions(files []types.EnvFile, outputPath, password string) types.PackOptions {
	return types.PackOptions{
		Files:       files,
		OutputPath:  outputPath,
		Password:    password,
		Description: "Test archive created by test suite",
	}
}

// CreateValidUnpackOptions creates valid UnpackOptions for testing
func CreateValidUnpackOptions(archivePath, password, targetDir string) types.UnpackOptions {
	return types.UnpackOptions{
		ArchivePath: archivePath,
		Password:    password,
		TargetDir:   targetDir,
		Overwrite:   true,
		Backup:      false,
	}
}

// WaitForFile waits for a file to exist (useful for async operations)
func WaitForFile(t *testing.T, path string, timeout time.Duration) {
	start := time.Now()
	for time.Since(start) < timeout {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("File %s did not appear within timeout %v", path, timeout)
}

// WaitForFileToDisappear waits for a file to be deleted (useful for cleanup tests)
func WaitForFileToDisappear(t *testing.T, path string, timeout time.Duration) {
	start := time.Now()
	for time.Since(start) < timeout {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("File %s did not disappear within timeout %v", path, timeout)
}

// GetTestDataPath returns the path to test data files
func GetTestDataPath(filename string) string {
	return filepath.Join("testdata", filename)
}

// CreateMinimalTestConfig creates a minimal config for basic testing
func CreateMinimalTestConfig() *types.Config {
	return &types.Config{
		DefaultDepth:    2,
		EnvPatterns:     []string{`\.env$`},
		ExcludePatterns: []string{`node_modules/`},
		MaxFileSize:     1024 * 1024, // 1MB
	}
}

// Helper function to check if a string contains a substring
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || 
		(len(str) > len(substr) && 
		(str[:len(substr)] == substr || 
		str[len(str)-len(substr):] == substr || 
		contains(str[1:], substr))))
}

// MockTime returns a fixed time for consistent testing
func MockTime() time.Time {
	return time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
}

// AssertTimeBetween checks if a time is between two bounds
func AssertTimeBetween(t *testing.T, actual, start, end time.Time) {
	if actual.Before(start) || actual.After(end) {
		t.Errorf("Time %v is not between %v and %v", actual, start, end)
	}
}

// CreateLargeTestFile creates a file with specified size for testing
func CreateLargeTestFile(t *testing.T, path string, sizeBytes int64) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create file %s: %v", path, err)
	}
	defer file.Close()

	// Write data in chunks to avoid memory issues
	chunkSize := int64(1024)
	chunk := make([]byte, chunkSize)
	for i := range chunk {
		chunk[i] = byte('A' + (i % 26))
	}

	written := int64(0)
	for written < sizeBytes {
		toWrite := chunkSize
		if written + chunkSize > sizeBytes {
			toWrite = sizeBytes - written
		}
		
		if _, err := file.Write(chunk[:toWrite]); err != nil {
			t.Fatalf("Failed to write to file %s: %v", path, err)
		}
		written += toWrite
	}
}