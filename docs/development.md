# Development Guide

Everything you need to contribute to goingenv.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Building](#building)
- [Testing](#testing)
- [Contributing](#contributing)
- [CI/CD Pipeline](#cicd-pipeline)
- [Release Process](#release-process)
- [Development Tools](#development-tools)

## Getting Started

### Prerequisites

- **Go 1.23+**: [Download Go](https://golang.org/dl/)
- **Git**: Version control
- **Make**: Build automation (optional but recommended)

### Quick Setup

```bash
git clone https://github.com/spencerjireh/goingenv.git
cd goingenv
go mod tidy
make build
make test
./goingenv --help
```

## Development Setup

### IDE Configuration

**VS Code:** Go (Google), Go Test Explorer, GitLens

**GoLand/IntelliJ:** Go plugin (built-in), Makefile Language plugin

### Environment Setup

```bash
export GOPATH=$HOME/go
export GO111MODULE=on

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/air-verse/air@latest
```

### Development Commands

```bash
make dev              # Build with race detection
make watch            # Hot-reload with Air
make watch-run ARGS="status ."  # Hot-reload with specific args
make fmt              # Format code
make lint             # Lint code
make ci-full          # Run all CI checks locally
```

### Hot-Reload with Air

Air automatically rebuilds on file changes. Config is in `.air.toml`.

```bash
make watch            # Rebuilds on change (doesn't run)
make dev-watch        # Rebuilds and runs the TUI
make watch-run ARGS="status ."  # Rebuilds and runs with args
```

## Project Structure

```
goingenv/
├── cmd/goingenv/          # Entry point
├── internal/
│   ├── archive/           # Tar-based pack/unpack
│   ├── cli/               # Cobra CLI commands
│   ├── config/            # Configuration management
│   ├── crypto/            # AES-256-GCM encryption
│   ├── scanner/           # File discovery
│   └── tui/               # Bubbletea terminal UI
├── pkg/
│   ├── types/             # Interfaces and mocks
│   └── utils/             # Helpers
├── test/
│   ├── integration/       # Integration tests
│   └── testutils/         # Shared test helpers
├── docs/                  # Documentation
├── public/                # Website (GitHub Pages)
└── assets/                # Brand assets
```

### Key Packages

| Package | Responsibility |
|---|---|
| `internal/cli/` | Cobra commands: `init`, `pack`, `unpack`, `list`, `status` |
| `internal/tui/` | Bubbletea screen-based state machine |
| `internal/archive/` | Tar compression, delegates encryption to crypto |
| `internal/crypto/` | AES-256-GCM with PBKDF2 key derivation |
| `internal/scanner/` | Regex pattern matching with depth-limited `filepath.Walk` |
| `internal/config/` | Loads/saves `~/.goingenv.json` |
| `pkg/types/` | Interfaces (`Scanner`, `Archiver`, `Cryptor`, `ConfigManager`) + func-field mocks |

## Building

```bash
make build            # Current platform
make dev              # With race detection
make release-local    # All platforms into dist/

# Manual cross-compile
GOOS=linux GOARCH=amd64 go build -o goingenv-linux-amd64 ./cmd/goingenv

# Custom version
go build -ldflags="-X main.Version=1.0.0 -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o goingenv ./cmd/goingenv
```

## Testing

### Running Tests

```bash
make test-complete    # Full suite (unit + integration + e2e + functional)
make test             # Unit + integration
make test-functional  # Automated workflow tests
make test-unit        # Unit tests only
make test-integration # Integration tests only
make test-coverage    # With coverage report
make test-bench       # Benchmarks
```

### Test Structure

- **Unit tests**: Alongside source (`*_test.go`), table-driven
- **Integration tests**: `test/integration/` -- multi-phase workflows with real services
- **Test utilities**: `test/testutils/` -- `CreateTempGoingEnvDir()`, `CreateTempEnvFiles()`, `BuildBinary()`, `RunCLI()`

### Initialization in Tests

Archive tests require `.goingenv/` setup:

```go
func TestArchiveOperations(t *testing.T) {
    tmpDir := testutils.CreateTempEnvFiles(t)
    defer os.RemoveAll(tmpDir)
    testutils.CreateTempGoingEnvDir(t, tmpDir)
    // archive operations now work
}
```

## Contributing

### Workflow

1. Fork and clone
2. Create feature branch: `git checkout -b feature/my-feature`
3. Make changes, add tests
4. Run `make test-complete && make ci-full`
5. Commit with conventional format: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`
6. Push and create PR

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Keep functions small and focused
- Add comments for exported functions

### Adding CLI Commands

```go
// internal/cli/newcommand.go
func newNewCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "new",
        Short: "Description",
        RunE:  runNewCommand,
    }
    cmd.Flags().StringP("option", "o", "", "Option description")
    return cmd
}
```

### Adding TUI Screens

```go
// Add screen constant in model.go
const ScreenNewFeature Screen = "new_feature"

// Add render in view.go
func (m *Model) renderNewFeature() string { ... }

// Add key handling in update.go
func (m *Model) handleNewFeatureKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) { ... }
```

## CI/CD Pipeline

### Quick Reference

| Task | Command |
|---|---|
| Run all CI checks locally | `make ci-full` |
| Build release binaries locally | `make release-local` |
| Create a release | `git tag -a v1.2.3 -m "Release v1.2.3" && git push origin v1.2.3` |

### Before Pushing Code

```bash
make ci-full
```

This runs: dependency updates, unit tests with race detection, linting, security scanning, and cross-compilation tests.

### Common Fixes

```bash
# Lint failure
make fmt && go mod tidy

# Test failure
make test-complete

# Security scan failure
make vuln-check
go get -u [vulnerable-package]
go mod tidy
```

### Workflows

The pipeline is split into three workflow files:

- **`ci.yml`** -- Runs on PRs and pushes to `main`/`develop`. Validates code quality.
- **`release.yml`** -- Runs on `v*` tag pushes. Builds binaries and publishes a GitHub Release.
- **`pages.yml`** -- Deploys `public/` to GitHub Pages on changes.

### CI Workflow (`ci.yml`)

Jobs:

- **lint** -- gofmt, go vet, go mod tidy, golangci-lint
- **test** -- Matrix: Ubuntu/macOS x Go 1.23/stable. Unit, integration, and functional tests with race detection.
- **security** -- govulncheck, gosec, SARIF upload to GitHub Security tab
- **test-install-script** -- Depends on lint, test, security. Validates install.sh on Ubuntu and macOS.

```
lint ──────────┐
               │
test ──────────┼──> test-install-script
               │
security ──────┘
```

### Release Workflow (`release.yml`)

Triggered by pushing a tag matching `v*`.

- **build-release** -- Matrix: Linux/macOS x AMD64/ARM64. Builds optimized binaries with `-trimpath` and embedded version info. Creates tar.gz archives with SHA256 checksums.
- **create-release** -- Downloads artifacts, generates release notes, creates versioned install.sh, publishes GitHub Release.

```
build-release (4 platforms) ──> create-release
```

### Running CI Locally

```bash
make ci-full          # All checks
make ci-lint          # Lint only
make ci-test          # Tests only
make ci-security      # Security only
make ci-cross-compile # Cross-compilation
```

## Release Process

Releases are triggered by pushing a Git tag:

```bash
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3
```

- Stable versions (`v1.2.3`) are marked as latest release
- Prereleases (`v1.2.3-alpha.1`, `v1.2.3-rc.1`) are marked as prerelease

### Build Variables

| Variable | Source |
|---|---|
| `VERSION` | Git tag (e.g., `v1.2.3`) |
| `BUILD_TIME` | UTC build timestamp |
| `GIT_COMMIT` | Full commit SHA |

### Local Release Test

```bash
make release-local    # Builds all platforms into dist/
ls -la dist/
```

## Development Tools

### Debugging

```bash
# Debug build
go build -gcflags="all=-N -l" -o goingenv-debug ./cmd/goingenv
dlv exec ./goingenv-debug

# TUI debug logging
./goingenv --verbose
```

### Profiling

```bash
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

## Links

- [Issues](https://github.com/spencerjireh/goingenv/issues)
- [Security Guide](../SECURITY.md)
- [Cobra Docs](https://cobra.dev/)
- [Bubbletea Docs](https://github.com/charmbracelet/bubbletea)
