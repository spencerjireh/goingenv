#!/bin/bash

# Test Release Script
# This script helps test the release process locally before pushing tags

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${GREEN}[TEST]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Function to simulate GitHub Actions environment
simulate_github_actions() {
    local version="$1"
    
    log "Simulating GitHub Actions release build..."
    
    # Set environment variables like GitHub Actions would
    export VERSION="$version"
    export BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    export GIT_COMMIT=$(git rev-parse --short HEAD)
    
    log "Environment:"
    echo "  VERSION: $VERSION"
    echo "  BUILD_TIME: $BUILD_TIME" 
    echo "  GIT_COMMIT: $GIT_COMMIT"
    echo ""
    
    # Clean and build
    make clean
    
    # Test CI first
    log "Running CI checks..."
    make ci-full
    
    # Build release artifacts
    log "Building release artifacts..."
    make release-all
    make release-checksums
    
    # Verify archives
    log "Verifying archives..."
    cd dist/
    
    echo "Created archives:"
    ls -la *.tar.gz
    echo ""
    
    echo "Checksums:"
    cat checksums.txt
    echo ""
    
    # Test archive extraction
    for archive in *.tar.gz; do
        log "Testing extraction of $archive..."
        temp_dir=$(mktemp -d)
        tar -xzf "$archive" -C "$temp_dir"
        
        binary=$(find "$temp_dir" -name "goingenv-*" -type f)
        if [[ -n "$binary" ]]; then
            echo "  ✅ Binary found: $(basename $binary)"
            echo "  Size: $(du -h $binary | cut -f1)"
        else
            error "  ❌ Binary not found in $archive"
            rm -rf "$temp_dir"
            return 1
        fi
        
        rm -rf "$temp_dir"
    done
    
    cd ..
    
    log "✅ All archives verified successfully"
}

# Function to test install script compatibility
test_install_script() {
    local version="$1"
    
    log "Testing install script compatibility..."
    
    # Check install script syntax
    if ! bash -n install.sh; then
        error "Install script has syntax errors"
        return 1
    fi
    
    log "✅ Install script syntax is valid"
    
    # Test help function
    if ! ./install.sh --help >/dev/null; then
        error "Install script help function failed"
        return 1
    fi
    
    log "✅ Install script help works"
    
    # Test dry run (this will fail because version doesn't exist yet)
    log "Testing install script dry run..."
    DEBUG=1 NO_SUDO=1 SKIP_SHELL_INTEGRATION=1 ./install.sh --version "v$version" 2>&1 | head -10
    
    log "✅ Install script dry run completed (expected download failure)"
}

# Function to check git state
check_git_state() {
    log "Checking git repository state..."
    
    # Check if working directory is clean
    if ! git diff-index --quiet HEAD --; then
        warn "Working directory has uncommitted changes:"
        git status --porcelain
        echo ""
        read -p "Continue anyway? [y/N]: " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Aborted by user"
            exit 1
        fi
    else
        log "✅ Working directory is clean"
    fi
    
    # Check current branch
    current_branch=$(git branch --show-current)
    log "Current branch: $current_branch"
    
    # Check if on main/develop
    if [[ "$current_branch" != "main" && "$current_branch" != "develop" ]]; then
        warn "Not on main or develop branch"
        read -p "Continue anyway? [y/N]: " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Aborted by user"
            exit 1
        fi
    fi
    
    # Check for unpushed commits
    if [[ $(git log @{u}.. --oneline | wc -l) -gt 0 ]]; then
        warn "There are unpushed commits:"
        git log @{u}.. --oneline
        echo ""
        read -p "Continue anyway? [y/N]: " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Aborted by user"
            exit 1
        fi
    else
        log "✅ All commits are pushed"
    fi
}

# Function to validate version format
validate_version() {
    local version="$1"
    
    if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-.*)?$ ]]; then
        error "Invalid version format: $version"
        echo "Expected format: 1.0.0 or 1.0.0-alpha.1"
        return 1
    fi
    
    # Check if tag already exists
    if git tag -l | grep -q "^v$version$"; then
        error "Tag v$version already exists"
        git tag -l | grep "^v$version$"
        return 1
    fi
    
    log "✅ Version format is valid: $version"
}

# Main function
main() {
    local version="$1"
    
    echo -e "${BLUE}GoingEnv Release Test${NC}"
    echo -e "${BLUE}===================${NC}"
    echo ""
    
    if [[ -z "$version" ]]; then
        read -p "Enter version to test (e.g., 1.0.0): " version
    fi
    
    # Validate inputs
    validate_version "$version"
    
    # Check git state
    check_git_state
    
    # Run simulation
    simulate_github_actions "$version"
    
    # Test install script
    test_install_script "$version"
    
    echo ""
    log "🎉 Release test completed successfully!"
    echo ""
    echo -e "${BLUE}Next steps:${NC}"
    echo "1. If everything looks good, create the real release:"
    echo "   make tag-release"
    echo "   make push-release-tag"
    echo ""
    echo "2. Monitor the GitHub Actions workflow:"
    echo "   https://github.com/$(git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\/[^/.]*\).*/\1/')/actions"
    echo ""
    echo "3. Test the install script once release is live:"
    echo "   curl -sSL https://raw.githubusercontent.com/$(git config --get remote.origin.url | sed 's/.*github.com[:/]\([^/]*\/[^/.]*\).*/\1/')/main/install.sh | bash -s -- --version v$version"
}

# Show help
if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    cat << EOF
GoingEnv Release Test Script

USAGE:
    $0 [VERSION]

DESCRIPTION:
    Tests the release process locally before creating actual releases.
    Simulates GitHub Actions build process and validates install script.

EXAMPLES:
    $0              # Interactive version input
    $0 1.0.0        # Test specific version
    $0 1.0.0-rc.1   # Test prerelease version

WHAT IT TESTS:
    - Git repository state
    - Version format validation
    - CI checks (tests, linting, security)
    - Cross-platform builds
    - Archive creation and extraction
    - Install script compatibility
    - Checksum generation

EOF
    exit 0
fi

# Run main function
main "$@"