# Installation Guide

This guide provides detailed installation instructions for goingenv on Linux and macOS systems.

## Quick Installation

### Automatic Installation (Recommended)

**One-line install (Linux & macOS):**
```bash
curl -sSL https://raw.githubusercontent.com/spencerjirehcebrian/goingenv/main/install.sh | bash
```

**Install development version:**
```bash
curl -sSL https://raw.githubusercontent.com/spencerjirehcebrian/goingenv/develop/install.sh | bash
```

> **Branch Selection:**
> - `main` - Stable releases, recommended for general use
> - `develop` - Latest development features, may include unreleased functionality

**Or using wget:**
```bash
wget -qO- https://raw.githubusercontent.com/spencerjirehcebrian/goingenv/main/install.sh | bash
```

### Secure Installation (Download and Inspect)

For security-conscious users, download and inspect the script first:

```bash
# Download the installer
curl -sSL https://raw.githubusercontent.com/spencerjirehcebrian/goingenv/main/install.sh -o install.sh

# Review the script (recommended)
cat install.sh

# Make executable and run
chmod +x install.sh
./install.sh
```

## Installation Options

### Custom Version
```bash
# Install specific version
curl -sSL https://raw.githubusercontent.com/spencerjirehcebrian/goingenv/main/install.sh | bash -s -- --version v0.1.0-beta.11

# Or with local script
./install.sh --version v1.0.0
```

### Custom Installation Directory
```bash
# Install to custom directory
./install.sh --dir /opt/bin

# Install to user directory (no sudo required)
./install.sh --dir ~/.local/bin --no-sudo
```

### Non-Interactive Installation
```bash
# Skip all prompts (useful for automation)
./install.sh --yes

# Skip shell integration
./install.sh --skip-shell
```

### Environment Variables
```bash
# Set installation preferences via environment variables
export GOINGENV_VERSION=v1.0.0
export INSTALL_DIR=~/.local/bin
export NO_SUDO=1
export SKIP_SHELL_INTEGRATION=1
./install.sh
```

## Manual Installation

### Prerequisites
- curl or wget
- tar
- gzip

### Download Pre-built Binary

1. **Visit the releases page:**
   https://github.com/spencerjirehcebrian/goingenv/releases

2. **Download the appropriate binary for your platform:**

   **Linux x86_64:**
   ```bash
   curl -sSL https://github.com/spencerjirehcebrian/goingenv/releases/download/v1.0.0/goingenv-v1.0.0-linux-amd64.tar.gz -o goingenv.tar.gz
   ```

   **Linux ARM64:**
   ```bash
   curl -sSL https://github.com/spencerjirehcebrian/goingenv/releases/download/v1.0.0/goingenv-v1.0.0-linux-arm64.tar.gz -o goingenv.tar.gz
   ```

   **macOS Intel:**
   ```bash
   curl -sSL https://github.com/spencerjirehcebrian/goingenv/releases/download/v1.0.0/goingenv-v1.0.0-darwin-amd64.tar.gz -o goingenv.tar.gz
   ```

   **macOS Apple Silicon:**
   ```bash
   curl -sSL https://github.com/spencerjirehcebrian/goingenv/releases/download/v1.0.0/goingenv-v1.0.0-darwin-arm64.tar.gz -o goingenv.tar.gz
   ```

3. **Extract and install:**
   ```bash
   # Extract the archive
   tar -xzf goingenv.tar.gz
   
   # Make executable
   chmod +x goingenv-*
   
   # Move to PATH directory
   sudo mv goingenv-* /usr/local/bin/goingenv
   # Or for user installation:
   mkdir -p ~/.local/bin
   mv goingenv-* ~/.local/bin/goingenv
   ```

4. **Add to PATH (if using ~/.local/bin):**
   ```bash
   echo 'export PATH="$PATH:$HOME/.local/bin"' >> ~/.bashrc
   source ~/.bashrc
   ```

### Build from Source

**Prerequisites:**
- Go 1.21 or later
- Git

**Build steps:**
```bash
# Clone the repository
git clone https://github.com/spencerjirehcebrian/goingenv.git
cd goingenv

# Install dependencies
go mod tidy

# Build the application
make build

# Install globally (optional)
sudo cp goingenv /usr/local/bin/
# Or install for current user
mkdir -p ~/.local/bin
cp goingenv ~/.local/bin/
```

## Verification

After installation, verify that GoingEnv is working correctly:

```bash
# Check version
goingenv --version

# Test basic functionality
goingenv --help

# Check installation
which goingenv
```

### First-time Setup

Before using GoingEnv in a project, you must initialize it:

```bash
# Navigate to your project directory
cd /path/to/your/project

# Initialize GoingEnv (required first step)
goingenv init

# Verify initialization worked
goingenv status
```

This creates the necessary `.goingenv` directory structure in your project.

## Post-Installation Setup

### Create Configuration Directory
The installer automatically creates `~/.goingenv/`, but you can verify:
```bash
ls -la ~/.goingenv/
```

### Shell Integration
If the installer didn't add GoingEnv to your PATH, add it manually:

**Bash:**
```bash
echo 'export PATH="$PATH:$HOME/.local/bin"' >> ~/.bashrc
source ~/.bashrc
```

**Zsh:**
```bash
echo 'export PATH="$PATH:$HOME/.local/bin"' >> ~/.zshrc
source ~/.zshrc
```

**Fish:**
```bash
fish_add_path ~/.local/bin
```

## Troubleshooting

### Common Issues

**1. Permission Denied**
```bash
# If you get permission errors, try user installation:
./install.sh --no-sudo --dir ~/.local/bin
```

**2. Command Not Found**
```bash
# Check if the binary exists:
ls -la ~/.local/bin/goingenv
/usr/local/bin/goingenv

# Check PATH:
echo $PATH

# Manually add to PATH:
export PATH="$PATH:~/.local/bin"
```

**3. Download Failures**
```bash
# Check internet connection and try again
# Or download manually from GitHub releases
```

**4. Architecture Mismatch**
```bash
# Check your architecture:
uname -m

# Supported architectures:
# - x86_64 (Intel/AMD 64-bit)
# - arm64/aarch64 (ARM 64-bit)
```

### Debug Mode
Run the installer with debug output:
```bash
DEBUG=1 ./install.sh
```

### Getting Help
- **GitHub Issues:** https://github.com/spencerjirehcebrian/goingenv/issues
- **Documentation:** https://github.com/spencerjirehcebrian/goingenv
- **Discussions:** https://github.com/spencerjirehcebrian/goingenv/discussions

## Uninstallation

### Using the Installer
```bash
# Download and run uninstaller
curl -sSL https://raw.githubusercontent.com/spencerjirehcebrian/goingenv/main/install.sh | bash -s -- --uninstall

# Or with local script
./install.sh --uninstall
```

### Manual Removal
```bash
# Remove binary
sudo rm -f /usr/local/bin/goingenv
rm -f ~/.local/bin/goingenv

# Remove configuration (optional)
rm -rf ~/.goingenv

# Remove from shell profile (manual)
# Edit ~/.bashrc, ~/.zshrc, etc. and remove GoingEnv PATH entries
```

## Updating

### Using the Installer
```bash
# Install latest version (will overwrite existing)
./install.sh

# Install specific version
./install.sh --version v1.1.0
```

### Manual Update
1. Download new version from releases
2. Replace existing binary
3. Verify installation

## Platform-Specific Notes

### Linux
- Tested on Ubuntu, Debian, CentOS, Fedora, Alpine Linux
- Requires glibc (most distributions) or musl (Alpine)
- ARM64 support for Raspberry Pi and ARM servers

### macOS
- Supports macOS 10.15+ (Catalina and later)
- Universal support for Intel and Apple Silicon Macs
- No special permissions required for user installation

### Windows (Future)
Windows support is planned for future releases. Currently, you can:
- Use WSL (Windows Subsystem for Linux)
- Use Git Bash with Linux binaries
- Build from source with Go for Windows

## Security Considerations

### Installation Security
- Always download from official GitHub releases
- Verify checksums when available
- Review installer script before execution
- Use HTTPS URLs only

### Runtime Security
- GoingEnv stores encrypted archives in `~/.goingenv/`
- Debug logs (when enabled) are stored in `~/.goingenv/debug/`
- No network connections made during normal operation
- All encryption happens locally

## License

GoingEnv is open source software. See the LICENSE file for details.