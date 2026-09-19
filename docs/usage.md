# Usage Guide

Reference for passwords, environments, the review tools, output formats,
scripting, file selection and configuration. The [README](../README.md) covers
installing and the commands.

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

## Passwords

Every command that opens an archive takes the password from the first of these
that supplies one:

1. `--password-stdin` -- the first line of standard input. Never prompts. Only
   that line is consumed, so `run --password-stdin` leaves the rest of stdin to
   the command it runs.
2. `--password-env VAR` -- a variable you name. Set but empty is an error.
3. `GOINGENV_PASSWORD_<ENV>` when `--env` is given, then `GOINGENV_PASSWORD`.
   The environment name is upper-cased and hyphens become underscores, so
   `prod-eu` and `prod_eu` both read `GOINGENV_PASSWORD_PROD_EU`.
4. An interactive prompt.

```bash
printf '%s\n' "$PW" | goingenv unpack --password-stdin
GOINGENV_PASSWORD=... goingenv pack
GOINGENV_PASSWORD_PROD=... goingenv unpack --env prod
```

At an interactive prompt, `pack` asks for the password twice and refuses a
mismatch, since a typo would produce an archive nobody can open. The other
sources are taken as given.

## Environments

`--env NAME` keeps separate archive streams under one `.goingenv/` directory:

```bash
goingenv pack --env prod        # writes .goingenv/prod-<timestamp>.enc
goingenv unpack --env prod      # picks the newest prod-*.enc
goingenv list --all --env prod  # only the prod archives
```

Without `--env`, archives are `archive-<timestamp>.enc` and the newest of
those is picked. A name is `[a-z0-9][a-z0-9_-]*`; `archive` is reserved.
An archive named explicitly with `-o` keeps that name and is never chosen as
"newest", so it does not take part in environment selection.

## Reviewing changes

An archive is an opaque blob in a pull request. Two tools make a change
reviewable without exposing a value.

### Manifest

`pack --manifest` writes `<archive>.manifest.json` beside the archive:

```json
{
  "created_at": "2026-09-19T12:00:00Z",
  "description": "...",
  "env": "prod",
  "archive": "prod-20260919-120000.enc",
  "files": [
    {"path": ".env", "sha256": "...", "keys": ["DATABASE_URL", "STRIPE_KEY"]}
  ]
}
```

It is meant to be committed, so a diff of it shows that `STRIPE_KEY` was added
to `.env` without showing what it was set to. It **publishes file paths and key
names**; if those are themselves sensitive, leave it off. `manifest: true` in
the config turns it on for every pack; `--no-manifest` turns it off for one.

### diff

`goingenv diff [FROM [TO]]` compares by key and never prints a value. The sides
follow git: `FROM` defaults to the newest archive for `--env`, `TO` to the env
files on disk.

```bash
goingenv diff                     # newest archive -> working tree: what would pack change?
goingenv diff old.enc             # old.enc -> working tree
goingenv diff old.enc new.enc     # old.enc -> new.enc
goingenv diff --exit-code         # exit 1 when anything differs, for CI
```

Porcelain rows carry a kind tag first, like `status`:

```
file<TAB>path<TAB>added|removed
key<TAB>path<TAB>KEY<TAB>added|removed|changed
```

A file present on one side only lists every key as added or removed; a file on
both sides lists only the keys that differ. Both archives must open with the
same password.

## run

`goingenv run [flags] -- COMMAND [ARGS...]` decrypts an archive in memory and
runs the command with its variables set. Nothing is written to disk.

```bash
goingenv run -- npm start
goingenv run --env prod -- ./deploy.sh --region eu
goingenv run --file apps/api/.env --file .env.local -- go test ./...
```

- The root `.env` is loaded by default; `--file` names entries by their path
  inside the archive and may be repeated, later files winning on a duplicate.
- Archive values override variables already in the environment.
- Flag parsing stops at `COMMAND`, so its own flags pass through.
- The command's exit status becomes goingenv's; 127 means it was not found.
- `--format` has no effect: stdout belongs to the command.

### Env file syntax

`run`, `diff` and the manifest parse env files with these rules and no
variable expansion: blank lines and `#` comments are skipped; a leading
`export ` is stripped; the key is everything before the first `=` and must
match `^[A-Za-z_][A-Za-z0-9_]*$`, otherwise the line is ignored; double-quoted
values may span lines and understand `\n \r \t \" \\`; single-quoted values
are literal and may span lines; an unquoted value ends at ` #`.

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
GOINGENV_PASSWORD=... goingenv pack

# Fail the job when the committed archive is behind the working tree
goingenv diff --exit-code --format porcelain

# Which keys does the newest prod archive define, without decrypting anything
jq -r '.files[] | .path as $p | .keys[] | "\($p)\t\(.)"' .goingenv/prod-*.enc.manifest.json
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
  "max_file_size": 10485760,
  "manifest": false
}
```

- `default_depth` -- how many directories deep to scan, 1 to 50
- `env_patterns` -- filename regexes to include
- `env_exclude_patterns` -- filename regexes to drop again
- `exclude_patterns` -- directory paths to skip entirely
- `max_file_size` -- bytes; larger files are ignored
- `manifest` -- write `<archive>.manifest.json` on every pack; `--manifest` and
  `--no-manifest` override it for one run

`goingenv status --verbose` prints the configuration actually in force,
including which file it came from.
