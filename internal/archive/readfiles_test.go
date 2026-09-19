package archive

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"goingenv/pkg/types"
	"goingenv/test/testutils"
)

// packFixture writes two env files, packs them and returns the archive path.
func packFixture(t *testing.T, service *Service, env string) (tmpDir, archivePath string) {
	t.Helper()
	tmpDir = t.TempDir()
	files := map[string]string{
		".env":            "TOP=1\nSECRET=hunter2\n",
		"config/.env.dev": "DEV=true\n",
	}
	var envFiles []types.EnvFile
	for rel, content := range files {
		path := filepath.Join(tmpDir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		envFiles = append(envFiles, types.EnvFile{Path: path, RelativePath: rel, Size: int64(len(content)), ModTime: time.Now()})
	}
	archivePath = filepath.Join(tmpDir, "test.enc")
	err := service.Pack(types.PackOptions{Files: envFiles, OutputPath: archivePath, Password: "pw", Description: "fixture", Env: env})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	return tmpDir, archivePath
}

func TestService_ReadFiles(t *testing.T) {
	service := NewService(testutils.FastCrypto())
	tmpDir, archivePath := packFixture(t, service, "prod")

	// Remove the sources so a stray write to disk would be visible.
	if err := os.RemoveAll(filepath.Join(tmpDir, "config")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(tmpDir, ".env")); err != nil {
		t.Fatal(err)
	}

	archive, files, err := service.ReadFiles(archivePath, "pw")
	if err != nil {
		t.Fatalf("ReadFiles: %v", err)
	}
	if archive.Env != "prod" {
		t.Errorf("Env = %q, want prod", archive.Env)
	}
	if archive.Version != MetadataVersion {
		t.Errorf("Version = %q, want %q", archive.Version, MetadataVersion)
	}
	if len(files) != 2 {
		t.Fatalf("got %d entries, want 2: %v", len(files), files)
	}
	if string(files[".env"]) != "TOP=1\nSECRET=hunter2\n" {
		t.Errorf(".env = %q", files[".env"])
	}
	if string(files["config/.env.dev"]) != "DEV=true\n" {
		t.Errorf("config/.env.dev = %q", files["config/.env.dev"])
	}
	if _, ok := files["metadata.json"]; ok {
		t.Error("metadata.json must not be returned as a file")
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "test.enc" {
		t.Errorf("ReadFiles wrote to disk: %v", entries)
	}
}

func TestService_ReadFilesWrongPassword(t *testing.T) {
	service := NewService(testutils.FastCrypto())
	_, archivePath := packFixture(t, service, "")

	_, _, err := service.ReadFiles(archivePath, "nope")
	if !errors.Is(err, types.ErrDecryptFailed) {
		t.Errorf("err = %v, want ErrDecryptFailed through the ArchiveError", err)
	}
	var archiveErr *types.ArchiveError
	if !errors.As(err, &archiveErr) || archiveErr.Operation != "read" {
		t.Errorf("err = %v, want ArchiveError{read}", err)
	}
}

// A blob from before format v1 must surface the legacy sentinel through
// every reader so the CLI can name the remedy.
func TestService_LegacyArchiveSurfacesSentinel(t *testing.T) {
	service := NewService(testutils.FastCrypto())
	dir := t.TempDir()
	legacy := make([]byte, 128)
	if _, err := rand.Read(legacy); err != nil {
		t.Fatal(err)
	}
	if string(legacy[:4]) == "GENV" {
		legacy[0] ^= 0xFF
	}
	path := filepath.Join(dir, "old.enc")
	if err := os.WriteFile(path, legacy, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := service.List(path, "pw"); !errors.Is(err, types.ErrLegacyArchive) {
		t.Errorf("List err = %v, want ErrLegacyArchive", err)
	}
	if _, _, err := service.ReadFiles(path, "pw"); !errors.Is(err, types.ErrLegacyArchive) {
		t.Errorf("ReadFiles err = %v, want ErrLegacyArchive", err)
	}
	_, err := service.Unpack(types.UnpackOptions{ArchivePath: path, Password: "pw", TargetDir: dir})
	if !errors.Is(err, types.ErrLegacyArchive) {
		t.Errorf("Unpack err = %v, want ErrLegacyArchive", err)
	}
}
