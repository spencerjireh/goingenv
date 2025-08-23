package types

import (
	"time"
)

// MockScanner implements Scanner interface for testing
type MockScanner struct {
	ScanFilesFunc    func(opts ScanOptions) ([]EnvFile, error)
	ValidateFileFunc func(path string) error
}

func (m *MockScanner) ScanFiles(opts ScanOptions) ([]EnvFile, error) {
	if m.ScanFilesFunc != nil {
		return m.ScanFilesFunc(opts)
	}
	return []EnvFile{}, nil
}

func (m *MockScanner) ValidateFile(path string) error {
	if m.ValidateFileFunc != nil {
		return m.ValidateFileFunc(path)
	}
	return nil
}

// MockArchiver implements Archiver interface for testing
type MockArchiver struct {
	PackFunc                 func(opts PackOptions) error
	UnpackFunc               func(opts UnpackOptions) error
	ListFunc                 func(archivePath, password string) (*Archive, error)
	GetAvailableArchivesFunc func(dir string) ([]string, error)
}

func (m *MockArchiver) Pack(opts PackOptions) error {
	if m.PackFunc != nil {
		return m.PackFunc(opts)
	}
	return nil
}

func (m *MockArchiver) Unpack(opts UnpackOptions) error {
	if m.UnpackFunc != nil {
		return m.UnpackFunc(opts)
	}
	return nil
}

func (m *MockArchiver) List(archivePath, password string) (*Archive, error) {
	if m.ListFunc != nil {
		return m.ListFunc(archivePath, password)
	}
	return &Archive{}, nil
}

func (m *MockArchiver) GetAvailableArchives(dir string) ([]string, error) {
	if m.GetAvailableArchivesFunc != nil {
		return m.GetAvailableArchivesFunc(dir)
	}
	return []string{}, nil
}

// MockCryptor implements Cryptor interface for testing
type MockCryptor struct {
	EncryptFunc          func(data []byte, password string) ([]byte, error)
	DecryptFunc          func(data []byte, password string) ([]byte, error)
	ValidatePasswordFunc func(data []byte, password string) error
}

func (m *MockCryptor) Encrypt(data []byte, password string) ([]byte, error) {
	if m.EncryptFunc != nil {
		return m.EncryptFunc(data, password)
	}
	return data, nil // Simple mock - just return input
}

func (m *MockCryptor) Decrypt(data []byte, password string) ([]byte, error) {
	if m.DecryptFunc != nil {
		return m.DecryptFunc(data, password)
	}
	return data, nil
}

func (m *MockCryptor) ValidatePassword(data []byte, password string) error {
	if m.ValidatePasswordFunc != nil {
		return m.ValidatePasswordFunc(data, password)
	}
	return nil
}

// MockConfigManager implements ConfigManager interface for testing
type MockConfigManager struct {
	LoadFunc       func() (*Config, error)
	SaveFunc       func(config *Config) error
	GetDefaultFunc func() *Config
	ValidateFunc   func(config *Config) error
}

func (m *MockConfigManager) Load() (*Config, error) {
	if m.LoadFunc != nil {
		return m.LoadFunc()
	}
	return &Config{}, nil
}

func (m *MockConfigManager) Save(config *Config) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(config)
	}
	return nil
}

func (m *MockConfigManager) GetDefault() *Config {
	if m.GetDefaultFunc != nil {
		return m.GetDefaultFunc()
	}
	return &Config{
		DefaultDepth:    3,
		EnvPatterns:     []string{`\.env`},
		ExcludePatterns: []string{`node_modules/`},
		MaxFileSize:     10 * 1024 * 1024,
	}
}

func (m *MockConfigManager) Validate(config *Config) error {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(config)
	}
	return nil
}

// NewMockEnvFile creates a test EnvFile for testing
func NewMockEnvFile(path, relativePath string, size int64) EnvFile {
	return EnvFile{
		Path:         path,
		RelativePath: relativePath,
		Size:         size,
		ModTime:      time.Now(),
		Checksum:     "mock-checksum-" + relativePath,
	}
}

// NewMockArchive creates a test Archive for testing
func NewMockArchive(description string, files []EnvFile) *Archive {
	return &Archive{
		Description: description,
		CreatedAt:   time.Now(),
		Files:       files,
		TotalSize:   calculateTotalSize(files),
		Version:     "1.0.0",
	}
}

func calculateTotalSize(files []EnvFile) int64 {
	var total int64
	for _, file := range files {
		total += file.Size
	}
	return total
}
