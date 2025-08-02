package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"Bytes", 512, "512 B"},
		{"Kilobytes", 1024, "1.0 KB"},
		{"Megabytes", 1024 * 1024, "1.0 MB"},
		{"Gigabytes", 1024 * 1024 * 1024, "1.0 GB"},
		{"Large number", 1536, "1.5 KB"},
		{"Zero", 0, "0 B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatSize(%d) = %s; want %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Valid filename", "my-file_name.txt", "my-file_name_txt"},
		{"With spaces", "my file name", "my_file_name"},
		{"With special chars", "file@#$%.txt", "file____txt"},
		{"Empty string", "", ""},
		{"Only alphanumeric", "abc123", "abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename(%s) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCalculateFileChecksum(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test-checksum-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test content
	testContent := "Hello, World!"
	if _, err := tmpFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Calculate checksum
	checksum, err := CalculateFileChecksum(tmpFile.Name())
	if err != nil {
		t.Fatalf("CalculateFileChecksum failed: %v", err)
	}

	// Expected SHA-256 hash of "Hello, World!"
	expected := "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"
	if checksum != expected {
		t.Errorf("CalculateFileChecksum() = %s; want %s", checksum, expected)
	}
}

func TestCategorizeEnvFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"Main env", ".env", "Main"},
		{"Local env", ".env.local", "Local"},
		{"Development env", ".env.development", "Development"},
		{"Production env", ".env.production", "Production"},
		{"Staging env", ".env.staging", "Staging"},
		{"Test env", ".env.test", "Test"},
		{"Other env", ".env.custom", "Other"},
		{"Invalid", "not-env", "Other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CategorizeEnvFile(tt.filename)
			if result != tt.expected {
				t.Errorf("CategorizeEnvFile(%s) = %s; want %s", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestEnsureDir(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test-ensure-dir-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test creating a new directory
	newDir := filepath.Join(tmpDir, "new", "nested", "dir")
	err = EnsureDir(newDir)
	if err != nil {
		t.Errorf("EnsureDir failed: %v", err)
	}

	// Check if directory was created
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("Directory was not created")
	}

	// Test with existing directory (should not error)
	err = EnsureDir(newDir)
	if err != nil {
		t.Errorf("EnsureDir failed on existing directory: %v", err)
	}
}

func TestFileExists(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-exists-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Test existing file
	if !FileExists(tmpFile.Name()) {
		t.Error("FileExists returned false for existing file")
	}

	// Test non-existing file
	if FileExists("/path/that/does/not/exist") {
		t.Error("FileExists returned true for non-existing file")
	}
}

func TestGetFileSize(t *testing.T) {
	// Create a temporary file with known content
	tmpFile, err := os.CreateTemp("", "test-size-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testContent := "Test content for size calculation"
	expectedSize := int64(len(testContent))

	if _, err := tmpFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Test file size
	size, err := GetFileSize(tmpFile.Name())
	if err != nil {
		t.Errorf("GetFileSize failed: %v", err)
	}

	if size != expectedSize {
		t.Errorf("GetFileSize() = %d; want %d", size, expectedSize)
	}

	// Test non-existing file
	_, err = GetFileSize("/path/that/does/not/exist")
	if err == nil {
		t.Error("GetFileSize should fail for non-existing file")
	}
}

func TestIsValidEnvFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"Valid .env", ".env", true},
		{"Valid .env.local", ".env.local", true},
		{"Valid .env.development", ".env.development", true},
		{"Valid .env.production", ".env.production", true},
		{"Valid .env.test", ".env.test", true},
		{"Valid .env.staging", ".env.staging", true},
		{"Invalid extension", "file.txt", false},
		{"No extension", "env", false},
		{"Wrong pattern", "env.local", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidEnvFile(tt.filename)
			if result != tt.expected {
				t.Errorf("IsValidEnvFile(%s) = %v; want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestGetRelativePath(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		fullPath string
		expected string
		wantErr  bool
	}{
		{
			name:     "Simple relative path",
			basePath: "/home/user",
			fullPath: "/home/user/project/.env",
			expected: "project/.env",
			wantErr:  false,
		},
		{
			name:     "Same directory",
			basePath: "/home/user",
			fullPath: "/home/user/.env",
			expected: ".env",
			wantErr:  false,
		},
		{
			name:     "Complex nested path",
			basePath: "/home/user/projects",
			fullPath: "/home/user/projects/app/config/.env.local",
			expected: "app/config/.env.local",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetRelativePath(tt.basePath, tt.fullPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRelativePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("GetRelativePath() = %s; want %s", result, tt.expected)
			}
		})
	}
}