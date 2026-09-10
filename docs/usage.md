# Usage Guide

Reference for output formats, scripting, file selection and configuration. The
[README](../README.md) covers installing and the six commands.

## Install options

The install script takes flags after `-s --` when piped:

```bash
# A specific version
curl -sSL https://raw.githubusercontent.com/spencerjireh/goingenv/main/install.sh | bash -s -- --version <tag>

# From the release asset, pinning the installer itself to that version
curl -sSL https://github.com/spencerjireh/goingenv/releases/download/<tag>/install.sh | bash

# Remove old backups and duplicate installations while upgrading
curl -sSL https://raw.githubusercontent.com/spencerjireh/goingenv/main/install.sh | bash -s -- --cleanup-all
```

Run the installer with `--help` for the full list, including `--dir` to choose
an install location, `--yes` for non-interactive runs, `--no-sudo`,
`--skip-shell` to leave your shell profile alone, and `--uninstall`.

Re-running the installer upgrades in place.

## Output formats

Every command takes a persistent `--format`:

| Format | Contract |
|---|---|
| `text` (default) | Human-readable. **Not stable, do not parse.** `table` is accepted as an alias |
| `json` | One JSON object on stdout. Field names are a contract |
| `porcelain` | Tab-separated records, no header, stable column order |
| `csv` | `list` only, predating the flag |

In `json` and `porcelain`, stdout carries only the payload and all human output
moves to stderr, so the result pipes cleanly.

`json` field names and `porcelain` column order are a contract. `text` is the
default and is **not** stable -- do not parse it. Errors go to stderr, as a JSON
object under `--format json`, and the exit code remains the primary signal.

## Scripting

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

## Which files get picked up

Patterns are **regular expressions matched against the base filename**, not
globs, and they are unanchored. The default is a single pattern, `\.env.*`, so
anything containing `.env` matches -- including `.env.backup`, `.env.example`
and `myapp.environment`. Scanning is depth-limited, 10 directories by default.

Use `--env-exclude` to keep a file out by name:

```bash
goingenv pack --env-exclude '\.example$'
```

`--exclude` is a different thing: it matches directory paths, so it can skip a
whole tree but never an individual file.

## Configuration

Settings are read from the first of these that exists:

| Location | Purpose |
|---|---|
| `.goingenv/config.json` | Project-local. Commit it to pin how this repository is scanned |
| `~/.goingenv.json` | Your personal default, for projects that do not pin one |
| built-in defaults | Used when neither file exists |

The first file found wins outright -- the two are not merged -- so checking out
a repository determines its file set regardless of personal configuration. Keys
omitted from a file fall back to the built-in default, so a config may set only
what it cares about.

`goingenv init --project-config` writes one for you. Saves always go to
`~/.goingenv.json`; nothing writes to the committed project file implicitly.

### Config file

```json
{
  "default_depth": 10,
  "env_patterns": ["\\.env.*"],
  "env_exclude_patterns": ["\\.env\\.example$"],
  "exclude_patterns": ["node_modules/", "\\.git/"],
  "max_file_size": 10485760
}
```

- `default_depth` -- how many directories deep to scan, 1 to 50
- `env_patterns` -- filename regexes to include
- `env_exclude_patterns` -- filename regexes to drop again
- `exclude_patterns` -- directory paths to skip entirely
- `max_file_size` -- bytes; larger files are ignored

`goingenv status --verbose` prints the configuration actually in force,
including which file it came from.
