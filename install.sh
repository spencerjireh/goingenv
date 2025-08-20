#!/bin/bash

# GoingEnv Installation Script
# This script installs GoingEnv on Linux and macOS systems
# Usage: curl -sSL https://raw.githubusercontent.com/spencerjirehcebrian/goingenv/main/install.sh | bash
# Or: wget -qO- https://raw.githubusercontent.com/spencerjirehcebrian/goingenv/main/install.sh | bash

set -e

# Configuration
REPO_OWNER="spencerjirehcebrian"
REPO_NAME="goingenv"
BINARY_NAME="goingenv"
GITHUB_REPO="https://github.com/${REPO_OWNER}/${REPO_NAME}"
GITHUB_API="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}"

# Default settings (can be overridden by environment variables)
DEFAULT_INSTALL_DIR="${HOME}/.local/bin"
SYSTEM_INSTALL_DIR="/usr/local/bin"
VERSION="${GOINGENV_VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-}"
SKIP_SHELL_INTEGRATION="${SKIP_SHELL_INTEGRATION:-0}"
NO_SUDO="${NO_SUDO:-0}"
YES="${YES:-0}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Disable colors if not a TTY
if [[ ! -t 1 ]]; then
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    PURPLE=''
    CYAN=''
    NC=''
fi

# Logging functions
log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

debug() {
    if [[ "${DEBUG:-0}" == "1" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $1"
    fi
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Detect OS and architecture
detect_platform() {
    local os arch

    # Detect OS
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        *)          error "Unsupported operating system: $(uname -s)"
                    exit 1 ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        arm64|aarch64)  arch="arm64" ;;
        armv7l)         arch="arm" ;;
        i386|i686)      arch="386" ;;
        *)              error "Unsupported architecture: $(uname -m)"
                        exit 1 ;;
    esac

    echo "${os}-${arch}"
}

# Check for required tools
check_dependencies() {
    local missing_deps=()

    if ! command_exists curl && ! command_exists wget; then
        missing_deps+=("curl or wget")
    fi

    if ! command_exists tar; then
        missing_deps+=("tar")
    fi

    if ! command_exists gzip; then
        missing_deps+=("gzip")
    fi

    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        error "Missing required dependencies: ${missing_deps[*]}"
        echo "Please install the missing dependencies and try again."
        exit 1
    fi
}

# Download file using curl or wget
download_file() {
    local url="$1"
    local output="$2"

    debug "Downloading $url to $output"

    if command_exists curl; then
        curl -sSL -o "$output" "$url"
    elif command_exists wget; then
        wget -q -O "$output" "$url"
    else
        error "Neither curl nor wget is available"
        exit 1
    fi
}

# Get the latest release version from GitHub API
get_latest_version() {
    local version_url="${GITHUB_API}/releases/latest"
    local version

    debug "Fetching latest version from: $version_url"

    if command_exists curl; then
        version=$(curl -sSL "$version_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command_exists wget; then
        version=$(wget -qO- "$version_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        error "Cannot fetch latest version: no curl or wget available"
        exit 1
    fi

    if [[ -z "$version" ]]; then
        error "Failed to fetch latest version"
        exit 1
    fi

    echo "$version"
}

# Verify checksum if available
verify_checksum() {
    local file="$1"
    local expected_checksum="$2"

    if [[ -z "$expected_checksum" ]]; then
        debug "No checksum provided, skipping verification"
        return 0
    fi

    if command_exists sha256sum; then
        local actual_checksum
        actual_checksum=$(sha256sum "$file" | cut -d' ' -f1)
        if [[ "$actual_checksum" == "$expected_checksum" ]]; then
            debug "Checksum verification passed"
            return 0
        else
            error "Checksum verification failed"
            error "Expected: $expected_checksum"
            error "Actual: $actual_checksum"
            return 1
        fi
    elif command_exists shasum; then
        local actual_checksum
        actual_checksum=$(shasum -a 256 "$file" | cut -d' ' -f1)
        if [[ "$actual_checksum" == "$expected_checksum" ]]; then
            debug "Checksum verification passed"
            return 0
        else
            error "Checksum verification failed"
            error "Expected: $expected_checksum"
            error "Actual: $actual_checksum"
            return 1
        fi
    else
        warn "No checksum utility available, skipping verification"
        return 0
    fi
}

# Determine installation directory
determine_install_dir() {
    local install_dir

    # Use custom install directory if specified
    if [[ -n "$INSTALL_DIR" ]]; then
        install_dir="$INSTALL_DIR"
    # Try system directory if not disabled and we can write to it
    elif [[ "$NO_SUDO" != "1" ]] && [[ -w "$SYSTEM_INSTALL_DIR" || $(id -u) -eq 0 ]]; then
        install_dir="$SYSTEM_INSTALL_DIR"
    # Fall back to user directory
    else
        install_dir="$DEFAULT_INSTALL_DIR"
    fi

    echo "$install_dir"
}

# Check if directory is in PATH
is_in_path() {
    local dir="$1"
    case ":$PATH:" in
        *":$dir:"*) return 0 ;;
        *) return 1 ;;
    esac
}

# Add directory to shell profile
add_to_path() {
    local dir="$1"
    local shell_profile

    # Determine shell profile file
    if [[ -n "$BASH_VERSION" ]]; then
        if [[ -f "$HOME/.bash_profile" ]]; then
            shell_profile="$HOME/.bash_profile"
        else
            shell_profile="$HOME/.bashrc"
        fi
    elif [[ -n "$ZSH_VERSION" ]]; then
        shell_profile="$HOME/.zshrc"
    elif [[ "$SHELL" == */fish ]]; then
        # Fish shell uses a different method
        if command_exists fish; then
            fish -c "set -U fish_user_paths $dir \$fish_user_paths"
            log "Added $dir to fish PATH"
            return 0
        fi
    else
        shell_profile="$HOME/.profile"
    fi

    # Add to profile if not already present
    if [[ -f "$shell_profile" ]] && grep -q "$dir" "$shell_profile"; then
        debug "$dir already in $shell_profile"
        return 0
    fi

    echo "" >> "$shell_profile"
    echo "# Added by GoingEnv installer" >> "$shell_profile"
    echo "export PATH=\"\$PATH:$dir\"" >> "$shell_profile"
    log "Added $dir to PATH in $shell_profile"
    warn "Please restart your shell or run: source $shell_profile"
}

# Check for existing installation
check_existing_installation() {
    local install_dir="$1"
    local binary_path="$install_dir/$BINARY_NAME"

    if [[ -f "$binary_path" ]]; then
        local current_version
        current_version=$("$binary_path" --version 2>/dev/null | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' || echo "unknown")
        
        if [[ "$YES" != "1" ]]; then
            echo -e "\n${YELLOW}Existing installation found:${NC}"
            echo "  Path: $binary_path"
            echo "  Version: $current_version"
            echo -e "  Target version: $VERSION\n"
            
            read -p "Do you want to overwrite the existing installation? [y/N]: " -r
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                log "Installation cancelled by user"
                exit 0
            fi
        else
            log "Overwriting existing installation at $binary_path"
        fi

        # Backup existing binary
        local backup_path="${binary_path}.backup.$(date +%Y%m%d_%H%M%S)"
        if cp "$binary_path" "$backup_path"; then
            log "Backed up existing binary to $backup_path"
        else
            warn "Failed to backup existing binary"
        fi
    fi
}

# Install the binary
install_binary() {
    local platform="$1"
    local version="$2"
    local install_dir="$3"
    
    local archive_name="${BINARY_NAME}-${version}-${platform}.tar.gz"
    local download_url="${GITHUB_REPO}/releases/download/${version}/${archive_name}"
    local temp_dir
    temp_dir=$(mktemp -d)
    local archive_path="$temp_dir/$archive_name"

    debug "Archive name: $archive_name"
    debug "Download URL: $download_url"
    debug "Temp directory: $temp_dir"

    # Ensure install directory exists
    if [[ ! -d "$install_dir" ]]; then
        log "Creating installation directory: $install_dir"
        if ! mkdir -p "$install_dir"; then
            error "Failed to create installation directory: $install_dir"
            if [[ "$install_dir" == "$SYSTEM_INSTALL_DIR" ]]; then
                warn "Try running with sudo or set INSTALL_DIR to a writable location"
            fi
            exit 1
        fi
    fi

    # Download the archive
    log "Downloading GoingEnv $version for $platform..."
    if ! download_file "$download_url" "$archive_path"; then
        error "Failed to download GoingEnv"
        error "URL: $download_url"
        rm -rf "$temp_dir"
        exit 1
    fi

    # Extract the archive
    log "Extracting archive..."
    if ! tar -xzf "$archive_path" -C "$temp_dir"; then
        error "Failed to extract archive"
        rm -rf "$temp_dir"
        exit 1
    fi

    # Find the binary in the extracted files
    local binary_path
    binary_path=$(find "$temp_dir" -name "$BINARY_NAME" -type f | head -1)
    
    if [[ -z "$binary_path" ]]; then
        error "Binary not found in archive"
        rm -rf "$temp_dir"
        exit 1
    fi

    # Install the binary
    log "Installing GoingEnv to $install_dir..."
    if ! cp "$binary_path" "$install_dir/$BINARY_NAME"; then
        error "Failed to install binary to $install_dir"
        if [[ "$install_dir" == "$SYSTEM_INSTALL_DIR" ]]; then
            warn "Try running with sudo: sudo $0"
        fi
        rm -rf "$temp_dir"
        exit 1
    fi

    # Make binary executable
    chmod +x "$install_dir/$BINARY_NAME"

    # Cleanup
    rm -rf "$temp_dir"

    log "GoingEnv $version installed successfully to $install_dir/$BINARY_NAME"
}

# Setup shell integration
setup_shell_integration() {
    local install_dir="$1"

    if [[ "$SKIP_SHELL_INTEGRATION" == "1" ]]; then
        debug "Skipping shell integration"
        return 0
    fi

    # Check if install directory is in PATH
    if ! is_in_path "$install_dir"; then
        if [[ "$YES" == "1" ]]; then
            add_to_path "$install_dir"
        else
            echo -e "\n${YELLOW}The installation directory is not in your PATH:${NC}"
            echo "  Directory: $install_dir"
            echo "  Current PATH: $PATH"
            echo ""
            read -p "Do you want to add it to your PATH? [Y/n]: " -r
            if [[ ! $REPLY =~ ^[Nn]$ ]]; then
                add_to_path "$install_dir"
            else
                warn "You may need to add $install_dir to your PATH manually"
                warn "Or use the full path: $install_dir/$BINARY_NAME"
            fi
        fi
    else
        debug "$install_dir is already in PATH"
    fi

    # Setup GoingEnv directory
    local goingenv_dir="$HOME/.goingenv"
    if [[ ! -d "$goingenv_dir" ]]; then
        log "Creating GoingEnv directory: $goingenv_dir"
        mkdir -p "$goingenv_dir"
    fi
}

# Verify installation
verify_installation() {
    local install_dir="$1"
    local binary_path="$install_dir/$BINARY_NAME"

    if [[ ! -f "$binary_path" ]]; then
        error "Installation verification failed: binary not found at $binary_path"
        exit 1
    fi

    if [[ ! -x "$binary_path" ]]; then
        error "Installation verification failed: binary is not executable"
        exit 1
    fi

    # Test binary execution
    local version_output
    if version_output=$("$binary_path" --version 2>&1); then
        log "Installation verified successfully"
        echo -e "${GREEN}Installed version:${NC} $version_output"
    else
        error "Installation verification failed: binary execution failed"
        error "Output: $version_output"
        exit 1
    fi
}

# Show usage instructions
show_usage_instructions() {
    local install_dir="$1"
    local binary_path="$install_dir/$BINARY_NAME"

    echo ""
    echo -e "${GREEN}🎉 GoingEnv installation completed successfully!${NC}"
    echo ""
    echo -e "${CYAN}Usage:${NC}"
    
    if is_in_path "$install_dir"; then
        echo "  $BINARY_NAME --help"
        echo "  $BINARY_NAME --verbose  # Interactive mode with debug logging"
        echo "  $BINARY_NAME status     # Show current status"
    else
        echo "  $binary_path --help"
        echo "  $binary_path --verbose  # Interactive mode with debug logging"
        echo "  $binary_path status     # Show current status"
    fi
    
    echo ""
    echo -e "${CYAN}Documentation:${NC}"
    echo "  GitHub: $GITHUB_REPO"
    echo "  Issues: $GITHUB_REPO/issues"
    echo ""
    echo -e "${CYAN}Configuration:${NC}"
    echo "  Config directory: ~/.goingenv/"
    echo "  Debug logs: ~/.goingenv/debug/ (when using --verbose)"
    echo ""
}

# Uninstall function
uninstall() {
    local binary_locations=(
        "/usr/local/bin/$BINARY_NAME"
        "$HOME/.local/bin/$BINARY_NAME"
        "$HOME/bin/$BINARY_NAME"
    )

    local found_installations=()
    
    # Find all installations
    for location in "${binary_locations[@]}"; do
        if [[ -f "$location" ]]; then
            found_installations+=("$location")
        fi
    done

    # Also check custom INSTALL_DIR if specified
    if [[ -n "$INSTALL_DIR" && -f "$INSTALL_DIR/$BINARY_NAME" ]]; then
        found_installations+=("$INSTALL_DIR/$BINARY_NAME")
    fi

    if [[ ${#found_installations[@]} -eq 0 ]]; then
        log "No GoingEnv installations found"
        exit 0
    fi

    echo -e "${YELLOW}Found GoingEnv installations:${NC}"
    for installation in "${found_installations[@]}"; do
        local version
        version=$("$installation" --version 2>/dev/null | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' || echo "unknown")
        echo "  $installation ($version)"
    done

    if [[ "$YES" != "1" ]]; then
        echo ""
        read -p "Do you want to remove all installations? [y/N]: " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log "Uninstall cancelled by user"
            exit 0
        fi
    fi

    # Remove installations
    for installation in "${found_installations[@]}"; do
        if rm -f "$installation"; then
            log "Removed: $installation"
        else
            error "Failed to remove: $installation"
        fi
    done

    # Ask about removing user data
    if [[ "$YES" != "1" && -d "$HOME/.goingenv" ]]; then
        echo ""
        read -p "Do you want to remove user data (~/.goingenv)? [y/N]: " -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            if rm -rf "$HOME/.goingenv"; then
                log "Removed user data directory"
            else
                error "Failed to remove user data directory"
            fi
        fi
    fi

    log "Uninstall completed"
}

# Show help
show_help() {
    cat << EOF
GoingEnv Installation Script

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --help              Show this help message
    --uninstall         Uninstall GoingEnv
    --version VERSION   Install specific version (default: latest)
    --dir PATH          Custom installation directory
    --yes               Skip interactive prompts
    --no-sudo           Don't attempt system-wide installation
    --skip-shell        Skip shell integration setup

ENVIRONMENT VARIABLES:
    GOINGENV_VERSION    Version to install (e.g., v1.0.0)
    INSTALL_DIR         Custom installation directory
    YES                 Skip prompts (1 to enable)
    NO_SUDO             Avoid system-wide installation (1 to enable)
    SKIP_SHELL_INTEGRATION  Skip PATH setup (1 to enable)
    DEBUG               Enable debug output (1 to enable)

EXAMPLES:
    # Install latest version
    $0

    # Install specific version
    $0 --version v1.0.0

    # Install to custom directory
    $0 --dir /opt/bin

    # Non-interactive installation
    $0 --yes

    # Uninstall
    $0 --uninstall

EOF
}

# Main installation function
main() {
    local platform install_dir

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help|-h)
                show_help
                exit 0
                ;;
            --uninstall)
                uninstall
                exit 0
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            --yes|-y)
                YES=1
                shift
                ;;
            --no-sudo)
                NO_SUDO=1
                shift
                ;;
            --skip-shell)
                SKIP_SHELL_INTEGRATION=1
                shift
                ;;
            *)
                error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done

    # Show header
    echo -e "${PURPLE}GoingEnv Installer${NC}"
    echo -e "${PURPLE}==================${NC}"
    echo ""

    # Check dependencies
    check_dependencies

    # Detect platform
    platform=$(detect_platform)
    log "Detected platform: $platform"

    # Get version to install
    if [[ "$VERSION" == "latest" ]]; then
        VERSION=$(get_latest_version)
        log "Latest version: $VERSION"
    else
        log "Installing version: $VERSION"
    fi

    # Determine installation directory
    install_dir=$(determine_install_dir)
    log "Installation directory: $install_dir"

    # Check for existing installation
    check_existing_installation "$install_dir"

    # Install the binary
    install_binary "$platform" "$VERSION" "$install_dir"

    # Setup shell integration
    setup_shell_integration "$install_dir"

    # Verify installation
    verify_installation "$install_dir"

    # Show usage instructions
    show_usage_instructions "$install_dir"
}

# Handle errors and cleanup
trap 'error "Installation failed"; exit 1' ERR

# Run main function
main "$@"