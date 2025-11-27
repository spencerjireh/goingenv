# Agent Guidelines for goingenv

## Build & Test Commands
- **Build**: `make build` or `go build -o goingenv ./cmd/goingenv`
- **Test all**: `make test` or `go test ./...`
- **Test single**: `go test ./internal/crypto -run TestEncrypt` or `go test -v ./pkg/utils/...`
- **Lint**: `make lint` or `golangci-lint run ./...`
- **Full CI locally**: `make ci-full` (runs all checks: test, lint, build, security)

## Code Style & Conventions

### Imports
- Standard library first, then third-party, then local packages with blank line separation
- Use explicit package names: `"goingenv/pkg/types"` not `"./types"`

### Structure & Organization
- Private code in `internal/` (archive, cli, config, crypto, scanner, tui)
- Public APIs in `pkg/` (types, utils, password)
- Services implement interfaces defined in `pkg/types/types.go`
- Each service follows pattern: `NewService()` constructor, interface implementation

### Naming & Types
- Use PascalCase for exported types/functions, camelCase for unexported
- Struct fields use PascalCase for JSON serialization
- Constants use PascalCase (e.g., `SaltSize`, `KeySize`)
- Interfaces: `Scanner`, `Archiver`, `Cryptor`, `ConfigManager`

### Error Handling
- Return custom error types: `ScanError`, `ArchiveError`, `CryptoError`, `ValidationError`
- Wrap errors with context: `fmt.Errorf("failed to create cipher: %w", err)`
- Validate inputs early and return descriptive errors
- Check for empty/nil values before operations

### Testing
- Table-driven tests preferred (see `encryption_test.go`, `scanner_test.go`)
- Use `test/testutils/helpers.go` for shared test utilities
- Integration tests require `.goingenv` directory: call `testutils.CreateTempGoingEnvDir(t, tmpDir)`
- Use `t.TempDir()` for temporary test directories

## Special Requirements
- **Initialization**: Operations require `.goingenv/` directory (run `goingenv init` first)
- **Go version**: 1.21+ required (see go.mod)
- **Crypto**: AES-256-GCM with PBKDF2 (100,000 iterations, 32-byte salt)
- **Linter config**: `.golangci.yml` enables gocyclo, gocritic, gosec, misspell
