# GoingEnv Testing Guide

This document provides comprehensive information about testing in the GoingEnv project, including test structure, running tests, and writing new tests.

## Table of Contents

- [Test Architecture](#test-architecture)
- [Running Tests](#running-tests)
- [Test Categories](#test-categories)
- [Writing Tests](#writing-tests)
- [Test Utilities](#test-utilities)
- [Continuous Integration](#continuous-integration)
- [Coverage Reports](#coverage-reports)
- [Performance Testing](#performance-testing)
- [Troubleshooting](#troubleshooting)

## Test Architecture

GoingEnv uses a comprehensive testing strategy with multiple layers:

```
test/
├── integration/           # End-to-end integration tests
│   └── full_workflow_test.go
├── testutils/            # Shared test utilities and helpers
│   └── helpers.go
pkg/
├── types/
│   ├── mocks.go         # Mock implementations for interfaces
│   └── types.go
└── utils/
    ├── utils_test.go    # Utility function tests
    └── utils.go
internal/
├── crypto/
│   ├── encryption_test.go  # Crypto service tests
│   └── encryption.go
├── scanner/
│   ├── scanner_test.go     # File scanner tests
│   └── scanner.go
└── ...
```

### Test Types

1. **Unit Tests**: Test individual functions and methods in isolation
2. **Integration Tests**: Test complete workflows and component interactions
3. **Mock Tests**: Use mock implementations to test interfaces and dependencies
4. **Performance Tests**: Benchmark critical operations
5. **Error Handling Tests**: Validate error scenarios and edge cases

## Running Tests

### Quick Start

```bash
# Run all tests
make test

# Run automated functional tests
make test-functional

# Run complete test suite (unit + integration + functional)
make test-complete

# Run with coverage report
make test-coverage

# Run only unit tests
make test-unit

# Run only integration tests
make test-integration
```

### Detailed Test Commands

#### Basic Test Execution

```bash
# Run all tests with verbose output
make test-verbose

# Run tests with race condition detection
go test -race ./...

# Run tests in short mode (skips long-running tests)
go test -short ./...
```

#### Coverage Analysis

```bash
# Generate HTML coverage report
make test-coverage

# Coverage for CI (text output only)
make test-coverage-ci

# View coverage in browser
open coverage.html
```

#### Development Workflow

```bash
# Watch mode - automatically run tests when files change
make test-watch

# Run benchmarks
make test-bench

# Clean test artifacts
make test-clean
```

#### Specific Test Patterns

```bash
# Run specific test function
go test -run TestSpecificFunction ./pkg/utils

# Run tests matching pattern
go test -run "TestCrypto.*" ./internal/crypto

# Run integration tests only
go test -run "TestFull.*" ./test/integration
```

## Test Categories

### Unit Tests

**Location**: Alongside source files (`*_test.go`)

**Purpose**: Test individual functions, methods, and components in isolation

**Examples**:
```bash
# Utility functions
go test ./pkg/utils

# Crypto operations
go test ./internal/crypto

# File scanning
go test ./internal/scanner
```

### Integration Tests

**Location**: `test/integration/`

**Purpose**: Test complete workflows and component interactions

**Key Integration Tests**:
- Full workflow (scan → pack → list → unpack)
- Error handling scenarios
- Configuration management
- Large file handling
- Concurrent operations

### Functional Tests

**Command**: `make test-functional`

**Purpose**: Automated end-to-end testing that validates real-world usage scenarios

**What it tests**:
1. **Application Build**: Ensures the binary compiles successfully
2. **Test Environment Setup**: Creates realistic .env file scenarios
3. **All-Inclusive Pattern Detection**: Tests that `\.env.*` pattern detects all variants
4. **Exclusion Pattern Functionality**: Validates that exclusion patterns work correctly
5. **Pack/Unpack Workflow**: Tests the complete archive creation and extraction process
6. **Configuration Management**: Validates config backup, usage, and restoration
7. **Cleanup**: Ensures all test artifacts are properly removed

**Test Files Created**:
- `.env` - Basic environment file
- `.env.local` - Local overrides
- `.env.development` - Development settings
- `.env.custom` - Custom configuration
- `.env.backup` - Backup file (used for exclusion testing)
- `.env.new_format` - Novel format testing
- `regular.txt` - Non-env file (should be ignored)

**Validation Steps**:
- ✅ All 6 .env files detected with default all-inclusive pattern
- ✅ 5 files detected when `.env.backup` is excluded
- ✅ Pack operation creates encrypted archive successfully
- ✅ Unpack operation restores files correctly
- ✅ Configuration backup and restoration works safely

**Safe Testing**:
- Automatically backs up existing `~/.goingenv.json` config
- Uses temporary test directory (`test_env_files_functional/`)
- Restores original configuration after testing
- Cleans up all test artifacts automatically

### Mock Tests

**Location**: Uses mocks from `pkg/types/mocks.go`

**Purpose**: Test interfaces and dependencies without external dependencies

**Mock Implementations Available**:
- `MockScanner` - File scanning interface
- `MockArchiver` - Archive operations interface  
- `MockCryptor` - Encryption interface
- `MockConfigManager` - Configuration interface

## Writing Tests

### Test File Structure

```go
package mypackage

import (
    "testing"
    "goingenv/test/testutils"
    "goingenv/pkg/types"
)

func TestMyFunction(t *testing.T) {
    // Test implementation
}

func BenchmarkMyFunction(b *testing.B) {
    // Benchmark implementation
}
```

### Using Test Utilities

The `test/testutils` package provides helpful utilities:

```go
// Create test environment
tmpDir := testutils.CreateTempEnvFiles(t)
defer os.RemoveAll(tmpDir)

// Create test configuration
config := testutils.CreateTestConfig()

// Assert file operations
testutils.AssertFileExists(t, "/path/to/file")
testutils.AssertNoError(t, err)

// Compare files
if !testutils.CompareFiles(t, file1, file2) {
    t.Error("Files don't match")
}
```

### Mock Usage Examples

```go
func TestWithMocks(t *testing.T) {
    // Create mock scanner
    mockScanner := &types.MockScanner{
        ScanFilesFunc: func(opts types.ScanOptions) ([]types.EnvFile, error) {
            return testutils.CreateTestEnvFiles(3), nil
        },
    }
    
    // Test with mock
    files, err := mockScanner.ScanFiles(types.ScanOptions{})
    testutils.AssertNoError(t, err)
    // ... continue test
}
```

### Error Testing Patterns

```go
func TestErrorHandling(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "valid", false},
        {"invalid input", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := myFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("myFunction() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Test Utilities

### Helper Functions

The `testutils` package provides extensive helper functions:

#### File Operations
- `CreateTempEnvFiles(t)` - Creates temporary test environment
- `CreateTempFile(t, pattern, content)` - Creates temporary file
- `WriteTestFile(t, path, content)` - Writes test file
- `AssertFileExists(t, path)` - Validates file existence
- `CompareFiles(t, path1, path2)` - Compares file contents

#### Test Data Creation
- `CreateTestConfig()` - Standard test configuration
- `CreateTestEnvFiles(count)` - Creates test EnvFile structs
- `CreateValidArchiveOptions(...)` - Creates PackOptions for testing

#### Assertions
- `AssertNoError(t, err)` - Validates no error occurred
- `AssertStringContains(t, str, substr)` - String containment
- `AssertSliceContains(t, slice, item)` - Slice containment

#### Time and Cleanup
- `WaitForFile(t, path, timeout)` - Waits for file creation
- `CleanupTempFiles(t, paths...)` - Cleanup utility
- `MockTime()` - Fixed time for consistent testing

### Mock Implementations

Comprehensive mocks are available for all major interfaces:

```go
// Example: Custom mock behavior
mockCryptor := &types.MockCryptor{
    EncryptFunc: func(data []byte, password string) ([]byte, error) {
        // Custom encryption logic for testing
        return append([]byte("encrypted:"), data...), nil
    },
}
```

## Continuous Integration

### CI Test Commands

```bash
# Standard CI test run
make test-coverage-ci

# Full validation for CI
make check-full
```

### GitHub Actions Integration

The testing strategy is designed to work with CI/CD pipelines:

```yaml
# Example GitHub Actions step
- name: Run Tests
  run: |
    make test-coverage-ci
    make check-full
```

## Coverage Reports

### Generating Coverage

```bash
# HTML report (opens in browser)
make test-coverage

# Text summary
make test-coverage-ci
```

### Coverage Targets

- **Unit Tests**: Aim for >90% coverage
- **Integration Tests**: Focus on critical paths
- **Overall**: Target >85% overall coverage

### Viewing Coverage

```bash
# Generate and view HTML report
make test-coverage
open coverage.html

# Command line summary
go tool cover -func=coverage.out
```

## Performance Testing

### Benchmark Tests

```bash
# Run all benchmarks
make test-bench

# Run specific benchmarks
go test -bench=BenchmarkEncrypt ./internal/crypto

# Memory profiling
go test -bench=. -benchmem ./...
```

### Performance Monitoring

Key performance areas tested:
- Encryption/decryption operations
- File scanning performance
- Large file handling
- Memory usage patterns

### Benchmark Examples

```go
func BenchmarkScanFiles(b *testing.B) {
    tmpDir := createLargeTestDir(&testing.T{}, 100)
    defer os.RemoveAll(tmpDir)
    
    service := scanner.NewService(config)
    opts := types.ScanOptions{
        RootPath: tmpDir,
        MaxDepth: 5,
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := service.ScanFiles(opts)
        if err != nil {
            b.Fatalf("ScanFiles failed: %v", err)
        }
    }
}
```

## Troubleshooting

### Common Issues

#### Test Failures

```bash
# Run with verbose output for debugging
go test -v ./...

# Run specific failing test
go test -v -run TestSpecificFunction ./package
```

#### Coverage Issues

```bash
# Clean test cache
go clean -testcache

# Remove old coverage files
make test-clean
```

#### Mock Generation

```bash
# Install mockgen if missing
go install github.com/golang/mock/mockgen@latest

# Generate mocks
make generate-mocks
```

### Environment Issues

#### File Permissions
```bash
# Ensure test directories are writable
chmod -R 755 test/
```

#### Temporary Files
```bash
# Clean up test artifacts
make test-clean

# Manual cleanup
rm -rf /tmp/goingenv-test-*
```

### Debugging Tests

#### Adding Debug Output

```go
func TestDebug(t *testing.T) {
    t.Logf("Debug info: %v", someValue)
    // Use t.Logf for debug output that only shows on failure
}
```

#### Test Data Inspection

```go
// Print test data for debugging
t.Logf("Test data: %+v", testData)

// Use testutils helpers for consistent output
testutils.AssertStringContains(t, output, expectedSubstring)
```

## Best Practices

### Test Organization

1. **Group related tests** using subtests
2. **Use descriptive test names** that explain the scenario
3. **Keep tests focused** - one assertion per test when possible
4. **Use table-driven tests** for multiple scenarios

### Test Data Management

1. **Use testutils helpers** for consistent test data
2. **Clean up temporary files** with defer statements
3. **Use mock implementations** to isolate components
4. **Create realistic test scenarios** that match production use

### Error Testing

1. **Test both success and failure cases**
2. **Validate specific error types** where appropriate
3. **Test edge cases** and boundary conditions
4. **Use structured error testing** with table-driven tests

### Performance Considerations

1. **Use `testing.Short()`** for long-running tests
2. **Profile memory usage** in benchmarks
3. **Test with realistic data sizes**
4. **Monitor test execution time**

---

## Quick Reference

### Essential Commands

```bash
make test                 # Run all tests
make test-functional     # Automated functional workflow tests
make test-complete       # Complete test suite (unit + integration + functional)
make test-coverage       # Generate coverage report
make test-unit          # Unit tests only
make test-integration   # Integration tests only
make test-watch         # Watch mode
make test-clean         # Clean artifacts
```

### Test File Locations

- Unit tests: `*_test.go` files alongside source
- Integration tests: `test/integration/`
- Test utilities: `test/testutils/`
- Mocks: `pkg/types/mocks.go`

### Key Testing Packages

- `testing` - Go standard testing
- `goingenv/test/testutils` - Project test utilities
- `goingenv/pkg/types` - Mock implementations

For more information, see the individual test files and the project's main documentation.