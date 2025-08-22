# goingenv

**Share envs the easy way**

Tired of complex tools just to share .env files? goingenv is perfect for small teams, personal projects, and private repositories. Get you and your teammates sharing encrypted environments in seconds - no third parties, no setup headaches.

**Made by devs who want easier solutions.**

Strong encryption, zero config needed. Just encrypt, commit, share. Your envs are secure in seconds, not hours. Perfect for small-scale projects where simplicity matters more than enterprise features.

## Website

**[Website](https://spencerjirehcebrian.github.io/goingenv/)** - Installation guide, usage examples, and documentation

> [!WARNING]  
> goingenv hasn't undergone formal security audits, though it's been used successfully in production. Great for private repositories and internal team projects. Always exercise caution with sensitive data.

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

**First-time setup:**

```bash
# Initialize goingenv in your project directory
cd /path/to/your/project
goingenv init
```

**Interactive mode (recommended for beginners):**

```bash
goingenv
```

**Command-line usage:**

```bash
# Check what files would be processed
goingenv status

# Create encrypted backup (interactive password prompt)
goingenv pack

# Create backup with environment variable password
export MY_PASSWORD="your-secure-password"
goingenv pack --password-env MY_PASSWORD -o backup.enc
unset MY_PASSWORD

# List archive contents
goingenv list -f backup.enc --password-env MY_PASSWORD

# Restore from backup
goingenv unpack -f backup.enc --password-env MY_PASSWORD
```

## Documentation

- **[Installation Guide](INSTALL.md)** - Detailed installation instructions and troubleshooting
- **[User Guide](USAGE.md)** - Complete usage examples and workflows
- **[Developer Guide](DEVELOPMENT.md)** - Building, testing, and contributing
- **[Security Guide](SECURITY.md)** - Security considerations and best practices

## Example Workflow

```bash
# 1. Install goingenv
curl -sSL https://raw.githubusercontent.com/spencerjirehcebrian/goingenv/main/install.sh | bash

# 2. Navigate to your project
cd /path/to/your/project

# 3. Initialize goingenv (required first step)
goingenv init

# 4. Check what would be archived
goingenv status

# 5. Create encrypted backup (interactive password prompt)
goingenv pack -o project-backup.enc

# 6. Later, restore from backup
goingenv unpack -f project-backup.enc
```

## Common Commands

| Command              | Description                      |
| -------------------- | -------------------------------- |
| `goingenv init`      | Initialize goingenv in project   |
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

**Star this repo if goingenv helps secure your environment files!**
