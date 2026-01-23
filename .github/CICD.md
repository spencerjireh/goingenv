# CI/CD Pipeline Architecture

This document describes the unified CI/CD pipeline for goingenv.

## Quick Reference

| Task | Command |
|------|---------|
| Run all CI checks | `make ci-full` |
| Test release locally | `./scripts/test-release.sh VERSION` |
| Create patch release | `git commit -m "fix: message [release]"` |
| Create minor release | `git commit -m "feat: message [release] [minor]"` |
| Create major release | `git commit -m "feat!: message [release] [major]"` |
| One-command alpha | `make quick-alpha` |
| Check release status | `make check-release-status` |

### Before Pushing Code

```bash
make ci-full
```

This runs: dependency updates, unit tests with race detection, linting, build verification, security scanning, and cross-compilation tests.

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

---

## Overview

The pipeline is defined in a single workflow file (`.github/workflows/pipeline.yml`) with three distinct stages:

```
STAGE 1: VALIDATE → STAGE 2: BUILD → STAGE 3: RELEASE
```

## Pipeline Stages

### Stage 1: Validate (Always Runs)

Runs on all pushes and pull requests to `main` and `develop` branches.

**Jobs:**
- **lint** - Code quality checks
  - Go formatting verification (`gofmt`)
  - Static analysis (`go vet`)
  - Module tidiness check (`go mod tidy`)
  - Comprehensive linting (`golangci-lint`)

- **test** - Comprehensive testing
  - Matrix: Ubuntu/macOS × Go 1.21/1.22
  - Unit tests with race detection
  - Integration tests
  - Functional workflow tests
  - **All tests must pass** (no failures ignored)

- **security** - Security scanning
  - Vulnerability scanning (`govulncheck`)
  - Security code analysis (`gosec`)
  - SARIF upload for GitHub Security tab

- **build-verify** - Build verification
  - Depends on: lint, test, security
  - Current platform build test
  - Cross-compilation verification (Linux/macOS, AMD64/ARM64)

- **test-install-script** - Installation testing
  - Depends on: lint, test, security
  - Matrix: Ubuntu/macOS
  - Syntax validation
  - Help command testing
  - Dry run validation

### Stage 2: Build (Main Branch Only)

Runs only on pushes to `main` branch, after Stage 1 passes.

**Jobs:**
- **check-release** - Release decision logic
  - Depends on: build-verify, test-install-script
  - Checks for `[release]` flag in commit message
  - Calculates next version based on flags:
    - `[major]` - Major version bump (X.0.0)
    - `[minor]` - Minor version bump (0.X.0)
    - Default - Patch version bump (0.0.X)
  - Outputs: `should-release`, `next-version`

- **build-release** - Multi-platform builds
  - Depends on: check-release
  - Only if: `should-release == true`
  - Matrix: Linux/macOS × AMD64/ARM64
  - Creates optimized release binaries
  - Generates SHA256 checksums
  - Uploads build artifacts

### Stage 3: Release (Main Branch + Release Flag)

Runs only when Stage 2 completes successfully and release is triggered.

**Jobs:**
- **create-release** - GitHub Release creation
  - Depends on: check-release, build-release
  - Only if: `should-release == true`
  - Downloads all build artifacts
  - Creates Git tag
  - Generates release notes (auto + custom)
  - Publishes GitHub Release with binaries

- **post-release-validation** - Release verification
  - Depends on: check-release, create-release
  - Only if: `should-release == true`
  - Matrix: Ubuntu/macOS
  - Validates install script with new release
  - Tests direct binary download
  - Verifies binary functionality

- **notify-success** - Success notification
  - Depends on: check-release, post-release-validation
  - Outputs release information
  - Provides install commands

- **notify-failure** - Failure notification
  - Depends on: all jobs
  - Only runs on failure
  - Provides debugging guidance

## Triggering Releases

### Automatic Release (Recommended)

Include `[release]` in your commit message when pushing to `main`:

```bash
git commit -m "feat: add new feature [release]"
git push origin main
```

**Version Control:**
- `[release]` - Patch bump (1.0.0 → 1.0.1)
- `[release] [minor]` - Minor bump (1.0.0 → 1.1.0)
- `[release] [major]` - Major bump (1.0.0 → 2.0.0)

### Manual Release

Use GitHub Actions UI:

1. Go to Actions → Pipeline → Run workflow
2. Select `main` branch
3. Choose version type (patch/minor/major) or provide custom version
4. Click "Run workflow"

## Key Improvements Over Old System

### Before (ci.yml + auto-release.yml)

- Tests could fail but pipeline continued
- No dependency between CI and release
- Duplicate security/lint checks
- Two separate workflow files
- Confusing concurrency control

### After (pipeline.yml)

- **Zero tolerance for test failures** - All tests must pass
- **Proper job dependencies** - Release only after successful validation
- **No duplication** - Each check runs once
- **Single unified workflow** - Clear stage progression
- **Smart concurrency** - Per-branch cancellation

## Job Dependencies

```
lint ────────────────┐
                     │
test ────────────────┼──→ build-verify ──┐
                     │                    │
security ────────────┤                    ├──→ check-release ──→ build-release ──→ create-release ──→ post-release-validation ──→ notify-success
                     │                    │
test-install-script ─┘                    │
                                          │
notify-failure ←──────────────────────────┘ (on any failure)
```

## Environment Variables

The pipeline uses these environment variables during builds:

- `VERSION` - Semantic version (e.g., 1.2.3)
- `BUILD_TIME` - Timestamp of build
- `GIT_COMMIT` - Short Git commit SHA

## Caching Strategy

Go modules and build cache are shared across jobs using GitHub Actions cache:

**Cache Key:** `${{ runner.os }}-go-${{ go-version }}-${{ hashFiles('**/go.sum') }}`

This significantly speeds up builds by reusing dependencies.

## Security Features

1. **SARIF Upload** - Security findings appear in GitHub Security tab
2. **Checksum Verification** - SHA256 checksums for all release assets
3. **Automated Scanning** - Every commit is scanned for vulnerabilities
4. **Post-Release Validation** - Ensures releases are installable

## Monitoring

### View Pipeline Status

- Check Actions tab: `https://github.com/[owner]/goingenv/actions`
- PR status checks show inline results
- Email notifications on failure (configurable)

### Common Issues

**Pipeline fails on lint:**
```bash
make fmt
go mod tidy
```

**Pipeline fails on test:**
```bash
make test-complete
```

**Pipeline fails on security:**
```bash
make vuln-check
make security-scan
```

**Build fails on cross-compilation:**
```bash
make ci-cross-compile
```

## Local Testing

Run the same checks locally before pushing:

```bash
# Run all CI checks
make ci-full

# Individual checks
make ci-lint
make ci-test
make ci-security
make ci-build
make ci-cross-compile
```

## Pages Deployment

The `pages.yml` workflow is separate and deploys documentation to GitHub Pages:

- Triggers on pushes to `index.html`
- Independent of main pipeline
- Simple static file deployment

## Future Enhancements

Potential improvements to consider:

- [ ] Add code coverage reporting
- [ ] Add performance benchmarking
- [ ] Add Docker image builds
- [ ] Add Homebrew tap updates
- [ ] Add changelog automation
- [ ] Add release notes from commit messages
- [ ] Add semantic-release integration
