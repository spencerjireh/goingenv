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

// scanContext holds compiled patterns for scanning
type scanContext struct {
	root        string
	maxDepth    int
	maxFileSize int64
	include     []*regexp.Regexp
	exclude     []*regexp.Regexp
	envExclude  []*regexp.Regexp
}

// newScanContext creates a scan context with compiled patterns
func newScanContext(opts *types.ScanOptions, cfg *types.Config) (*scanContext, error) {
	include, err := compilePatterns(opts.Patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to compile env patterns: %w", err)
	}

	exclude, err := compilePatterns(opts.ExcludePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to compile exclude patterns: %w", err)
	}

	envExclude, err := compilePatterns(opts.EnvExcludePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to compile env exclude patterns: %w", err)
	}

	return &scanContext{
		root:        opts.RootPath,
		maxDepth:    opts.MaxDepth,
		maxFileSize: cfg.MaxFileSize,
		include:     include,
		exclude:     exclude,
		envExclude:  envExclude,
	}, nil
}

// matchesAny returns true if name matches any pattern (pure function)
func matchesAny(name string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p.MatchString(name) {
			return true
		}
	}
	return false
}

// exceedsDepth returns true if path exceeds max depth (pure function)
func exceedsDepth(relPath string, maxDepth int) bool {
	return strings.Count(relPath, string(filepath.Separator)) > maxDepth
}

// shouldSkipDir returns true if directory should be skipped
func (sc *scanContext) shouldSkipDir(path string) bool {
	return matchesAny(path+"/", sc.exclude)
}

// shouldInclude returns true if file should be included
func (sc *scanContext) shouldInclude(name string, size int64) bool {
	if size > sc.maxFileSize {
		return false
	}
	if !matchesAny(name, sc.include) {
		return false
	}
	if matchesAny(name, sc.envExclude) {
		return false
	}
	return true
}

// applyDefaults fills in missing options from config
func applyDefaults(opts *types.ScanOptions, cfg *types.Config) {
	if opts.RootPath == "" {
		opts.RootPath = "."
	}
	if opts.MaxDepth == 0 {
		opts.MaxDepth = cfg.DefaultDepth
	}
	if len(opts.Patterns) == 0 {
		opts.Patterns = cfg.EnvPatterns
	}
	if len(opts.EnvExcludePatterns) == 0 {
		opts.EnvExcludePatterns = cfg.EnvExcludePatterns
	}
	if len(opts.ExcludePatterns) == 0 {
		opts.ExcludePatterns = cfg.ExcludePatterns
	}
}

// ScanFiles scans for environment files based on the provided options
func (s *Service) ScanFiles(opts *types.ScanOptions) ([]types.EnvFile, error) {
	applyDefaults(opts, s.config)

	sc, err := newScanContext(opts, s.config)
	if err != nil {
		return nil, err
	}

	var files []types.EnvFile
	err = filepath.Walk(opts.RootPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return &types.ScanError{Path: path, Err: walkErr}
		}

		relPath, relErr := filepath.Rel(sc.root, path)
		if relErr != nil {
			return &types.ScanError{Path: path, Err: relErr}
		}

		if exceedsDepth(relPath, sc.maxDepth) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			if sc.shouldSkipDir(path) {
				return filepath.SkipDir
			}
			return nil
		}

		if !sc.shouldInclude(info.Name(), info.Size()) {
			return nil
		}

		checksum, checksumErr := s.calculateChecksum(path)
		if checksumErr != nil {
			return &types.ScanError{
				Path: path,
				Err:  fmt.Errorf("failed to calculate checksum: %w", checksumErr),
			}
		}

		files = append(files, types.EnvFile{
			Path:         path,
			RelativePath: relPath,
			Size:         info.Size(),
			ModTime:      info.ModTime(),
			Checksum:     checksum,
		})
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
	defer file.Close()

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
func GetFileStats(files []types.EnvFile) types.FileStats {
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

	var averageSize int64
	if len(files) > 0 {
		averageSize = totalSize / int64(len(files))
	}

	return types.FileStats{
		TotalFiles:     len(files),
		TotalSize:      totalSize,
		AverageSize:    averageSize,
		FilesByPattern: filesByPattern,
	}
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
