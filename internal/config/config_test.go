package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"goingenv/pkg/types"
)

func TestManager_Load_NoConfigFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create manager with non-existent config path
	manager := &Manager{
		configPath: filepath.Join(tmpDir, ".goingenv.json"),
	}

	config, err := manager.Load()
	if err != nil {
		t.Errorf("Load() should return default config, got error: %v", err)
	}

	if config == nil {
		t.Fatal("Load() returned nil config")
	}

	// Verify default values
	defaultConfig := manager.GetDefault()
	if config.DefaultDepth != defaultConfig.DefaultDepth {
		t.Errorf("DefaultDepth = %d, want %d", config.DefaultDepth, defaultConfig.DefaultDepth)
	}
	if config.MaxFileSize != defaultConfig.MaxFileSize {
		t.Errorf("MaxFileSize = %d, want %d", config.MaxFileSize, defaultConfig.MaxFileSize)
	}
}

func TestManager_Load_ValidConfig(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, ".goingenv.json")

	// Create valid config file
	validConfig := &types.Config{
		DefaultDepth:       5,
		EnvPatterns:        []string{`\.env.*`, `\.secret`},
		EnvExcludePatterns: []string{`\.env\.example`},
		ExcludePatterns:    []string{`node_modules/`, `vendor/`},
		MaxFileSize:        5 * 1024 * 1024,
	}

	data, err := json.MarshalIndent(validConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	manager := &Manager{configPath: configPath}
	config, err := manager.Load()
	if err != nil {
		t.Errorf("Load() error = %v", err)
	}

	if config.DefaultDepth != validConfig.DefaultDepth {
		t.Errorf("DefaultDepth = %d, want %d", config.DefaultDepth, validConfig.DefaultDepth)
	}
	if config.MaxFileSize != validConfig.MaxFileSize {
		t.Errorf("MaxFileSize = %d, want %d", config.MaxFileSize, validConfig.MaxFileSize)
	}
}

func TestManager_Load_InvalidJSON(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, ".goingenv.json")

	// Create invalid JSON file
	if err := os.WriteFile(configPath, []byte("{invalid json}"), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	manager := &Manager{configPath: configPath}
	_, err = manager.Load()
	if err == nil {
		t.Error("Load() should fail for invalid JSON")
	}
}

func TestManager_Load_InvalidConfig(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, ".goingenv.json")

	// Create config with invalid depth
	invalidConfig := &types.Config{
		DefaultDepth: 100, // Invalid - should be between 1 and 10
		EnvPatterns:  []string{`\.env`},
		MaxFileSize:  1024,
	}

	data, err := json.MarshalIndent(invalidConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	manager := &Manager{configPath: configPath}
	_, err = manager.Load()
	if err == nil {
		t.Error("Load() should fail for invalid config")
	}
}

func TestManager_Save(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, ".goingenv.json")
	manager := &Manager{configPath: configPath}

	config := &types.Config{
		DefaultDepth:       3,
		EnvPatterns:        []string{`\.env`},
		EnvExcludePatterns: []string{},
		ExcludePatterns:    []string{`node_modules/`},
		MaxFileSize:        10 * 1024 * 1024,
	}

	err = manager.Save(config)
	if err != nil {
		t.Errorf("Save() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Verify permissions are restrictive (0600)
	info, err := os.Stat(configPath)
	if err != nil {
		t.Errorf("Failed to stat config file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Config file permissions = %v, want 0600", info.Mode().Perm())
	}

	// Verify content
	loadedConfig, err := manager.Load()
	if err != nil {
		t.Errorf("Failed to load saved config: %v", err)
	}

	if loadedConfig.DefaultDepth != config.DefaultDepth {
		t.Errorf("Saved DefaultDepth = %d, want %d", loadedConfig.DefaultDepth, config.DefaultDepth)
	}
}

func TestManager_Save_InvalidConfig(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, ".goingenv.json")
	manager := &Manager{configPath: configPath}

	invalidConfig := &types.Config{
		DefaultDepth: 0, // Invalid
		EnvPatterns:  []string{},
		MaxFileSize:  -1,
	}

	err = manager.Save(invalidConfig)
	if err == nil {
		t.Error("Save() should fail for invalid config")
	}
}

func TestManager_Validate(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		name    string
		config  *types.Config
		wantErr bool
		errType string
	}{
		{
			name: "Valid config",
			config: &types.Config{
				DefaultDepth: 3,
				EnvPatterns:  []string{`\.env`},
				MaxFileSize:  1024,
			},
			wantErr: false,
		},
		{
			name: "DefaultDepth too low",
			config: &types.Config{
				DefaultDepth: 0,
				EnvPatterns:  []string{`\.env`},
				MaxFileSize:  1024,
			},
			wantErr: true,
			errType: "DefaultDepth",
		},
		{
			name: "DefaultDepth too high",
			config: &types.Config{
				DefaultDepth: 11,
				EnvPatterns:  []string{`\.env`},
				MaxFileSize:  1024,
			},
			wantErr: true,
			errType: "DefaultDepth",
		},
		{
			name: "Empty EnvPatterns",
			config: &types.Config{
				DefaultDepth: 3,
				EnvPatterns:  []string{},
				MaxFileSize:  1024,
			},
			wantErr: true,
			errType: "EnvPatterns",
		},
		{
			name: "Invalid MaxFileSize",
			config: &types.Config{
				DefaultDepth: 3,
				EnvPatterns:  []string{`\.env`},
				MaxFileSize:  0,
			},
			wantErr: true,
			errType: "MaxFileSize",
		},
		{
			name: "Negative MaxFileSize",
			config: &types.Config{
				DefaultDepth: 3,
				EnvPatterns:  []string{`\.env`},
				MaxFileSize:  -100,
			},
			wantErr: true,
			errType: "MaxFileSize",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.wantErr {
				if validationErr, ok := err.(*types.ValidationError); ok {
					if validationErr.Field != tt.errType {
						t.Errorf("ValidationError.Field = %s, want %s", validationErr.Field, tt.errType)
					}
				}
			}
		})
	}
}

func TestManager_GetDefault(t *testing.T) {
	manager := NewManager()
	config := manager.GetDefault()

	if config == nil {
		t.Fatal("GetDefault() returned nil")
	}

	if config.DefaultDepth != 3 {
		t.Errorf("DefaultDepth = %d, want 3", config.DefaultDepth)
	}

	if config.MaxFileSize != DefaultMaxFileSize {
		t.Errorf("MaxFileSize = %d, want %d", config.MaxFileSize, DefaultMaxFileSize)
	}

	if len(config.EnvPatterns) == 0 {
		t.Error("EnvPatterns should not be empty")
	}

	if len(config.ExcludePatterns) == 0 {
		t.Error("ExcludePatterns should not be empty")
	}
}

func TestGetGoingEnvDir(t *testing.T) {
	dir := GetGoingEnvDir()
	if dir != ".goingenv" {
		t.Errorf("GetGoingEnvDir() = %s, want .goingenv", dir)
	}
}

func TestEnsureGoingEnvDir(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Create temp directory and change to it
	tmpDir, err := os.MkdirTemp("", "goingenv-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		os.Chdir(originalDir)
		os.RemoveAll(tmpDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	err = EnsureGoingEnvDir()
	if err != nil {
		t.Errorf("EnsureGoingEnvDir() error = %v", err)
	}

	// Verify directory exists
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	info, err := os.Stat(goingenvDir)
	if os.IsNotExist(err) {
		t.Error(".goingenv directory was not created")
	}

	// Verify directory permissions are restrictive
	if info.Mode().Perm() != 0700 {
		t.Errorf("Directory permissions = %v, want 0700", info.Mode().Perm())
	}

	// Verify .gitignore was created
	gitignorePath := filepath.Join(goingenvDir, ".gitignore")
	info, err = os.Stat(gitignorePath)
	if os.IsNotExist(err) {
		t.Error(".gitignore was not created")
	}

	// Verify .gitignore permissions are restrictive
	if info.Mode().Perm() != 0600 {
		t.Errorf(".gitignore permissions = %v, want 0600", info.Mode().Perm())
	}
}

func TestIsInitialized(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		os.Chdir(originalDir)
		os.RemoveAll(tmpDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Not initialized yet
	if IsInitialized() {
		t.Error("IsInitialized() should return false before initialization")
	}

	// Initialize
	if err := InitializeProject(); err != nil {
		t.Fatalf("InitializeProject() error: %v", err)
	}

	// Now initialized
	if !IsInitialized() {
		t.Error("IsInitialized() should return true after initialization")
	}
}

func TestInitializeProject(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		os.Chdir(originalDir)
		os.RemoveAll(tmpDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	err = InitializeProject()
	if err != nil {
		t.Errorf("InitializeProject() error = %v", err)
	}

	// Verify directory exists with correct permissions
	goingenvDir := filepath.Join(tmpDir, ".goingenv")
	info, err := os.Stat(goingenvDir)
	if os.IsNotExist(err) {
		t.Error(".goingenv directory was not created")
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("Directory permissions = %v, want 0700", info.Mode().Perm())
	}

	// Verify .gitignore exists with correct permissions
	gitignorePath := filepath.Join(goingenvDir, ".gitignore")
	info, err = os.Stat(gitignorePath)
	if os.IsNotExist(err) {
		t.Error(".gitignore was not created")
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf(".gitignore permissions = %v, want 0600", info.Mode().Perm())
	}

	// Verify .gitignore content
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Errorf("Failed to read .gitignore: %v", err)
	}

	if len(content) == 0 {
		t.Error(".gitignore should not be empty")
	}
}

func TestInitializeProject_Idempotent(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		os.Chdir(originalDir)
		os.RemoveAll(tmpDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize twice - should not fail
	if err := InitializeProject(); err != nil {
		t.Fatalf("First InitializeProject() error: %v", err)
	}

	if err := InitializeProject(); err != nil {
		t.Errorf("Second InitializeProject() error: %v", err)
	}

	// Should still be properly initialized
	if !IsInitialized() {
		t.Error("IsInitialized() should return true")
	}
}

func TestGetDefaultArchivePath(t *testing.T) {
	path := GetDefaultArchivePath()

	if path == "" {
		t.Error("GetDefaultArchivePath() returned empty string")
	}

	if !filepath.HasPrefix(path, GetGoingEnvDir()) {
		t.Error("Archive path should be in .goingenv directory")
	}

	if filepath.Ext(path) != ".enc" {
		t.Errorf("Archive path should have .enc extension, got %s", filepath.Ext(path))
	}
}

func TestNewManager(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.configPath == "" {
		t.Error("configPath should not be empty")
	}
}
