# Changelog

All notable changes to goingenv will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.4.0] - 2026-09-10

### Added
- **Machine-readable output on every command** - a persistent `--format` flag taking `text` (default), `json` and `porcelain`. In a machine format stdout carries only the payload and all human output moves to stderr, so `goingenv status --format json | jq` works. JSON field names and porcelain column order are a contract; `text` is not stable and should not be parsed
- `status --format json` reports which configuration file is in force and whether it is the project-local or the per-user one
- Smoke tests for `internal/tui`, which previously had none

### Fixed
- **`list --format json` was not pipeable.** The branded header and archive summary were printed to stdout ahead of the JSON, so the one documented machine format could not be consumed by anything
- **The TUI archive picker listed only the first archive.** Window size never reached the file picker, so its height stayed 0 and every archive after the first was unreachable. On the previous dependency set it listed none at all
- Errors are printed to stderr rather than stdout, and reported exactly once
- `curl ... | bash -s -- --help` printed `USAGE: bash [OPTIONS]` and examples like `bash --version v1.3.0`. Under a pipe `$0` is literally `bash`, so every command in the help text was inert if pasted
- `install.sh` no longer masks the exit status of the `date` call that builds a backup path

### Changed
- **Minimum Go version is now 1.26** (was 1.24), required by `golang.org/x/crypto` v0.56.0. This affects building from source only; released binaries are unaffected
- Dependencies updated: bubbletea 0.24 to 1.3, bubbles 0.16 to 0.21, lipgloss 0.8 to 1.1, cobra 1.7 to 1.10, termenv 0.15 to 0.16, x/crypto 0.41 to 0.56, x/term 0.34 to 0.45
- `list` no longer declares its own `--format` flag, using the persistent one instead. `table` remains accepted as an alias for `text`, and `csv` is still available on `list`
- Coverage now includes `test/cli` and `test/e2e`, which spawn a compiled binary and previously counted for nothing: 42.8% to 63.4%, with `cmd/goingenv` going from 0% to 100%. No tests were added; those suites were already running but uncounted
- `shellcheck` runs over `install.sh` and its test suite in `make lint` and CI

### Removed
- Unused TUI style helpers `GetScreenStyle`, `GetResponsiveWidth`, `RenderWithIcon`, `RenderCard` and the three screen-size style variables only they referenced

## [1.3.0] - 2026-09-10

### Added
- **Project-local configuration** - a committed `.goingenv/config.json` now takes precedence over `~/.goingenv.json`, so a repository can pin how it is scanned regardless of each developer's personal config. Precedence is whole-file; the two are never merged
- **`goingenv init --project-config`** - writes that project config. Opt-in, and never overwrites an existing one, since it is a committed file
- **`goingenv pack --env-exclude`** - exclude env files by base filename, for example `--env-exclude '\.example$'`. Previously `EnvExcludePatterns` was reachable only by hand-editing the config file

### Changed
- A config file may now omit keys; anything absent falls back to the built-in default instead of failing validation as a zero value
- `--exclude` help text now says it matches directory paths, not filenames, which is what it has always done
- `status` and the TUI now show the config file actually in use, rather than the `.goingenv` directory path

### Removed
- `config.EnsureGoingEnvDir` - unused, and duplicated `InitializeProject`

## [1.2.0] - 2026-09-10

### Fixed
- **`curl ... | bash` installed nothing.** The installer's "run unless sourced"
  guard compared `BASH_SOURCE[0]` to `$0`. When bash reads a script from stdin
  there is no script file, so the comparison failed and `main` never ran: the
  documented install command printed nothing and exited 0.
- **PATH was written to the wrong file for zsh users.** The installer selected a
  shell profile using `$BASH_VERSION`, which is always set inside a bash script,
  so the zsh branch was unreachable. macOS users got their PATH appended to a
  bash profile zsh never reads, then "command not found" in the next terminal.
  Shell detection now uses `$SHELL`.
- Re-running the installer to upgrade no longer fails in non-interactive mode.
- `--uninstall` no longer consumes its own script when piped to bash; both of
  its prompts are now TTY-guarded like the others.
- The PATH check no longer treats the install directory as an unanchored regex,
  which could silently skip adding it, and the export is prepended rather than
  appended so a new install wins over an older copy earlier in PATH.
- `goingenv init` no longer rewrites the user-level `~/.goingenv.json` on every
  run; it writes only when no config exists.
- `-X main.GitCommit` was passed at build time with no corresponding variable,
  so the linker discarded it silently.

### Added
- The installer verifies each download's SHA-256 against the release's
  `checksums.txt` and refuses to install on a mismatch. `--skip-checksum`
  (`SKIP_CHECKSUM=1`) is the documented escape hatch.
- `--version` now reports the commit and build time alongside the version.
- Releases publish SBOMs and a build provenance attestation, verifiable with
  `gh attestation verify <archive> --repo spencerjireh/goingenv`.

### Changed
- Releases are built by GoReleaser and gated on the full CI suite; a tag can no
  longer publish untested code.
- Minimum Go version for building from source is now 1.24.


## [1.1.0] - 2026-01-23

### Added
- **`goingenv init` command** - Required initialization step for each project directory
- **Brand design system** - `docs/design.md` documenting logo, colors, and UI specifications
- **CLI output system** - Consistent branded output with prefix indicators (`[●]`, `[+]`, `[!]`, `[x]`, `[>]`, `[?]`, `[-]`, `[~]`)
- **Relative timestamps** - New `FormatTimeAgo` utility showing "2 hours ago" style times
- TTY-aware color detection for CLI output
- CI/CD pipeline with GitHub Actions: lint, test matrix, security scanning and release
- Automated release creation with cross-platform binaries
- Install script for Linux and macOS with platform detection
- Debug logging system for TUI mode with --verbose flag
- Documentation split into separate security, development and design guides
- New TUI initialization screen for uninitialized projects
- Initialization requirement verification tests

### Changed
- **BREAKING**: All commands now require `goingenv init` to be run first in each project directory
- **BREAKING**: Simplified `status` command - removed `--directory`, `--archives`, `--files`, `--config`, `--stats`, `--recommendations` flags; directory is now a positional argument
- **Updated color palette** - Changed from purple (`#7D56F4`) to teal (`#22d3a7`) brand color
- **TUI redesign** - Borderless components, chevron menu selection, branded header with version
- TUI now shows initialization screen when project is not initialized
- Archive operations no longer auto-create `.goingenv` directory
- CLI commands now display branded header `[●] goingenv v{version}`
- Encrypted archives (`.enc` files) are now shareable via git by default - no auto-gitignore modification
- README restructured around install, usage and the command table
- Makefile gained CI and release targets
- TUI shows a debug-mode indicator when `--verbose` is set
- Updated documentation to reflect initialization requirement

### Security
- Added security scanning with gosec and govulncheck
- Implemented checksum verification for releases
- Install script verifies downloads against the release checksums
- Archive operations no longer create `.goingenv/` implicitly, so a mistyped path cannot scatter directories

## [1.0.0] - 2025-08-19

### Added
- Initial release of goingenv
- Environment file scanning and detection
- AES-256-GCM encryption with PBKDF2 key derivation
- Interactive terminal UI with Bubbletea
- Command-line interface with Cobra
- Support for multiple environment file patterns
- File integrity verification with SHA-256 checksums
- Configurable scan depth and exclude patterns
- Archive management with metadata

### Security
- AES-256 encryption with PBKDF2 key derivation
- Secure random salt and nonce generation
- Password-based archive protection
- File integrity verification

---

## Release Process

Per-release notes are generated on GitHub from commit history by GoReleaser.
This file records notable changes by hand; add entries to [Unreleased] as you
go, and promote them to a version section when cutting a release.

### Version Types

- **Major (X.0.0)**: Breaking changes, major new features
- **Minor (0.X.0)**: New features, backward compatible
- **Patch (0.0.X)**: Bug fixes, security updates
- **Prerelease (0.0.0-alpha.1)**: Development versions

[Unreleased]: https://github.com/spencerjireh/goingenv/compare/v1.4.0...HEAD
[1.4.0]: https://github.com/spencerjireh/goingenv/releases/tag/v1.4.0
[1.3.0]: https://github.com/spencerjireh/goingenv/releases/tag/v1.3.0
[1.2.0]: https://github.com/spencerjireh/goingenv/releases/tag/v1.2.0
[1.1.0]: https://github.com/spencerjireh/goingenv/releases/tag/v1.1.0
[1.0.0]: https://github.com/spencerjireh/goingenv/releases/tag/v1.0.0
