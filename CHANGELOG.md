# Changelog

All notable changes to goingenv will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **`goingenv init` command** - Required initialization step for each project directory
- **Brand design system** - New `DESIGN.md` documenting logo, colors, and UI specifications
- **CLI output system** - Consistent branded output with prefix indicators (`[*]`, `[+]`, `[!]`, `[x]`, `[>]`, `[?]`, `[-]`, `[~]`)
- **Relative timestamps** - New `FormatTimeAgo` utility showing "2 hours ago" style times
- TTY-aware color detection for CLI output
- **Automatic release system** - Pushes to main branch automatically create stable releases
- Comprehensive CI/CD pipeline with GitHub Actions
- Automated release creation with cross-platform binaries
- Install script for Linux and macOS with platform detection
- Debug logging system for TUI mode with --verbose flag
- Comprehensive documentation split into specialized guides
- New TUI initialization screen for uninitialized projects
- Initialization requirement verification tests
- Semantic version control via commit message flags ([major], [minor], [skip-release])

### Changed
- **BREAKING**: All commands now require `goingenv init` to be run first in each project directory
- **BREAKING**: Simplified `status` command - removed `--directory`, `--archives`, `--files`, `--config`, `--stats`, `--recommendations` flags; directory is now a positional argument
- **Updated color palette** - Changed from purple (`#7D56F4`) to teal (`#22d3a7`) brand color
- **TUI redesign** - Borderless components, chevron menu selection, branded header with version
- TUI now shows initialization screen when project is not initialized
- Archive operations no longer auto-create `.goingenv` directory
- CLI commands now display branded header `[*] goingenv v{version}`
- Encrypted archives (`.enc` files) are now shareable via git by default - no auto-gitignore modification
- Restructured README.md for better user experience
- Enhanced Makefile with CI and release targets
- Improved TUI with debug mode indicators
- Updated documentation to reflect initialization requirement

### Security
- Added security scanning with gosec and nancy
- Implemented checksum verification for releases
- Enhanced install script with security features
- Improved initialization workflow prevents accidental directory creation

## [1.0.0] - 2025-08-19

### Added
- Initial release of GoingEnv
- Environment file scanning and detection
- AES-256-GCM encryption for secure archiving
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

This changelog is automatically enhanced by GitHub Actions during releases.
Manual entries can be added to the [Unreleased] section above.

### Version Types

- **Major (X.0.0)**: Breaking changes, major new features
- **Minor (0.X.0)**: New features, backward compatible
- **Patch (0.0.X)**: Bug fixes, security updates
- **Prerelease (0.0.0-alpha.1)**: Development versions