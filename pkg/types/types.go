package types

import (
	"errors"
	"time"

	"goingenv/pkg/utils"
)

// Sentinel errors that callers distinguish with errors.Is. Both survive the
// CryptoError and ArchiveError wrappers through their Unwrap methods.
var (
	// ErrLegacyArchive marks a blob written before format v1: it has no header,
	// so this version cannot read it. The message names the remedy.
	ErrLegacyArchive = errors.New("archive predates format v1: unpack it with goingenv v1.6.0, then re-pack it with this version")
	// ErrDecryptFailed is the generic authentication failure. A wrong password
	// and a corrupted file are indistinguishable by design.
	ErrDecryptFailed = errors.New("wrong password, or the file is corrupted")
)

// EnvFile represents a detected environment file
type EnvFile struct {
	Path         string    `json:"path"`
	RelativePath string    `json:"relative_path"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	Checksum     string    `json:"checksum"`
}

// Archive represents the structure of an encrypted archive
type Archive struct {
	CreatedAt   time.Time `json:"created_at"`
	Files       []EnvFile `json:"files"`
	TotalSize   int64     `json:"total_size"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Env         string    `json:"env,omitempty"`
}

// Config holds application configuration
type Config struct {
	DefaultDepth       int      `json:"default_depth" validate:"min=1,max=10"`
	EnvPatterns        []string `json:"env_patterns" validate:"required,min=1"`
	EnvExcludePatterns []string `json:"env_exclude_patterns"`
	ExcludePatterns    []string `json:"exclude_patterns"`
	MaxFileSize        int64    `json:"max_file_size"`
	Manifest           bool     `json:"manifest"`
}

// App holds all the application dependencies
type App struct {
	Config    *Config
	Scanner   Scanner
	Archiver  Archiver
	Crypto    Cryptor
	ConfigMgr ConfigManager
}

// ScanOptions represents options for file scanning
type ScanOptions struct {
	RootPath           string
	MaxDepth           int
	Patterns           []string
	EnvExcludePatterns []string
	ExcludePatterns    []string
}

// PackOptions represents options for packing files
type PackOptions struct {
	Files       []EnvFile
	OutputPath  string
	Password    string
	Description string
	Env         string // environment name recorded in the metadata; "" when unnamed
}

// UnpackOptions represents options for unpacking files.
//
// Include and Exclude are glob patterns (filepath.Match) tested against each
// entry's relative path. An entry is written only when it matches some
// Include pattern (or Include is empty) and matches no Exclude pattern. The
// archiver applies them so the files written and the files reported can never
// disagree.
type UnpackOptions struct {
	ArchivePath string
	Password    string
	TargetDir   string
	Overwrite   bool
	Backup      bool
	Include     []string
	Exclude     []string
}

// Selects reports whether an entry with the given relative path passes the
// Include and Exclude filters.
func (o *UnpackOptions) Selects(relativePath string) bool {
	if len(o.Include) > 0 && !utils.MatchesAnyGlob(relativePath, o.Include) {
		return false
	}
	return !utils.MatchesAnyGlob(relativePath, o.Exclude)
}

// UnpackResult reports what Unpack wrote and what it left alone.
type UnpackResult struct {
	Extracted []string // relative paths written to TargetDir
	Skipped   []string // relative paths that already existed and were not overwritten
}

// Interfaces for better testability and decoupling

// Scanner interface for file scanning operations
type Scanner interface {
	ScanFiles(opts *ScanOptions) ([]EnvFile, error)
	ValidateFile(path string) error
}

// Archiver interface for archive operations
type Archiver interface {
	Pack(opts PackOptions) error
	Unpack(opts UnpackOptions) (*UnpackResult, error)
	List(archivePath, password string) (*Archive, error)
	// ReadFiles decrypts an archive entirely in memory and returns its
	// metadata plus every entry's content keyed by relative path.
	ReadFiles(archivePath, password string) (*Archive, map[string][]byte, error)
	GetAvailableArchives(dir string) ([]string, error)
}

// Cryptor interface for encryption operations
type Cryptor interface {
	Encrypt(data []byte, password string) ([]byte, error)
	Decrypt(data []byte, password string) ([]byte, error)
	ValidatePassword(data []byte, password string) error
}

// ConfigManager interface for configuration management
type ConfigManager interface {
	Load() (*Config, error)
	Save(config *Config) error
	GetDefault() *Config
	Validate(config *Config) error
}

// UIState represents the state of the TUI
type UIState struct {
	CurrentScreen string
	Message       string
	Error         string
	Loading       bool
	SelectedFile  string
	Width         int
	Height        int
}

// MenuItem represents a menu item in the TUI
type MenuItem struct {
	Title       string
	Description string
	Icon        string
	Action      string
}

// Custom error types for better error handling

// ScanError represents an error during file scanning
type ScanError struct {
	Path string
	Err  error
}

func (e *ScanError) Error() string {
	return "scan error at " + e.Path + ": " + e.Err.Error()
}

func (e *ScanError) Unwrap() error { return e.Err }

// ArchiveError represents an error during archive operations
type ArchiveError struct {
	Operation string
	Path      string
	Err       error
}

func (e *ArchiveError) Error() string {
	if e.Path == "" {
		return e.Operation + " error: " + e.Err.Error()
	}
	return e.Operation + " error for " + e.Path + ": " + e.Err.Error()
}

func (e *ArchiveError) Unwrap() error { return e.Err }

// CryptoError represents an error during cryptographic operations
type CryptoError struct {
	Operation string
	Err       error
}

func (e *CryptoError) Error() string {
	return "crypto " + e.Operation + " error: " + e.Err.Error()
}

func (e *CryptoError) Unwrap() error { return e.Err }

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return "validation error for " + e.Field + ": " + e.Message
}

// FileStats represents statistics about scanned files
type FileStats struct {
	TotalFiles     int
	TotalSize      int64
	AverageSize    int64
	FilesByPattern map[string]int
}
