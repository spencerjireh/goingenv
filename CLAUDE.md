# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

goingenv is a Go CLI tool for managing environment files with AES-256-GCM encryption. It features both a CLI (Cobra) and an interactive TUI (Bubbletea) interface. Designed for small teams to securely share encrypted .env files without third-party services.

## Build and Development Commands

```bash
# Build
make build              # Build for current platform
make dev                # Build with race detector

# Testing
make test-complete      # Full test suite (unit + integration + functional)
make ci-test            # CI tests with race detection
go test -v ./pkg/... ./internal/...  # Run specific package tests

# Linting
make fmt && make lint   # Format and lint code
make ci-full            # Run all CI checks locally

# Hot-reload development (requires air)
make watch              # Auto-rebuild on changes
make watch-run ARGS="status ."  # Run with specific args on changes
```

## Architecture

```
internal/
  cli/        # Cobra commands (init, pack, unpack, list, status)
  tui/        # Bubbletea TUI components
  crypto/     # AES-256-GCM encryption with PBKDF2 key derivation
  archive/    # Archive creation/extraction with compression
  scanner/    # Environment file detection
  config/     # Configuration management
  constants/  # Defaults and constants

pkg/
  types/      # Shared types and interfaces
  utils/      # Utility functions
  password/   # Secure password handling

cmd/goingenv/ # Entry point
```

## Key Design Constraints

- **Initialization required**: Projects must run `goingenv init` before using pack/unpack commands. This creates a `.goingenv` directory.
- **Two-mode operation**: No args launches TUI; subcommands use CLI mode.
- **Encryption**: AES-256-GCM with PBKDF2-SHA256 (100k iterations), 32-byte salt, 12-byte nonce per encryption.
- **Archive storage**: Encrypted archives go to `.goingenv/` directory within the project.

## Testing

- Unit tests are alongside source code (`*_test.go`)
- Integration tests in `test/integration/`
- For archive tests, use `testutils.CreateTempGoingEnvDir()` to set up the required `.goingenv` directory
- Run `make test-complete` before commits

## Linting

golangci-lint is configured with:
- Max cyclomatic complexity: 15
- Security scanning via gosec (with some exclusions for CLI patterns)
- gofmt, goimports, errcheck, staticcheck enabled

## Release Workflow

Releases require `[release]` flag in commit message when pushing to main:
- `[release]` - Patch version bump (default)
- `[release] [minor]` - Minor version bump
- `[release] [major]` - Major version bump
- No `[release]` flag - CI validation only, no release

Manual releases: `make quick-alpha`, `make quick-beta`, `make quick-stable`, or use GitHub Actions workflow_dispatch
