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
VERSION=1.0.0
BUILD_TIME=$(shell date +%FT%T%z)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Build targets for different platforms
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME).exe
BINARY_DARWIN=$(BINARY_NAME)_darwin

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

# Build optimized release version
release-build:
	@echo -e "$(BLUE)Building optimized release version...$(NC)"
	go build $(LDFLAGS) -trimpath -o $(BINARY_NAME) $(MAIN_PATH)
	@echo -e "$(GREEN)Release build completed: $(BINARY_NAME)$(NC)"

# Build release binaries for all supported platforms
release-all:
	@echo -e "$(BLUE)Building release binaries for all platforms...$(NC)"
	@mkdir -p dist
	
	# Linux AMD64
	@echo -e "$(BLUE)Building for Linux AMD64...$(NC)"
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -trimpath -o dist/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	cd dist && tar -czf $(BINARY_NAME)-v$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	
	# Linux ARM64
	@echo -e "$(BLUE)Building for Linux ARM64...$(NC)"
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -trimpath -o dist/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	cd dist && tar -czf $(BINARY_NAME)-v$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	
	# macOS AMD64
	@echo -e "$(BLUE)Building for macOS AMD64...$(NC)"
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -trimpath -o dist/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	cd dist && tar -czf $(BINARY_NAME)-v$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	
	# macOS ARM64 (Apple Silicon)
	@echo -e "$(BLUE)Building for macOS ARM64...$(NC)"
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -trimpath -o dist/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	cd dist && tar -czf $(BINARY_NAME)-v$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	
	@echo "Release binaries created in dist/ directory"
	@echo "Archives ready for GitHub release:"
	@ls -la dist/*.tar.gz

# Create checksums for release files
release-checksums:
	@echo "Generating checksums...$(NC)"
	@cd dist && sha256sum *.tar.gz > checksums.txt
	@echo "Checksums generated in dist/checksums.txt"

# Complete release preparation
release: clean release-all release-checksums
	@echo "Release v$(VERSION) prepared successfully!"
	@echo ""
	@echo "Upload these files to GitHub release:"
	@ls -la dist/
	@echo ""
	@echo "Install script download URL will be:"
	@echo "https://github.com/$(shell git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\/[^/.]*\).*/\1/')/releases/download/v$(VERSION)/"

# CI-friendly targets
ci-test:
	@echo -e "$(BLUE)Running CI tests...$(NC)"
	@echo -e "$(BLUE)Running unit tests with race detection...$(NC)"
	go test -race -timeout=5m ./pkg/... ./internal/...
	@echo -e "$(BLUE)Running integration tests...$(NC)"
	go test -v -timeout=2m ./test/integration/...
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

ci-build:
	@echo -e "$(BLUE)Running CI build verification...$(NC)"
	go build -o /tmp/$(BINARY_NAME) $(MAIN_PATH)
	/tmp/$(BINARY_NAME) --version
	@echo -e "$(GREEN)Build verification passed$(NC)"

ci-security:
	@echo -e "$(BLUE)Running security checks...$(NC)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
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
ci-full: deps ci-test ci-lint ci-build ci-security ci-cross-compile
	@echo -e "$(GREEN)All CI checks passed locally!$(NC)"

# Release management targets
pre-release-check:
	@echo -e "$(BLUE)Running pre-release checks...$(NC)"
	@echo "Current branch: $(shell git branch --show-current)"
	@echo "Latest commit: $(shell git log -1 --oneline)"
	@echo ""
	
	# Ensure working directory is clean
	@if ! git diff-index --quiet HEAD --; then \
		echo "$(RED)ERROR:$(NC) Working directory is not clean. Please commit or stash changes."; \
		exit 1; \
	fi
	
	# Run full CI suite
	make ci-full
	
	@echo ""
	@echo -e "$(GREEN)Pre-release checks passed!$(NC)"
	@echo "Ready to create release tag."

tag-release: pre-release-check
	@echo -e "$(BLUE)Creating release tag...$(NC)"
	@echo ""
	@read -p "Enter version (e.g., 1.0.0): " version; \
	if [[ ! "$$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-.*)?$$ ]]; then \
		echo "$(RED)ERROR:$(NC) Invalid version format. Use: 1.0.0 or 1.0.0-alpha.1"; \
		exit 1; \
	fi; \
	tag="v$$version"; \
	echo "Creating tag: $$tag"; \
	git tag -a "$$tag" -m "Release $$tag"; \
	echo ""; \
	echo "Tag created. Push with:"; \
	echo "  git push origin $$tag"; \
	echo ""; \
	echo "This will trigger automated release creation on GitHub."

push-release-tag:
	@echo "Pushing latest tag to trigger release...$(NC)"
	@latest_tag=$$(git tag --sort=-version:refname | head -1); \
	if [[ -z "$$latest_tag" ]]; then \
		echo "$(RED)ERROR:$(NC) No tags found. Create a tag first with 'make tag-release'"; \
		exit 1; \
	fi; \
	echo "Pushing tag: $$latest_tag"; \
	git push origin "$$latest_tag"; \
	echo ""; \
	echo -e "$(CYAN)Release triggered! Monitor progress at:$(NC)"; \
	echo "  https://github.com/$(shell git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\/[^/.]*\).*/\1/')/actions"

release-local: clean
	@echo -e "$(BLUE)Creating local release simulation...$(NC)"
	@version=$$(git describe --tags --always 2>/dev/null || echo "dev"); \
	echo "Building release for version: $$version"; \
	make release-all; \
	echo ""; \
	echo " Local release created in dist/"; \
	echo "This simulates what GitHub Actions will build."

check-release-status:
	@echo "Checking latest release status...$(NC)"
	@latest_tag=$$(git tag --sort=-version:refname | head -1); \
	if [[ -z "$$latest_tag" ]]; then \
		echo "No releases found"; \
		exit 0; \
	fi; \
	echo "Latest tag: $$latest_tag"; \
	echo ""; \
	echo "GitHub release:"; \
	if command -v gh >/dev/null 2>&1; then \
		gh release view "$$latest_tag" 2>/dev/null || echo "  Release not found on GitHub"; \
	else \
		echo " Install 'gh' CLI to check release status"; \
	fi; \
	echo ""; \
	echo "Install command:";

# Automated functional testing workflow
test-functional:
	@echo -e "$(BLUE)Running functional test workflow...$(NC)"
	@echo "Step 1: Building application..."
	@make build > /dev/null
	@echo -e "$(GREEN)✓$(NC) Build completed"
	
	@echo "Step 2: Creating test environment files..."
	@mkdir -p test_env_files_functional
	@echo "TEST=value" > test_env_files_functional/.env
	@echo "LOCAL=test" > test_env_files_functional/.env.local
	@echo "DEV=true" > test_env_files_functional/.env.development
	@echo "CUSTOM=value" > test_env_files_functional/.env.custom
	@echo "BACKUP=old" > test_env_files_functional/.env.backup
	@echo "NEW=format" > test_env_files_functional/.env.new_format
	@echo "IGNORED=value" > test_env_files_functional/regular.txt
	@echo -e "$(GREEN)✓$(NC) Test files created (6 .env files + 1 regular file)"
	
	@echo "Step 3: Backing up existing config..."
	@if [ -f ~/.goingenv.json ]; then \
		cp ~/.goingenv.json ~/.goingenv.json.test-backup; \
		echo -e "$(YELLOW)!$(NC) Existing config backed up"; \
	else \
		echo -e "$(GREEN)✓$(NC) No existing config to backup"; \
	fi
	
	@echo "Step 4: Testing all-inclusive pattern (no config)..."
	@rm -f ~/.goingenv.json
	@files_detected=$$(./goingenv status test_env_files_functional/ | grep -c "\.env"); \
	if [ "$$files_detected" -eq 6 ]; then \
		echo -e "$(GREEN)✓$(NC) All-inclusive pattern working ($$files_detected/6 files detected)"; \
	else \
		echo -e "$(RED)✗$(NC) All-inclusive pattern failed ($$files_detected/6 files detected)"; \
		exit 1; \
	fi
	
	@echo "Step 5: Testing exclusion patterns..."
	@echo '{"default_depth": 3, "env_patterns": ["\\\\.env.*"], "env_exclude_patterns": ["\\\\.env\\\\.backup$$"], "exclude_patterns": ["node_modules/", "\\\\.git/"], "max_file_size": 10485760}' > ~/.goingenv.json
	@files_detected=$$(./goingenv status test_env_files_functional/ | grep -c "\.env"); \
	if [ "$$files_detected" -eq 5 ]; then \
		echo -e "$(GREEN)✓$(NC) Exclusion patterns working ($$files_detected/5 files detected, .env.backup excluded)"; \
	else \
		echo -e "$(RED)✗$(NC) Exclusion patterns failed ($$files_detected/5 files detected)"; \
		exit 1; \
	fi
	
	@echo "Step 6: Testing pack/unpack functionality..."
	@echo "Step 6a: Initializing goingenv in test directory..."
	@cd test_env_files_functional && ../goingenv init > /dev/null 2>&1
	@echo -e "$(GREEN)✓$(NC) goingenv initialized in test directory"
	@cd test_env_files_functional && echo "test123" | ../goingenv pack --password-env TEST_PASSWORD -o functional-test.enc > /dev/null 2>&1 || TEST_PASSWORD="test123" ../goingenv pack --password-env TEST_PASSWORD -o functional-test.enc > /dev/null
	@if [ -f test_env_files_functional/.goingenv/functional-test.enc ]; then \
		echo -e "$(GREEN)✓$(NC) Pack functionality working"; \
	else \
		echo -e "$(RED)✗$(NC) Pack functionality failed"; \
		exit 1; \
	fi
	@mkdir -p test_env_files_functional/unpacked
	@cd test_env_files_functional && TEST_PASSWORD="test123" ../goingenv unpack -f .goingenv/functional-test.enc --password-env TEST_PASSWORD -t unpacked > /dev/null
	@unpacked_files=$$(find test_env_files_functional/unpacked -name ".env*" | wc -l); \
	if [ "$$unpacked_files" -eq 5 ]; then \
		echo -e "$(GREEN)✓$(NC) Unpack functionality working ($$unpacked_files files restored)"; \
	else \
		echo -e "$(RED)✗$(NC) Unpack functionality failed ($$unpacked_files files restored)"; \
		exit 1; \
	fi
	
	@echo "Step 7: Cleaning up..."
	@rm -rf test_env_files_functional
	@rm -f ~/.goingenv.json
	@if [ -f ~/.goingenv.json.test-backup ]; then \
		mv ~/.goingenv.json.test-backup ~/.goingenv.json; \
		echo -e "$(YELLOW)!$(NC) Original config restored"; \
	else \
		echo -e "$(GREEN)✓$(NC) Cleanup completed"; \
	fi
	
	@echo ""
	@echo -e "$(GREEN)🎉 All functional tests passed!$(NC)"
	@echo "✓ All-inclusive .env.* pattern detection"
	@echo "✓ Exclusion pattern functionality" 
	@echo "✓ Pack/unpack workflow"
	@echo "✓ Configuration management"

# Complete test suite including functional tests
test-complete: clean
	@echo -e "$(BLUE)Running complete test suite...$(NC)"
	@echo ""
	@make ci-test
	@echo ""
	@make test-functional
	@echo ""
	@echo -e "$(GREEN)🎉 Complete test suite passed!$(NC)"
	@echo "✓ Unit tests with race detection"
	@echo "✓ Integration tests" 
	@echo "✓ Functional workflow tests"

# Quick release commands for common scenarios
release-alpha: pre-release-check
	@echo -e "$(BLUE)Creating alpha release...$(NC)"
	@next_version=$$(git tag --sort=-version:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+-alpha\.[0-9]+$$' | head -1 | sed 's/v\([0-9]*\)\.\([0-9]*\)\.\([0-9]*\)-alpha\.\([0-9]*\)/\1.\2.\3-alpha.\4/' | awk -F'-alpha\.' '{print $$1 "-alpha." ($$2+1)}'); \
	if [[ -z "$$next_version" ]]; then \
		next_version="1.0.0-alpha.1"; \
	fi; \
	echo "Next alpha version: $$next_version"; \
	read -p "Proceed with v$$next_version? [Y/n]: " -r; \
	if [[ $$REPLY =~ ^[Nn]$$ ]]; then \
		echo "Cancelled"; \
		exit 1; \
	fi; \
	tag="v$$next_version"; \
	echo "Creating tag: $$tag"; \
	git tag -a "$$tag" -m "Release $$tag"; \
	echo "Tag created. Use 'make push-release-tag' to publish."

release-beta: pre-release-check
	@echo -e "$(BLUE)Creating beta release...$(NC)"
	@next_version=$$(git tag --sort=-version:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+-beta\.[0-9]+$$' | head -1 | sed 's/v\([0-9]*\)\.\([0-9]*\)\.\([0-9]*\)-beta\.\([0-9]*\)/\1.\2.\3-beta.\4/' | awk -F'-beta\.' '{print $$1 "-beta." ($$2+1)}'); \
	if [[ -z "$$next_version" ]]; then \
		next_version="1.0.0-beta.1"; \
	fi; \
	echo "Next beta version: $$next_version"; \
	read -p "Proceed with v$$next_version? [Y/n]: " -r; \
	if [[ $$REPLY =~ ^[Nn]$$ ]]; then \
		echo "Cancelled"; \
		exit 1; \
	fi; \
	tag="v$$next_version"; \
	echo "Creating tag: $$tag"; \
	git tag -a "$$tag" -m "Release $$tag"; \
	echo "Tag created. Use 'make push-release-tag' to publish."

release-stable: pre-release-check
	@echo -e "$(BLUE)Creating stable release...$(NC)"
	@echo "$(YELLOW)WARNING:$(NC)  This creates a production release!"
	@read -p "Enter stable version (e.g., 1.0.0): " version; \
	if [[ ! "$$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$$ ]]; then \
		echo "$(RED)ERROR:$(NC) Invalid stable version format. Use: 1.0.0"; \
		exit 1; \
	fi; \
	tag="v$$version"; \
	echo "Creating STABLE tag: $$tag"; \
	read -p "Are you sure? This will be marked as latest release [y/N]: " -r; \
	if [[ ! $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "Cancelled"; \
		exit 1; \
	fi; \
	git tag -a "$$tag" -m "Release $$tag"; \
	echo "Stable tag created. Use 'make push-release-tag' to publish."

# One-command release flows
quick-alpha: release-alpha push-release-tag
	@echo -e "$(GREEN)Alpha release published!$(NC)"

quick-beta: release-beta push-release-tag  
	@echo -e "$(GREEN)Beta release published!$(NC)"

quick-stable: release-stable push-release-tag
	@echo -e "$(GREEN)Stable release published!$(NC)"

# Release with custom version
release-version:
	@echo -e "$(BLUE)Creating custom version release...$(NC)"
	@read -p "Enter version (e.g., 1.0.0 or 1.0.0-rc.1): " version; \
	if [[ ! "$$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-.*)?$$ ]]; then \
		echo "$(RED)ERROR:$(NC) Invalid version format. Use: 1.0.0 or 1.0.0-alpha.1"; \
		exit 1; \
	fi; \
	echo "$$version" | make tag-release

# Clean build artifacts
clean:
	@printf "$(BLUE)Cleaning build artifacts...$(NC)\n"
	go clean
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-dev
	rm -f $(BINARY_UNIX)
	rm -f $(BINARY_WINDOWS)
	rm -f $(BINARY_DARWIN)
	rm -f $(BINARY_NAME)-linux-*
	rm -f $(BINARY_NAME)-darwin-*
	rm -f $(BINARY_NAME)-windows-*
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

test-coverage:
	@echo -e "$(BLUE)Running tests with coverage...$(NC)"
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@echo "📊 Coverage summary:"
	@go tool cover -func=coverage.out | tail -1

test-coverage-ci:
	@echo -e "$(BLUE)Running tests with coverage for CI...$(NC)"
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -func=coverage.out

test-watch:
	@echo -e "$(BLUE)Running tests in watch mode (requires entr)...$(NC)"
	@if command -v entr >/dev/null 2>&1; then \
		find . -name '*.go' | entr -c go test ./...; \
	else \
		echo "$(YELLOW)WARNING:$(NC)  entr not installed. Install with your package manager."; \
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

# Build for all platforms
build-all: build-linux build-darwin build-windows
	@echo "Multi-platform build completed$(NC)\""

# Build for Linux (multiple architectures)
build-linux:
	@echo -e "$(BLUE)Building for Linux...$(NC)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=linux GOARCH=386 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-386 $(MAIN_PATH)
	@echo "Linux builds completed$(NC)\""

# Build for macOS (multiple architectures)
build-darwin:
	@echo -e "$(BLUE)Building for macOS...$(NC)"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo "macOS builds completed$(NC)\""

# Build for Windows (multiple architectures)
build-windows:
	@echo -e "$(BLUE)Building for Windows...$(NC)"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-386.exe $(MAIN_PATH)
	@echo "Windows builds completed$(NC)\""

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

# Create release archives and checksums (legacy target)
release-legacy: clean build-all
	@echo -e "$(BLUE)Creating release archives...$(NC)"
	mkdir -p dist
	
	# Linux AMD64
	tar -czf dist/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 README.md
	
	# Linux ARM64
	tar -czf dist/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64 README.md
	
	# Linux 386
	tar -czf dist/$(BINARY_NAME)-$(VERSION)-linux-386.tar.gz $(BINARY_NAME)-linux-386 README.md
	
	# macOS AMD64
	tar -czf dist/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 README.md
	
	# macOS ARM64 (Apple Silicon)
	tar -czf dist/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 README.md
	
	# Windows AMD64
	zip -j dist/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe README.md
	
	# Windows 386
	zip -j dist/$(BINARY_NAME)-$(VERSION)-windows-386.zip $(BINARY_NAME)-windows-386.exe README.md
	
	@echo "Release archives created in dist/"

# Generate checksums for releases
checksums:
	@echo "Generating checksums...$(NC)"
	cd dist && sha256sum * > checksums.txt
	cd dist && md5sum * > checksums.md5
	@echo "Checksums generated"

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

# Development server (for TUI testing)
dev-server: dev
	@echo "Starting development server with file watching...$(NC)"
	@echo "Press Ctrl+C to stop"
	./$(BINARY_NAME)-dev

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
		gosec ./...; \
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
		echo "📚 Documentation server: http://localhost:6060/pkg/goingenv/"; \
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
	@echo " release-build  - Build optimized release version"
	@echo " build-all      - Build for all platforms"
	@echo " build-linux    - Build for Linux (amd64, arm64, 386)"
	@echo " build-darwin   - Build for macOS (amd64, arm64)"
	@echo " build-windows  - Build for Windows (amd64, 386)"
	@echo ""
	@echo "Development Commands:"
	@echo " clean          - Clean build artifacts"
	@echo " deps           - Install and update dependencies"
	@echo " fmt            - Format code"
	@echo " vet            - Vet code for issues"
	@echo " lint           - Run linter (requires golangci-lint)"
	@echo " check          - Run all checks (fmt, vet, lint, test)"
	@echo " check-full     - Run comprehensive checks including integration tests"
	@echo ""
	@echo "Test Commands:"
	@echo " test           - Run all tests"
	@echo " test-unit      - Run unit tests only"
	@echo " test-integration - Run integration tests only"
	@echo " test-functional - Run automated functional workflow tests"
	@echo " test-complete  - Run complete test suite (unit + integration + functional)"
	@echo " test-coverage  - Run tests with coverage report"
	@echo " test-coverage-ci - Run tests with coverage for CI"
	@echo " test-watch     - Run tests in watch mode (requires entr)"
	@echo " test-verbose   - Run tests with verbose output"
	@echo " test-bench     - Run benchmarks"
	@echo " test-clean     - Clean test artifacts"
	@echo " generate-mocks - Generate mock implementations"
	@echo " bench          - Run benchmarks"
	@echo ""
	@echo "Release Commands:"
	@echo " release        - Create release archives"
	@echo " checksums      - Generate checksums for releases"
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
	@echo " make ci-full                  # Run all CI checks locally"
	@echo " make tag-release              # Create and tag new release"
	@echo " make push-release-tag         # Push tag to trigger GitHub release"
	@echo " make quick-alpha              # Create and publish alpha release"
	@echo " make quick-beta               # Create and publish beta release"  
	@echo " make quick-stable             # Create and publish stable release"
	@echo " make run ARGS='pack'          # Run pack command (interactive password)"
	@echo " make demo-scenario            # Full demo with sample files"
	@echo ""
	@echo "Note: Pushing to main branch automatically creates stable releases."
	@echo "Use commit message flags: [major], [minor], [skip-release]"

# Phony targets
.PHONY: build dev release-build clean deps fmt vet lint test test-unit test-integration \
        test-functional test-complete test-coverage test-coverage-ci test-watch test-verbose test-bench test-clean \
        generate-mocks bench check check-full build-all build-linux build-darwin \
        build-windows install uninstall release release-all release-checksums checksums run run-pack run-unpack \
        run-list run-status demo clean-demo demo-scenario dev-server profile \
        profile-mem security-scan vuln-check docs stats help \
        ci-test ci-lint ci-build ci-security ci-cross-compile ci-full \
        pre-release-check tag-release push-release-tag release-local check-release-status \
        release-alpha release-beta release-stable quick-alpha quick-beta quick-stable release-version