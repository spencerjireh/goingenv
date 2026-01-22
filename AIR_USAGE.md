# Air Hot-Reload Quick Reference

Air is now configured for goingenv development. It automatically rebuilds your project when Go files change.

## Quick Start

```bash
# Install Air (one-time)
make deps  # or manually: go install github.com/air-verse/air@latest

# Start hot-reload development
make watch              # Just rebuild on changes
make dev-watch          # Rebuild and run (launches TUI)
make watch-run ARGS="status demo/"  # Rebuild and run with args
```

## Usage Examples

### 1. Basic Development (Recommended)
```bash
# Terminal 1: Watch and auto-rebuild
make watch

# Terminal 2: Test your changes
./tmp/goingenv status .
./tmp/goingenv pack
./tmp/goingenv --help
```

### 2. TUI Development
```bash
# Launches TUI automatically on each rebuild
make dev-watch
```

### 3. CLI Command Development
```bash
# Auto-rebuild and run specific command
make watch-run ARGS="status demo/"
make watch-run ARGS="list"
make watch-run ARGS="--help"
```

### 4. Direct Air Usage
```bash
# Use Air directly with custom config
air                          # Uses .air.toml
air -- status .              # Pass args to binary
air -d                       # Debug mode
air -c custom.air.toml       # Custom config
```

## Configuration

Configuration is in `.air.toml`. Key settings:

- **Build output**: `./tmp/goingenv`
- **Excluded dirs**: `dist, test, .goingenv, demo, tmp, vendor`
- **Watch extensions**: `go, tpl, tmpl, html`
- **Build delay**: 1000ms (prevents rapid rebuilds)

## Tips

1. **Faster iteration**: Use `./tmp/goingenv` directly instead of `make build`
2. **Test data**: Use `make demo` to create test environment files
3. **Clear artifacts**: `make clean` removes tmp/ directory
4. **Debug builds**: Edit `.air.toml` cmd to add `-race` or `-gcflags="all=-N -l"`

## Troubleshooting

**Air not found?**
```bash
# Ensure GOPATH/bin is in PATH
export PATH="$(go env GOPATH)/bin:$PATH"

# Or use full path
$(go env GOPATH)/bin/air
```

**Build errors?**
```bash
# Check build log
cat tmp/build-errors.log

# Run build manually to see full errors
go build -o tmp/goingenv ./cmd/goingenv
```

**Want to rebuild everything?**
```bash
make clean && make watch
```

## Comparison with Previous Workflow

| Old (entr) | New (Air) | Benefit |
|------------|-----------|---------|
| `find . -name '*.go' \| entr -c make dev` | `make watch` | Simpler command |
| Manual shell script | Built-in config | Easier to maintain |
| No build artifacts management | Automatic tmp/ cleanup | Cleaner workspace |
| Single command only | Multiple workflows | More flexible |

## See Also

- Full documentation: `DEVELOPMENT.md`
- Makefile targets: `make help`
- Air documentation: https://github.com/air-verse/air
