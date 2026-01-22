package archive

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"goingenv/internal/config"
	"goingenv/pkg/types"
)

// Service implements the Archiver interface
type Service struct {
	crypto types.Cryptor
}

// NewService creates a new archive service
func NewService(crypto types.Cryptor) *Service {
	return &Service{
		crypto: crypto,
	}
}

// Pack creates an encrypted archive of the given files
func (s *Service) Pack(opts types.PackOptions) error {
	if len(opts.Files) == 0 {
		return &types.ArchiveError{
			Operation: "pack",
			Path:      opts.OutputPath,
			Err:       fmt.Errorf("no files to pack"),
		}
	}

	// Calculate total size
	var totalSize int64
	for _, file := range opts.Files {
		totalSize += file.Size
	}

	// Create archive metadata
	archive := types.Archive{
		CreatedAt:   time.Now(),
		Files:       opts.Files,
		TotalSize:   totalSize,
		Description: opts.Description,
		Version:     "1.0.0", // You might want to make this configurable
	}

	// Create temporary file for the tar archive
	tmpFile, err := os.CreateTemp("", "goingenv-*.tar")
	if err != nil {
		return &types.ArchiveError{
			Operation: "pack",
			Path:      opts.OutputPath,
			Err:       fmt.Errorf("failed to create temporary file: %w", err),
		}
	}
	// Secure temp file permissions immediately
	if err := tmpFile.Chmod(0600); err != nil {
		os.Remove(tmpFile.Name())
		tmpFile.Close()
		return &types.ArchiveError{
			Operation: "pack",
			Path:      opts.OutputPath,
			Err:       fmt.Errorf("failed to secure temporary file: %w", err),
		}
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(tmpFile)
	defer tarWriter.Close()

	// Write metadata first
	if err := s.writeMetadata(tarWriter, archive); err != nil {
		return &types.ArchiveError{
			Operation: "pack",
			Path:      opts.OutputPath,
			Err:       fmt.Errorf("failed to write metadata: %w", err),
		}
	}

	// Write files to tar
	for _, file := range opts.Files {
		if err := s.writeFileToTar(tarWriter, file); err != nil {
			return &types.ArchiveError{
				Operation: "pack",
				Path:      file.Path,
				Err:       fmt.Errorf("failed to write file to archive: %w", err),
			}
		}
	}

	// Close tar writer to flush data
	if err := tarWriter.Close(); err != nil {
		return &types.ArchiveError{
			Operation: "pack",
			Path:      opts.OutputPath,
			Err:       fmt.Errorf("failed to close tar writer: %w", err),
		}
	}

	// Read tar data
	if _, err := tmpFile.Seek(0, 0); err != nil {
		return &types.ArchiveError{
			Operation: "pack",
			Path:      opts.OutputPath,
			Err:       fmt.Errorf("failed to seek to beginning: %w", err),
		}
	}

	tarData, err := io.ReadAll(tmpFile)
	if err != nil {
		return &types.ArchiveError{
			Operation: "pack",
			Path:      opts.OutputPath,
			Err:       fmt.Errorf("failed to read tar data: %w", err),
		}
	}

	// Encrypt the data
	encryptedData, err := s.crypto.Encrypt(tarData, opts.Password)
	if err != nil {
		return &types.ArchiveError{
			Operation: "pack",
			Path:      opts.OutputPath,
			Err:       fmt.Errorf("failed to encrypt data: %w", err),
		}
	}

	// Write encrypted data to output file with restrictive permissions
	if err := os.WriteFile(opts.OutputPath, encryptedData, 0600); err != nil {
		return &types.ArchiveError{
			Operation: "pack",
			Path:      opts.OutputPath,
			Err:       fmt.Errorf("failed to write encrypted file: %w", err),
		}
	}

	return nil
}

// isPathSafe validates that targetPath is safely within basePath
// Returns false if path traversal is detected
func isPathSafe(basePath, targetPath string) bool {
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return false
	}
	// Ensure the base path ends with separator for accurate prefix matching
	if !strings.HasSuffix(absBase, string(filepath.Separator)) {
		absBase += string(filepath.Separator)
	}
	return strings.HasPrefix(absTarget, absBase)
}

// Unpack decrypts and extracts files from an archive
func (s *Service) Unpack(opts types.UnpackOptions) error {
	// Read encrypted file
	encryptedData, err := os.ReadFile(opts.ArchivePath)
	if err != nil {
		return &types.ArchiveError{
			Operation: "unpack",
			Path:      opts.ArchivePath,
			Err:       fmt.Errorf("failed to read archive: %w", err),
		}
	}

	// Decrypt the data
	tarData, err := s.crypto.Decrypt(encryptedData, opts.Password)
	if err != nil {
		return &types.ArchiveError{
			Operation: "unpack",
			Path:      opts.ArchivePath,
			Err:       fmt.Errorf("failed to decrypt archive: %w", err),
		}
	}

	// Create tar reader
	tarReader := tar.NewReader(strings.NewReader(string(tarData)))

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return &types.ArchiveError{
				Operation: "unpack",
				Path:      opts.ArchivePath,
				Err:       fmt.Errorf("failed to read tar header: %w", err),
			}
		}

		// Skip metadata file
		if header.Name == "metadata.json" {
			continue
		}

		// Validate path safety before extraction
		if filepath.IsAbs(header.Name) || strings.Contains(header.Name, "..") {
			return &types.ArchiveError{
				Operation: "unpack",
				Path:      header.Name,
				Err:       fmt.Errorf("unsafe path detected: %s", header.Name),
			}
		}

		targetPath := filepath.Join(opts.TargetDir, header.Name)

		// Double-check resolved path is within target directory
		if !isPathSafe(opts.TargetDir, targetPath) {
			return &types.ArchiveError{
				Operation: "unpack",
				Path:      header.Name,
				Err:       fmt.Errorf("path traversal detected: %s", header.Name),
			}
		}

		// Create directory if needed with restrictive permissions
		if err := os.MkdirAll(filepath.Dir(targetPath), 0700); err != nil {
			return &types.ArchiveError{
				Operation: "unpack",
				Path:      targetPath,
				Err:       fmt.Errorf("failed to create directory: %w", err),
			}
		}

		// Handle existing files
		if _, err := os.Stat(targetPath); err == nil {
			if !opts.Overwrite {
				fmt.Printf("Skipping existing file: %s\n", targetPath)
				continue
			}
			if opts.Backup {
				backupPath := targetPath + ".backup"
				if err := os.Rename(targetPath, backupPath); err != nil {
					return &types.ArchiveError{
						Operation: "unpack",
						Path:      targetPath,
						Err:       fmt.Errorf("failed to create backup: %w", err),
					}
				}
			}
		}

		// Extract file
		if err := s.extractFile(tarReader, targetPath, header); err != nil {
			return &types.ArchiveError{
				Operation: "unpack",
				Path:      targetPath,
				Err:       fmt.Errorf("failed to extract file: %w", err),
			}
		}
	}

	return nil
}

// List returns the contents of an archive without extracting
func (s *Service) List(archivePath, password string) (*types.Archive, error) {
	// Read encrypted file
	encryptedData, err := os.ReadFile(archivePath)
	if err != nil {
		return nil, &types.ArchiveError{
			Operation: "list",
			Path:      archivePath,
			Err:       fmt.Errorf("failed to read archive: %w", err),
		}
	}

	// Decrypt the data
	tarData, err := s.crypto.Decrypt(encryptedData, password)
	if err != nil {
		return nil, &types.ArchiveError{
			Operation: "list",
			Path:      archivePath,
			Err:       fmt.Errorf("failed to decrypt archive: %w", err),
		}
	}

	// Create tar reader
	tarReader := tar.NewReader(strings.NewReader(string(tarData)))

	// Read metadata (should be first entry)
	header, err := tarReader.Next()
	if err != nil {
		return nil, &types.ArchiveError{
			Operation: "list",
			Path:      archivePath,
			Err:       fmt.Errorf("failed to read metadata: %w", err),
		}
	}

	if header.Name != "metadata.json" {
		return nil, &types.ArchiveError{
			Operation: "list",
			Path:      archivePath,
			Err:       fmt.Errorf("invalid archive format: missing metadata"),
		}
	}

	metadataBytes, err := io.ReadAll(tarReader)
	if err != nil {
		return nil, &types.ArchiveError{
			Operation: "list",
			Path:      archivePath,
			Err:       fmt.Errorf("failed to read metadata: %w", err),
		}
	}

	var archive types.Archive
	if err := json.Unmarshal(metadataBytes, &archive); err != nil {
		return nil, &types.ArchiveError{
			Operation: "list",
			Path:      archivePath,
			Err:       fmt.Errorf("failed to unmarshal metadata: %w", err),
		}
	}

	return &archive, nil
}

// GetAvailableArchives returns a list of available archive files
func (s *Service) GetAvailableArchives(dir string) ([]string, error) {
	var archives []string

	if dir == "" {
		dir = config.GetGoingEnvDir()
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return archives, nil
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".enc") {
			archives = append(archives, filepath.Join(dir, file.Name()))
		}
	}

	return archives, nil
}

// writeMetadata writes archive metadata to tar
func (s *Service) writeMetadata(tarWriter *tar.Writer, archive types.Archive) error {
	metadataJSON, err := json.Marshal(archive)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	header := &tar.Header{
		Name: "metadata.json",
		Mode: 0600,
		Size: int64(len(metadataJSON)),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write metadata header: %w", err)
	}

	if _, err := tarWriter.Write(metadataJSON); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// writeFileToTar writes a file to the tar archive
func (s *Service) writeFileToTar(tarWriter *tar.Writer, file types.EnvFile) error {
	fileInfo, err := os.Stat(file.Path)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", file.Path, err)
	}

	header := &tar.Header{
		Name:    file.RelativePath,
		Mode:    int64(fileInfo.Mode()),
		Size:    fileInfo.Size(),
		ModTime: fileInfo.ModTime(),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write header for %s: %w", file.Path, err)
	}

	fileContent, err := os.Open(file.Path)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", file.Path, err)
	}
	defer fileContent.Close()

	if _, err := io.Copy(tarWriter, fileContent); err != nil {
		return fmt.Errorf("failed to write file %s: %w", file.Path, err)
	}

	return nil
}

// extractFile extracts a single file from tar to the filesystem
func (s *Service) extractFile(tarReader *tar.Reader, targetPath string, header *tar.Header) error {
	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", targetPath, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, tarReader); err != nil {
		return fmt.Errorf("failed to extract file %s: %w", targetPath, err)
	}

	// Set file permissions (use restrictive permissions, masking to safe defaults)
	// Only preserve read/write bits for owner, strip group/world permissions
	safeMode := os.FileMode(header.Mode) & 0600
	if safeMode == 0 {
		safeMode = 0600 // Default to owner read/write if no permissions
	}
	if err := os.Chmod(targetPath, safeMode); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := os.Chtimes(targetPath, time.Now(), header.ModTime); err != nil {
		return fmt.Errorf("failed to set modification time: %w", err)
	}

	return nil
}
