package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"goingenv/pkg/types"
)

const (
	ConfigFileName        = ".goingenv.json"
	ProjectConfigFileName = "config.json"
	DefaultMaxFileSize    = 10 * 1024 * 1024 // 10MB
)

// Manager implements the ConfigManager interface.
//
// Reads and writes deliberately target different files. A project may ship its
// own committed .goingenv/config.json to pin how this repository is scanned, and
// that file wins on load — but it is checked in, so nothing may write to it
// implicitly. Saves always go to the user's own ~/.goingenv.json.
type Manager struct {
	loadPath string
	savePath string
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		loadPath: ResolveConfigPath(),
		savePath: homeConfigPath(),
	}
}

// NewManagerWithPath creates a configuration manager backed by an explicit
// path, used for both loading and saving. Tests use this to avoid touching the
// real user home directory, and 'init --project-config' uses it to write a
// project config explicitly.
func NewManagerWithPath(path string) *Manager {
	return &Manager{loadPath: path, savePath: path}
}

// LoadPath reports the file this manager reads configuration from.
func (m *Manager) LoadPath() string {
	return m.loadPath
}

// Exists reports whether a configuration file is already present at the path
// this manager would write to.
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.savePath)
	return err == nil
}

// Load loads configuration from file or returns default if not found.
//
// Keys absent from the file keep their built-in default, so a config may set
// only what it cares about. Without that, a file containing just env_patterns
// would unmarshal DefaultDepth as 0 and fail validation.
func (m *Manager) Load() (*types.Config, error) {
	if _, statErr := os.Stat(m.loadPath); os.IsNotExist(statErr) {
		return m.GetDefault(), nil
	}

	data, readErr := os.ReadFile(m.loadPath)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", m.loadPath, readErr)
	}

	config := *m.GetDefault()
	if parseErr := json.Unmarshal(data, &config); parseErr != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", m.loadPath, parseErr)
	}

	// Validate loaded config
	if validateErr := m.Validate(&config); validateErr != nil {
		return nil, fmt.Errorf("invalid configuration in %s: %w", m.loadPath, validateErr)
	}

	return &config, nil
}

// Save saves configuration to file
func (m *Manager) Save(config *types.Config) error {
	if err := m.Validate(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Create the containing directory only when it is actually missing.
	// MkdirAll would be a no-op on an existing directory, but being explicit
	// keeps this from reading like it re-permissions the user's home.
	dir := filepath.Dir(m.savePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.savePath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetDefault returns the default configuration
func (m *Manager) GetDefault() *types.Config {
	return &types.Config{
		DefaultDepth: 10,
		EnvPatterns: []string{
			`\.env.*`,
		},
		EnvExcludePatterns: []string{},
		ExcludePatterns: []string{
			`node_modules/`,
			`\.git/`,
			`vendor/`,
			`dist/`,
			`build/`,
			`target/`,
			`bin/`,
			`obj/`,
			`\.next/`,
			`\.nuxt/`,
			`coverage/`,
		},
		MaxFileSize: DefaultMaxFileSize,
	}
}

// Validate validates the configuration
func (m *Manager) Validate(config *types.Config) error {
	if config.DefaultDepth < 1 || config.DefaultDepth > 50 {
		return &types.ValidationError{
			Field:   "DefaultDepth",
			Value:   config.DefaultDepth,
			Message: "must be between 1 and 50",
		}
	}

	if len(config.EnvPatterns) == 0 {
		return &types.ValidationError{
			Field:   "EnvPatterns",
			Value:   config.EnvPatterns,
			Message: "must have at least one pattern",
		}
	}

	if config.MaxFileSize <= 0 {
		return &types.ValidationError{
			Field:   "MaxFileSize",
			Value:   config.MaxFileSize,
			Message: "must be greater than 0",
		}
	}

	return nil
}

// GetGoingEnvDir returns the .goingenv directory path
func GetGoingEnvDir() string {
	return ".goingenv"
}

// homeConfigPath returns the per-user configuration file path.
func homeConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ConfigFileName
	}
	return filepath.Join(home, ConfigFileName)
}

// ProjectConfigPath returns the project-local configuration file path, relative
// to the current working directory.
func ProjectConfigPath() string {
	return filepath.Join(GetGoingEnvDir(), ProjectConfigFileName)
}

// ResolveConfigPath returns the configuration file to read: the project-local
// one when the current directory has it, otherwise the per-user one.
//
// Resolution happens once, against the working directory, and does not walk
// upwards — consistent with GetGoingEnvDir, which is relative for the same
// reason. Commands are expected to run from the project root.
func ResolveConfigPath() string {
	projectPath := ProjectConfigPath()
	if _, statErr := os.Stat(projectPath); statErr == nil {
		return projectPath
	}
	return homeConfigPath()
}

// GetDefaultArchivePath generates a default archive path with timestamp
func GetDefaultArchivePath() string {
	return filepath.Join(GetGoingEnvDir(), fmt.Sprintf("archive-%s.enc",
		getCurrentTimestamp()))
}

// getCurrentTimestamp returns current timestamp in format suitable for filenames
func getCurrentTimestamp() string {
	return time.Now().Format("20060102-150405")
}

// IsInitialized checks if GoingEnv has been initialized in the current directory
func IsInitialized() bool {
	dir := GetGoingEnvDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}

	// Check if the directory contains the expected files
	gitignorePath := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		return false
	}

	return true
}

// InitializeProject initializes GoingEnv in the current directory
func InitializeProject() error {
	dir := GetGoingEnvDir()

	// Create .goingenv directory with restrictive permissions
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create .goingenv directory: %w", err)
	}

	// Create .gitignore for temporary files only
	gitignorePath := filepath.Join(dir, ".gitignore")
	gitignoreContent := "# GoingEnv directory gitignore\n# Ignore temporary files\n*.tmp\n*.temp\n"

	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0o600); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	return nil
}
