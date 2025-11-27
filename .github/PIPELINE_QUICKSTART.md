# Pipeline Quick Start

Fast reference for working with the goingenv CI/CD pipeline.

## Before Pushing Code

Run the full CI suite locally:

```bash
make ci-full
```

This runs:
- Dependency updates
- Unit tests with race detection
- Linting (format, vet, golangci-lint)
- Build verification
- Security scanning
- Cross-compilation tests

## Creating a Pull Request

1. Run `make ci-full` locally
2. Push to a feature branch
3. Create PR to `main` or `develop`
4. Wait for validation stage to pass (lint, test, security, build-verify)
5. Get review and merge

## Releasing a New Version

### Automatic Release (Recommended)

Merge to `main` with `[release]` in the commit message:

```bash
# Patch release (1.0.0 → 1.0.1)
git commit -m "fix: resolve auth issue [release]"

# Minor release (1.0.0 → 1.1.0)
git commit -m "feat: add new command [release] [minor]"

# Major release (1.0.0 → 2.0.0)
git commit -m "feat!: breaking API changes [release] [major]"
```

### Manual Release

1. Go to GitHub Actions → Pipeline
2. Click "Run workflow"
3. Select `main` branch
4. Choose version type or enter custom version
5. Click "Run workflow"

## Pipeline Stages

```
VALIDATE (always) → BUILD (main only) → RELEASE (main + flag)
```

### Stage 1: Validate
- Runs on all pushes and PRs
- Must pass for PR to be mergeable
- Includes: lint, test, security, build-verify

### Stage 2: Build
- Runs only on `main` branch
- Creates release binaries if `[release]` flag present
- Builds for: Linux/macOS × AMD64/ARM64

### Stage 3: Release
- Runs only when build succeeds and `[release]` flag present
- Creates GitHub Release
- Uploads binaries with checksums
- Validates installation

## Common Commands

```bash
# Full CI suite
make ci-full

# Individual checks
make ci-lint          # Linting only
make ci-test          # Tests only
make ci-security      # Security scan only
make ci-build         # Build verification only
make ci-cross-compile # Cross-compilation only

# Fix common issues
make fmt              # Format code
go mod tidy           # Tidy modules
make test-complete    # Run all tests locally
```

## What Happens on Push

### To Feature Branch
- Validation stage runs
- No builds or releases

### To Main Branch (no [release] flag)
- Validation stage runs
- Build stage skipped
- No release created

### To Main Branch (with [release] flag)
- Validation stage runs
- Build stage creates binaries
- Release stage publishes to GitHub
- Post-validation verifies installation

## Troubleshooting

### Lint Failure
```bash
make fmt
go mod tidy
git add -A
git commit -m "style: fix formatting"
```

### Test Failure
```bash
make test-complete
# Fix failing tests
git add -A
git commit -m "test: fix failing tests"
```

### Security Scan Failure
```bash
make vuln-check
go get -u [vulnerable-package]
go mod tidy
git add -A
git commit -m "security: update vulnerable dependency"
```

### Build Failure
```bash
make build
# Fix build errors
git add -A
git commit -m "fix: resolve build issues"
```

## Pipeline Notifications

- PRs show inline status checks
- Failed jobs appear in GitHub Actions tab
- GitHub sends email on failures (check Settings → Notifications)

## Advanced

### Skip CI for Documentation
Add to commit message:
```
[skip ci]
```

Note: Pushes to `*.md` files already skip CI by default.

### Force Re-run Pipeline
Go to Actions → Failed workflow → Re-run all jobs

### View Pipeline Logs
Actions tab → Select workflow run → Click on job → Expand step

## Getting Help

- Full documentation: [CICD.md](CICD.md)
- View workflow: `.github/workflows/pipeline.yml`
- Issues: Create an issue on GitHub
