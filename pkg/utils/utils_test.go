package utils

import (
	"os"
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
		{"With special chars", "file@#$%.txt", "file_____txt"},
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

func TestJoinResults(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "Empty slice",
			input:    []string{},
			expected: "",
		},
		{
			name:     "Single item",
			input:    []string{"first"},
			expected: "  • first",
		},
		{
			name:     "Multiple items",
			input:    []string{"first", "second", "third"},
			expected: "  • first\n  • second\n  • third",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JoinResults(tt.input)
			if result != tt.expected {
				t.Errorf("JoinResults(%v) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFilterFilesByPatterns(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		patterns []string
		expected []string
	}{
		{
			name:     "No matches",
			files:    []string{"file1.txt", "file2.go"},
			patterns: []string{"*.env"},
			expected: []string{},
		},
		{
			name:     "Some matches",
			files:    []string{".env", "file.txt", ".env.local"},
			patterns: []string{"*.env*"},
			expected: []string{".env", ".env.local"},
		},
		{
			name:     "Multiple patterns",
			files:    []string{".env", "config.yaml", ".env.local", "app.json"},
			patterns: []string{"*.env*", "*.yaml"},
			expected: []string{".env", "config.yaml", ".env.local"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterFilesByPatterns(tt.files, tt.patterns)
			if len(result) != len(tt.expected) {
				t.Errorf("FilterFilesByPatterns() = %v; want %v", result, tt.expected)
				return
			}

			// Check each expected item is in result
			expectedMap := make(map[string]bool)
			for _, expected := range tt.expected {
				expectedMap[expected] = true
			}

			for _, item := range result {
				if !expectedMap[item] {
					t.Errorf("Unexpected item in result: %s", item)
				}
			}
		})
	}
}
