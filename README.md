<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo-full.svg">
    <source media="(prefers-color-scheme: light)" srcset="assets/logo-full-light.svg">
    <img alt="goingenv" src="assets/logo-full-light.svg" width="440">
  </picture>
</p>

<p align="center">
  <strong>Share envs the easy way</strong>
</p>

<p align="center">
  <a href="https://github.com/spencerjireh/goingenv/actions/workflows/ci.yml"><img src="https://github.com/spencerjireh/goingenv/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/spencerjireh/goingenv/releases/latest"><img src="https://img.shields.io/github/v/release/spencerjireh/goingenv?color=22d3a7&label=release" alt="Release"></a>
  <a href="https://github.com/spencerjireh/goingenv/blob/main/LICENSE"><img src="https://img.shields.io/github/license/spencerjireh/goingenv?color=6b7a8f" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/spencerjireh/goingenv"><img src="https://goreportcard.com/badge/github.com/spencerjireh/goingenv" alt="Go Report Card"></a>
</p>

---

Secure your `.env` files with AES-256-GCM encryption. No dependencies, no configuration. Pack, encrypt, and share environment files across your team through Git.

> [!WARNING]
> **Disclaimer** -- This project was developed with AI assistance and has not undergone a formal security audit. While it has been used in production environments, perform your own security assessment before using it with sensitive data.

## Features

| | |
|---|---|
| **Smart Scanning** | Auto-detects `.env`, `.env.local`, `.env.production`, and more |
| **AES-256-GCM** | Industry-standard encryption with PBKDF2 key derivation |
| **Interactive TUI** | Beautiful terminal interface with real-time preview |
| **CLI Mode** | Script-friendly commands for CI/CD and automation |
| **Integrity Checks** | SHA-256 checksums ensure data integrity |
| **Cross-Platform** | Linux, macOS (Intel & Apple Silicon), and Windows via WSL |

## Quick Start

### Install

```bash
curl -sSL https://raw.githubusercontent.com/spencerjireh/goingenv/main/install.sh | bash
```

<details>
<summary>More installation options</summary>

**Install a specific version:**

```bash
curl -sSL https://raw.githubusercontent.com/spencerjireh/goingenv/main/install.sh | bash -s -- --version v1.1.1
```

**Install from release asset (version-locked):**

```bash
curl -sSL https://github.com/spencerjireh/goingenv/releases/download/v1.1.1/install.sh | bash
```

**Upgrade and cleanup old installations:**

```bash
curl -sSL https://raw.githubusercontent.com/spencerjireh/goingenv/main/install.sh | bash -s -- --cleanup-all
```

**Manual:** Download the binary from [releases](https://github.com/spencerjireh/goingenv/releases), extract, and move to your PATH.

</details>

### Usage

```bash
# Initialize in your project
goingenv init

# Launch interactive TUI
goingenv

# Or use CLI commands directly
goingenv status              # See detected files
goingenv pack                # Encrypt env files (interactive password)
goingenv unpack -f backup    # Decrypt and restore
goingenv list -f backup      # View archive contents
```

## How It Works

```
 Your Project                    Encrypted Archive
 ─────────────                   ─────────────────
 .env                 pack
 .env.local          ─────>     envs.goingenv
 .env.production      AES-256    (single encrypted file)
                      
                      unpack
                     <─────      Share via Git
```

**1. Initialize** -- Run `goingenv init` to set up the `.goingenv/` directory in your project.

**2. Pack** -- Scans for environment files, bundles them into a tar archive, and encrypts with AES-256-GCM. You provide the password.

**3. Share** -- Commit the encrypted `.goingenv` archive to your repo. Share the password through a secure channel.

**4. Unpack** -- Team members run `goingenv unpack` with the shared password to restore the environment files.

## Commands

| Command | Description |
|---|---|
| `goingenv` | Launch interactive TUI |
| `goingenv init` | Initialize goingenv in project |
| `goingenv pack` | Encrypt and archive env files |
| `goingenv unpack` | Decrypt and restore files |
| `goingenv list` | View archive contents |
| `goingenv status` | Show detected files and archives |
| `goingenv --verbose` | Enable debug logging |

### Password via Environment Variable

```bash
export GOINGENV_PASSWORD="your-secure-password"
goingenv pack -o backup.enc --password-env GOINGENV_PASSWORD
goingenv unpack -f backup.enc --password-env GOINGENV_PASSWORD
unset GOINGENV_PASSWORD
```

## Supported Platforms

| Platform | Architecture |
|---|---|
| Linux | x86_64, ARM64 |
| macOS | Intel, Apple Silicon |
| Windows | via [WSL](https://docs.microsoft.com/en-us/windows/wsl/install) |

## File Patterns Detected

`.env`, `.env.local`, `.env.development`, `.env.staging`, `.env.production`, `.env.test`, and custom patterns via `~/.goingenv.json`.

## Documentation

- [Developer Guide](docs/development.md) -- Building, testing, CI/CD, and contributing
- [Design System](docs/design.md) -- Brand identity, TUI specs, and CLI output design
- [Security Guide](SECURITY.md) -- Security considerations and best practices
- [Website](https://spencerjireh.github.io/goingenv/) -- Installation guide and usage examples

## License

[MIT](LICENSE)
