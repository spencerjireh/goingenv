# GoingEnv Makefile

# Variables
BINARY_NAME=goingenv
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME).exe
BINARY_DARWIN=$(BINARY_NAME)_darwin
VERSION=1.0.0
BUILD_TIME=$(shell date +%FT%T%z)
LDFLAGS=-ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Default target
.DEFAULT_GOAL := build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Build for development with race detector
dev:
	@echo "Building development version with race detector..."
	go build -race -o $(BINARY_NAME)-dev .

# Clean build artifacts
clean:
	@echo "Cleaning..."
	go clean
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-dev
	rm -f $(BINARY_UNIX)
	rm -f $(BINARY_WINDOWS)
	rm -f $(BINARY_DARWIN)
	rm -f $(BINARY_NAME)-linux-*
	rm -f $(BINARY_NAME)-darwin-*
	rm -f $(BINARY_NAME)-windows-*

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...

# Run all checks
check: fmt vet lint test

# Build for all platforms
build-all: build-linux build-darwin build-windows

# Build for Linux
build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 .

# Build for macOS
build-darwin:
	@echo "Building for macOS..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .

# Build for Windows
build-windows:
	@echo "Building for Windows..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe .

# Install the binary globally
install: build
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) .

# Uninstall the binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	rm -f $(GOPATH)/bin/$(BINARY_NAME)

# Create release archives
release: build-all
	@echo "Creating release archives..."
	mkdir -p dist
	
	# Linux AMD64
	tar -czf dist/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 README.md
	
	# Linux ARM64
	tar -czf dist/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64 README.md
	
	# macOS AMD64
	tar -czf dist/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 README.md
	
	# macOS ARM64
	tar -czf dist/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 README.md
	
	# Windows AMD64
	zip -j dist/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe README.md

# Generate checksums for releases
checksums:
	@echo "Generating checksums..."
	cd dist && sha256sum * > checksums.txt

# Run the application in development mode
run:
	@echo "Running $(BINARY_NAME)..."
	go run . $(ARGS)

# Run with demo data
demo:
	@echo "Setting up demo environment..."
	mkdir -p demo
	echo "DATABASE_URL=postgres://localhost:5432/myapp" > demo/.env
	echo "API_KEY=demo-api-key-12345" > demo/.env.local
	echo "DEBUG=true" > demo/.env.development
	echo "NODE_ENV=production" > demo/.env.production
	cd demo && ../$(BINARY_NAME) $(ARGS)

# Clean demo data
clean-demo:
	@echo "Cleaning demo environment..."
	rm -rf demo

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  dev            - Build development version with race detector"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Install dependencies"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code (requires golangci-lint)"
	@echo "  vet            - Vet code"
	@echo "  check          - Run all checks (fmt, vet, lint, test)"
	@echo "  build-all      - Build for all platforms"
	@echo "  build-linux    - Build for Linux (amd64, arm64)"
	@echo "  build-darwin   - Build for macOS (amd64, arm64)"
	@echo "  build-windows  - Build for Windows (amd64)"
	@echo "  install        - Install binary globally"
	@echo "  uninstall      - Uninstall binary"
	@echo "  release        - Create release archives"
	@echo "  checksums      - Generate checksums for releases"
	@echo "  run            - Run the application (use ARGS= for arguments)"
	@echo "  demo           - Set up demo environment and run"
	@echo "  clean-demo     - Clean demo environment"
	@echo "  help           - Show this help message"

# Phony targets
.PHONY: build dev clean deps test test-coverage fmt lint vet check build-all build-linux build-darwin build-windows install uninstall release checksums run demo clean-demo help