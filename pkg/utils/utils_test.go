package utils

import (
	"os"
	"testing"
	"time"
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
	if _, writeErr := tmpFile.WriteString(testContent); writeErr != nil {
		t.Fatalf("Failed to write to temp file: %v", writeErr)
	}
	_ = tmpFile.Close()

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

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "Just now",
			time:     now.Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "1 minute ago",
			time:     now.Add(-1 * time.Minute),
			expected: "1 minute ago",
		},
		{
			name:     "Multiple minutes ago",
			time:     now.Add(-45 * time.Minute),
			expected: "45 minutes ago",
		},
		{
			name:     "1 hour ago",
			time:     now.Add(-1 * time.Hour),
			expected: "1 hour ago",
		},
		{
			name:     "Multiple hours ago",
			time:     now.Add(-5 * time.Hour),
			expected: "5 hours ago",
		},
		{
			name:     "1 day ago",
			time:     now.Add(-24 * time.Hour),
			expected: "1 day ago",
		},
		{
			name:     "Multiple days ago",
			time:     now.Add(-3 * 24 * time.Hour),
			expected: "3 days ago",
		},
		{
			name:     "1 week ago",
			time:     now.Add(-7 * 24 * time.Hour),
			expected: "1 week ago",
		},
		{
			name:     "Multiple weeks ago",
			time:     now.Add(-21 * 24 * time.Hour),
			expected: "3 weeks ago",
		},
		{
			name:     "Over 30 days ago",
			time:     now.Add(-60 * 24 * time.Hour),
			expected: now.Add(-60 * 24 * time.Hour).Format("2006-01-02"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTimeAgo(tt.time)
			if result != tt.expected {
				t.Errorf("FormatTimeAgo() = %q; want %q", result, tt.expected)
			}
		})
	}
}
