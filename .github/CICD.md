# CI/CD Pipeline

## Quick Reference

| Task | Command |
|------|---------|
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

---

## Overview

The pipeline is split into two workflow files:

- **`ci.yml`** -- Runs on PRs and pushes to `main`/`develop`. Validates code quality.
- **`release.yml`** -- Runs on `v*` tag pushes. Builds binaries and publishes a GitHub Release.

## CI Workflow (`ci.yml`)

Triggered on all pushes and pull requests to `main` and `develop` branches.

**Jobs:**

- **lint** -- Code quality checks (gofmt, go vet, go mod tidy, golangci-lint)
- **test** -- Matrix: Ubuntu/macOS x Go 1.23/stable. Unit tests with race detection, integration tests, functional tests. All tests must pass.
- **security** -- Vulnerability scanning (govulncheck), security code analysis (gosec), SARIF upload to GitHub Security tab.
- **test-install-script** -- Depends on lint, test, security. Validates install.sh syntax, help output, and dry run on Ubuntu and macOS.

```
lint ──────────┐
               │
test ──────────┼──→ test-install-script
               │
security ──────┘
```

## Release Workflow (`release.yml`)

Triggered by pushing a tag matching `v*`. The tag name is the version.

**Jobs:**

- **build-release** -- Matrix: Linux/macOS x AMD64/ARM64. Builds optimized binaries with `-trimpath` and embedded version info. Creates tar.gz archives with SHA256 checksums.
- **create-release** -- Depends on build-release. Downloads artifacts, generates release notes via GitHub API, creates versioned install.sh, publishes GitHub Release.

```
build-release (4 platforms) ──→ create-release
```

### Creating a Release

```bash
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3
```

- Stable versions (`v1.2.3`) are marked as the latest release.
- Prereleases (`v1.2.3-alpha.1`, `v1.2.3-rc.1`) are marked as prerelease.

## Environment Variables

The pipeline embeds these during builds:

- `VERSION` -- From the Git tag (e.g., `v1.2.3`)
- `BUILD_TIME` -- UTC timestamp of build
- `GIT_COMMIT` -- Full Git commit SHA

## Local Testing

```bash
# Run all CI checks
make ci-full

# Individual checks
make ci-lint
make ci-test
make ci-security
make ci-cross-compile

# Build release binaries locally
make release-local
```

## Pages Deployment

The `pages.yml` workflow is separate and deploys documentation to GitHub Pages. It triggers on pushes to `index.html` and is independent of the main pipeline.
