package cli

import (
	"time"

	"goingenv/pkg/types"
)

// Payload types for --format json.
//
// These are a contract. Field names are what scripts key on, so treat renaming
// or removing one as a breaking change; adding a field is not. They are
// declared here rather than inline at each call site so the whole contract can
// be reviewed in one place.
//
// Times are RFC 3339 (Go's default for time.Time), sizes are bytes as numbers
// rather than pre-formatted strings, and paths are reported as the command saw
// them.

// FileInfo describes one environment file. Shared by status, pack and unpack
// so a script can handle files the same way whichever command produced them.
type FileInfo struct {
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	Checksum string    `json:"checksum,omitempty"`
}

// ArchiveInfo describes one archive on disk.
type ArchiveInfo struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// ConfigInfo reports the effective configuration and, importantly, where it
// came from. Since a project-local .goingenv/config.json takes whole-file
// precedence over ~/.goingenv.json, "which config is in force" is the question
// a script most often needs answered.
type ConfigInfo struct {
	Path        string   `json:"path"`
	Source      string   `json:"source"` // "project" or "user"
	Depth       int      `json:"depth"`
	MaxFileSize int64    `json:"max_file_size"`
	Patterns    []string `json:"patterns"`
	Excludes    []string `json:"excludes,omitempty"`
	EnvExcludes []string `json:"env_excludes,omitempty"`
	Manifest    bool     `json:"manifest"`
}

// StatusPayload is the `goingenv status --format json` document.
type StatusPayload struct {
	Directory   string        `json:"directory"`
	Initialized bool          `json:"initialized"`
	Config      ConfigInfo    `json:"config"`
	EnvFiles    []FileInfo    `json:"env_files"`
	Archives    []ArchiveInfo `json:"archives"`
	TotalSize   int64         `json:"total_size"`
}

// ListPayload is the `goingenv list --format json` document.
type ListPayload struct {
	Archive ListArchiveInfo `json:"archive"`
	Files   []FileInfo      `json:"files"`
	Count   int             `json:"count"`
}

// ListArchiveInfo is the archive metadata decrypted from its header.
type ListArchiveInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	CreatedAt   time.Time `json:"created_at"`
	Version     string    `json:"version"`
	Description string    `json:"description,omitempty"`
	Env         string    `json:"env,omitempty"`
}

// PackPayload is the `goingenv pack --format json` document. DryRun is
// reported explicitly so a script can tell a rehearsal from a real archive
// without inspecting the filesystem.
type PackPayload struct {
	Archive   string     `json:"archive"`
	Env       string     `json:"env,omitempty"`
	Manifest  string     `json:"manifest,omitempty"`
	Files     []FileInfo `json:"files"`
	Count     int        `json:"count"`
	TotalSize int64      `json:"total_size"`
	DryRun    bool       `json:"dry_run"`
}

// UnpackPayload is the `goingenv unpack --format json` document. Files and
// Count describe what was written; Skipped lists the entries left alone
// because they already existed and --overwrite was not given. Both are always
// present, empty rather than null.
type UnpackPayload struct {
	Archive string     `json:"archive"`
	Target  string     `json:"target"`
	Files   []FileInfo `json:"files"`
	Count   int        `json:"count"`
	Skipped []string   `json:"skipped"`
	DryRun  bool       `json:"dry_run"`
}

// DiffSide names one side of a diff: an archive, or the env files on disk.
type DiffSide struct {
	Kind string `json:"kind"` // "archive" or "worktree"
	Path string `json:"path,omitempty"`
}

// KeyDiff is one key that differs between the two sides. Values are never
// included: the diff exists so a change can be reviewed without them.
type KeyDiff struct {
	Key    string `json:"key"`
	Change string `json:"change"` // "added", "removed" or "changed"
}

// FileDiff is one file that differs. A file that exists on only one side is
// "added" or "removed" and lists every key with the same change, so a
// consumer counting key changes needs no special case. A file on both sides
// with differing keys is "modified".
type FileDiff struct {
	Path   string    `json:"path"`
	Status string    `json:"status"`
	Keys   []KeyDiff `json:"keys"`
}

// DiffPayload is the `goingenv diff --format json` document. Files holds only
// the files that differ; Changed is false when it is empty.
type DiffPayload struct {
	From    DiffSide   `json:"from"`
	To      DiffSide   `json:"to"`
	Files   []FileDiff `json:"files"`
	Changed bool       `json:"changed"`
}

// InitPayload is the `goingenv init --format json` document. Created is false
// when the project was already initialised, which is a success, not an error.
type InitPayload struct {
	Created       bool   `json:"created"`
	GoingEnvDir   string `json:"goingenv_dir"`
	ProjectConfig string `json:"project_config,omitempty"`
}

// toFileInfo converts a scanned or archived env file to its payload form.
func toFileInfo(f *types.EnvFile) FileInfo {
	return FileInfo{
		Path:     f.RelativePath,
		Size:     f.Size,
		Modified: f.ModTime,
		Checksum: f.Checksum,
	}
}

// toFileInfos converts a slice, always returning a non-nil slice so the JSON
// carries `[]` rather than `null` when nothing matched. A consumer iterating
// the field should not have to special-case empty.
func toFileInfos(files []types.EnvFile) []FileInfo {
	out := make([]FileInfo, 0, len(files))
	for i := range files {
		out = append(out, toFileInfo(&files[i]))
	}
	return out
}

// Porcelain column orders. Documented here, and in docs/, because they are the
// part scripts hard-code by position.
//
//	status env file   path <TAB> size <TAB> modified
//	status archive    name <TAB> size <TAB> modified
//	list              path <TAB> size <TAB> modified <TAB> checksum
//	pack              path <TAB> size <TAB> modified
//	unpack            path <TAB> size <TAB> modified
//	init              dir  <TAB> created
//	diff file         file <TAB> path <TAB> added|removed
//	diff key          key  <TAB> path <TAB> KEY <TAB> added|removed|changed
//
// status and diff emit two record kinds, so their rows are prefixed with a
// type column to keep them distinguishable in a single stream.
const (
	porcelainKindEnv     = "env"
	porcelainKindArchive = "archive"
	porcelainKindFile    = "file"
	porcelainKindKey     = "key"
)
