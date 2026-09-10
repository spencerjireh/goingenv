<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo-full.svg">
    <source media="(prefers-color-scheme: light)" srcset="assets/logo-full-light.svg">
    <img alt="goingenv" src="assets/logo-full-light.svg" width="440">
  </picture>
</p>

<p align="center">
  <a href="https://github.com/spencerjireh/goingenv/actions/workflows/ci.yml"><img src="https://github.com/spencerjireh/goingenv/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/spencerjireh/goingenv/releases/latest"><img src="https://img.shields.io/github/v/release/spencerjireh/goingenv?color=22d3a7&label=release" alt="Release"></a>
  <a href="https://github.com/spencerjireh/goingenv/blob/main/LICENSE.md"><img src="https://img.shields.io/github/license/spencerjireh/goingenv?color=6b7a8f" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/spencerjireh/goingenv"><img src="https://goreportcard.com/badge/github.com/spencerjireh/goingenv" alt="Go Report Card"></a>
</p>

---

Bundle your `.env` files into one AES-256-GCM encrypted archive, commit it alongside your
code, and let teammates restore it with a shared password. A CLI and a terminal UI over the
same commands.

> [!WARNING]
> **Disclaimer** -- This project was developed with AI assistance and has not undergone a
> formal security audit. Perform your own assessment before using it with sensitive data.

## Install

```bash
curl -sSL https://raw.githubusercontent.com/spencerjireh/goingenv/main/install.sh | bash
```

Linux and macOS, x86_64 and ARM64. On Windows, use
[WSL](https://docs.microsoft.com/en-us/windows/wsl/install). The script verifies the
download against the release checksums and refuses to install if they do not match.

<details>
<summary>Other ways to install</summary>

Replace `<tag>` with one from the [releases page](https://github.com/spencerjireh/goingenv/releases).

```bash
# A specific version
curl -sSL https://raw.githubusercontent.com/spencerjireh/goingenv/main/install.sh | bash -s -- --version <tag>

# From the release asset, pinning the installer itself to that version
curl -sSL https://github.com/spencerjireh/goingenv/releases/download/<tag>/install.sh | bash

# Remove old backups and duplicate installations while upgrading
curl -sSL https://raw.githubusercontent.com/spencerjireh/goingenv/main/install.sh | bash -s -- --cleanup-all
```

**Manual:** download a binary from [releases](https://github.com/spencerjireh/goingenv/releases), extract it, and move it onto your `PATH`.

**From source:** needs Go 1.26 or newer. `go build ./cmd/goingenv`.

</details>

## Usage

```bash
goingenv init                # Create .goingenv/ in your project
goingenv                     # Launch the terminal UI
goingenv status              # See which files would be packed
goingenv pack                # Encrypt them into .goingenv/archive-<timestamp>.enc
goingenv unpack -f <archive> # Decrypt and restore
goingenv list -f <archive>   # Inspect an archive without extracting
```

```
 Your project                     .goingenv/
 ────────────                     ──────────
 .env                  pack
 .env.local           ─────>      archive-20260910-124213.enc
 .env.production       AES-256    (one encrypted file, safe to commit)

                       unpack
                      <─────      Teammate clones and restores
```

The password is never stored. Share it through a channel that is not the repository.

## Commands

| Command | Description |
|---|---|
| `goingenv` | Launch the terminal UI |
| `goingenv init` | Set up `.goingenv/` in the current project |
| `goingenv init --project-config` | Also write a committed `.goingenv/config.json` |
| `goingenv pack` | Encrypt env files into an archive |
| `goingenv unpack` | Decrypt an archive and restore files |
| `goingenv list` | Show an archive's contents |
| `goingenv status` | Show detected files and existing archives |

Every command takes `--format text` (default), `--format json` or `--format porcelain`.
`list` also accepts `csv`, and `table` as an alias for `text`.

## Scripting

In `json` and `porcelain`, stdout carries only the payload and all human output moves to
stderr, so the result pipes cleanly:

```bash
# Which env files would be packed, and how big are they?
goingenv status --format json | jq -r '.env_files[] | "\(.size)\t\(.path)"'

# Fail a CI job if no archive exists
test "$(goingenv status --format json | jq '.archives | length')" -gt 0

# Which config is in force -- the committed one, or the developer's?
goingenv status --format json | jq -r '.config.source'   # "project" or "user"

# Tab-separated, for cut and awk, no jq required
goingenv status --format porcelain | awk -F'\t' '$1 == "env" { print $2 }'

# Non-interactive password, for CI
GOINGENV_PASSWORD=... goingenv pack --password-env GOINGENV_PASSWORD
```

`json` field names and `porcelain` column order are a contract. `text` is the default and
is **not** stable -- do not parse it. Errors go to stderr, as a JSON object under
`--format json`, and the exit code remains the primary signal.

## Which files get picked up

Patterns are **regular expressions matched against the base filename**, not globs, and they
are unanchored. The default is a single pattern, `\.env.*`, so anything containing `.env`
matches -- including `.env.backup`, `.env.example` and `myapp.environment`. Scanning is
depth-limited, 10 directories by default.

Use `--env-exclude` to keep a file out by name:

```bash
goingenv pack --env-exclude '\.example$'
```

`--exclude` is a different thing: it matches directory paths, so it can skip a whole tree
but never an individual file.

### Configuration

Settings are read from the first of these that exists:

| Location | Purpose |
|---|---|
| `.goingenv/config.json` | Project-local. Commit it to pin how this repository is scanned |
| `~/.goingenv.json` | Your personal default, for projects that do not pin one |
| built-in defaults | Used when neither file exists |

The first file found wins outright -- the two are not merged -- so checking out a repository
determines its file set regardless of personal configuration. Keys omitted from a file fall
back to the built-in default, so a config may set only what it cares about.

`goingenv init --project-config` writes one for you. Saves always go to `~/.goingenv.json`;
nothing writes to the committed project file implicitly.

## Documentation

- [Security Guide](SECURITY.md) -- threat model, encryption details, verifying a download
- [Developer Guide](docs/development.md) -- building, testing, CI, contributing
- [Design System](docs/design.md) -- brand, TUI specs, CLI output design
- [Website](https://spencerjireh.github.io/goingenv/)

## License

[MIT](LICENSE.md)
