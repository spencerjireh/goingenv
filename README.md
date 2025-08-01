# GoingEnv 📦

**Environment File Manager with Encryption**

GoingEnv is a powerful CLI tool designed to securely manage environment files across your projects. It scans, encrypts, and archives your `.env` files with AES-256 encryption, making it easy to backup, transfer, and restore your environment configurations safely.

## 🚀 Features

### Core Functionality
- **Smart Scanning**: Automatically detects common environment file patterns (`.env`, `.env.local`, `.env.production`, etc.)
- **Secure Encryption**: AES-256 encryption with PBKDF2 key derivation
- **Archive Management**: Create compressed, encrypted archives with metadata
- **Integrity Verification**: SHA-256 checksums ensure data integrity
- **Recursive Search**: Configurable depth scanning with exclude patterns

### Interactive Terminal UI
- **Modern TUI**: Beautiful terminal interface built with Bubbletea
- **Navigation**: Intuitive navigation with arrow keys or vim-style keys (h/j/k/l)
- **Real-time Preview**: Live preview of detected files during operations
- **Progress Indicators**: Visual progress bars for encryption/decryption
- **Secure Input**: Hidden password input for security

### Command Line Interface
- **Scriptable**: Full CLI support for automation and CI/CD
- **Flexible Options**: Comprehensive flag support for all operations
- **Multiple Archives**: Support for named archives and versioning

## 📥 Installation

### Prerequisites
- Go 1.21 or later

### Install from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/goingenv.git
cd goingenv

# Install dependencies
go mod tidy

# Build the application
go build -o goingenv

# Install globally (optional)
go install
```

### Quick Build

```bash
# Build for current platform
go build -o goingenv

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o goingenv-linux-amd64
GOOS=windows GOARCH=amd64 go build -o goingenv-windows-amd64.exe
GOOS=darwin GOARCH=amd64 go build -o goingenv-darwin-amd64
```

## 🎯 Quick Start

### Interactive Mode (Recommended for Beginners)

```bash
# Launch the interactive terminal interface
./goingenv
```

Navigate the menu with arrow keys:
- 📦 **Pack Environment Files** - Scan and encrypt your env files
- 📂 **Unpack Archive** - Decrypt and restore archived files
- 📋 **List Archive Contents** - Browse archive contents
- 📊 **Status** - View current directory and available archives
- ⚙️ **Settings** - Configure default options
- ❓ **Help** - View documentation and examples

### Command Line Mode (For Automation)

```bash
# Pack environment files with password
./goingenv pack -k "your-secure-password"

# Pack from specific directory
./goingenv pack -k "password" -d /path/to/project

# Pack with custom output name
./goingenv pack -k "password" -o backup-2024.enc

# Unpack files (will prompt for password)
./goingenv unpack

# Unpack specific archive
./goingenv unpack -f .goingenv/backup-2024.enc -k "password"

# Unpack with overwrite protection
./goingenv unpack -f archive.enc -k "password" --backup

# List archive contents
./goingenv list -f .goingenv/archive.enc -k "password"

# Show status and available archives
./goingenv status
```

## 📖 Detailed Usage

### Environment File Detection

EnvCase automatically detects these environment file patterns:
- `.env`
- `.env.local`
- `.env.development`
- `.env.production`
- `.env.staging`
- `.env.test`
- `.env.example`

### Directory Structure

```
your-project/
├── .env
├── .env.local
├── .env.production
├── src/
│   └── config/
│       └── .env.development
└── .goingenv/
    ├── .gitignore              # Auto-generated
    ├── archive-20240801.enc    # Default archive
    └── backup-prod.enc         # Custom named archive
```

### Encryption Details

- **Algorithm**: AES-256-GCM
- **Key Derivation**: PBKDF2 with 100,000 iterations
- **Salt**: 32-byte random salt per archive
- **Nonce**: 12-byte random nonce per encryption
- **Integrity**: Built-in authentication with GCM mode

### Archive Structure

Each encrypted archive contains:
- **Metadata**: JSON with creation time, file list, sizes, checksums
- **File Contents**: Original file data with preserved paths
- **Permissions**: File permissions and modification times
- **Integrity**: SHA-256 checksums for verification

## 🛠️ Advanced Configuration

### Exclude Patterns

The following directories are automatically excluded from scanning:
- `node_modules/`
- `.git/`
- `vendor/`
- `dist/`
- `build/`

### Customization

You can modify the default patterns by editing the `config` variable in the source code:

```go
var config = Config{
    DefaultDepth: 3,
    EnvPatterns: []string{
        `\.env$`,
        `\.env\.local$`,
        `\.env\.development$`,
        `\.env\.production$`,
        // Add your custom patterns here
    },
    ExcludePatterns: []string{
        `node_modules/`,
        `\.git/`,
        // Add your custom exclusions here
    },
}
```

## 🔒 Security Best Practices

### Password Security
- Use strong, unique passwords for each archive
- Consider using a password manager
- Never store passwords in scripts or environment variables
- Use interactive mode for manual operations

### File Handling
- Regularly verify archive integrity with `list` command
- Create multiple backups with different passwords
- Store archives in secure, backed-up locations
- Use `.gitignore` to prevent accidental commits

### Access Control
```bash
# Set restrictive permissions on archives
chmod 600 .goingenv/*.enc

# Secure the entire .goingenv directory
chmod 700 .goingenv
```

## 📋 Command Reference

### Global Flags
All commands support these options:
- `--help`: Show command help
- `--version`: Show version information

### pack
Pack and encrypt environment files.

```bash
goingenv pack [flags]

Flags:
  -d, --directory string   Directory to scan (default: current directory)
  -k, --key string        Encryption password
  -o, --output string     Output archive name (default: auto-generated)
```

### unpack
Unpack and decrypt archived files.

```bash
goingenv unpack [flags]

Flags:
      --backup         Create backups of existing files before overwriting
  -f, --file string    Archive file to unpack (default: most recent)
  -k, --key string    Decryption password
      --overwrite      Overwrite existing files without prompting
```

### list
List archive contents without extracting.

```bash
goingenv list [flags]

Flags:
  -f, --file string   Archive file to list (required)
  -k, --key string   Decryption password
```

### status
Show current directory status and available archives.

```bash
goingenv status
```

## 🎨 Interactive Interface

### Navigation
- **Arrow Keys**: Navigate menus and options
- **Vim Keys**: Use h/j/k/l for navigation (alternative)
- **Enter**: Select current option
- **Escape**: Go back to previous screen
- **q / Ctrl+C**: Quit application

### Screens
1. **Main Menu**: Choose primary actions
2. **File Scanner**: Real-time preview of detected files
3. **Password Input**: Secure password entry (hidden characters)
4. **Progress View**: Visual progress bars during operations
5. **Archive Browser**: Navigate and select archive files
6. **Content Viewer**: Browse archive contents
7. **Status Display**: Current directory and archive information

## 🔧 Troubleshooting

### Common Issues

**Archive not found**
```bash
# Check available archives
./goingenv status

# List .goingenv directory contents
ls -la .goingenv/
```

**Decryption failed**
- Verify the password is correct
- Check if the archive file is corrupted
- Ensure you're using the same password used for encryption

**Permission denied**
```bash
# Fix permissions on .goingenv directory
chmod 755 .goingenv
chmod 644 .goingenv/*.enc
```

**No environment files found**
- Check if files match the expected patterns
- Verify the scan directory is correct
- Increase scan depth if files are in subdirectories

### Debug Mode

Enable verbose output by modifying the source code or by checking file operations:

```bash
# Check what files would be detected
find . -name ".env*" -type f | head -20

# Test file patterns
grep -r "\.env" . --include="*.env*"
```

## 🚀 Development

### Project Structure
```
goingenv/
├── main.go              # Main application logic
├── go.mod              # Go module dependencies
├── go.sum              # Dependency checksums
├── README.md           # This documentation
└── .gitignore          # Git ignore patterns
```

### Key Components
- **CLI Framework**: Cobra for command-line interface
- **TUI Framework**: Bubbletea for interactive terminal UI
- **Styling**: Lipgloss for beautiful terminal styling
- **Encryption**: Go's crypto/aes and crypto/cipher packages
- **Archive Format**: Standard tar format with JSON metadata

### Building from Source
```bash
# Install dependencies
go mod tidy

# Run tests (if available)
go test ./...

# Build development version
go build -race -o goingenv-dev

# Build optimized release version
go build -ldflags="-s -w" -o goingenv
```

### Contributing
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License. See the LICENSE file for details.

## 🙏 Acknowledgments

- [Charm](https://charm.sh/) for the excellent Bubbletea and Lipgloss libraries
- [Cobra](https://cobra.dev/) for the powerful CLI framework
- The Go community for cryptographic libraries and best practices

## 📞 Support

- **Issues**: Report bugs and request features on GitHub
- **Documentation**: Check this README and inline help (`envcase --help`)
- **Security**: For security-related issues, please email privately

---

**Made with ❤️ and Go**

GoingEnv helps keep your environment variables secure while making them easy to manage across projects and teams.