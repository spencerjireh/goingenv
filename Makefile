# goingenv Makefile
#
# Tool versions (go, golangci-lint, actionlint, shellcheck, gosec, govulncheck,
# goreleaser, syft, air) are pinned in mise.toml. Run `make bootstrap` once, then
# everything here uses exactly the versions CI uses.

# Recipes run under bash, not sh. Under sh, `echo -e "..."` prints a literal
# "-e " -- which this file used to do in roughly forty places.
SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c

# Remove a target file if its recipe fails, so a partial build is never
# mistaken for a good one.
.DELETE_ON_ERROR:

# Colors for output
GREEN := \033[0;32m
RED := \033[0;31m
YELLOW := \033[1;33m
BLUE := \033[0;34m
CYAN := \033[0;36m
NC := \033[0m

BINARY_NAME := goingenv
MAIN_PATH := ./cmd/goingenv

# `git describe --tags --abbrev=0` returns the newest reachable tag regardless
# of where HEAD is, so a dev build forty commits past v1.0.0 called itself
# v1.0.0. This reports the real position, and marks a dirty tree.
# The leading "v" is kept, matching what goreleaser stamps into releases.
VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

.DEFAULT_GOAL := build

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------

# Install the pinned toolchain and warm the module cache.
bootstrap:
	@printf "$(BLUE)Installing pinned toolchain...$(NC)\n"
	mise install
	go mod download
	@printf "$(GREEN)Toolchain ready:$(NC)\n"
	@go version
	@golangci-lint --version

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

build:
	@printf "$(BLUE)Building $(BINARY_NAME) $(VERSION)...$(NC)\n"
	go build $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@printf "$(GREEN)Build completed: $(BINARY_NAME)$(NC)\n"

# Development build with race detector and debug symbols.
dev:
	@printf "$(BLUE)Building development version with race detector...$(NC)\n"
	go build -race -gcflags="all=-N -l" -o $(BINARY_NAME)-dev $(MAIN_PATH)
	@printf "$(GREEN)Development build completed: $(BINARY_NAME)-dev$(NC)\n"

install: build
	@printf "$(BLUE)Installing $(BINARY_NAME) globally...$(NC)\n"
	go install $(LDFLAGS) $(MAIN_PATH)
	@printf "$(GREEN)$(BINARY_NAME) installed$(NC)\n"

uninstall:
	@GOBIN=$$(go env GOBIN); \
	if [ -z "$$GOBIN" ]; then GOBIN=$$(go env GOPATH)/bin; fi; \
	rm -f "$$GOBIN/$(BINARY_NAME)"
	@printf "$(GREEN)$(BINARY_NAME) uninstalled$(NC)\n"

clean:
	@printf "$(BLUE)Cleaning build artifacts...$(NC)\n"
	go clean
	rm -rf $(COVERDIR)
	rm -f $(BINARY_NAME) $(BINARY_NAME)-dev coverage.out coverage.unit.out coverage.bin.out coverage.html
	rm -rf dist/ build/
	@printf "$(GREEN)Clean completed$(NC)\n"

# ---------------------------------------------------------------------------
# CI-equivalent targets
#
# These mirror .github/workflows/ci.yml. Tools are pinned by mise, so a missing
# tool is an error rather than a warning -- `make ci-full` used to print
# "All CI checks passed locally!" on a machine with no linter installed.
# ---------------------------------------------------------------------------

ci-test:
	@printf "$(BLUE)Running unit tests with race detection...$(NC)\n"
	go test -race -timeout=5m ./pkg/... ./internal/...
	@printf "$(BLUE)Running integration tests...$(NC)\n"
	go test -race -timeout=3m ./test/integration/...
	@printf "$(BLUE)Running CLI tests...$(NC)\n"
	go test -timeout=5m ./test/cli/...
	@printf "$(BLUE)Running e2e tests...$(NC)\n"
	go test -timeout=5m ./test/e2e/...
	@printf "$(BLUE)Running install script tests...$(NC)\n"
	bash test/install/test_install.sh
	@printf "$(GREEN)All tests passed$(NC)\n"

ci-lint:
	@printf "$(BLUE)Checking formatting...$(NC)\n"
	@if [ -n "$$(gofmt -s -l .)" ]; then \
		printf "$(RED)Not formatted:$(NC)\n"; gofmt -s -l .; exit 1; \
	fi
	@printf "$(BLUE)Running go vet...$(NC)\n"
	go vet ./...
	@printf "$(BLUE)Checking go mod tidy...$(NC)\n"
	go mod tidy -diff
	@printf "$(BLUE)Running golangci-lint...$(NC)\n"
	golangci-lint run --config=.golangci.yml
	@printf "$(BLUE)Linting workflows...$(NC)\n"
	actionlint
	@printf "$(BLUE)Linting shell scripts...$(NC)\n"
	shellcheck $(SHELL_SOURCES)
	@printf "$(GREEN)Linting passed$(NC)\n"

ci-security:
	@printf "$(BLUE)Running govulncheck...$(NC)\n"
	govulncheck ./...
	@printf "$(BLUE)Running gosec...$(NC)\n"
	gosec -quiet -exclude=G115,G117,G204,G304,G407,G703 ./...
	@printf "$(GREEN)Security checks passed$(NC)\n"

# Everything CI runs. release-local covers cross-compilation, which is why the
# old ci-cross-compile target (four go builds into /tmp) is gone.
ci-full: ci-lint ci-test ci-security release-local
	@printf "$(GREEN)All CI checks passed locally$(NC)\n"

# ---------------------------------------------------------------------------
# Release
# ---------------------------------------------------------------------------

# Build release artifacts locally with the same goreleaser config CI uses.
# Snapshot mode needs no tag and publishes nothing. Note that snapshot archive
# names carry a -SNAPSHOT- suffix, so they differ from a real release.
release-local:
	@printf "$(BLUE)Building local release artifacts...$(NC)\n"
	@mkdir -p build && cp install.sh build/install.sh
	goreleaser release --snapshot --clean
	@printf "$(GREEN)Local release built in dist/$(NC)\n"
	@ls -la dist/

release-check:
	goreleaser check

# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------

test:
	go test ./...

test-unit:
	go test -short ./pkg/... ./internal/...

test-integration:
	go test -race -timeout=3m ./test/integration/...

test-cli:
	go test -timeout=5m ./test/cli/...

test-e2e:
	go test -timeout=5m ./test/e2e/...

test-install:
	bash test/install/test_install.sh

test-complete: ci-test

# Coverage spans two mechanisms that have to be merged by hand.
#
#   1. The in-process packages -- pkg, internal, test/integration -- produce a
#      TEXT profile straight from `go test -coverprofile`.
#   2. test/cli and test/e2e spawn a separately built binary, so they used to
#      contribute nothing at all however much they exercised. Building that
#      binary with `go build -cover` and pointing GOCOVERDIR at $(COVERDIR)
#      makes each subprocess emit BINARY-format data, which
#      `go tool covdata textfmt` converts into a text profile that can be
#      concatenated onto (1).
#
# Both sides must share a covermode -- atomic, forced by -race on (1) -- or
# `go tool cover` rejects the result. Coverage is opt-in via
# GOINGENV_TEST_COVERDIR: with it unset the test harness builds and runs
# exactly as it always has.
COVERDIR      := $(CURDIR)/coverage-data
COVER_PKGS    := ./cmd/...,./internal/...,./pkg/...
COVER_RUN     := ./pkg/... ./internal/... ./test/integration/...
COVER_BIN_RUN := ./test/cli/... ./test/e2e/...

coverage-collect:
	rm -rf $(COVERDIR) coverage.out coverage.unit.out coverage.bin.out
	mkdir -p $(COVERDIR)
	go test -race -covermode=atomic -coverpkg=$(COVER_PKGS) \
		-coverprofile=coverage.unit.out $(COVER_RUN)
	@# -count=1 is load-bearing: a cached test result never re-runs the
	@# spawned binary, so $(COVERDIR) would stay empty and coverage would
	@# silently collapse back to the in-process number. The instrumented
	@# binary is also slower, hence the longer timeout.
	GOINGENV_TEST_COVERDIR=$(COVERDIR) go test -count=1 -timeout=10m $(COVER_BIN_RUN)
	@# Fail loudly rather than quietly reporting a worse number if the
	@# subprocess plumbing ever breaks -- a dropped -cover flag would
	@# otherwise regress coverage with CI still green.
	@test -n "$$(ls -A $(COVERDIR) 2>/dev/null)" || { \
		printf "$(RED)No coverage data emitted by the spawned binary.$(NC)\n"; \
		printf "Check that BuildBinary passed -cover and that GOCOVERDIR reached the subprocess.\n"; \
		exit 1; }
	go tool covdata textfmt -i=$(COVERDIR) -o=coverage.bin.out
	@# tail -n +2 strips the second profile's "mode:" line.
	@{ cat coverage.unit.out; tail -n +2 coverage.bin.out; } > coverage.out

test-coverage: coverage-collect
	go tool cover -html=coverage.out -o coverage.html
	@printf "$(GREEN)Coverage report: coverage.html$(NC)\n"
	@go tool cover -func=coverage.out | tail -1

test-coverage-ci: coverage-collect
	@go tool cover -func=coverage.out | tail -1

test-bench:
	go test -bench=. -benchmem ./...

test-clean:
	rm -rf $(COVERDIR)
	rm -f coverage.out coverage.unit.out coverage.bin.out coverage.html
	go clean -testcache

# ---------------------------------------------------------------------------
# Code quality
# ---------------------------------------------------------------------------

fmt:
	gofmt -s -w .
	@printf "$(GREEN)Code formatted$(NC)\n"

vet:
	go vet ./...

# Enumerated rather than globbed. A glob also picks up build/install.sh -- the
# gitignored byte-identical copy goreleaser stages for release -- and reports
# every finding twice. `git ls-files '*.sh'` would exclude it for free but
# evaluates to empty in a source tarball with no .git, and shellcheck with no
# arguments reads stdin and hangs.
SHELL_SOURCES := install.sh test/install/test_install.sh

shellcheck:
	shellcheck $(SHELL_SOURCES)

lint:
	golangci-lint run --config=.golangci.yml
	actionlint
	shellcheck $(SHELL_SOURCES)

vuln-check:
	govulncheck ./...

deps:
	go mod download
	go mod tidy
	go mod verify
	@printf "$(GREEN)Dependencies updated$(NC)\n"

check: fmt vet lint test
	@printf "$(GREEN)All checks passed$(NC)\n"

# ---------------------------------------------------------------------------
# Running
# ---------------------------------------------------------------------------

run:
	go run $(MAIN_PATH) $(ARGS)

# ---------------------------------------------------------------------------
# TUI development
#
# The sandbox is a throwaway project with sample .env files, so the TUI has
# something to show without touching a real one.
# ---------------------------------------------------------------------------

TUI_SANDBOX := /tmp/goingenv-sandbox

tui-sandbox:
	@if [ ! -d "$(TUI_SANDBOX)/.goingenv" ]; then \
		mkdir -p "$(TUI_SANDBOX)"; \
		printf "DB_HOST=localhost\nDB_PORT=5432\n" > "$(TUI_SANDBOX)/.env"; \
		printf "SECRET_KEY=dev-secret-123\n" > "$(TUI_SANDBOX)/.env.local"; \
		printf "API_KEY=prod-key-456\nSTRIPE_KEY=sk_live_xxx\n" > "$(TUI_SANDBOX)/.env.production"; \
		printf "DEBUG=true\nLOG_LEVEL=verbose\n" > "$(TUI_SANDBOX)/.env.development"; \
		printf "REDIS_URL=redis://localhost:6379\n" > "$(TUI_SANDBOX)/.env.staging"; \
		cd "$(TUI_SANDBOX)" && "$(CURDIR)/$(BINARY_NAME)" init > /dev/null 2>&1; \
		printf "$(GREEN)Sandbox created: $(TUI_SANDBOX) (5 env files)$(NC)\n"; \
	else \
		printf "$(BLUE)Sandbox ready: $(TUI_SANDBOX)$(NC)\n"; \
	fi

tui: build tui-sandbox
	@cd "$(TUI_SANDBOX)" && "$(CURDIR)/$(BINARY_NAME)"

# Hot-reload. .air.toml already runs the binary inside the sandbox, which is
# why the old watch / watch-run / dev-watch targets were all identical to this
# one and have been removed.
tui-watch: tui-sandbox
	@printf "$(BLUE)Watching for changes... TUI runs in $(TUI_SANDBOX)$(NC)\n"
	air

tui-clean:
	@rm -rf "$(TUI_SANDBOX)"
	@printf "$(GREEN)Sandbox removed$(NC)\n"

# ---------------------------------------------------------------------------
# Misc
# ---------------------------------------------------------------------------

stats:
	@printf "Go files:     %s\n" "$$(find . -name '*.go' -not -path './dist/*' | wc -l | tr -d ' ')"
	@printf "Lines of Go:  %s\n" "$$(find . -name '*.go' -not -path './dist/*' -exec cat {} + | wc -l | tr -d ' ')"
	@printf "Packages:     %s\n" "$$(go list ./... | wc -l | tr -d ' ')"
	@printf "Dependencies: %s\n" "$$(go list -m all | wc -l | tr -d ' ')"

help:
	@printf "$(CYAN)goingenv build system$(NC)\n"
	@printf "\n"
	@printf "$(BLUE)Setup$(NC)\n"
	@printf "  bootstrap         Install the pinned toolchain (mise) and warm caches\n"
	@printf "\n"
	@printf "$(BLUE)Build$(NC)\n"
	@printf "  build             Build for the current platform\n"
	@printf "  dev               Build with race detector and debug symbols\n"
	@printf "  install           Install the binary globally\n"
	@printf "  uninstall         Remove the installed binary\n"
	@printf "  clean             Remove build artifacts\n"
	@printf "\n"
	@printf "$(BLUE)Test$(NC)\n"
	@printf "  test              Run every test\n"
	@printf "  test-unit         Unit tests only\n"
	@printf "  test-integration  Integration tests (in-process, race enabled)\n"
	@printf "  test-cli          CLI tests (spawns the built binary)\n"
	@printf "  test-e2e          End-to-end tests\n"
	@printf "  test-install      install.sh checksum verification tests\n"
	@printf "  test-complete     Everything CI runs\n"
	@printf "  test-coverage     Coverage report incl. CLI/E2E (writes coverage.html)\n"
	@printf "  test-bench        Benchmarks\n"
	@printf "\n"
	@printf "$(BLUE)Quality$(NC)\n"
	@printf "  fmt               Format the code\n"
	@printf "  vet               go vet\n"
	@printf "  lint              golangci-lint, actionlint and shellcheck\n"
	@printf "  shellcheck        install.sh and its test suite only\n"
	@printf "  vuln-check        govulncheck\n"
	@printf "  check             fmt + vet + lint + test\n"
	@printf "\n"
	@printf "$(BLUE)CI equivalents$(NC)\n"
	@printf "  ci-full           Everything CI runs (lint, test, security, release)\n"
	@printf "  ci-lint           Formatting, vet, tidy, golangci-lint, actionlint, shellcheck\n"
	@printf "  ci-test           The full test suite\n"
	@printf "  ci-security       govulncheck and gosec\n"
	@printf "\n"
	@printf "$(BLUE)Release$(NC)\n"
	@printf "  release-local     Build release artifacts locally (goreleaser snapshot)\n"
	@printf "  release-check     Validate .goreleaser.yaml\n"
	@printf "\n"
	@printf "$(BLUE)TUI development$(NC)\n"
	@printf "  tui               Build and launch the TUI in a sandbox\n"
	@printf "  tui-watch         Same, with hot-reload via air\n"
	@printf "  tui-clean         Remove the sandbox\n"
	@printf "\n"
	@printf "$(BLUE)Examples$(NC)\n"
	@printf "  make bootstrap                  # first-time setup\n"
	@printf "  make ci-full                    # everything CI runs, before pushing\n"
	@printf "  make run ARGS='status .'        # run a subcommand\n"
	@printf "\n"
	@printf "Releases: git tag -a v1.2.3 -m 'Release v1.2.3' && git push origin v1.2.3\n"

.PHONY: bootstrap build dev install uninstall clean \
        ci-test ci-lint ci-security ci-full \
        release-local release-check \
        test test-unit test-integration test-cli test-e2e test-install \
        test-complete test-coverage test-coverage-ci coverage-collect test-bench test-clean \
        fmt vet lint shellcheck vuln-check deps check \
        run tui tui-sandbox tui-watch tui-clean \
        stats help
