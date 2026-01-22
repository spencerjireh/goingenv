# Hot-Reload Testing Guide

Complete guide for testing goingenv commands with Air hot-reload workflow.

## Table of Contents

- [Setup (One-Time)](#setup-one-time)
- [The Development Loop](#the-development-loop)
- [Testing Pack Command](#testing-pack-command)
- [Speed Up Testing](#speed-up-testing)
- [Real Example Workflow](#real-example-workflow)
- [Recommended Terminal Layout](#recommended-terminal-layout)

## Setup (One-Time)

### Terminal 1 - Start the Watcher

```bash
cd /Users/spencerjirehcebrian/Projects/goingenv
make watch
```

This stays running and auto-rebuilds to `./tmp/goingenv` whenever you save a `.go` file.

### Terminal 2 - Prepare Test Environment

```bash
cd /Users/spencerjirehcebrian/Projects/goingenv
make demo                    # Creates demo/project1, project2, etc. with .env files
cd demo/project1
../../goingenv init          # Initialize .goingenv directory
```

## The Development Loop

Now you're ready for rapid iteration:

```
┌─────────────────────────────────────────────┐
│ 1. Edit code in your editor                │
│    (e.g., internal/cli/pack.go)            │
└──────────────┬──────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────┐
│ 2. Save the file (Cmd+S / Ctrl+S)         │
└──────────────┬──────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────┐
│ 3. Terminal 1 shows:                       │
│    "Building..." → "Build finished"        │
└──────────────┬──────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────┐
│ 4. Terminal 2: Test immediately            │
│    ./../../tmp/goingenv pack ...           │
└──────────────┬──────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────┐
│ 5. See results, repeat from step 1        │
└─────────────────────────────────────────────┘
```

**Time per iteration: 3-5 seconds** (vs 10-15 seconds without hot-reload)

## Testing Pack Command

From `demo/project1/` directory:

```bash
# Basic pack test
TEST_PASSWORD="dev123" ../../tmp/goingenv pack --password-env TEST_PASSWORD

# Check what will be packed first
../../tmp/goingenv status

# Pack with custom output name
TEST_PASSWORD="dev123" ../../tmp/goingenv pack --password-env TEST_PASSWORD -o my-backup.enc

# Verify the archive was created
TEST_PASSWORD="dev123" ../../tmp/goingenv list -f .goingenv/my-backup.enc --password-env TEST_PASSWORD

# Test different scenarios
TEST_PASSWORD="dev123" ../../tmp/goingenv pack --password-env TEST_PASSWORD -d ../project2

# Test error cases
echo "WRONG_PASS" | ../../tmp/goingenv pack
```

## Testing Other Commands

```bash
# Test status command
../../tmp/goingenv status
../../tmp/goingenv status ../project2

# Test list command
TEST_PASSWORD="dev123" ../../tmp/goingenv list -f .goingenv/backup.enc --password-env TEST_PASSWORD

# Test unpack command
mkdir -p unpacked
TEST_PASSWORD="dev123" ../../tmp/goingenv unpack -f .goingenv/backup.enc --password-env TEST_PASSWORD -t unpacked

# Test init command
cd /tmp/test-project
/path/to/goingenv/tmp/goingenv init

# Test help and flags
../../tmp/goingenv --help
../../tmp/goingenv pack --help
```

## Speed Up Testing

### Method 1: Create Aliases

In Terminal 2, create shortcuts:

```bash
# Set these up once
alias tpack='TEST_PASSWORD="dev123" ../../tmp/goingenv pack --password-env TEST_PASSWORD'
alias tstatus='../../tmp/goingenv status'
alias tlist='TEST_PASSWORD="dev123" ../../tmp/goingenv list -f .goingenv/backup.enc --password-env TEST_PASSWORD'
alias tunpack='TEST_PASSWORD="dev123" ../../tmp/goingenv unpack -f .goingenv/backup.enc --password-env TEST_PASSWORD -t unpacked'

# Now you can just type:
tstatus      # Check files
tpack        # Run pack
tlist        # Verify archive
tunpack      # Extract files
```

### Method 2: Create a Test Script

Create `demo/project1/quick-test.sh`:

```bash
#!/bin/bash
set -e

echo "🔍 Checking files to pack..."
../../tmp/goingenv status

echo ""
echo "📦 Packing files..."
TEST_PASSWORD="dev123" ../../tmp/goingenv pack --password-env TEST_PASSWORD -o test.enc

echo ""
echo "✅ Listing packed files..."
TEST_PASSWORD="dev123" ../../tmp/goingenv list -f .goingenv/test.enc --password-env TEST_PASSWORD

echo ""
echo "📂 Unpacking to verify..."
rm -rf unpacked
mkdir unpacked
TEST_PASSWORD="dev123" ../../tmp/goingenv unpack -f .goingenv/test.enc --password-env TEST_PASSWORD -t unpacked

echo ""
echo "🔍 Comparing files..."
diff -r . unpacked/ --exclude=.goingenv --exclude=unpacked --exclude=quick-test.sh || true

echo ""
echo "✨ Test complete!"
```

Make it executable and run:

```bash
chmod +x quick-test.sh
./quick-test.sh
```

### Method 3: Full Workflow Test Script

Create `demo/test-full-workflow.sh`:

```bash
#!/bin/bash
set -e

GOINGENV="../../tmp/goingenv"
TEST_PASS="dev123"

echo "=== Testing Full Workflow ==="

# Test 1: Status
echo ""
echo "TEST 1: Status Command"
cd project1
$GOINGENV status
cd ..

# Test 2: Pack
echo ""
echo "TEST 2: Pack Command"
cd project1
TEST_PASSWORD=$TEST_PASS $GOINGENV pack --password-env TEST_PASSWORD -o workflow-test.enc
cd ..

# Test 3: List
echo ""
echo "TEST 3: List Command"
cd project1
TEST_PASSWORD=$TEST_PASS $GOINGENV list -f .goingenv/workflow-test.enc --password-env TEST_PASSWORD
cd ..

# Test 4: Unpack
echo ""
echo "TEST 4: Unpack Command"
cd project1
rm -rf workflow-unpacked
mkdir workflow-unpacked
TEST_PASSWORD=$TEST_PASS $GOINGENV unpack -f .goingenv/workflow-test.enc --password-env TEST_PASSWORD -t workflow-unpacked
echo "Files unpacked: $(find workflow-unpacked -type f | wc -l)"
cd ..

# Test 5: Init (in new directory)
echo ""
echo "TEST 5: Init Command"
mkdir -p /tmp/goingenv-test-init
cd /tmp/goingenv-test-init
$GOINGENV init
echo "Initialized: $(ls -la .goingenv)"
cd -

echo ""
echo "=== All Tests Passed! ==="
```

## Real Example Workflow

Let's say you want to add a progress bar to the pack command:

### Terminal 1 (Watcher)
```bash
make watch
# Shows: "Watching for changes..."
```

### Terminal 2 (Testing)
```bash
cd demo/project1

# Run once to see current behavior
TEST_PASSWORD="test" ../../tmp/goingenv pack --password-env TEST_PASSWORD
```

### Your Editor
```go
// Edit internal/cli/pack.go
// Add progress bar code:

func runPackCommand(cmd *cobra.Command, args []string) error {
    // ... existing code ...
    
    // Add progress bar
    fmt.Println("Packing files...")
    for i, file := range files {
        fmt.Printf("\rProgress: %d/%d", i+1, len(files))
    }
    fmt.Println()
    
    // ... rest of code ...
}

// Save (Cmd+S)
```

### Terminal 1 (Automatically)
```
Building...
Build finished in 1.2s
```

### Terminal 2 (You Run)
```bash
# Immediately test your changes
TEST_PASSWORD="test" ../../tmp/goingenv pack --password-env TEST_PASSWORD
# See the progress bar!

# Found a bug? Edit again and test immediately
```

**Repeat: Edit → Save → Test (3 seconds total!)**

## Workflow Comparison

### OLD WAY (No Hot-Reload)
```bash
vim internal/cli/pack.go     # Edit
# Save
make build                   # Wait 2-5 seconds
./goingenv pack             # Test
# Found a bug?
vim internal/cli/pack.go     # Edit again
make build                   # Wait again...
./goingenv pack             # Test again...
```
**Time per iteration: ~10-15 seconds**

### NEW WAY (With Air)
```bash
# Terminal 1 already running: make watch
vim internal/cli/pack.go     # Edit & Save
# Air rebuilds automatically (1-2 seconds)
./tmp/goingenv pack         # Test immediately
# Found a bug?
vim internal/cli/pack.go     # Edit & Save
# Air rebuilds automatically
./tmp/goingenv pack         # Test immediately
```
**Time per iteration: ~3-5 seconds**

## Recommended Terminal Layout

```
┌─────────────────────┬─────────────────────┐
│                     │                     │
│   Your Editor       │  Terminal 1         │
│   (VS Code/Vim)     │  make watch         │
│                     │  (shows builds)     │
│                     │                     │
│                     │  Building...        │
│                     │  Build finished!    │
│                     │                     │
├─────────────────────┴─────────────────────┤
│                                           │
│   Terminal 2                              │
│   cd demo/project1                        │
│   ../../tmp/goingenv pack ...             │
│   (run tests here)                        │
│                                           │
│   $ TEST_PASSWORD="dev123" ../../tmp/     │
│     goingenv pack --password-env TEST_... │
│   ✓ Packed 6 files                        │
│                                           │
└───────────────────────────────────────────┘
```

This layout lets you see:
- Your code (editor)
- Build status (Terminal 1)
- Test results (Terminal 2)

All at once!

## Debugging Tips

### Build Errors

If Terminal 1 shows build errors:

```bash
# Check detailed build log
cat tmp/build-errors.log

# Or run build manually to see full output
go build -o tmp/goingenv ./cmd/goingenv
```

### Runtime Errors

```bash
# Capture error output
../../tmp/goingenv pack 2>&1 | tee error.log

# Run with verbose flags (if your app supports it)
../../tmp/goingenv --verbose pack
```

### Test Data Issues

```bash
# Clean and recreate test environment
cd ../..
make clean-demo
make demo
cd demo/project1
../../goingenv init
```

## Key Points to Remember

1. **Terminal 1 = Watcher** - Start once, leave it running all day
2. **Terminal 2 = Tester** - Run commands here repeatedly
3. **Binary location** - Always use `./tmp/goingenv` (not `./goingenv`)
4. **Test data** - `make demo` creates persistent test files
5. **Speed** - Edit → Save → Test in seconds, not minutes
6. **Clean slate** - Run `make clean` if builds act strange

## Advanced Usage

### Test Multiple Commands in Sequence

```bash
# Create a comprehensive test
cd demo/project1

echo "=== Testing All Commands ===" && \
../../tmp/goingenv status && \
TEST_PASSWORD="test" ../../tmp/goingenv pack --password-env TEST_PASSWORD && \
TEST_PASSWORD="test" ../../tmp/goingenv list -f .goingenv/backup.enc --password-env TEST_PASSWORD && \
rm -rf test-unpack && \
mkdir test-unpack && \
TEST_PASSWORD="test" ../../tmp/goingenv unpack -f .goingenv/backup.enc --password-env TEST_PASSWORD -t test-unpack && \
echo "=== All Commands Successful ==="
```

### Test Error Conditions

```bash
# Test with wrong password
echo "wrongpass" | ../../tmp/goingenv pack

# Test with missing .goingenv
cd /tmp/no-goingenv
/path/to/goingenv/tmp/goingenv pack

# Test with invalid archive
../../tmp/goingenv list -f /dev/null
```

### Performance Testing

```bash
# Time the pack operation
time TEST_PASSWORD="test" ../../tmp/goingenv pack --password-env TEST_PASSWORD

# Test with large files
dd if=/dev/zero of=.env.large bs=1M count=10
../../tmp/goingenv status
time TEST_PASSWORD="test" ../../tmp/goingenv pack --password-env TEST_PASSWORD
```

## Troubleshooting

### Air Not Building

```bash
# Check if Air is watching
# Terminal 1 should show "watching..." status

# Restart Air
# Ctrl+C in Terminal 1, then:
make watch
```

### Wrong Binary Being Used

```bash
# Make sure you're using the tmp version
which goingenv              # Shows system install
ls -la tmp/goingenv         # Shows hot-reload version

# Always use: ../../tmp/goingenv
```

### Test Environment Broken

```bash
cd /Users/spencerjirehcebrian/Projects/goingenv
make clean-demo
make demo
cd demo/project1
../../goingenv init
```

## See Also

- [AIR_USAGE.md](AIR_USAGE.md) - Air configuration and setup
- [DEVELOPMENT.md](DEVELOPMENT.md) - Full development guide
- [Makefile](Makefile) - All available commands
