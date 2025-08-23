package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FormatSize formats file size in human-readable format
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// CalculateFileChecksum calculates SHA-256 checksum of a file
func CalculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// SanitizeFilename sanitizes a filename for safe use in file paths
func SanitizeFilename(filename string) string {
	result := ""
	for _, char := range filename {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_' {
			result += string(char)
		} else {
			result += "_"
		}
	}
	return result
}

// JoinResults joins a slice of strings with bullet points
func JoinResults(results []string) string {
	result := ""
	for i, res := range results {
		if i > 0 {
			result += "\n"
		}
		result += "  • " + res
	}
	return result
}

// CategorizeEnvFile categorizes environment files by type
func CategorizeEnvFile(filename string) string {
	switch {
	case strings.HasPrefix(filename, ".env.local"):
		return "Local"
	case strings.HasPrefix(filename, ".env.development") || strings.HasPrefix(filename, ".env.dev"):
		return "Development"
	case strings.HasPrefix(filename, ".env.production") || strings.HasPrefix(filename, ".env.prod"):
		return "Production"
	case strings.HasPrefix(filename, ".env.staging") || strings.HasPrefix(filename, ".env.stage"):
		return "Staging"
	case strings.HasPrefix(filename, ".env.test") || strings.HasPrefix(filename, ".env.testing"):
		return "Test"
	case filename == ".env":
		return "Main"
	default:
		return "Other"
	}
}

// FilterFilesByPatterns filters files based on glob patterns
func FilterFilesByPatterns(relativePaths []string, patterns []string) []string {
	var filtered []string

	for _, filePath := range relativePaths {
		for _, pattern := range patterns {
			if matched, _ := filepath.Match(pattern, filePath); matched {
				filtered = append(filtered, filePath)
				break
			}
		}
	}

	return filtered
}
