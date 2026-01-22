package archive

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"goingenv/internal/crypto"
	"goingenv/pkg/types"
)

func TestService_Pack(t *testing.T) {
	cryptoService := crypto.NewService()
	service := NewService(cryptoService)

	// Create temp directory for test files
	tmpDir, err := os.MkdirTemp("", "goingenv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testContent := []byte("TEST_VAR=test_value\nAPI_KEY=secret123")
	testFilePath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(testFilePath, testContent, 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	outputPath := filepath.Join(tmpDir, "output.enc")

	tests := []struct {
		name    string
		opts    types.PackOptions
		wantErr bool
	}{
		{
			name: "Valid pack",
			opts: types.PackOptions{
				Files: []types.EnvFile{
					{
						Path:         testFilePath,
						RelativePath: ".env",
						Size:         int64(len(testContent)),
						ModTime:      time.Now(),
					},
				},
				OutputPath:  outputPath,
				Password:    "testpassword123",
				Description: "Test archive",
			},
			wantErr: false,
		},
		{
			name: "Empty files",
			opts: types.PackOptions{
				Files:       []types.EnvFile{},
				OutputPath:  outputPath,
				Password:    "testpassword123",
				Description: "Empty archive",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Pack(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Pack() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.wantErr {
				if archiveErr, ok := err.(*types.ArchiveError); ok {
					if archiveErr.Operation != "pack" {
						t.Errorf("Expected operation 'pack', got %s", archiveErr.Operation)
					}
				}
			}

			if !tt.wantErr {
				// Verify file was created
				if _, err := os.Stat(tt.opts.OutputPath); os.IsNotExist(err) {
					t.Error("Output file was not created")
				}

				// Verify permissions are restrictive
				info, err := os.Stat(tt.opts.OutputPath)
				if err != nil {
					t.Errorf("Failed to stat output file: %v", err)
				}
				if info.Mode().Perm() != 0o600 {
					t.Errorf("Output file permissions = %v, want 0o600", info.Mode().Perm())
				}
			}
		})
	}
}

func TestService_Unpack(t *testing.T) {
	cryptoService := crypto.NewService()
	service := NewService(cryptoService)

	// Create temp directory for test files
	tmpDir, err := os.MkdirTemp("", "goingenv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create and pack test file
	testContent := []byte("TEST_VAR=test_value\nAPI_KEY=secret123")
	testFilePath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(testFilePath, testContent, 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	archivePath := filepath.Join(tmpDir, "test.enc")
	password := "testpassword123"

	packOpts := types.PackOptions{
		Files: []types.EnvFile{
			{
				Path:         testFilePath,
				RelativePath: ".env",
				Size:         int64(len(testContent)),
				ModTime:      time.Now(),
			},
		},
		OutputPath: archivePath,
		Password:   password,
	}

	if err := service.Pack(packOpts); err != nil {
		t.Fatalf("Failed to pack test archive: %v", err)
	}

	// Remove original file
	os.Remove(testFilePath)

	targetDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		t.Fatalf("Failed to create target dir: %v", err)
	}

	tests := []struct {
		name    string
		opts    types.UnpackOptions
		wantErr bool
	}{
		{
			name: "Valid unpack",
			opts: types.UnpackOptions{
				ArchivePath: archivePath,
				Password:    password,
				TargetDir:   targetDir,
				Overwrite:   true,
			},
			wantErr: false,
		},
		{
			name: "Wrong password",
			opts: types.UnpackOptions{
				ArchivePath: archivePath,
				Password:    "wrongpassword",
				TargetDir:   targetDir,
				Overwrite:   true,
			},
			wantErr: true,
		},
		{
			name: "Non-existent archive",
			opts: types.UnpackOptions{
				ArchivePath: filepath.Join(tmpDir, "nonexistent.enc"),
				Password:    password,
				TargetDir:   targetDir,
				Overwrite:   true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Unpack(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unpack() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify file was extracted
				extractedPath := filepath.Join(tt.opts.TargetDir, ".env")
				data, err := os.ReadFile(extractedPath)
				if err != nil {
					t.Errorf("Failed to read extracted file: %v", err)
				}
				if !bytes.Equal(data, testContent) {
					t.Errorf("Extracted content doesn't match original")
				}
			}
		})
	}
}

func TestService_List(t *testing.T) {
	cryptoService := crypto.NewService()
	service := NewService(cryptoService)

	// Create temp directory for test files
	tmpDir, err := os.MkdirTemp("", "goingenv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create and pack test files
	testContent := []byte("TEST_VAR=test_value")
	testFilePath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(testFilePath, testContent, 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	archivePath := filepath.Join(tmpDir, "test.enc")
	password := "testpassword123"
	description := "Test archive description"

	packOpts := types.PackOptions{
		Files: []types.EnvFile{
			{
				Path:         testFilePath,
				RelativePath: ".env",
				Size:         int64(len(testContent)),
				ModTime:      time.Now(),
			},
		},
		OutputPath:  archivePath,
		Password:    password,
		Description: description,
	}

	if err := service.Pack(packOpts); err != nil {
		t.Fatalf("Failed to pack test archive: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		password string
		wantErr  bool
	}{
		{
			name:     "Valid list",
			path:     archivePath,
			password: password,
			wantErr:  false,
		},
		{
			name:     "Wrong password",
			path:     archivePath,
			password: "wrongpassword",
			wantErr:  true,
		},
		{
			name:     "Non-existent file",
			path:     filepath.Join(tmpDir, "nonexistent.enc"),
			password: password,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive, err := service.List(tt.path, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if archive == nil {
					t.Error("Expected archive to be returned")
					return
				}
				if len(archive.Files) != 1 {
					t.Errorf("Expected 1 file, got %d", len(archive.Files))
				}
				if archive.Description != description {
					t.Errorf("Description = %s, want %s", archive.Description, description)
				}
			}
		})
	}
}

func TestService_GetAvailableArchives(t *testing.T) {
	cryptoService := crypto.NewService()
	service := NewService(cryptoService)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test .enc files
	for i := 0; i < 3; i++ {
		encPath := filepath.Join(tmpDir, "test"+string(rune('0'+i))+".enc")
		if writeErr := os.WriteFile(encPath, []byte("test"), 0o600); writeErr != nil {
			t.Fatalf("Failed to create test enc file: %v", writeErr)
		}
	}

	// Create non-enc file
	if writeErr := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0o600); writeErr != nil {
		t.Fatalf("Failed to create test txt file: %v", writeErr)
	}

	archives, err := service.GetAvailableArchives(tmpDir)
	if err != nil {
		t.Errorf("GetAvailableArchives() error = %v", err)
	}

	if len(archives) != 3 {
		t.Errorf("Expected 3 archives, got %d", len(archives))
	}

	for _, archive := range archives {
		if !strings.HasSuffix(archive, ".enc") {
			t.Errorf("Expected .enc file, got %s", archive)
		}
	}
}

func TestService_GetAvailableArchives_NonExistentDir(t *testing.T) {
	cryptoService := crypto.NewService()
	service := NewService(cryptoService)

	archives, err := service.GetAvailableArchives("/nonexistent/path")
	if err != nil {
		t.Errorf("GetAvailableArchives() should not error for non-existent dir: %v", err)
	}

	if len(archives) != 0 {
		t.Errorf("Expected 0 archives for non-existent dir, got %d", len(archives))
	}
}

func TestSafePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "goingenv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name    string
		relPath string
		baseDir string
		wantErr bool
	}{
		{
			name:    "Safe path - same directory",
			relPath: "file.txt",
			baseDir: tmpDir,
			wantErr: false,
		},
		{
			name:    "Safe path - subdirectory",
			relPath: "subdir/file.txt",
			baseDir: tmpDir,
			wantErr: false,
		},
		{
			name:    "Unsafe path - parent directory traversal",
			relPath: "../file.txt",
			baseDir: tmpDir,
			wantErr: true,
		},
		{
			name:    "Unsafe path - double parent traversal",
			relPath: "../../file.txt",
			baseDir: tmpDir,
			wantErr: true,
		},
		{
			name:    "Unsafe path - absolute path",
			relPath: "/etc/passwd",
			baseDir: tmpDir,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := safePath(tt.relPath, tt.baseDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("safePath(%q, %q) error = %v, wantErr %v", tt.relPath, tt.baseDir, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == "" {
				t.Errorf("safePath(%q, %q) returned empty path for valid input", tt.relPath, tt.baseDir)
			}
		})
	}
}

func TestService_Unpack_PathTraversalPrevention(t *testing.T) {
	cryptoService := crypto.NewService()
	service := NewService(cryptoService)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a malicious tar archive with path traversal
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Add a file with path traversal
	header := &tar.Header{
		Name: "../../../etc/malicious",
		Mode: 0o600,
		Size: int64(len("malicious content")),
	}
	if headerErr := tw.WriteHeader(header); headerErr != nil {
		t.Fatalf("Failed to write tar header: %v", headerErr)
	}
	if _, writeErr := tw.Write([]byte("malicious content")); writeErr != nil {
		t.Fatalf("Failed to write tar content: %v", writeErr)
	}
	_ = tw.Close()

	// Encrypt the malicious tar
	password := "testpassword123"
	encryptedData, err := cryptoService.Encrypt(buf.Bytes(), password)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	archivePath := filepath.Join(tmpDir, "malicious.enc")
	if writeErr := os.WriteFile(archivePath, encryptedData, 0o600); writeErr != nil {
		t.Fatalf("Failed to write archive: %v", writeErr)
	}

	targetDir := filepath.Join(tmpDir, "extracted")
	if mkdirErr := os.MkdirAll(targetDir, 0o700); mkdirErr != nil {
		t.Fatalf("Failed to create target dir: %v", mkdirErr)
	}

	// Attempt to unpack - should fail with path traversal error
	err = service.Unpack(types.UnpackOptions{
		ArchivePath: archivePath,
		Password:    password,
		TargetDir:   targetDir,
		Overwrite:   true,
	})

	if err == nil {
		t.Error("Unpack should fail for path traversal attempt")
	}

	if archiveErr, ok := err.(*types.ArchiveError); ok {
		if !strings.Contains(archiveErr.Err.Error(), "unsafe path") &&
			!strings.Contains(archiveErr.Err.Error(), "path traversal") {
			t.Errorf("Expected path traversal error, got: %v", err)
		}
	}
}

func TestService_Unpack_AbsolutePathPrevention(t *testing.T) {
	cryptoService := crypto.NewService()
	service := NewService(cryptoService)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a malicious tar archive with absolute path
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Add a file with absolute path
	header := &tar.Header{
		Name: "/etc/malicious",
		Mode: 0o600,
		Size: int64(len("malicious content")),
	}
	if headerErr := tw.WriteHeader(header); headerErr != nil {
		t.Fatalf("Failed to write tar header: %v", headerErr)
	}
	if _, writeErr := tw.Write([]byte("malicious content")); writeErr != nil {
		t.Fatalf("Failed to write tar content: %v", writeErr)
	}
	_ = tw.Close()

	// Encrypt the malicious tar
	password := "testpassword123"
	encryptedData, err := cryptoService.Encrypt(buf.Bytes(), password)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	archivePath := filepath.Join(tmpDir, "malicious.enc")
	if writeErr := os.WriteFile(archivePath, encryptedData, 0o600); writeErr != nil {
		t.Fatalf("Failed to write archive: %v", writeErr)
	}

	targetDir := filepath.Join(tmpDir, "extracted")
	if mkdirErr := os.MkdirAll(targetDir, 0o700); mkdirErr != nil {
		t.Fatalf("Failed to create target dir: %v", mkdirErr)
	}

	// Attempt to unpack - should fail with unsafe path error
	err = service.Unpack(types.UnpackOptions{
		ArchivePath: archivePath,
		Password:    password,
		TargetDir:   targetDir,
		Overwrite:   true,
	})

	if err == nil {
		t.Error("Unpack should fail for absolute path attempt")
	}

	if archiveErr, ok := err.(*types.ArchiveError); ok {
		if !strings.Contains(archiveErr.Err.Error(), "unsafe path") {
			t.Errorf("Expected unsafe path error, got: %v", err)
		}
	}
}

func TestService_PackUnpack_RoundTrip(t *testing.T) {
	cryptoService := crypto.NewService()
	service := NewService(cryptoService)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple test files
	files := map[string][]byte{
		".env":           []byte("DATABASE_URL=postgres://localhost\nAPI_KEY=secret"),
		".env.local":     []byte("LOCAL_VAR=value"),
		"config/.env.db": []byte("DB_HOST=localhost\nDB_PORT=5432"),
	}

	var envFiles []types.EnvFile
	for relPath, content := range files {
		fullPath := filepath.Join(tmpDir, relPath)
		dir := filepath.Dir(fullPath)
		if mkdirErr := os.MkdirAll(dir, 0o700); mkdirErr != nil {
			t.Fatalf("Failed to create dir: %v", mkdirErr)
		}
		if writeErr := os.WriteFile(fullPath, content, 0o600); writeErr != nil {
			t.Fatalf("Failed to create test file: %v", writeErr)
		}
		envFiles = append(envFiles, types.EnvFile{
			Path:         fullPath,
			RelativePath: relPath,
			Size:         int64(len(content)),
			ModTime:      time.Now(),
		})
	}

	archivePath := filepath.Join(tmpDir, "roundtrip.enc")
	password := "roundtrip-password-123"

	// Pack
	err = service.Pack(types.PackOptions{
		Files:       envFiles,
		OutputPath:  archivePath,
		Password:    password,
		Description: "Round trip test",
	})
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	// Delete original files
	for relPath := range files {
		os.Remove(filepath.Join(tmpDir, relPath))
	}

	// Unpack to new location
	extractDir := filepath.Join(tmpDir, "extracted")
	if mkdirErr := os.MkdirAll(extractDir, 0o700); mkdirErr != nil {
		t.Fatalf("Failed to create extract dir: %v", mkdirErr)
	}

	err = service.Unpack(types.UnpackOptions{
		ArchivePath: archivePath,
		Password:    password,
		TargetDir:   extractDir,
		Overwrite:   true,
	})
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}

	// Verify all files were extracted correctly
	for relPath, expectedContent := range files {
		extractedPath := filepath.Join(extractDir, relPath)
		actualContent, err := os.ReadFile(extractedPath)
		if err != nil {
			t.Errorf("Failed to read extracted file %s: %v", relPath, err)
			continue
		}
		if !bytes.Equal(actualContent, expectedContent) {
			t.Errorf("Content mismatch for %s:\nExpected: %s\nActual: %s", relPath, expectedContent, actualContent)
		}
	}
}

func TestService_Unpack_OverwriteAndBackup(t *testing.T) {
	cryptoService := crypto.NewService()
	service := NewService(cryptoService)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file and archive
	testContent := []byte("ORIGINAL=content")
	testFilePath := filepath.Join(tmpDir, ".env")
	if writeErr := os.WriteFile(testFilePath, testContent, 0o600); writeErr != nil {
		t.Fatalf("Failed to create test file: %v", writeErr)
	}

	archivePath := filepath.Join(tmpDir, "test.enc")
	password := "testpassword123"

	err = service.Pack(types.PackOptions{
		Files: []types.EnvFile{
			{
				Path:         testFilePath,
				RelativePath: ".env",
				Size:         int64(len(testContent)),
				ModTime:      time.Now(),
			},
		},
		OutputPath: archivePath,
		Password:   password,
	})
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	// Create existing file in target directory
	targetDir := filepath.Join(tmpDir, "target")
	if mkdirErr := os.MkdirAll(targetDir, 0o700); mkdirErr != nil {
		t.Fatalf("Failed to create target dir: %v", mkdirErr)
	}

	existingContent := []byte("EXISTING=file")
	existingPath := filepath.Join(targetDir, ".env")
	if writeErr := os.WriteFile(existingPath, existingContent, 0o600); writeErr != nil {
		t.Fatalf("Failed to create existing file: %v", writeErr)
	}

	// Test with backup=true
	err = service.Unpack(types.UnpackOptions{
		ArchivePath: archivePath,
		Password:    password,
		TargetDir:   targetDir,
		Overwrite:   true,
		Backup:      true,
	})
	if err != nil {
		t.Fatalf("Unpack with backup failed: %v", err)
	}

	// Verify backup was created
	backupPath := existingPath + ".backup"
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Errorf("Backup file not created: %v", err)
	}
	if !bytes.Equal(backupContent, existingContent) {
		t.Errorf("Backup content mismatch")
	}

	// Verify file was overwritten
	newContent, err := os.ReadFile(existingPath)
	if err != nil {
		t.Errorf("Failed to read extracted file: %v", err)
	}
	if !bytes.Equal(newContent, testContent) {
		t.Errorf("File was not overwritten correctly")
	}
}

func BenchmarkPack(b *testing.B) {
	cryptoService := crypto.NewService()
	service := NewService(cryptoService)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testContent := bytes.Repeat([]byte("TEST=value\n"), 1000)
	testFilePath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(testFilePath, testContent, 0o600); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	outputPath := filepath.Join(tmpDir, "bench.enc")
	password := "benchmarkpassword"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := service.Pack(types.PackOptions{
			Files: []types.EnvFile{
				{
					Path:         testFilePath,
					RelativePath: ".env",
					Size:         int64(len(testContent)),
					ModTime:      time.Now(),
				},
			},
			OutputPath: outputPath,
			Password:   password,
		})
		if err != nil {
			b.Fatalf("Pack failed: %v", err)
		}
	}
}

func BenchmarkUnpack(b *testing.B) {
	cryptoService := crypto.NewService()
	service := NewService(cryptoService)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goingenv-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create and pack test file
	testContent := bytes.Repeat([]byte("TEST=value\n"), 1000)
	testFilePath := filepath.Join(tmpDir, ".env")
	if writeErr := os.WriteFile(testFilePath, testContent, 0o600); writeErr != nil {
		b.Fatalf("Failed to create test file: %v", writeErr)
	}

	archivePath := filepath.Join(tmpDir, "bench.enc")
	password := "benchmarkpassword"

	err = service.Pack(types.PackOptions{
		Files: []types.EnvFile{
			{
				Path:         testFilePath,
				RelativePath: ".env",
				Size:         int64(len(testContent)),
				ModTime:      time.Now(),
			},
		},
		OutputPath: archivePath,
		Password:   password,
	})
	if err != nil {
		b.Fatalf("Pack failed: %v", err)
	}

	targetDir := filepath.Join(tmpDir, "extracted")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = os.RemoveAll(targetDir)
		_ = os.MkdirAll(targetDir, 0o700) //nolint:errcheck // benchmark setup

		err := service.Unpack(types.UnpackOptions{
			ArchivePath: archivePath,
			Password:    password,
			TargetDir:   targetDir,
			Overwrite:   true,
		})
		if err != nil {
			b.Fatalf("Unpack failed: %v", err)
		}
	}
}
