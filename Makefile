# GoingEnv Makefile - Refactored Structure

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
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	@echo "Main path: $(MAIN_PATH)"
	go build $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Build completed: $(BINARY_NAME)"

# Build for development with race detector and debug info
dev:
	@echo "Building development version with race detector..."
	go build -race -gcflags="all=-N -l" -o $(BINARY_NAME)-dev $(MAIN_PATH)
	@echo "✅ Development build completed: $(BINARY_NAME)-dev"

# Build optimized release version
release-build:
	@echo "Building optimized release version..."
	go build $(LDFLAGS) -trimpath -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Release build completed: $(BINARY_NAME)"

# Build release binaries for all supported platforms
release-all:
	@echo "Building release binaries for all platforms..."
	@mkdir -p dist
	
	# Linux AMD64
	@echo "Building for Linux AMD64..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -trimpath -o dist/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	cd dist && tar -czf $(BINARY_NAME)-v$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	
	# Linux ARM64
	@echo "Building for Linux ARM64..."
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -trimpath -o dist/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	cd dist && tar -czf $(BINARY_NAME)-v$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	
	# macOS AMD64
	@echo "Building for macOS AMD64..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -trimpath -o dist/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	cd dist && tar -czf $(BINARY_NAME)-v$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	
	# macOS ARM64 (Apple Silicon)
	@echo "Building for macOS ARM64..."
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -trimpath -o dist/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	cd dist && tar -czf $(BINARY_NAME)-v$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	
	@echo "✅ Release binaries created in dist/ directory"
	@echo "Archives ready for GitHub release:"
	@ls -la dist/*.tar.gz

# Create checksums for release files
release-checksums:
	@echo "Generating checksums..."
	@cd dist && sha256sum *.tar.gz > checksums.txt
	@echo "✅ Checksums generated in dist/checksums.txt"

# Complete release preparation
release: clean release-all release-checksums
	@echo "🎉 Release v$(VERSION) prepared successfully!"
	@echo ""
	@echo "Upload these files to GitHub release:"
	@ls -la dist/
	@echo ""
	@echo "Install script download URL will be:"
	@echo "https://github.com/$(shell git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\/[^/.]*\).*/\1/')/releases/download/v$(VERSION)/"

# CI-friendly targets
ci-test:
	@echo "Running CI tests..."
	go test -v -race -coverprofile=coverage.out ./pkg/... ./internal/...
	go test -v ./test/integration/...
	@echo "✅ All tests passed"

ci-lint:
	@echo "Running CI linting..."
	golangci-lint run ./...
	go vet ./...
	@echo "✅ Linting passed"

ci-build:
	@echo "Running CI build verification..."
	go build -o /tmp/$(BINARY_NAME) $(MAIN_PATH)
	/tmp/$(BINARY_NAME) --version
	@echo "✅ Build verification passed"

ci-security:
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "⚠️  gosec not installed, skipping security scan"; \
	fi
	@echo "✅ Security checks completed"

ci-cross-compile:
	@echo "Testing cross-compilation..."
	GOOS=linux GOARCH=amd64 go build -o /tmp/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 go build -o /tmp/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 go build -o /tmp/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build -o /tmp/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo "✅ Cross-compilation successful"

# Run all CI checks locally
ci-full: deps ci-test ci-lint ci-build ci-security ci-cross-compile
	@echo "🎉 All CI checks passed locally!"

# Release management targets
pre-release-check:
	@echo "Running pre-release checks..."
	@echo "Current branch: $(shell git branch --show-current)"
	@echo "Latest commit: $(shell git log -1 --oneline)"
	@echo ""
	
	# Ensure working directory is clean
	@if ! git diff-index --quiet HEAD --; then \
		echo "❌ Working directory is not clean. Please commit or stash changes."; \
		exit 1; \
	fi
	
	# Run full CI suite
	make ci-full
	
	@echo ""
	@echo "✅ Pre-release checks passed!"
	@echo "Ready to create release tag."

tag-release: pre-release-check
	@echo "Creating release tag..."
	@echo ""
	@read -p "Enter version (e.g., 1.0.0): " version; \
	if [[ ! "$$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-.*)?$$ ]]; then \
		echo "❌ Invalid version format. Use: 1.0.0 or 1.0.0-alpha.1"; \
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
	@echo "Pushing latest tag to trigger release..."
	@latest_tag=$$(git tag --sort=-version:refname | head -1); \
	if [[ -z "$$latest_tag" ]]; then \
		echo "❌ No tags found. Create a tag first with 'make tag-release'"; \
		exit 1; \
	fi; \
	echo "Pushing tag: $$latest_tag"; \
	git push origin "$$latest_tag"; \
	echo ""; \
	echo "🚀 Release triggered! Monitor progress at:"; \
	echo "  https://github.com/$(shell git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\/[^/.]*\).*/\1/')/actions"

release-local: clean
	@echo "Creating local release simulation..."
	@version=$$(git describe --tags --always 2>/dev/null || echo "dev"); \
	echo "Building release for version: $$version"; \
	make release-all; \
	echo ""; \
	echo "✅ Local release created in dist/"; \
	echo "This simulates what GitHub Actions will build."

check-release-status:
	@echo "Checking latest release status..."
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
		echo "  Install 'gh' CLI to check release status"; \
	fi; \
	echo ""; \
	echo "Install command:"; \
	echo "  curl -sSL https://raw.githubusercontent.com/$(shell git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\/[^/.]*\).*/\1/')/main/install.sh | bash -s -- --version $$latest_tag"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
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
	@echo "✅ Clean completed"

# Install dependencies and tidy modules
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	go mod verify
	@echo "✅ Dependencies updated"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted"

# Vet code for issues
vet:
	@echo "Vetting code..."
	go vet ./...
	@echo "✅ Code vetted"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
		echo "✅ Linting completed"; \
	else \
		echo "⚠️  golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Test targets
test:
	@echo "Running all tests..."
	go test -v ./...
	@echo "✅ All tests completed"

test-unit:
	@echo "Running unit tests..."
	go test -v -short ./pkg/... ./internal/...
	@echo "✅ Unit tests completed"

test-integration:
	@echo "Running integration tests..."
	go test -v -run TestFull ./test/integration/...
	@echo "✅ Integration tests completed"

test-coverage:
	@echo "Running tests with coverage..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"
	@echo "📊 Coverage summary:"
	@go tool cover -func=coverage.out | tail -1

test-coverage-ci:
	@echo "Running tests with coverage for CI..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -func=coverage.out

test-watch:
	@echo "Running tests in watch mode (requires entr)..."
	@if command -v entr >/dev/null 2>&1; then \
		find . -name '*.go' | entr -c go test ./...; \
	else \
		echo "⚠️  entr not installed. Install with your package manager."; \
	fi

test-verbose:
	@echo "Running tests with verbose output..."
	go test -v -race ./... -args -test.v

test-bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...
	@echo "✅ Benchmarks completed"

test-clean:
	@echo "Cleaning test artifacts..."
	rm -f coverage.out coverage.html
	rm -rf test/tmp/*
	go clean -testcache
	@echo "✅ Test artifacts cleaned"

# Mock generation (if using gomock)
generate-mocks:
	@echo "Generating mocks..."
	@if command -v mockgen >/dev/null 2>&1; then \
		mockgen -source=pkg/types/types.go -destination=pkg/types/mocks_generated.go -package=types; \
		echo "✅ Mocks generated"; \
	else \
		echo "⚠️  mockgen not installed. Install with: go install github.com/golang/mock/mockgen@latest"; \
	fi

# Run benchmarks
bench: test-bench

# Run all checks (format, vet, lint, test)
check: fmt vet lint test
	@echo "✅ All checks passed"

# Run comprehensive checks including integration tests
check-full: fmt vet lint test-unit test-integration
	@echo "✅ All comprehensive checks passed"

# Build for all platforms
build-all: build-linux build-darwin build-windows
	@echo "✅ Multi-platform build completed"

# Build for Linux (multiple architectures)
build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=linux GOARCH=386 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-386 $(MAIN_PATH)
	@echo "✅ Linux builds completed"

# Build for macOS (multiple architectures)
build-darwin:
	@echo "Building for macOS..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo "✅ macOS builds completed"

# Build for Windows (multiple architectures)
build-windows:
	@echo "Building for Windows..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-386.exe $(MAIN_PATH)
	@echo "✅ Windows builds completed"

# Install the binary globally
install: build
	@echo "Installing $(BINARY_NAME) globally..."
	go install $(LDFLAGS) $(MAIN_PATH)
	@echo "✅ $(BINARY_NAME) installed globally"

# Uninstall the binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@GOBIN=$$(go env GOBIN); \
	if [ -z "$$GOBIN" ]; then GOBIN=$$(go env GOPATH)/bin; fi; \
	rm -f "$$GOBIN/$(BINARY_NAME)"
	@echo "✅ $(BINARY_NAME) uninstalled"

# Create release archives and checksums
release: clean build-all
	@echo "Creating release archives..."
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
	
	@echo "✅ Release archives created in dist/"

# Generate checksums for releases
checksums:
	@echo "Generating checksums..."
	cd dist && sha256sum * > checksums.txt
	cd dist && md5sum * > checksums.md5
	@echo "✅ Checksums generated"

# Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	go run $(MAIN_PATH) $(ARGS)

# Run with specific command
run-pack:
	@echo "Running pack command..."
	go run $(MAIN_PATH) pack $(ARGS)

run-unpack:
	@echo "Running unpack command..."
	go run $(MAIN_PATH) unpack $(ARGS)

run-list:
	@echo "Running list command..."
	go run $(MAIN_PATH) list $(ARGS)

run-status:
	@echo "Running status command..."
	go run $(MAIN_PATH) status $(ARGS)

# Development utilities

# Set up demo environment with sample files
demo:
	@echo "Setting up demo environment..."
	mkdir -p demo/project1 demo/project2/config demo/project3
	echo "DATABASE_URL=postgres://localhost:5432/myapp" > demo/project1/.env
	echo "API_KEY=demo-api-key-12345" > demo/project1/.env.local
	echo "DEBUG=true" > demo/project1/.env.development
	echo "NODE_ENV=production" > demo/project1/.env.production
	echo "REDIS_URL=redis://localhost:6379" > demo/project2/.env
	echo "SECRET_KEY=super-secret-key" > demo/project2/config/.env.staging
	echo "AWS_REGION=us-east-1" > demo/project3/.env.test
	@echo "✅ Demo environment created in demo/"

# Clean demo environment
clean-demo:
	@echo "Cleaning demo environment..."
	rm -rf demo
	@echo "✅ Demo environment cleaned"

# Run demo scenario
demo-scenario: demo build
	@echo "Running demo scenario..."
	cd demo/project1 && ../../$(BINARY_NAME) status
	cd demo/project1 && ../../$(BINARY_NAME) pack -k "demo123" -o demo-backup.enc
	cd demo/project1 && ../../$(BINARY_NAME) list -f .goingenv/demo-backup.enc -k "demo123"
	@echo "✅ Demo scenario completed"

# Development server (for TUI testing)
dev-server: dev
	@echo "Starting development server with file watching..."
	@echo "Press Ctrl+C to stop"
	./$(BINARY_NAME)-dev

# Profile the application
profile:
	@echo "Running with CPU profiling..."
	go run $(MAIN_PATH) pack -k "test" -cpuprofile=cpu.prof
	go tool pprof cpu.prof

# Memory profile
profile-mem:
	@echo "Running with memory profiling..."
	go run $(MAIN_PATH) pack -k "test" -memprofile=mem.prof
	go tool pprof mem.prof

# Security scan (requires gosec)
security-scan:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
		echo "✅ Security scan completed"; \
	else \
		echo "⚠️  gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Dependency vulnerability check
vuln-check:
	@echo "Checking for vulnerabilities..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
		echo "✅ Vulnerability check completed"; \
	else \
		echo "⚠️  govulncheck not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

# Generate documentation
docs:
	@echo "Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "📚 Documentation server: http://localhost:6060/pkg/goingenv/"; \
		godoc -http=:6060; \
	else \
		echo "⚠️  godoc not installed. Install with: go install golang.org/x/tools/cmd/godoc@latest"; \
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
	@echo "GoingEnv Build System"
	@echo "===================="
	@echo ""
	@echo "Build Commands:"
	@echo "  build          - Build binary for current platform"
	@echo "  dev            - Build development version with race detector"
	@echo "  release-build  - Build optimized release version"
	@echo "  build-all      - Build for all platforms"
	@echo "  build-linux    - Build for Linux (amd64, arm64, 386)"
	@echo "  build-darwin   - Build for macOS (amd64, arm64)"
	@echo "  build-windows  - Build for Windows (amd64, 386)"
	@echo ""
	@echo "Development Commands:"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Install and update dependencies"
	@echo "  fmt            - Format code"
	@echo "  vet            - Vet code for issues"
	@echo "  lint           - Run linter (requires golangci-lint)"
	@echo "  check          - Run all checks (fmt, vet, lint, test)"
	@echo "  check-full     - Run comprehensive checks including integration tests"
	@echo ""
	@echo "Test Commands:"
	@echo "  test           - Run all tests"
	@echo "  test-unit      - Run unit tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-coverage-ci - Run tests with coverage for CI"
	@echo "  test-watch     - Run tests in watch mode (requires entr)"
	@echo "  test-verbose   - Run tests with verbose output"
	@echo "  test-bench     - Run benchmarks"
	@echo "  test-clean     - Clean test artifacts"
	@echo "  generate-mocks - Generate mock implementations"
	@echo "  bench          - Run benchmarks"
	@echo ""
	@echo "Release Commands:"
	@echo "  release        - Create release archives"
	@echo "  checksums      - Generate checksums for releases"
	@echo "  install        - Install binary globally"
	@echo "  uninstall      - Uninstall binary"
	@echo ""
	@echo "Run Commands:"
	@echo "  run            - Run application (use ARGS= for arguments)"
	@echo "  run-pack       - Run pack command (use ARGS= for arguments)"
	@echo "  run-unpack     - Run unpack command"
	@echo "  run-list       - Run list command"
	@echo "  run-status     - Run status command"
	@echo ""
	@echo "Demo Commands:"
	@echo "  demo           - Set up demo environment"
	@echo "  clean-demo     - Clean demo environment"
	@echo "  demo-scenario  - Run complete demo scenario"
	@echo ""
	@echo "Analysis Commands:"
	@echo "  profile        - Run with CPU profiling"
	@echo "  profile-mem    - Run with memory profiling"
	@echo "  security-scan  - Run security scan (requires gosec)"
	@echo "  vuln-check     - Check for vulnerabilities (requires govulncheck)"
	@echo "  stats          - Show project statistics"
	@echo ""
	@echo "Documentation:"
	@echo "  docs           - Start documentation server"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build                    # Build for current platform"
	@echo "  make ci-full                  # Run all CI checks locally"
	@echo "  make tag-release              # Create and tag new release"
	@echo "  make push-release-tag         # Push tag to trigger GitHub release"
	@echo "  make run ARGS='pack -k pass'  # Run pack command"
	@echo "  make demo-scenario            # Full demo with sample files"

# Phony targets
.PHONY: build dev release-build clean deps fmt vet lint test test-unit test-integration \
        test-coverage test-coverage-ci test-watch test-verbose test-bench test-clean \
        generate-mocks bench check check-full build-all build-linux build-darwin \
        build-windows install uninstall release release-all release-checksums checksums run run-pack run-unpack \
        run-list run-status demo clean-demo demo-scenario dev-server profile \
        profile-mem security-scan vuln-check docs stats help \
        ci-test ci-lint ci-build ci-security ci-cross-compile ci-full \
        pre-release-check tag-release push-release-tag release-local check-release-status