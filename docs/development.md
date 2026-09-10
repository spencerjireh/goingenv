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

- **[mise](https://mise.jdx.dev/)**: installs the pinned toolchain. Everything
  else -- Go, golangci-lint, gosec, govulncheck, goreleaser, syft, air -- comes
  from `mise.toml`, so your versions match CI exactly.
- **Git** and **Make**

Building from source without mise needs Go 1.24+ (the minimum `go.mod`
declares).

### Quick Setup

```bash
git clone https://github.com/spencerjireh/goingenv.git
cd goingenv
mise install       # or: make bootstrap
make build
make ci-test
./goingenv --help
```

## Development Setup

### IDE Configuration

**VS Code:** Go (Google), Go Test Explorer, GitLens

**GoLand/IntelliJ:** Go plugin (built-in), Makefile Language plugin

### Environment Setup

```bash
make bootstrap
```

That installs everything pinned in `mise.toml` and warms the module cache.
Installing these tools with `go install ...@latest` instead will drift from the
versions CI uses, which is what `mise.toml` exists to prevent.

### Development Commands

```bash
make dev              # Build with race detection
make tui-watch        # Hot-reload the TUI with Air
make run ARGS="status ."        # Run a subcommand
make fmt              # Format code
make lint             # Lint code
make ci-full          # Run all CI checks locally
```

### TUI Development Sandbox

A persistent sandbox at `/tmp/goingenv-sandbox` provides sample `.env` files for iterating on the TUI without touching real project data. The sandbox is created once and reused across runs.

```bash
make tui              # Build and launch TUI in the sandbox
make tui-watch        # Same as tui but with hot-reload via air
make tui-clean        # Remove the sandbox directory
```

### Hot-Reload with Air

Air rebuilds on file changes. Config is in `.air.toml`, which already runs the
rebuilt binary inside the sandbox -- so `make tui-watch` is the only target
needed. (There used to be three more, all identical to it.)

```bash
make tui-watch        # Rebuild and relaunch the TUI on every save
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
| `internal/config/` | Resolves and loads project/user config, saves to `~/.goingenv.json` |
| `pkg/types/` | Interfaces (`Scanner`, `Archiver`, `Cryptor`, `ConfigManager`) + func-field mocks |

## Building

```bash
make build            # Current platform
make dev              # With race detection
make release-local    # All platforms into dist/

# Manual cross-compile
GOOS=linux GOARCH=amd64 go build -o goingenv-linux-amd64 ./cmd/goingenv

# Custom version
go build -ldflags="-X main.Version=v1.2.3 -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o goingenv ./cmd/goingenv
```

## Testing

### Running Tests

```bash
make ci-test          # Everything CI runs
make test-unit        # Unit tests only
make test-integration # Integration tests (in process, race enabled)
make test-cli         # CLI tests (spawns the built binary)
make test-e2e         # End-to-end tests
make test-install     # install.sh checksum verification tests
make test-coverage    # With coverage report
make test-bench       # Benchmarks
```

Tests never touch your real `~/.goingenv.json`: every spawned binary runs with
`HOME` pointed at a temp directory. If you add a test that drives the config
manager in process, use `config.NewManagerWithPath()` rather than
`config.NewManager()`.

Config resolution is relative to the working directory, so a `.goingenv/config.json`
in a test's temp project outranks the temp `HOME`. Tests that need to exercise
user-level config must not leave a project config behind.

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

### Adding a TUI Tab

The TUI is a tabbed layout; there is no screen state machine. Each tab lives in
its own `internal/tui/tab_*.go` and implements the `Tab` interface.

```go
// tab_newfeature.go -- implement the Tab interface
func (t *NewFeatureTab) Update(msg tea.Msg) (Tab, tea.Cmd) { ... }
func (t *NewFeatureTab) View(width, height int) string     { ... }

// Register the TabID and construct it in model.go
```

For a multi-step (wizard) tab, give each step its own `update<Step>` method and
keep `Update` as a dispatch switch over `t.step`. `tab_pack.go` is the model to
follow -- one large switch here is how the linter's complexity limit gets hit.

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

Runs linting, the full test suite, security scanning, and a snapshot release
build. Tools come from `mise.toml`, so a missing tool is now an error rather
than a warning -- this target used to report success while silently skipping
checks whose tools were not installed.

### Common Fixes

```bash
# Formatting or tidy failure
make fmt && go mod tidy

# Test failure
make ci-test

# Vulnerability found
make vuln-check
go get -u [vulnerable-package] && go mod tidy
```

### Workflows

- **`ci.yml`** -- PRs and pushes to `main`/`develop`, plus `workflow_call` from
  the release workflow.
- **`release.yml`** -- `v*` tag pushes. Calls `ci.yml` first, then GoReleaser.
- **`pages.yml`** -- deploys `public/` to GitHub Pages.

### CI Workflow (`ci.yml`)

```
changes ──┬──> lint ───────────────┐
          ├──> test (matrix) ──────┤
          ├──> security ───────────┼──> ci-ok
          └──> test-install-script ┘
```

- **changes** -- path filter. Non-code changes skip the jobs below, but `ci-ok`
  still reports.
- **lint** -- gofmt, go vet, `go mod tidy -diff`, golangci-lint.
- **test** -- Ubuntu/macOS x Go minimum/stable. Unit (race), integration
  (race), CLI, and E2E tests. The minimum-version leg sets `GOTOOLCHAIN=local`,
  without which Go silently downloads a newer toolchain and the leg tests
  nothing.
- **security** -- govulncheck, gosec, SARIF upload to the Security tab.
- **test-install-script** -- install.sh checksum verification on Ubuntu and
  macOS. Runs in parallel; it does not depend on the other jobs.
- **ci-ok** -- always runs and aggregates the rest.

**Branch protection should require `ci-ok` and nothing else.** The other jobs
are skipped for docs-only changes, so requiring them directly would block such
PRs on checks that never report.

### Release Workflow (`release.yml`)

```
ci (calls ci.yml) ──> release (goreleaser + provenance)
```

A tag cannot publish without CI passing first. GoReleaser builds all four
platforms in one job, writes `checksums.txt`, generates SBOMs, and publishes a
version-stamped `install.sh` alongside the archives. A provenance attestation
is recorded via OIDC.

The final step re-checks the artifact contract `install.sh` depends on --
archive naming, single-binary contents, and a matching `checksums.txt` entry --
before anything is published.

### Running CI Locally

```bash
make ci-full          # everything
make ci-lint          # lint only
make ci-test          # tests only
make ci-security      # security only
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

All three are injected with `-ldflags -X` and reported by `goingenv --version`.

| Variable | Release source | Local build source |
|---|---|---|
| `main.Version` | Git tag (e.g. `v1.2.3`) | `git describe --tags --dirty --always` |
| `main.BuildTime` | Commit date (reproducible builds) | Build timestamp |
| `main.GitCommit` | Full commit SHA | Short SHA |

### Local Release Test

```bash
make release-local    # goreleaser snapshot; builds all platforms, publishes nothing
ls -la dist/
```

Snapshot archive names carry a `-SNAPSHOT-` suffix, so they differ from a real
release. To check the artifact contract by hand:

```bash
tar -tzf dist/goingenv-*-darwin-arm64.tar.gz   # must print only: goingenv
cat dist/checksums.txt
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
