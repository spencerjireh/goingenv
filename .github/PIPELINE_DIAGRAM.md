# Pipeline Architecture Diagram

## High-Level Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         GITHUB PUSH / PULL REQUEST                      │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          STAGE 1: VALIDATE                              │
│                    (Runs on all pushes & PRs)                           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │   Lint   │  │   Test   │  │ Security │  │  Build   │              │
│  │          │  │          │  │          │  │  Verify  │              │
│  │ • gofmt  │  │ • Unit   │  │ • gosec  │  │          │              │
│  │ • go vet │  │ • Integ  │  │ • vulns  │  │ • Build  │              │
│  │ • tidy   │  │ • Func   │  │ • SARIF  │  │ • Cross  │              │
│  │ • golint │  │ • Race   │  │          │  │   Comp   │              │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘              │
│       │             │             │             │                      │
│       └─────────────┴─────────────┴─────────────┘                      │
│                            │                                           │
│                            ▼                                           │
│                   ┌─────────────────┐                                  │
│                   │  Install Script │                                  │
│                   │      Test       │                                  │
│                   └────────┬────────┘                                  │
└────────────────────────────┼────────────────────────────────────────────┘
                             │
                             ▼
                    ┌────────────────┐
                    │  All Passed?   │
                    └────────┬───────┘
                             │
                    ┌────────┴────────┐
                    │                 │
                   YES               NO
                    │                 │
                    ▼                 ▼
         ┌──────────────────┐  ┌──────────┐
         │ Main Branch?     │  │  Failed  │
         └────────┬─────────┘  │  Status  │
                  │            └──────────┘
         ┌────────┴────────┐
         │                 │
        YES               NO
         │                 │
         ▼                 ▼
┌─────────────────┐  ┌──────────┐
│  STAGE 2: BUILD │  │ Success  │
└─────────────────┘  │  Status  │
         │           └──────────┘
         ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          STAGE 2: BUILD                                 │
│                      (Main branch only)                                 │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│                   ┌─────────────────────┐                              │
│                   │  Check for Release  │                              │
│                   │   [release] flag?   │                              │
│                   └─────────┬───────────┘                              │
│                             │                                           │
│                    ┌────────┴────────┐                                 │
│                    │                 │                                 │
│                   YES               NO                                 │
│                    │                 │                                 │
│                    ▼                 ▼                                 │
│         ┌──────────────────┐  ┌──────────┐                            │
│         │  Build Release   │  │   Skip   │                            │
│         │   Artifacts      │  │  Stage   │                            │
│         │                  │  └──────────┘                            │
│         │ ┌──────────────┐ │                                          │
│         │ │ Linux AMD64  │ │                                          │
│         │ │ Linux ARM64  │ │                                          │
│         │ │ macOS AMD64  │ │                                          │
│         │ │ macOS ARM64  │ │                                          │
│         │ └──────────────┘ │                                          │
│         │                  │                                          │
│         │ • Checksums      │                                          │
│         │ • Tar archives   │                                          │
│         │ • Artifacts      │                                          │
│         └────────┬─────────┘                                          │
└──────────────────┼──────────────────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        STAGE 3: RELEASE                                 │
│                  (Main branch + [release] flag)                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────┐      │
│  │                    Create Release                            │      │
│  │                                                              │      │
│  │  1. Calculate version (major/minor/patch)                   │      │
│  │  2. Create Git tag                                          │      │
│  │  3. Generate release notes                                  │      │
│  │  4. Upload artifacts + checksums                            │      │
│  │  5. Publish GitHub Release                                  │      │
│  └─────────────────────┬───────────────────────────────────────┘      │
│                        │                                               │
│                        ▼                                               │
│  ┌─────────────────────────────────────────────────────────────┐      │
│  │              Post-Release Validation                         │      │
│  │                                                              │      │
│  │  • Wait for release availability                            │      │
│  │  • Test install script with new version                     │      │
│  │  • Test direct binary download                              │      │
│  │  • Verify binary execution                                  │      │
│  │  • Confirm --version and --help work                        │      │
│  └─────────────────────┬───────────────────────────────────────┘      │
│                        │                                               │
│                        ▼                                               │
│                  ┌──────────┐                                          │
│                  │ Success! │                                          │
│                  └──────────┘                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Job Dependencies Graph

```
                    ┌──────────────────────────────────────┐
                    │         PARALLEL EXECUTION           │
                    ├──────────────────────────────────────┤
                    │                                      │
        ┌───────────┼─────────┬──────────┬────────────────┤
        │           │         │          │                │
        ▼           ▼         ▼          ▼                ▼
    ┌──────┐   ┌──────┐  ┌─────────┐ ┌──────────────┐   │
    │ Lint │   │ Test │  │Security │ │Test Install  │   │
    │      │   │      │  │  Scan   │ │    Script    │   │
    └───┬──┘   └───┬──┘  └────┬────┘ └──────┬───────┘   │
        │          │          │              │           │
        └──────────┴──────────┴──────────────┘           │
                       │                                 │
                       ▼                                 │
                ┌─────────────┐                          │
                │Build Verify │                          │
                └──────┬──────┘                          │
                       │                                 │
                       └─────────────────────────────────┘
                                    │
                                    ▼
                            ┌───────────────┐
                            │Check Release  │
                            └───────┬───────┘
                                    │
                                    ▼
                            ┌───────────────┐
                            │Build Release  │
                            │  (Matrix: 4)  │
                            └───────┬───────┘
                                    │
                                    ▼
                            ┌───────────────┐
                            │Create Release │
                            └───────┬───────┘
                                    │
                                    ▼
                         ┌──────────────────────┐
                         │Post-Release Validate │
                         │     (Matrix: 2)      │
                         └──────────┬───────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    │                               │
                    ▼                               ▼
            ┌───────────────┐              ┌───────────────┐
            │Notify Success │              │Notify Failure │
            └───────────────┘              └───────────────┘
```

## Conditional Execution

```
┌─────────────────────────────────────────────────────────┐
│                   TRIGGER CONDITIONS                    │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  VALIDATE Stage:                                        │
│  • Always runs on push to main/develop                  │
│  • Always runs on PRs to main/develop                   │
│  • Skips on *.md changes (except this triggers)         │
│                                                         │
│  BUILD Stage:                                           │
│  • Only if: github.ref == 'refs/heads/main'             │
│  • Only if: VALIDATE stage passed                       │
│                                                         │
│  RELEASE Stage:                                         │
│  • Only if: github.ref == 'refs/heads/main'             │
│  • Only if: BUILD stage passed                          │
│  • Only if: commit message contains '[release]'         │
│    OR workflow_dispatch triggered                       │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## Version Calculation Logic

```
Commit Message          Version Type       Example
─────────────────────────────────────────────────────
[release]          →    patch         →   1.0.0 → 1.0.1
[release] [minor]  →    minor         →   1.0.0 → 1.1.0
[release] [major]  →    major         →   1.0.0 → 2.0.0

Manual Dispatch:
• Version override  →    exact         →   v1.2.3
• Version type      →    major/minor   →   calculated
```

## Matrix Builds

### Test Matrix (3 jobs)
```
┌─────────────┬──────────┬──────────┐
│     OS      │ Go 1.21  │ Go 1.22  │
├─────────────┼──────────┼──────────┤
│ Ubuntu      │    ✓     │    ✓     │
│ macOS       │    -     │    ✓     │
└─────────────┴──────────┴──────────┘
```

### Build Matrix (4 jobs)
```
┌─────────────┬──────────┬──────────┐
│     OS      │  AMD64   │  ARM64   │
├─────────────┼──────────┼──────────┤
│ Linux       │    ✓     │    ✓     │
│ macOS       │    ✓     │    ✓     │
└─────────────┴──────────┴──────────┘
```

### Validation Matrix (2 jobs)
```
┌─────────────┐
│     OS      │
├─────────────┤
│ Ubuntu      │
│ macOS       │
└─────────────┘
```

## Caching Strategy

```
Cache Key Format:
${{ runner.os }}-go-${{ go-version }}-${{ hashFiles('**/go.sum') }}

Examples:
• Linux-go-1.22-abc123def456
• macOS-go-1.21-abc123def456

Cache Hierarchy:
1. Exact match (os + version + sum)
2. Fallback to version (os + version)
3. Fallback to os (os only)

Cached Paths:
• ~/.cache/go-build
• ~/go/pkg/mod
```

## Pipeline Metrics

```
┌──────────────────────────┬──────────┬──────────┐
│         Stage            │   Jobs   │ Approx.  │
│                          │          │   Time   │
├──────────────────────────┼──────────┼──────────┤
│ VALIDATE                 │    6     │  3-5min  │
│ BUILD (if triggered)     │    5     │  2-3min  │
│ RELEASE (if triggered)   │    4     │  3-4min  │
├──────────────────────────┼──────────┼──────────┤
│ Total (full release)     │   15     │  8-12min │
└──────────────────────────┴──────────┴──────────┘
```

## See Also

- [CICD.md](CICD.md) - Full documentation
- [PIPELINE_QUICKSTART.md](PIPELINE_QUICKSTART.md) - Quick reference
- [MIGRATION_SUMMARY.md](MIGRATION_SUMMARY.md) - What changed
