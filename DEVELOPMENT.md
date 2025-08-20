# GoingEnv Development Guide

This guide provides everything you need to know for contributing to GoingEnv development.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Building](#building)
- [Testing](#testing)
- [Contributing](#contributing)
- [Release Process](#release-process)

## Getting Started

### Prerequisites

- **Go 1.21+**: [Download Go](https://golang.org/dl/)
- **Git**: Version control
- **Make**: Build automation (optional but recommended)

### Quick Setup

```bash
# Clone the repository
git clone https://github.com/spencerjirehcebrian/goingenv.git
cd goingenv

# Install dependencies
go mod tidy

# Build and test
make build
make test

# Run the application
./goingenv --help
```

## Development Setup

### IDE Configuration

**VS Code Extensions:**
- Go (Google)
- Go Test Explorer
- GitLens
- Better Comments

**GoLand/IntelliJ:**
- Go plugin (built-in)
- Makefile Language plugin

### Environment Setup

```bash
# Set up development environment
export GOPATH=$HOME/go
export GO111MODULE=on

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/vektra/mockery/v2@latest
```

### Development Commands

```bash
# Development build with race detection
make dev

# Run with live reload (install air first)
go install github.com/cosmtrek/air@latest
air

# Format code
make fmt

# Lint code
make lint

# Run all checks
make check

# CI-related commands
make ci-full      # Run all CI checks locally
make ci-test      # Run tests like CI
make ci-lint      # Run linting like CI
make ci-build     # Test build like CI
```

## Project Structure

```
goingenv/
├── cmd/
│   └── goingenv/           # Main application entry point
│       └── main.go
├── internal/               # Private application code
│   ├── archive/           # Archive operations
│   ├── cli/               # CLI commands and root
│   ├── config/            # Configuration management
│   ├── crypto/            # Encryption/decryption
│   ├── scanner/           # File scanning logic
│   └── tui/               # Terminal UI components
├── pkg/                   # Public API packages
│   ├── types/             # Shared types and interfaces
│   └── utils/             # Utility functions
├── test/                  # Test files
│   ├── integration/       # Integration tests
│   └── testutils/         # Test utilities
├── docs/                  # Additional documentation
├── scripts/               # Build and utility scripts
├── Makefile              # Build automation
├── go.mod                # Go module definition
├── go.sum                # Dependency checksums
├── install.sh            # Installation script
├── README.md             # Main documentation
├── INSTALL.md            # Installation guide
├── USAGE.md              # User guide
└── DEVELOPMENT.md        # This file
```

### Package Responsibilities

**`cmd/goingenv/`**: Application entry point and initialization

**`internal/cli/`**: Cobra-based CLI commands
- `root.go`: Root command and TUI launcher
- `pack.go`: Pack command implementation
- `unpack.go`: Unpack command implementation
- `list.go`: List command implementation
- `status.go`: Status command implementation

**`internal/tui/`**: Bubbletea-based terminal UI
- `model.go`: Main TUI model and state
- `view.go`: UI rendering logic
- `update.go`: Event handling and updates
- `commands.go`: Async command execution
- `styles.go`: UI styling and themes
- `debug.go`: Debug logging system

**`internal/archive/`**: Archive operations
- `archive.go`: Compression and extraction

**`internal/crypto/`**: Cryptography
- `encryption.go`: AES-256 encryption/decryption
- `encryption_test.go`: Crypto tests

**`internal/scanner/`**: File discovery
- `scanner.go`: Environment file detection
- `scanner_test.go`: Scanner tests

**`internal/config/`**: Configuration management
- `config.go`: Settings and defaults

**`pkg/types/`**: Shared types and interfaces
- `types.go`: Common data structures
- `mocks.go`: Mock implementations for testing

**`pkg/utils/`**: Utility functions
- `utils.go`: Helper functions
- `utils_test.go`: Utility tests

## Building

### Local Development

```bash
# Build for current platform
make build

# Build with race detection
make dev

# Build optimized release
make release-build

# Clean build artifacts
make clean
```

### Cross-Platform Building

```bash
# Build for all supported platforms
make release-all

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -o goingenv-linux-amd64 ./cmd/goingenv

# Available targets
make build-linux    # Linux AMD64
make build-darwin   # macOS AMD64
make build-windows  # Windows AMD64
```

### Build Configuration

Build-time variables in `Makefile`:
- `VERSION`: Application version
- `BUILD_TIME`: Build timestamp
- `GIT_COMMIT`: Git commit hash

Custom build:
```bash
go build -ldflags="-X main.Version=1.0.0 -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o goingenv ./cmd/goingenv
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run automated functional workflow tests
make test-functional

# Run complete test suite (recommended for development)
make test-complete

# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# Run tests with coverage
make test-coverage

# Run tests with verbose output
make test-verbose

# Run benchmarks
make test-bench
```

### Test Structure

**Unit Tests:**
- Located alongside source code (`*_test.go`)
- Test individual functions and methods
- Use table-driven tests where appropriate

**Integration Tests:**
- Located in `test/integration/`
- Test complete workflows
- Use temporary directories and files

**Test Utilities:**
- Located in `test/testutils/`
- Shared helpers for test setup
- Mock data generation

### Writing Tests

**Example unit test:**
```go
func TestScanFiles(t *testing.T) {
    tests := []struct {
        name     string
        opts     ScanOptions
        want     int
        wantErr  bool
    }{
        {
            name: "basic scan",
            opts: ScanOptions{
                RootPath: "testdata",
                MaxDepth: 2,
            },
            want:    3,
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            scanner := NewService(&Config{})
            files, err := scanner.ScanFiles(tt.opts)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("ScanFiles() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if len(files) != tt.want {
                t.Errorf("ScanFiles() = %v files, want %v", len(files), tt.want)
            }
        })
    }
}
```

**Example integration test:**
```go
func TestFullWorkflow(t *testing.T) {
    // Setup temporary directory
    tempDir := t.TempDir()
    
    // Create test files
    testutils.CreateTestFiles(t, tempDir)
    
    // Test pack operation
    // Test unpack operation
    // Verify results
}
```

### Mocking

Generate mocks for interfaces:
```bash
make generate-mocks
```

Using mocks in tests:
```go
func TestWithMock(t *testing.T) {
    mockScanner := &types.MockScanner{}
    mockScanner.On("ScanFiles", mock.Anything).Return([]types.EnvFile{}, nil)
    
    // Use mock in test
}
```

## Contributing

### Development Workflow

1. **Fork and Clone**
   ```bash
   git clone https://github.com/yourusername/goingenv.git
   cd goingenv
   git remote add upstream https://github.com/spencerjirehcebrian/goingenv.git
   ```

2. **Create Feature Branch**
   ```bash
   git checkout -b feature/amazing-feature
   ```

3. **Make Changes**
   - Write code following style guidelines
   - Add tests for new functionality
   - Update documentation

4. **Test Changes**
   ```bash
   make test-complete   # Run complete test suite (recommended)
   make check-full      # Run all checks including linting
   make test-functional # Quick functional validation
   ```

5. **Commit Changes**
   ```bash
   git add .
   git commit -m "feat: add amazing feature"
   ```

6. **Push and Create PR**
   ```bash
   git push origin feature/amazing-feature
   ```

### Code Style Guidelines

**Go Style:**
- Follow standard Go conventions
- Use `gofmt` for formatting
- Use meaningful variable names
- Add comments for exported functions
- Keep functions small and focused

**Commit Messages:**
Use conventional commits format:
- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation changes
- `test:` adding tests
- `refactor:` code refactoring
- `chore:` maintenance tasks

**Code Review:**
- All changes require review
- Address reviewer feedback
- Ensure CI passes
- Squash commits before merge

### Adding New Features

**1. CLI Commands:**
```go
// internal/cli/newcommand.go
func newNewCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "new",
        Short: "New command description",
        RunE:  runNewCommand,
    }
    
    // Add flags
    cmd.Flags().StringP("option", "o", "", "Option description")
    
    return cmd
}
```

**2. TUI Screens:**
```go
// internal/tui/model.go
const (
    ScreenNewFeature Screen = "new_feature"
)

// internal/tui/view.go
func (m *Model) renderNewFeature() string {
    // Render new screen
}

// internal/tui/update.go
func (m *Model) handleNewFeatureKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    // Handle key events
}
```

**3. New Packages:**
- Create in `internal/` for private code
- Create in `pkg/` for public APIs
- Add interfaces in `pkg/types/`
- Include comprehensive tests

## Release Process

### Version Management

GoingEnv uses semantic versioning (SemVer):
- `MAJOR.MINOR.PATCH` (e.g., `1.2.3`)
- Pre-release: `1.2.3-alpha.1`, `1.2.3-beta.1`, `1.2.3-rc.1`
- Tags must follow format: `v1.2.3` or `v1.2.3-alpha.1`

### Automated Release Workflow

**Simple Two-Step Process:**
```bash
# 1. Create and validate release
make tag-release
# This will:
# - Run pre-release checks (tests, linting, build verification)
# - Prompt for version number with validation
# - Create local git tag
# - Show push command

# 2. Trigger automated release
make push-release-tag
# This will:
# - Push the tag to GitHub
# - Trigger GitHub Actions release workflow
# - Provide monitoring link
```

**What GitHub Actions Does Automatically:**
1. **Validation**: Verify tag format and check for duplicates
2. **Quality Gates**: Run full CI suite (tests, linting, security)
3. **Build**: Create binaries for all platforms (Linux/macOS, AMD64/ARM64)
4. **Package**: Generate tar.gz archives with proper naming
5. **Checksums**: Create SHA256 checksums for security
6. **Release**: Create GitHub release with auto-generated notes
7. **Validation**: Test install script with new release
8. **Notification**: Success/failure feedback

### Manual Release Process (Fallback)

If you need to create a release manually:

**1. Prepare Release**
```bash
# Run pre-release checks
make pre-release-check

# Update version in Makefile if needed
vim Makefile  # Set VERSION=1.2.3

# Create tag manually
git tag -a v1.2.3 -m "Release v1.2.3"
```

**2. Build Release Artifacts**
```bash
# Build all platform binaries locally
make release-local

# Verify artifacts
ls -la dist/
```

**3. Create GitHub Release Manually**
```bash
# Push tag
git push origin v1.2.3

# Create release on GitHub web interface
# Upload files from dist/ directory
# Include release notes
```

### Pre-release Testing

**Local Simulation:**
```bash
# Test release build locally
make release-local

# Test install script with local files
# (Advanced: set up local HTTP server to test downloads)
```

**Validation Commands:**
```bash
# Check current release status
make check-release-status

# Verify CI passes before tagging
make ci-full
```

### Release Checklist

- [ ] Version updated in `Makefile`
- [ ] CHANGELOG.md updated
- [ ] Tests passing
- [ ] Documentation updated
- [ ] Release artifacts built
- [ ] GitHub release created
- [ ] Installation script tested
- [ ] Release announcement posted

### Hotfix Process

For critical bugs:

1. Create hotfix branch from main
2. Fix the issue
3. Test thoroughly
4. Create patch release (e.g., 1.2.4)
5. Fast-track review and merge

## Development Tools

### Useful Commands

```bash
# Code generation
go generate ./...

# Dependency management
go mod tidy
go mod verify
go mod why <package>

# Profiling
go build -o goingenv-prof ./cmd/goingenv
./goingenv-prof pack # with profiling flags

# Security scanning
make security-scan
```

### Debugging

**Debug Builds:**
```bash
# Build with debug symbols
go build -gcflags="all=-N -l" -o goingenv-debug ./cmd/goingenv

# Use with debugger (delve)
dlv exec ./goingenv-debug
```

**TUI Debug Mode:**
```bash
# Enable debug logging
./goingenv --verbose

# Check debug logs
tail -f ~/.goingenv/debug/tui_debug_*.log
```

### Performance

**Benchmarking:**
```bash
# Run benchmarks
make test-bench

# Profile CPU usage
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof
```

**Memory Profiling:**
```bash
# Profile memory usage
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

## Continuous Integration

### CI Pipeline

The project uses GitHub Actions for continuous integration with the following checks:

**Test Job:**
- Runs on Ubuntu and macOS
- Tests Go 1.21 and 1.22
- Unit and integration tests
- Race condition detection
- Coverage reporting

**Lint Job:**
- golangci-lint with comprehensive rules
- Go formatting verification
- `go vet` static analysis
- `go mod tidy` verification

**Security Job:**
- gosec security scanner
- Nancy vulnerability scanning
- SARIF report upload

**Build Job:**
- Build verification
- Cross-compilation testing
- Makefile target testing

**Install Script Job:**
- Syntax validation
- Help function testing
- Dry run verification

### Running CI Locally

```bash
# Run full CI suite locally
make ci-full

# Run individual CI jobs
make ci-test          # Test job
make ci-lint          # Lint job  
make ci-build         # Build job
make ci-security      # Security job
make ci-cross-compile # Cross-compilation

# Install required tools for local CI
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
go install github.com/sonatypecommunity/nancy@latest
```

### CI Configuration

- **Workflow file**: `.github/workflows/ci.yml`
- **Lint config**: `.golangci.yml`
- **Skip triggers**: Changes to `*.md` files only

### Branch Protection

When enabled, the following rules apply:
- All CI checks must pass
- Branches must be up-to-date
- At least one review required
- Admin enforcement enabled

## Getting Help

- **GitHub Discussions**: https://github.com/spencerjirehcebrian/goingenv/discussions
- **Issues**: https://github.com/spencerjirehcebrian/goingenv/issues
- **Go Documentation**: https://golang.org/doc/
- **Cobra Documentation**: https://cobra.dev/
- **Bubbletea Documentation**: https://github.com/charmbracelet/bubbletea