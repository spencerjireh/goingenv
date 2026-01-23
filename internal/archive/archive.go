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
	if chmodErr := tmpFile.Chmod(0o600); chmodErr != nil {
		_ = os.Remove(tmpFile.Name())
		_ = tmpFile.Close()
		return &types.ArchiveError{
			Operation: "pack",
			Path:      opts.OutputPath,
			Err:       fmt.Errorf("failed to secure temporary file: %w", chmodErr),
		}
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	defer func() { _ = tmpFile.Close() }()

	// Create tar writer
	tarWriter := tar.NewWriter(tmpFile)
	defer func() { _ = tarWriter.Close() }()

	// Write metadata first
	if metaErr := s.writeMetadata(tarWriter, &archive); metaErr != nil {
		return &types.ArchiveError{
			Operation: "pack",
			Path:      opts.OutputPath,
			Err:       fmt.Errorf("failed to write metadata: %w", metaErr),
		}
	}

	// Write files to tar
	for i := range opts.Files {
		if writeErr := s.writeFileToTar(tarWriter, &opts.Files[i]); writeErr != nil {
			return &types.ArchiveError{
				Operation: "pack",
				Path:      opts.Files[i].Path,
				Err:       fmt.Errorf("failed to write file to archive: %w", writeErr),
			}
		}
	}

	// Close tar writer to flush data
	if closeErr := tarWriter.Close(); closeErr != nil {
		return &types.ArchiveError{
			Operation: "pack",
			Path:      opts.OutputPath,
			Err:       fmt.Errorf("failed to close tar writer: %w", closeErr),
		}
	}

	// Read tar data
	if _, seekErr := tmpFile.Seek(0, 0); seekErr != nil {
		return &types.ArchiveError{
			Operation: "pack",
			Path:      opts.OutputPath,
			Err:       fmt.Errorf("failed to seek to beginning: %w", seekErr),
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
	if err := os.WriteFile(opts.OutputPath, encryptedData, 0o600); err != nil {
		return &types.ArchiveError{
			Operation: "pack",
			Path:      opts.OutputPath,
			Err:       fmt.Errorf("failed to write encrypted file: %w", err),
		}
	}

	return nil
}

// safePath validates and returns target path, or error if unsafe (pure function)
func safePath(name, baseDir string) (string, error) {
	if filepath.IsAbs(name) || strings.Contains(name, "..") {
		return "", fmt.Errorf("unsafe path detected: %s", name)
	}

	target := filepath.Join(baseDir, name) //nolint:gosec // G305: path is validated below

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %w", err)
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("failed to resolve target path: %w", err)
	}

	if !strings.HasSuffix(absBase, string(filepath.Separator)) {
		absBase += string(filepath.Separator)
	}
	if !strings.HasPrefix(absTarget, absBase) {
		return "", fmt.Errorf("path traversal detected: %s", name)
	}

	return target, nil
}

// ensureDir creates directory with restrictive permissions
func ensureDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o700)
}

// backupFile creates a backup of existing file
func backupFile(path string) error {
	return os.Rename(path, path+".backup")
}

// handleExisting handles existing file (skip, backup, or overwrite)
func handleExisting(path string, overwrite, backup bool) (skip bool, err error) {
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		return false, nil // file doesn't exist, proceed
	}

	if !overwrite {
		fmt.Printf("Skipping existing file: %s\n", path)
		return true, nil
	}

	if backup {
		if backupErr := backupFile(path); backupErr != nil {
			return false, fmt.Errorf("failed to create backup: %w", backupErr)
		}
	}
	return false, nil
}

// decryptArchive reads and decrypts archive data
func (s *Service) decryptArchive(archivePath, password string) ([]byte, error) {
	encryptedData, err := os.ReadFile(archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read archive: %w", err)
	}

	tarData, err := s.crypto.Decrypt(encryptedData, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt archive: %w", err)
	}

	return tarData, nil
}

// extractEntry extracts a single tar entry
func (s *Service) extractEntry(tarReader *tar.Reader, header *tar.Header, opts types.UnpackOptions) error {
	if header.Name == "metadata.json" {
		return nil // skip metadata
	}

	targetPath, pathErr := safePath(header.Name, opts.TargetDir)
	if pathErr != nil {
		return pathErr
	}

	if dirErr := ensureDir(targetPath); dirErr != nil {
		return fmt.Errorf("failed to create directory: %w", dirErr)
	}

	skip, existErr := handleExisting(targetPath, opts.Overwrite, opts.Backup)
	if existErr != nil {
		return existErr
	}
	if skip {
		return nil
	}

	return s.extractFile(tarReader, targetPath, header)
}

// Unpack decrypts and extracts files from an archive
func (s *Service) Unpack(opts types.UnpackOptions) error {
	tarData, err := s.decryptArchive(opts.ArchivePath, opts.Password)
	if err != nil {
		return &types.ArchiveError{
			Operation: "unpack",
			Path:      opts.ArchivePath,
			Err:       err,
		}
	}

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

		if extractErr := s.extractEntry(tarReader, header, opts); extractErr != nil {
			return &types.ArchiveError{
				Operation: "unpack",
				Path:      header.Name,
				Err:       extractErr,
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
func (s *Service) writeMetadata(tarWriter *tar.Writer, archive *types.Archive) error {
	metadataJSON, err := json.Marshal(archive)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	header := &tar.Header{
		Name: "metadata.json",
		Mode: 0o600,
		Size: int64(len(metadataJSON)),
	}

	if headerErr := tarWriter.WriteHeader(header); headerErr != nil {
		return fmt.Errorf("failed to write metadata header: %w", headerErr)
	}

	if _, writeErr := tarWriter.Write(metadataJSON); writeErr != nil {
		return fmt.Errorf("failed to write metadata: %w", writeErr)
	}

	return nil
}

// writeFileToTar writes a file to the tar archive
func (s *Service) writeFileToTar(tarWriter *tar.Writer, file *types.EnvFile) error {
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

	if headerErr := tarWriter.WriteHeader(header); headerErr != nil {
		return fmt.Errorf("failed to write header for %s: %w", file.Path, headerErr)
	}

	fileContent, err := os.Open(file.Path)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", file.Path, err)
	}
	defer fileContent.Close()

	if _, copyErr := io.Copy(tarWriter, fileContent); copyErr != nil {
		return fmt.Errorf("failed to write file %s: %w", file.Path, copyErr)
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
	safeMode := os.FileMode(header.Mode&0o777) & 0o600 //nolint:gosec // G115: mode is masked to safe range
	if safeMode == 0 {
		safeMode = 0o600 // Default to owner read/write if no permissions
	}
	if err := os.Chmod(targetPath, safeMode); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := os.Chtimes(targetPath, time.Now(), header.ModTime); err != nil {
		return fmt.Errorf("failed to set modification time: %w", err)
	}

	return nil
}
