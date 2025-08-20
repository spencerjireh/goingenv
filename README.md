# GoingEnv

**Secure Environment File Manager with Encryption**

GoingEnv is a CLI tool that scans, encrypts, and archives your `.env` files with AES-256 encryption. Perfect for securely backing up, transferring, and restoring environment configurations across development environments.

## Documentation

**[Documentation](https://spencerjirehcebrian.github.io/goingenv/)** - Installation guide, usage examples, and tutorials

> [!WARNING]
> Educational purposes only. Not audited for production use. Use at your own risk in sensitive environments.

## Key Features

- **Smart Scanning** - Auto-detects `.env`, `.env.local`, `.env.production`, etc.
- **AES-256 Encryption** - Military-grade security with PBKDF2 key derivation
- **Beautiful TUI** - Interactive terminal interface with real-time preview
- **Archive Management** - Compressed, encrypted archives with metadata
- **Integrity Checks** - SHA-256 checksums ensure data integrity
- **CLI & TUI Modes** - Perfect for both interactive use and automation
- **Cross-Platform** - Works on Linux, macOS (Intel & Apple Silicon)

## Quick Start

### Installation

**One-line installation (recommended):**

```bash
curl -sSL https://raw.githubusercontent.com/spencerjirehcebrian/goingenv/main/install.sh | bash
```

**Install latest development version:**

```bash
curl -sSL https://raw.githubusercontent.com/spencerjirehcebrian/goingenv/develop/install.sh | bash
```

**Install specific version:**

```bash
curl -sSL https://raw.githubusercontent.com/spencerjirehcebrian/goingenv/main/install.sh | bash -s -- --version v0.1.0-beta.11
```

**Manual installation:**

1. Download the appropriate binary from [releases](https://github.com/spencerjirehcebrian/goingenv/releases)
2. Extract and move to your PATH: `tar -xzf goingenv-*.tar.gz && mv goingenv /usr/local/bin/`

### Basic Usage

**Interactive mode (recommended for beginners):**

```bash
goingenv
```

**Command-line usage:**

```bash
# Check what files would be processed
goingenv status

# Create encrypted backup
goingenv pack -k "your-secure-password" -o backup.enc

# List archive contents
goingenv list -f backup.enc -k "your-password"

# Restore from backup
goingenv unpack -f backup.enc -k "your-password"
```

## Documentation

- **[Installation Guide](INSTALL.md)** - Detailed installation instructions and troubleshooting
- **[User Guide](USAGE.md)** - Complete usage examples and workflows
- **[Developer Guide](DEVELOPMENT.md)** - Building, testing, and contributing
- **[Security Guide](SECURITY.md)** - Security considerations and best practices

## Example Workflow

```bash
# 1. Install GoingEnv
curl -sSL https://raw.githubusercontent.com/spencerjirehcebrian/goingenv/main/install.sh | bash

# 2. Navigate to your project
cd /path/to/your/project

# 3. Check what would be archived
goingenv status

# 4. Create encrypted backup
goingenv pack -k "secure-password" -o project-backup.enc

# 5. Later, restore from backup
goingenv unpack -f project-backup.enc -k "secure-password"
```

## Common Commands

| Command              | Description                      |
| -------------------- | -------------------------------- |
| `goingenv`           | Launch interactive TUI           |
| `goingenv pack`      | Encrypt and archive env files    |
| `goingenv unpack`    | Decrypt and restore files        |
| `goingenv list`      | View archive contents            |
| `goingenv status`    | Show detected files and archives |
| `goingenv --verbose` | Enable debug logging             |

## Architecture

**Supported Platforms:**

- Linux (x86_64, ARM64)
- macOS (Intel, Apple Silicon)

**File Patterns Detected:**

- `.env`, `.env.local`, `.env.production`
- `.env.development`, `.env.staging`, `.env.test`
- Custom patterns via configuration

## Contributing

We welcome contributions! Please see our [Development Guide](DEVELOPMENT.md) for details on:

- Setting up the development environment
- Running tests
- Submitting pull requests
- Code style guidelines

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Links

- **GitHub:** https://github.com/spencerjirehcebrian/goingenv
- **Issues:** https://github.com/spencerjirehcebrian/goingenv/issues
- **Releases:** https://github.com/spencerjirehcebrian/goingenv/releases

---

**Star this repo if GoingEnv helps secure your environment files!**
