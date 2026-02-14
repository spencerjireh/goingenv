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
make test-complete      # Full test suite (unit + integration + e2e + functional)
make ci-test            # CI tests with race detection
go test -v ./pkg/... ./internal/...  # Run specific package tests
go test -v -run TestName ./internal/crypto/...  # Run a single test by name

# Linting
make fmt && make lint   # Format and lint code
make ci-full            # Run all CI checks locally

# Hot-reload development (requires air)
make watch              # Auto-rebuild on changes
make watch-run ARGS="status ."  # Run with specific args on changes
```

## Architecture

### Dependency Wiring and Two-Mode Operation

Entry point (`cmd/goingenv/main.go`) calls `cli.NewRootCommand()` which creates a Cobra root command. When invoked with no args, it launches the TUI via Bubbletea; with subcommands (`init`, `pack`, `unpack`, `list`, `status`), it uses CLI mode.

Both modes share the same service layer through the `types.App` struct, which acts as a dependency injection container:

```
ConfigManager.Load() → Config
  → Scanner = scanner.NewService(Config)
  → Crypto = crypto.NewService()
  → Archiver = archive.NewService(Crypto)
  → App{Scanner, Archiver, Crypto, ConfigManager}
```

### Key Interfaces (`pkg/types/types.go`)

All major services are defined as interfaces enabling mock-based testing:
- `Scanner` -- file detection via regex patterns with depth-limited `filepath.Walk`
- `Archiver` -- tar-based pack/unpack, delegates encryption to Cryptor
- `Cryptor` -- AES-256-GCM encrypt/decrypt (salt + nonce + ciphertext binary format)
- `ConfigManager` -- loads/saves `~/.goingenv.json`, checks project initialization

Mock implementations live in `pkg/types/mocks.go` (func-field based, not generated).

### TUI Structure (`internal/tui/`)

The TUI is a screen-based state machine. The model tracks `currentScreen` (ScreenMenu, ScreenPackPassword, ScreenPacking, etc.) and routes key presses accordingly in `update.go`. Async operations (scan, pack, unpack, list) run in goroutines via `commands.go` and return typed messages (e.g., `PackCompleteMsg`, `ErrorMsg`) back to the Bubbletea update loop.

### Password Handling

`pkg/password/` handles password acquisition with priority: env variable > interactive prompt. Passwords are cleared from memory via `ClearPassword()` (zeros bytes). CLI commands obtain passwords through `getPass(envVar)` in `cli/helpers.go` which returns a cleanup function used with defer.

### Configuration

Two config locations:
- **Project-level**: `.goingenv/` directory (created by `goingenv init`), contains `.gitignore` and encrypted archives
- **User-level**: `~/.goingenv.json` stores scan patterns, exclusions, max depth, max file size

`config.IsInitialized()` checks for `.goingenv/.gitignore` existence to determine if a project is set up.

## Testing

- Unit tests alongside source (`*_test.go`)
- Integration tests in `test/integration/` -- multi-phase (scan, pack, unpack, verify) with real services
- E2E tests in `test/e2e/` -- binary execution tests
- Shared helpers in `test/testutils/`:
  - `CreateTempGoingEnvDir()` -- required setup for archive tests (creates `.goingenv/` dir)
  - `CreateTempEnvFiles()` -- generates temp dir with sample .env files and excludable dirs
  - `BuildBinary()` -- compiles binary once via `sync.Once`, cached for test suite
  - `RunCLI()` / `RunCLIWithPassword()` -- execute binary and capture stdout/stderr/exit code
- Run `make test-complete` before commits

## Linting

golangci-lint is configured with:
- Max cyclomatic complexity: 15
- Security scanning via gosec (exclusions: G115, G117, G204, G304, G407, G703)
- gofmt, goimports, errcheck, staticcheck enabled

## Release Workflow

Releases require `[release]` flag in commit message when pushing to main:
- `[release]` -- Patch version bump (default)
- `[release] [minor]` -- Minor version bump
- `[release] [major]` -- Major version bump
- No `[release]` flag -- CI validation only, no release

Manual releases: `make quick-alpha`, `make quick-beta`, `make quick-stable`, or use GitHub Actions workflow_dispatch
