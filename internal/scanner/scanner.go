package scanner

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"goingenv/pkg/types"
)

// Service implements the Scanner interface
type Service struct {
	config *types.Config
}

// NewService creates a new scanner service
func NewService(config *types.Config) *Service {
	return &Service{
		config: config,
	}
}

// ScanFiles scans for environment files based on the provided options
func (s *Service) ScanFiles(opts types.ScanOptions) ([]types.EnvFile, error) {
	var files []types.EnvFile

	// Use default values if not provided
	if opts.RootPath == "" {
		opts.RootPath = "."
	}
	if opts.MaxDepth == 0 {
		opts.MaxDepth = s.config.DefaultDepth
	}
	if len(opts.Patterns) == 0 {
		opts.Patterns = s.config.EnvPatterns
	}
	if len(opts.EnvExcludePatterns) == 0 {
		opts.EnvExcludePatterns = s.config.EnvExcludePatterns
	}
	if len(opts.ExcludePatterns) == 0 {
		opts.ExcludePatterns = s.config.ExcludePatterns
	}

	// Compile regex patterns for efficiency
	envRegexes, err := compilePatterns(opts.Patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to compile env patterns: %w", err)
	}

	envExcludeRegexes, err := compilePatterns(opts.EnvExcludePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to compile env exclude patterns: %w", err)
	}

	excludeRegexes, err := compilePatterns(opts.ExcludePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to compile exclude patterns: %w", err)
	}

	err = filepath.Walk(opts.RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return &types.ScanError{Path: path, Err: err}
		}

		// Calculate relative path and depth
		relPath, err := filepath.Rel(opts.RootPath, path)
		if err != nil {
			return &types.ScanError{Path: path, Err: err}
		}

		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > opts.MaxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories, but check for exclusion patterns
		if info.IsDir() {
			// Check if this directory should be excluded
			for _, regex := range excludeRegexes {
				if regex.MatchString(path + "/") {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check file size limit
		if info.Size() > s.config.MaxFileSize {
			// Log or handle oversized files if needed
			return nil
		}

		// Check if file matches environment patterns
		matched := false
		for _, regex := range envRegexes {
			if regex.MatchString(info.Name()) {
				matched = true
				break
			}
		}

		if !matched {
			return nil
		}

		// Check if file matches env exclusion patterns
		for _, regex := range envExcludeRegexes {
			if regex.MatchString(info.Name()) {
				return nil // Skip this file as it matches exclusion pattern
			}
		}

		// Calculate checksum
		checksum, err := s.calculateChecksum(path)
		if err != nil {
			return &types.ScanError{
				Path: path,
				Err:  fmt.Errorf("failed to calculate checksum: %w", err),
			}
		}

		// Create EnvFile record
		envFile := types.EnvFile{
			Path:         path,
			RelativePath: relPath,
			Size:         info.Size(),
			ModTime:      info.ModTime(),
			Checksum:     checksum,
		}

		files = append(files, envFile)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// ValidateFile validates if a file is accessible and readable
func (s *Service) ValidateFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return &types.ScanError{
			Path: path,
			Err:  fmt.Errorf("file not accessible: %w", err),
		}
	}

	if info.IsDir() {
		return &types.ScanError{
			Path: path,
			Err:  fmt.Errorf("path is a directory, not a file"),
		}
	}

	if info.Size() > s.config.MaxFileSize {
		return &types.ScanError{
			Path: path,
			Err: fmt.Errorf("file size %d exceeds maximum allowed size %d",
				info.Size(), s.config.MaxFileSize),
		}
	}

	// Try to open file to ensure it's readable
	file, err := os.Open(path)
	if err != nil {
		return &types.ScanError{
			Path: path,
			Err:  fmt.Errorf("file not readable: %w", err),
		}
	}
	file.Close()

	return nil
}

// calculateChecksum calculates SHA-256 checksum of a file
func (s *Service) calculateChecksum(filePath string) (string, error) {
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

// compilePatterns compiles a slice of regex patterns
func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	var regexes []*regexp.Regexp

	for _, pattern := range patterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
		}
		regexes = append(regexes, regex)
	}

	return regexes, nil
}

// GetFileStats returns statistics about scanned files
func GetFileStats(files []types.EnvFile) map[string]interface{} {
	stats := make(map[string]interface{})
	var totalSize int64
	filesByPattern := make(map[string]int)

	for _, file := range files {
		totalSize += file.Size

		// Extract pattern from filename
		filename := filepath.Base(file.Path)
		if strings.HasPrefix(filename, ".env") {
			if strings.Contains(filename, ".") && filename != ".env" {
				suffix := strings.TrimPrefix(filename, ".env.")
				filesByPattern[".env."+suffix]++
			} else {
				filesByPattern[".env"]++
			}
		}
	}

	stats["total_files"] = len(files)
	stats["total_size"] = totalSize
	stats["files_by_pattern"] = filesByPattern
	stats["average_size"] = int64(0)

	if len(files) > 0 {
		stats["average_size"] = totalSize / int64(len(files))
	}

	return stats
}

// FilterFilesBySize filters files by size constraints
func FilterFilesBySize(files []types.EnvFile, minSize, maxSize int64) []types.EnvFile {
	var filtered []types.EnvFile

	for _, file := range files {
		if file.Size >= minSize && (maxSize == 0 || file.Size <= maxSize) {
			filtered = append(filtered, file)
		}
	}

	return filtered
}

// FilterFilesByPattern filters files by specific patterns
func FilterFilesByPattern(files []types.EnvFile, patterns []string) ([]types.EnvFile, error) {
	regexes, err := compilePatterns(patterns)
	if err != nil {
		return nil, err
	}

	var filtered []types.EnvFile

	for _, file := range files {
		filename := filepath.Base(file.Path)
		for _, regex := range regexes {
			if regex.MatchString(filename) {
				filtered = append(filtered, file)
				break
			}
		}
	}

	return filtered, nil
}
