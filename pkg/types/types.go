package types

import (
	"time"
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
}

// Config holds application configuration
type Config struct {
	DefaultDepth       int      `json:"default_depth" validate:"min=1,max=10"`
	EnvPatterns        []string `json:"env_patterns" validate:"required,min=1"`
	EnvExcludePatterns []string `json:"env_exclude_patterns"`
	ExcludePatterns    []string `json:"exclude_patterns"`
	MaxFileSize        int64    `json:"max_file_size"`
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
}

// UnpackOptions represents options for unpacking files
type UnpackOptions struct {
	ArchivePath string
	Password    string
	TargetDir   string
	Overwrite   bool
	Backup      bool
}

// Interfaces for better testability and decoupling

// Scanner interface for file scanning operations
type Scanner interface {
	ScanFiles(opts ScanOptions) ([]EnvFile, error)
	ValidateFile(path string) error
}

// Archiver interface for archive operations
type Archiver interface {
	Pack(opts PackOptions) error
	Unpack(opts UnpackOptions) error
	List(archivePath, password string) (*Archive, error)
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

// ArchiveError represents an error during archive operations
type ArchiveError struct {
	Operation string
	Path      string
	Err       error
}

func (e *ArchiveError) Error() string {
	return e.Operation + " error for " + e.Path + ": " + e.Err.Error()
}

// CryptoError represents an error during cryptographic operations
type CryptoError struct {
	Operation string
	Err       error
}

func (e *CryptoError) Error() string {
	return "crypto " + e.Operation + " error: " + e.Err.Error()
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return "validation error for " + e.Field + ": " + e.Message
}
