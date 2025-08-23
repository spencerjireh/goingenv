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
	ConfigFileName     = ".goingenv.json"
	DefaultMaxFileSize = 10 * 1024 * 1024 // 10MB
)

// Manager implements the ConfigManager interface
type Manager struct {
	configPath string
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		configPath: getConfigPath(),
	}
}

// Load loads configuration from file or returns default if not found
func (m *Manager) Load() (*types.Config, error) {
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		return m.GetDefault(), nil
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config types.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate loaded config
	if err := m.Validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Save saves configuration to file
func (m *Manager) Save(config *types.Config) error {
	if err := m.Validate(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Ensure config directory exists
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetDefault returns the default configuration
func (m *Manager) GetDefault() *types.Config {
	return &types.Config{
		DefaultDepth: 3,
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
	if config.DefaultDepth < 1 || config.DefaultDepth > 10 {
		return &types.ValidationError{
			Field:   "DefaultDepth",
			Value:   config.DefaultDepth,
			Message: "must be between 1 and 10",
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

// GetConfigPath returns the configuration file path
func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ConfigFileName
	}
	return filepath.Join(home, ConfigFileName)
}

// EnsureGoingEnvDir ensures the .goingenv directory exists
func EnsureGoingEnvDir() error {
	dir := GetGoingEnvDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create .goingenv directory: %w", err)
	}

	// Create .gitignore if it doesn't exist
	gitignorePath := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		gitignoreContent := "# GoingEnv directory gitignore\n# This allows *.enc files to be committed for safe env transfer\n# Ignore temporary files\n*.tmp\n*.temp\n"
		if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
			return fmt.Errorf("failed to create .gitignore: %w", err)
		}
	}

	return nil
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

	// Create .goingenv directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create .goingenv directory: %w", err)
	}

	// Create .gitignore with corrected content (NOT ignoring *.enc files)
	gitignorePath := filepath.Join(dir, ".gitignore")
	gitignoreContent := "# GoingEnv directory gitignore\n# This allows *.enc files to be committed for safe env transfer\n# Ignore temporary files\n*.tmp\n*.temp\n"

	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	return nil
}
