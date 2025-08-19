# GoingEnv 📦

**Secure Environment File Manager with Encryption**

GoingEnv is a CLI tool that scans, encrypts, and archives your `.env` files with AES-256 encryption. Perfect for securely backing up, transferring, and restoring environment configurations.

> **⚠️ WARNING:** Educational purposes only. Not audited for production use. Use at your own risk in sensitive environments.

## ✨ Key Features

- 🔍 **Smart Scanning** - Auto-detects `.env`, `.env.local`, `.env.production`, etc.
- 🔐 **AES-256 Encryption** - Military-grade security with PBKDF2 key derivation
- 🎨 **Beautiful TUI** - Interactive terminal interface with real-time preview
- 📦 **Archive Management** - Compressed, encrypted archives with metadata
- ✅ **Integrity Checks** - SHA-256 checksums ensure data integrity
- 🚀 **CLI & TUI Modes** - Perfect for both interactive use and automation

## 🚀 Quick Start

### Installation
```bash
# One-line install (Linux & macOS)
curl -sSL https://raw.githubusercontent.com/spencerjirehcebrian/goingenv/main/install.sh | bash
```

### Basic Usage
```bash
# Interactive mode (recommended for beginners)
goingenv

# Pack env files with encryption
goingenv pack -k "your-secure-password"

# Unpack archive
goingenv unpack -f backup.enc -k "your-password"

# View current status
goingenv status
```

## 📖 Documentation

- **[Installation Guide](INSTALL.md)** - Detailed installation instructions and troubleshooting
- **[User Guide](USAGE.md)** - Complete usage examples and workflows
- **[Developer Guide](DEVELOPMENT.md)** - Building, testing, and contributing
- **[Security Guide](SECURITY.md)** - Security considerations and best practices

## 💡 Example Workflow

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

## 🔧 Common Commands

| Command | Description |
|---------|-------------|
| `goingenv` | Launch interactive TUI |
| `goingenv pack` | Encrypt and archive env files |
| `goingenv unpack` | Decrypt and restore files |
| `goingenv list` | View archive contents |
| `goingenv status` | Show detected files and archives |
| `goingenv --verbose` | Enable debug logging |

## 🏗️ Architecture

**Supported Platforms:**
- Linux (x86_64, ARM64)
- macOS (Intel, Apple Silicon)

**File Patterns Detected:**
- `.env`, `.env.local`, `.env.production`
- `.env.development`, `.env.staging`, `.env.test`
- Custom patterns via configuration

## 🤝 Contributing

We welcome contributions! Please see our [Development Guide](DEVELOPMENT.md) for details on:
- Setting up the development environment
- Running tests
- Submitting pull requests
- Code style guidelines

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🔗 Links

- **GitHub:** https://github.com/spencerjirehcebrian/goingenv
- **Issues:** https://github.com/spencerjirehcebrian/goingenv/issues
- **Releases:** https://github.com/spencerjirehcebrian/goingenv/releases

---

⭐ **Star this repo if GoingEnv helps secure your environment files!**