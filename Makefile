# goingenv Makefile - Refactored Structure

# Color variables for output
GREEN=\033[0;32m
RED=\033[0;31m
YELLOW=\033[1;33m
BLUE=\033[0;34m
CYAN=\033[0;36m
PURPLE=\033[0;35m
NC=\033[0m

# Variables
BINARY_NAME=goingenv
MAIN_PATH=./cmd/goingenv
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "dev")
BUILD_TIME=$(shell date +%FT%T%z)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Default target
.DEFAULT_GOAL := build

# Build the binary for current platform
build:
	@printf "$(BLUE)Building $(BINARY_NAME) v$(VERSION)...$(NC)\n"
	@echo "Main path: $(MAIN_PATH)"
	go build $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@printf "$(GREEN)Build completed: $(BINARY_NAME)$(NC)\n"

# Build for development with race detector and debug info
dev:
	@echo -e "$(BLUE)Building development version with race detector...$(NC)"
	go build -race -gcflags="all=-N -l" -o $(BINARY_NAME)-dev $(MAIN_PATH)
	@echo -e "$(GREEN)Development build completed: $(BINARY_NAME)-dev$(NC)"

# CI-friendly targets
ci-test:
	@echo -e "$(BLUE)Running CI tests...$(NC)"
	@echo -e "$(BLUE)Running unit tests with race detection...$(NC)"
	go test -race -timeout=5m ./pkg/... ./internal/...
	@echo -e "$(BLUE)Running integration tests...$(NC)"
	go test -v -timeout=2m ./test/integration/...
	@echo -e "$(BLUE)Running e2e tests...$(NC)"
	go test -v -timeout=5m ./test/e2e/...
	@echo -e "$(GREEN)All tests passed$(NC)"

ci-lint:
	@echo -e "$(BLUE)Running CI linting...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)WARNING:$(NC)  golangci-lint not installed, using go vet only"; \
	fi
	go vet ./...
	@echo -e "$(GREEN)Linting passed$(NC)"

ci-security:
	@echo -e "$(BLUE)Running security checks...$(NC)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -exclude=G115,G117,G204,G304,G407,G703 ./...; \
	else \
		echo "$(YELLOW)WARNING:$(NC)  gosec not installed, skipping security scan"; \
	fi
	@echo -e "$(GREEN)Security checks completed$(NC)"

ci-cross-compile:
	@echo "Testing cross-compilation...$(NC)"
	GOOS=linux GOARCH=amd64 go build -o /tmp/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 go build -o /tmp/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 go build -o /tmp/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build -o /tmp/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo -e "$(GREEN)Cross-compilation successful$(NC)"

# Run all CI checks locally
ci-full: deps ci-test ci-lint ci-security ci-cross-compile
	@echo -e "$(GREEN)All CI checks passed locally!$(NC)"

# Build release binaries locally (simulates what GitHub Actions builds)
release-local: clean
	@printf "$(BLUE)Building local release binaries (v$(VERSION))...$(NC)\n"
	@mkdir -p dist
	@for platform in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64; do \
		os=$${platform%%-*}; arch=$${platform##*-}; \
		printf "$(BLUE)  Building $$platform...$(NC)\n"; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -trimpath -o dist/$(BINARY_NAME) $(MAIN_PATH); \
		cd dist && tar -czf $(BINARY_NAME)-v$(VERSION)-$$platform.tar.gz $(BINARY_NAME) && rm $(BINARY_NAME) && cd ..; \
	done
	@cd dist && shasum -a 256 *.tar.gz > checksums.txt
	@printf "$(GREEN)Local release built in dist/ (version: $(VERSION))$(NC)\n"
	@ls -la dist/

# Automated functional testing workflow
test-functional:
	@echo -e "$(BLUE)Running functional test workflow...$(NC)"
	@echo "Step 1: Building application..."
	@rm -rf test_env_files_functional
	@make build > /dev/null
	@echo -e "$(GREEN)[OK]$(NC) Build completed"

	@echo "Step 2: Creating test environment files..."
	@mkdir -p test_env_files_functional
	@echo "TEST=value" > test_env_files_functional/.env
	@echo "LOCAL=test" > test_env_files_functional/.env.local
	@echo "DEV=true" > test_env_files_functional/.env.development
	@echo "CUSTOM=value" > test_env_files_functional/.env.custom
	@echo "BACKUP=old" > test_env_files_functional/.env.backup
	@echo "NEW=format" > test_env_files_functional/.env.new_format
	@echo "IGNORED=value" > test_env_files_functional/regular.txt
	@echo -e "$(GREEN)[OK]$(NC) Test files created (6 .env files + 1 regular file)"
	
	@echo "Step 3: Backing up existing config..."
	@if [ -f ~/.goingenv.json ]; then \
		cp ~/.goingenv.json ~/.goingenv.json.test-backup; \
		echo -e "$(YELLOW)!$(NC) Existing config backed up"; \
	else \
		echo -e "$(GREEN)[OK]$(NC) No existing config to backup"; \
	fi
	
	@echo "Step 4: Testing all-inclusive pattern (no config)..."
	@rm -f ~/.goingenv.json
	@cd test_env_files_functional && ../goingenv init > /dev/null 2>&1
	@files_detected=$$(cd test_env_files_functional && ../goingenv status . | grep -c "\.env"); \
	if [ "$$files_detected" -eq 6 ]; then \
		echo -e "$(GREEN)[OK]$(NC) All-inclusive pattern working ($$files_detected/6 files detected)"; \
	else \
		echo -e "$(RED)[FAIL]$(NC) All-inclusive pattern failed ($$files_detected/6 files detected)"; \
		exit 1; \
	fi
	
	@echo "Step 5: Testing exclusion patterns..."
	@echo '{"default_depth": 10, "env_patterns": ["\\\\.env.*"], "env_exclude_patterns": ["\\\\.env\\\\.backup$$"], "exclude_patterns": ["node_modules/", "\\\\.git/"], "max_file_size": 10485760}' > ~/.goingenv.json
	@files_detected=$$(cd test_env_files_functional && ../goingenv status . | grep -c "\.env"); \
	if [ "$$files_detected" -eq 5 ]; then \
		echo -e "$(GREEN)[OK]$(NC) Exclusion patterns working ($$files_detected/5 files detected, .env.backup excluded)"; \
	else \
		echo -e "$(RED)[FAIL]$(NC) Exclusion patterns failed ($$files_detected/5 files detected)"; \
		exit 1; \
	fi
	
	@echo "Step 6: Testing pack/unpack functionality..."
	@echo "Step 6a: Initializing goingenv in test directory..."
	@cd test_env_files_functional && ../goingenv init > /dev/null 2>&1
	@echo -e "$(GREEN)[OK]$(NC) goingenv initialized in test directory"
	@cd test_env_files_functional && echo "test123" | ../goingenv pack --password-env TEST_PASSWORD -o functional-test.enc > /dev/null 2>&1 || TEST_PASSWORD="test123" ../goingenv pack --password-env TEST_PASSWORD -o functional-test.enc > /dev/null
	@if [ -f test_env_files_functional/.goingenv/functional-test.enc ]; then \
		echo -e "$(GREEN)[OK]$(NC) Pack functionality working"; \
	else \
		echo -e "$(RED)[FAIL]$(NC) Pack functionality failed"; \
		exit 1; \
	fi
	@mkdir -p test_env_files_functional/unpacked
	@cd test_env_files_functional && TEST_PASSWORD="test123" ../goingenv unpack -f .goingenv/functional-test.enc --password-env TEST_PASSWORD -t unpacked > /dev/null
	@unpacked_files=$$(find test_env_files_functional/unpacked -name ".env*" | wc -l); \
	if [ "$$unpacked_files" -eq 5 ]; then \
		echo -e "$(GREEN)[OK]$(NC) Unpack functionality working ($$unpacked_files files restored)"; \
	else \
		echo -e "$(RED)[FAIL]$(NC) Unpack functionality failed ($$unpacked_files files restored)"; \
		exit 1; \
	fi
	
	@echo "Step 7: Cleaning up..."
	@rm -rf test_env_files_functional
	@rm -f ~/.goingenv.json
	@if [ -f ~/.goingenv.json.test-backup ]; then \
		mv ~/.goingenv.json.test-backup ~/.goingenv.json; \
		echo -e "$(YELLOW)!$(NC) Original config restored"; \
	else \
		echo -e "$(GREEN)[OK]$(NC) Cleanup completed"; \
	fi
	
	@echo ""
	@echo -e "$(GREEN)All functional tests passed!$(NC)"
	@echo "[OK] All-inclusive .env.* pattern detection"
	@echo "[OK] Exclusion pattern functionality"
	@echo "[OK] Pack/unpack workflow"
	@echo "[OK] Configuration management"

# Complete test suite including functional tests
test-complete: clean
	@echo -e "$(BLUE)Running complete test suite...$(NC)"
	@echo ""
	@make ci-test
	@echo ""
	@make test-functional
	@echo ""
	@echo -e "$(GREEN)Complete test suite passed!$(NC)"
	@echo "[OK] Unit tests with race detection"
	@echo "[OK] Integration tests"
	@echo "[OK] E2E tests"
	@echo "[OK] Functional workflow tests"

# Clean build artifacts
clean:
	@printf "$(BLUE)Cleaning build artifacts...$(NC)\n"
	go clean
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-dev
	rm -rf dist/
	rm -f coverage.out coverage.html
	@printf "$(GREEN)Clean completed$(NC)\n"

# Install dependencies and tidy modules
deps:
	@echo -e "$(BLUE)Installing dependencies...$(NC)"
	go mod download
	go mod tidy
	go mod verify
	@echo "Dependencies updated"

# Format code
fmt:
	@echo "Formatting code...$(NC)"
	go fmt ./...
	@echo "Code formatted"

# Vet code for issues
vet:
	@echo "Vetting code...$(NC)"
	go vet ./...
	@echo "Code vetted"

# Run linter (requires golangci-lint)
lint:
	@echo -e "$(BLUE)Running linter...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
		echo "Linting completed$(NC)\""; \
	else \
		echo "$(YELLOW)WARNING:$(NC)  golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Test targets
test:
	@echo -e "$(BLUE)Running all tests...$(NC)"
	go test -v ./...
	@echo "All tests completed$(NC)\""

test-unit:
	@echo -e "$(BLUE)Running unit tests...$(NC)"
	go test -v -short ./pkg/... ./internal/...
	@echo "Unit tests completed$(NC)\""

test-integration:
	@echo -e "$(BLUE)Running integration tests...$(NC)"
	go test -v -run TestFull ./test/integration/...
	@echo "Integration tests completed$(NC)\""

test-e2e:
	@echo -e "$(BLUE)Running e2e tests...$(NC)"
	go test -v -timeout=5m ./test/e2e/...
	@echo -e "$(GREEN)E2E tests completed$(NC)"

test-coverage:
	@echo -e "$(BLUE)Running tests with coverage...$(NC)"
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@echo "Coverage summary:"
	@go tool cover -func=coverage.out | tail -1

test-coverage-ci:
	@echo -e "$(BLUE)Running tests with coverage for CI...$(NC)"
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -func=coverage.out

test-watch:
	@echo -e "$(BLUE)Running tests in watch mode (requires air)...$(NC)"
	@if command -v air >/dev/null 2>&1 || [ -f $(shell go env GOPATH)/bin/air ]; then \
		AIR_BIN=$$(command -v air || echo "$(shell go env GOPATH)/bin/air"); \
		$$AIR_BIN -c .air.toml -- test ./...; \
	else \
		echo "$(YELLOW)WARNING:$(NC)  air not installed. Install with: go install github.com/air-verse/air@latest"; \
	fi

test-verbose:
	@echo -e "$(BLUE)Running tests with verbose output...$(NC)"
	go test -v -race ./... -args -test.v

test-bench:
	@echo -e "$(BLUE)Running benchmarks...$(NC)"
	go test -bench=. -benchmem ./...
	@echo "Benchmarks completed$(NC)\""

test-clean:
	@echo "Cleaning test artifacts...$(NC)"
	rm -f coverage.out coverage.html
	rm -rf test/tmp/*
	go clean -testcache
	@echo "Test artifacts cleaned"

# Mock generation (if using gomock)
generate-mocks:
	@echo "Generating mocks...$(NC)"
	@if command -v mockgen >/dev/null 2>&1; then \
		mockgen -source=pkg/types/types.go -destination=pkg/types/mocks_generated.go -package=types; \
		echo "Mocks generated"; \
	else \
		echo "$(YELLOW)WARNING:$(NC)  mockgen not installed. Install with: go install github.com/golang/mock/mockgen@latest"; \
	fi

# Run benchmarks
bench: test-bench

# Run all checks (format, vet, lint, test)
check: fmt vet lint test
	@echo "All checks passed"

# Run comprehensive checks including integration tests
check-full: fmt vet lint test-unit test-integration
	@echo "All comprehensive checks passed"

# Install the binary globally
install: build
	@echo -e "$(BLUE)Installing $(BINARY_NAME) globally...$(NC)"
	go install $(LDFLAGS) $(MAIN_PATH)
	@echo "$(BINARY_NAME) installed globally"

# Uninstall the binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME)...$(NC)"
	@GOBIN=$$(go env GOBIN); \
	if [ -z "$$GOBIN" ]; then GOBIN=$$(go env GOPATH)/bin; fi; \
	rm -f "$$GOBIN/$(BINARY_NAME)"
	@echo "$(BINARY_NAME) uninstalled"

# Run the application
run:
	@echo -e "$(BLUE)Running $(BINARY_NAME)...$(NC)"
	go run $(MAIN_PATH) $(ARGS)

# Run with specific command
run-pack:
	@echo -e "$(BLUE)Running pack command...$(NC)"
	go run $(MAIN_PATH) pack $(ARGS)

run-unpack:
	@echo -e "$(BLUE)Running unpack command...$(NC)"
	go run $(MAIN_PATH) unpack $(ARGS)

run-list:
	@echo -e "$(BLUE)Running list command...$(NC)"
	go run $(MAIN_PATH) list $(ARGS)

run-status:
	@echo -e "$(BLUE)Running status command...$(NC)"
	go run $(MAIN_PATH) status $(ARGS)

# Development utilities

# Set up demo environment with sample files
demo:
	@echo "Setting up demo environment...$(NC)"
	mkdir -p demo/project1 demo/project2/config demo/project3
	echo "DATABASE_URL=postgres://localhost:5432/myapp" > demo/project1/.env
	echo "API_KEY=demo-api-key-12345" > demo/project1/.env.local
	echo "DEBUG=true" > demo/project1/.env.development
	echo "NODE_ENV=production" > demo/project1/.env.production
	echo "REDIS_URL=redis://localhost:6379" > demo/project2/.env
	echo "SECRET_KEY=super-secret-key" > demo/project2/config/.env.staging
	echo "AWS_REGION=us-east-1" > demo/project3/.env.test
	@echo "Demo environment created in demo/"

# Clean demo environment
clean-demo:
	@echo "Cleaning demo environment...$(NC)"
	rm -rf demo
	@echo "Demo environment cleaned"

# Run demo scenario
demo-scenario: demo build
	@echo -e "$(BLUE)Running demo scenario...$(NC)"
	cd demo/project1 && ../../$(BINARY_NAME) status
	cd demo/project1 && DEMO_PASSWORD="demo123" ../../$(BINARY_NAME) pack --password-env DEMO_PASSWORD -o demo-backup.enc
	cd demo/project1 && DEMO_PASSWORD="demo123" ../../$(BINARY_NAME) list -f .goingenv/demo-backup.enc --password-env DEMO_PASSWORD
	@echo "Demo scenario completed$(NC)\""

# Development server with hot-reload (for TUI testing)
dev-watch:
	@echo -e "$(BLUE)Starting development with hot-reload...$(NC)"
	@echo "Press Ctrl+C to stop"
	@if command -v air >/dev/null 2>&1 || [ -f $(shell go env GOPATH)/bin/air ]; then \
		AIR_BIN=$$(command -v air || echo "$(shell go env GOPATH)/bin/air"); \
		$$AIR_BIN; \
	else \
		echo "$(YELLOW)WARNING:$(NC)  air not installed. Install with: go install github.com/air-verse/air@latest"; \
		echo "Falling back to regular dev build..."; \
		make dev; \
	fi

# Development server (for TUI testing) - deprecated, use dev-watch
dev-server: dev-watch

# Watch and auto-rebuild (no execution)
watch:
	@echo -e "$(BLUE)Watching for changes and rebuilding...$(NC)"
	@if command -v air >/dev/null 2>&1 || [ -f $(shell go env GOPATH)/bin/air ]; then \
		AIR_BIN=$$(command -v air || echo "$(shell go env GOPATH)/bin/air"); \
		$$AIR_BIN; \
	else \
		echo "$(YELLOW)WARNING:$(NC)  air not installed. Install with: go install github.com/air-verse/air@latest"; \
	fi

# Watch and run with custom arguments
watch-run:
	@echo -e "$(BLUE)Watching and running with args: $(ARGS)...$(NC)"
	@if command -v air >/dev/null 2>&1 || [ -f $(shell go env GOPATH)/bin/air ]; then \
		AIR_BIN=$$(command -v air || echo "$(shell go env GOPATH)/bin/air"); \
		$$AIR_BIN -- $(ARGS); \
	else \
		echo "$(YELLOW)WARNING:$(NC)  air not installed. Install with: go install github.com/air-verse/air@latest"; \
	fi

# Profile the application
profile:
	@echo -e "$(BLUE)Running with CPU profiling...$(NC)"
	TEST_PASSWORD="test" go run $(MAIN_PATH) pack --password-env TEST_PASSWORD -cpuprofile=cpu.prof
	go tool pprof cpu.prof

# Memory profile
profile-mem:
	@echo -e "$(BLUE)Running with memory profiling...$(NC)"
	TEST_PASSWORD="test" go run $(MAIN_PATH) pack --password-env TEST_PASSWORD -memprofile=mem.prof
	go tool pprof mem.prof

# Security scan (requires gosec)
security-scan:
	@echo -e "$(BLUE)Running security scan...$(NC)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -exclude=G115,G117,G204,G304,G407,G703 ./...; \
		echo "Security scan completed$(NC)\""; \
	else \
		echo "$(YELLOW)WARNING:$(NC)  gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Dependency vulnerability check
vuln-check:
	@echo "Checking for vulnerabilities...$(NC)"
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
		echo "Vulnerability check completed$(NC)\""; \
	else \
		echo "$(YELLOW)WARNING:$(NC)  govulncheck not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

# Generate documentation
docs:
	@echo "Generating documentation...$(NC)"
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Documentation server: http://localhost:6060/pkg/goingenv/"; \
		godoc -http=:6060; \
	else \
		echo "$(YELLOW)WARNING:$(NC)  godoc not installed. Install with: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# Show project statistics
stats:
	@echo "Project Statistics:"
	@echo "=================="
	@echo "Go files: $$(find . -name '*.go' | wc -l)"
	@echo "Lines of code: $$(find . -name '*.go' -exec wc -l {} + | tail -1 | awk '{print $$1}')"
	@echo "Packages: $$(go list ./... | wc -l)"
	@echo "Dependencies: $$(go list -m all | wc -l)"
	@echo "Binary size: $$(if [ -f $(BINARY_NAME) ]; then ls -lh $(BINARY_NAME) | awk '{print $$5}'; else echo 'Not built'; fi)"

# Show help
help:
	@echo "goingenv Build System"
	@echo "===================="
	@echo ""
	@echo "Build Commands:"
	@echo " build          - Build binary for current platform"
	@echo " dev            - Build development version with race detector"
	@echo " release-local  - Build release binaries for all platforms locally"
	@echo ""
	@echo "Development Commands:"
	@echo " clean          - Clean build artifacts"
	@echo " deps           - Install and update dependencies"
	@echo " fmt            - Format code"
	@echo " vet            - Vet code for issues"
	@echo " lint           - Run linter (requires golangci-lint)"
	@echo " check          - Run all checks (fmt, vet, lint, test)"
	@echo " check-full     - Run comprehensive checks including integration tests"
	@echo " watch          - Watch for changes and auto-rebuild (requires air)"
	@echo " dev-watch      - Watch and run TUI on changes (requires air)"
	@echo " watch-run      - Watch and run with ARGS (requires air)"
	@echo ""
	@echo "Test Commands:"
	@echo " test           - Run all tests"
	@echo " test-unit      - Run unit tests only"
	@echo " test-integration - Run integration tests only"
	@echo " test-e2e       - Run e2e tests only"
	@echo " test-functional - Run automated functional workflow tests"
	@echo " test-complete  - Run complete test suite (unit + integration + e2e + functional)"
	@echo " test-coverage  - Run tests with coverage report"
	@echo " test-coverage-ci - Run tests with coverage for CI"
	@echo " test-watch     - Run tests in watch mode (requires air)"
	@echo " test-verbose   - Run tests with verbose output"
	@echo " test-bench     - Run benchmarks"
	@echo " test-clean     - Clean test artifacts"
	@echo " generate-mocks - Generate mock implementations"
	@echo " bench          - Run benchmarks"
	@echo ""
	@echo "Install Commands:"
	@echo " install        - Install binary globally"
	@echo " uninstall      - Uninstall binary"
	@echo ""
	@echo "Run Commands:"
	@echo " run            - Run application (use ARGS= for arguments)"
	@echo " run-pack       - Run pack command (use ARGS= for arguments)"
	@echo " run-unpack     - Run unpack command"
	@echo " run-list       - Run list command"
	@echo " run-status     - Run status command"
	@echo ""
	@echo "Demo Commands:"
	@echo " demo           - Set up demo environment"
	@echo " clean-demo     - Clean demo environment"
	@echo " demo-scenario  - Run complete demo scenario"
	@echo ""
	@echo "Analysis Commands:"
	@echo " profile        - Run with CPU profiling"
	@echo " profile-mem    - Run with memory profiling"
	@echo " security-scan  - Run security scan (requires gosec)"
	@echo " vuln-check     - Check for vulnerabilities (requires govulncheck)"
	@echo " stats          - Show project statistics"
	@echo ""
	@echo "Documentation:"
	@echo " docs           - Start documentation server"
	@echo " help           - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo " make build                    # Build for current platform"
	@echo " make watch                    # Hot-reload development"
	@echo " make watch-run ARGS='status demo/'  # Hot-reload with specific command"
	@echo " make ci-full                  # Run all CI checks locally"
	@echo " make release-local            # Build release binaries locally"
	@echo " make run ARGS='pack'          # Run pack command (interactive password)"
	@echo " make demo-scenario            # Full demo with sample files"
	@echo ""
	@echo "Releases: git tag -a v1.2.3 -m 'Release v1.2.3' && git push origin v1.2.3"

# Phony targets
.PHONY: build dev clean deps fmt vet lint test test-unit test-integration test-e2e \
        test-functional test-complete test-coverage test-coverage-ci test-watch test-verbose test-bench test-clean \
        generate-mocks bench check check-full release-local \
        install uninstall run run-pack run-unpack run-list run-status \
        demo clean-demo demo-scenario dev-server dev-watch watch watch-run profile \
        profile-mem security-scan vuln-check docs stats help \
        ci-test ci-lint ci-security ci-cross-compile ci-full