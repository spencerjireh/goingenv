# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

goingenv is a Go CLI tool for managing environment files with AES-256-GCM encryption. It features both a CLI (Cobra) and an interactive TUI (Bubbletea) interface. Designed for small teams to securely share encrypted .env files without third-party services.

## Build and Development Commands

```bash
# First-time setup: installs the toolchain pinned in mise.toml
make bootstrap

# Build
make build              # Build for current platform
make dev                # Build with race detector

# Testing
make ci-test            # Everything CI runs (unit, integration, cli, e2e, install)
make test-unit          # Unit tests only
make test-cli           # CLI tests (spawns the built binary)
make test-e2e           # End-to-end tests
make test-install       # install.sh checksum verification tests
go test -v ./pkg/... ./internal/...  # Run specific package tests
go test -v -run TestName ./internal/crypto/...  # Run a single test by name

# Linting
make fmt && make lint   # Format and lint code
make ci-full            # Run all CI checks locally -- run this before pushing

# Release artifacts (goreleaser snapshot, publishes nothing)
make release-local

# TUI development (sandbox in /tmp/goingenv-sandbox)
make tui                # Build and launch TUI in the sandbox (password: test1234)
make tui-watch          # Same as tui but with hot-reload via air
make tui-clean          # Remove the sandbox directory
```

## Architecture

### Dependency Wiring and Two-Mode Operation

Entry point (`cmd/goingenv/main.go`) calls `cli.NewRootCommand()` which creates a Cobra root command. When invoked with no args, it launches the TUI via Bubbletea; with subcommands (`init`, `pack`, `unpack`, `list`, `status`), it uses CLI mode.

Both modes share the same service layer through the `types.App` struct, which acts as a dependency injection container:

```
ConfigManager.Load() → Config
  → Scanner = scanner.NewService(Config)
  → Crypto = crypto.NewService()
  → Archiver = archive.NewService(Crypto)
  → App{Scanner, Archiver, Crypto, ConfigManager}
```

### Key Interfaces (`pkg/types/types.go`)

All major services are defined as interfaces enabling mock-based testing:
- `Scanner` -- file detection via regex patterns with depth-limited `filepath.Walk`
- `Archiver` -- tar-based pack/unpack, delegates encryption to Cryptor
- `Cryptor` -- AES-256-GCM encrypt/decrypt (salt + nonce + ciphertext binary format)
- `ConfigManager` -- loads from `.goingenv/config.json` or `~/.goingenv.json` (first wins), always saves to `~/.goingenv.json`, checks project initialization

Mock implementations live in `pkg/types/mocks.go` (func-field based, not generated).

### TUI Structure (`internal/tui/`)

The TUI is a tabbed layout, not a screen state machine. `model.go` holds the
root Bubbletea model: it tracks the active tab (`TabID`) and routes messages.
`Model.Update` dispatches via two helpers -- `routeTo` sends a message to one
tab, `broadcast` fans out to all of them (spinner ticks). Async completion
messages are routed to their owning tab rather than the active one, so a pack
finishing while the user is on another tab still lands correctly.

Each tab is its own file implementing the `Tab` interface: `tab_status.go`,
`tab_pack.go`, `tab_unpack.go`, `tab_list.go`, `tab_settings.go`. The wizard
tabs are per-step state machines, and each step's handling lives in its own
method (`updateIdle`, `updateScanning`, `updateReview`, ...) rather than one
large switch.

Supporting files: `layout.go` (frame, tab bar, step indicator, empty states),
`overlay.go` (modals and toasts), `keys.go`, `styles.go`, `debug.go`.

Async operations (scan, pack, unpack, list) run in goroutines via `commands.go`
and return typed messages (`PackCompleteMsg`, `ErrorMsg`, ...) to the update
loop.

### Password Handling

`pkg/password/` handles password acquisition with priority: env variable > interactive prompt. Passwords are cleared from memory via `ClearPassword()` (zeros bytes). CLI commands obtain passwords through `getPass(envVar)` in `cli/helpers.go` which returns a cleanup function used with defer.

### Configuration

Three config locations:
- **Project directory**: `.goingenv/` (created by `goingenv init`), contains `.gitignore` and encrypted archives
- **Project-level config**: `.goingenv/config.json`, optional and committed. Written only by `goingenv init --project-config`. When present it wins.
- **User-level config**: `~/.goingenv.json` stores scan patterns, exclusions, max depth, max file size

`config.ResolveConfigPath()` picks the first that exists, project before user.
Precedence is whole-file, not per-key -- the two are never merged -- so a repo
that ships a project config fully determines its own scan behaviour. Keys absent
from the chosen file fall back to the built-in defaults (`Load` unmarshals over
`GetDefault()`), so a partial config is valid.

**Reads and writes target different files.** `Manager` holds a `loadPath` (the
resolved winner) and a `savePath` (always `~/.goingenv.json`). Nothing writes to
the committed project config implicitly -- `init --project-config` does it
explicitly through `NewManagerWithPath`. This is why `init` never disturbs a
repo's pinned rules.

`config.IsInitialized()` checks for `.goingenv/.gitignore` existence to determine if a project is set up -- deliberately not `config.json`, which is optional.

## Testing

- Unit tests alongside source (`*_test.go`)
- Integration tests in `test/integration/` -- drives the services in process, so
  this is the suite where `-race` is meaningful
- CLI tests in `test/cli/` and E2E tests in `test/e2e/` -- both spawn the
  compiled binary as a subprocess, so `-race` would only instrument the
  harness. They *do* contribute coverage: see the Coverage section below
- TUI tests in `internal/tui/` -- smoke tests driving the root model the way
  the runtime does. Deliberately no golden files or exact-layout assertions:
  those break on every lipgloss release rather than surviving one. They cover
  eleven of the twelve items in the manual checklist in `docs/development.md`;
  the twelfth (alt-screen restore) is terminal state and stays manual
- Install script tests in `test/install/test_install.sh` -- covers checksum
  verification against a local fixture release
- Website tests in `test/site/` -- parse `public/index.html` into a DOM and
  assert structure, accessibility and the claims the page makes. The site has
  no build step, so these are the only check between an edit and production.
  Every test fails loudly when it finds no subjects, rather than passing
  vacuously
- Shared helpers in `test/testutils/`:
  - `CreateTempGoingEnvDir()` -- required setup for archive tests (creates `.goingenv/` dir)
  - `CreateTempEnvFiles()` -- generates temp dir with sample .env files and excludable dirs
  - `BuildBinary()` -- compiles binary once via `sync.Once`, cached for test suite
  - `RunCLI()` / `RunCLIWithPassword()` / `RunCLIWithEnv()` -- execute binary and
    capture stdout/stderr/exit code
- **Every spawned binary runs with `HOME` pointed at a temp directory**
  (`testutils.TestHome`), so tests never read or write the real
  `~/.goingenv.json`. Pass an explicit `HOME` in the env map to override it --
  `test/cli/config_test.go` does this to test user-level config behaviour. Note
  a `.goingenv/config.json` in the test's working directory outranks that HOME.
  Anything driving `config.NewManager()` in process should use
  `config.NewManagerWithPath()` instead.
- Run `make ci-full` before pushing

### Coverage

`make test-coverage` merges two mechanisms:

1. In-process packages (`pkg`, `internal`, `test/integration`) produce a text
   profile from `go test -coverprofile`.
2. `test/cli` and `test/e2e` spawn a binary built with `go build -cover`, which
   writes binary-format data into `$GOINGENV_TEST_COVERDIR`.
   `go tool covdata textfmt` converts it and the two profiles are concatenated.

Both sides must use `-covermode=atomic` or the concatenation is rejected.

Coverage is **opt-in** via `GOINGENV_TEST_COVERDIR`. Unset -- which is the case
for every plain `go test` run -- the harness builds and runs exactly as it
would without coverage. Instrumentation and the output directory are gated on
the same variable on purpose, so an instrumented binary can never run without
somewhere to write.

`coverage-collect` fails the build if the spawned binary emitted nothing.
Do not remove that check: without it, dropping `-cover` silently halves the
reported number while CI stays green.

## Output formats

Every command takes a persistent `--format`:

| Format | Contract |
|---|---|
| `text` (default) | Human-readable. **Not stable, do not parse.** |
| `json` | One JSON object on stdout. Field names are a contract |
| `porcelain` | Tab-separated records, no header, stable column order |
| `csv` | `list` only, predating the flag |

`table` is accepted as an alias for `text`.

The rule that makes this work: **in a machine format every human message goes
to stderr and stdout carries only the payload.** `Output` holds two writers for
this -- `humanW` and `stdout` -- which are the same stream in text mode. Add
new human output via the existing `Output` methods and it lands on the right
stream automatically.

Errors are reported once, by `cli.ReportError`, in the requested format, always
on stderr. Commands must not print their own errors. An unrecognised `--format`
is rejected rather than falling back to text, since a silent fallback hands a
script unparseable output with a zero exit code.

Payload schemas live in `internal/cli/machine.go`. Renaming or removing a field
is a breaking change; adding one is not.

## Linting

golangci-lint is configured with:
- Max cyclomatic complexity: 15
- Security scanning via gosec. `.golangci.yml` is the source of truth for the
  exclusion list (G115, G117, G204, G304, G407, G703); the SARIF run in
  `ci.yml` passes the same list and must be kept in sync
- gofmt, goimports, errcheck, staticcheck enabled

Shell scripts are linted too: `make shellcheck` covers `install.sh` and
`test/install/test_install.sh`, and `make lint` / `make ci-lint` include it.
The file list is enumerated in `SHELL_SOURCES` rather than globbed, so the
gitignored `build/install.sh` copy is not double-reported.

`make link-check` runs lychee over `LINK_SOURCES` (the markdown files and
`public/index.html`). It is **not** in `make lint` or `make ci-lint`: it needs
the network, so it runs on a schedule instead.

Tool versions are pinned in `mise.toml`, so `make lint` runs exactly the
golangci-lint and shellcheck versions CI runs.

## CI

`ci.yml` runs `lint`, `test` (Ubuntu/macOS x minimum/stable Go -- the minimum
leg is pinned to the version `go.mod` declares, with `GOTOOLCHAIN=local` so it
genuinely tests that floor), `security`, `test-install-script` and `web` in
parallel, gated on a `changes` path filter, with a final `ci-ok` job that
always reports. **`ci-ok` is the job branch protection should require** -- the
others are skipped on docs-only changes and would never report.

The `changes` filter has two outputs. `code` excludes markdown, `docs/`,
`public/` and `assets/`; `web` selects `public/`, `assets/`, `test/site/` and
`pages.yml`. Both are forced true on a tag build. Adding a filter means adding
an output, forwarding it in the `decide` step **and** adding the job to
`ci-ok`'s `needs` -- miss the last and the job reports without gating anything.

`pages.yml` runs `make test-site` before uploading. It triggers on push to
`main` rather than through `ci-ok`, so without that step a push that bypassed a
PR would deploy unchecked.

`link-check.yml` runs `make link-check` weekly. It is not a pull request check
on purpose: a dead external link is unrelated to the change under review, and a
rate-limiting host must not block a merge.

`release.yml` calls `ci.yml` via `workflow_call`, so a tag cannot publish
untested code.

## Release Workflow

Releases are triggered by pushing a Git tag:

```bash
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3
```

This runs CI, then GoReleaser (`.goreleaser.yaml`), which builds all four
platforms, writes `checksums.txt`, generates SBOMs via syft, and publishes the
release along with a version-stamped `install.sh`. A build provenance
attestation is recorded so artifacts can be verified with
`gh attestation verify <archive> --repo spencerjireh/goingenv`.

- Stable versions: `v1.2.3` (marked as latest release)
- Prereleases: `v1.2.3-alpha.1`, `v1.2.3-rc.1` (detected from the hyphen)
- Local build test: `make release-local` (goreleaser snapshot, publishes nothing)

**`install.sh` depends on the exact shape of the release artifacts** and
verifies downloads against `checksums.txt`. Three things are a contract:
archives named `goingenv-<tag>-<os>-<arch>.tar.gz` (tag keeping its `v`), each
containing exactly one file named `goingenv`, plus a `checksums.txt` asset.
`.goreleaser.yaml` documents them and the release workflow asserts them before
publishing.
