# goingenv Brand & Design System

This document defines the visual identity for goingenv across all touchpoints: website, TUI, icons, and marketing materials.

---

## Brand Overview

**Name:** goingenv
**Tagline:** Share envs the easy way
**Positioning:** Simple, secure environment file sharing for developers and small teams

**Brand Values:**
- Simplicity over complexity
- Security without friction
- Developer-native aesthetics
- Terminal-first design

---

## Logo System

### Primary Lockup

```
[●]goingenv
```

- **Format:** Logomark integrated with wordmark (no space)
- **Font:** JetBrains Mono Medium
- **Reads as:** A terminal command or CLI tool name
- **Unicode:** Uses Black Circle (U+25CF) for modern, softer aesthetic

### Logomark (Icon Only)

```
[●]
```

- **Usage:** Favicon, app icons, GitHub avatar, TUI badge, small contexts
- **Minimum size:** 16x16px (simplified at this size if needed)

### Construction

```
┌─────────────────────────────────────────────────┐
│                                                 │
│   [  ●  ]  g o i n g e n v                      │
│   ↑  ↑  ↑  ↑─────────────↑                      │
│   │  │  │  └── Wordmark (JetBrains Mono)        │
│   │  │  └───── Closing bracket                  │
│   │  └──────── Circle (brand color, U+25CF)     │
│   └─────────── Opening bracket                  │
│                                                 │
└─────────────────────────────────────────────────┘
```

### Spacing Rules

- No space between `]` and `g`
- Brackets and circle are part of the logomark
- When stacking, center the icon above the wordmark

**Stacked Version:**
```
  [●]
goingenv
```

---

## Color Palette

### Primary Colors

| Name       | Hex       | RGB           | Usage                           |
|------------|-----------|---------------|----------------------------------|
| Base       | `#121417` | 18, 20, 23    | Primary background               |
| Surface    | `#1c1f24` | 28, 31, 36    | Elevated surfaces, cards         |
| Signal     | `#22d3a7` | 34, 211, 167  | **Brand color**, CTAs, success   |

### Neutral Colors

| Name       | Hex       | RGB           | Usage                           |
|------------|-----------|---------------|----------------------------------|
| Text       | `#e9eaeb` | 233, 234, 235 | Primary text on dark            |
| Text Muted | `#8b9099` | 139, 144, 153 | Secondary text, descriptions    |
| Primary    | `#6b7a8f` | 107, 122, 143 | Borders, line numbers, brackets |

### Semantic Colors

| Name    | Hex       | RGB           | Usage                |
|---------|-----------|---------------|----------------------|
| Success | `#22d3a7` | 34, 211, 167  | Success states       |
| Error   | `#ff6b6b` | 255, 107, 107 | Error messages       |
| Warning | `#ffd93d` | 255, 217, 61  | Warning messages     |
| Info    | `#7c9cbc` | 124, 156, 188 | Informational, links |

### ANSI Terminal Mapping

For TUI implementation using 256-color or ANSI:

| Brand Color | ANSI Equivalent | 256-Color Code |
|-------------|-----------------|----------------|
| Signal      | Cyan (6)        | 43             |
| Text        | White (7)       | 255            |
| Text Muted  | Bright Black (8)| 245            |
| Primary     | Bright Black (8)| 103            |
| Error       | Red (1)         | 203            |
| Warning     | Yellow (3)      | 221            |

---

## Typography

### Font Stack

**Monospace (Primary):**
```
JetBrains Mono, IBM Plex Mono, SF Mono, Consolas, monospace
```

**Sans-serif (Secondary, documentation only):**
```
Inter, -apple-system, BlinkMacSystemFont, system-ui, sans-serif
```

### Usage Guidelines

| Context          | Font            | Weight  | Size        |
|------------------|-----------------|---------|-------------|
| Logo wordmark    | JetBrains Mono  | Medium  | —           |
| TUI all text     | Terminal default| Regular | Terminal    |
| Website headers  | Inter           | Medium  | 1.5-2rem    |
| Website body     | Inter           | Regular | 1rem        |
| Code blocks      | JetBrains Mono  | Regular | 0.875rem    |

---

## Logo Color Variants

### On Dark Background (Primary)

```
Brackets:  #6b7a8f (Primary/muted gray)
Circle:    #22d3a7 (Signal/teal)
Wordmark:  #e9eaeb (Text/light)
```

### On Light Background

```
Brackets:  #6b7a8f (Primary/muted gray)
Circle:    #22d3a7 (Signal/teal)
Wordmark:  #121417 (Base/dark)
```

### Monochrome (Single Color)

```
All elements: #22d3a7 (Signal/teal)
— or —
All elements: #e9eaeb (light) / #121417 (dark)
```

---

## Icon Specifications

### Favicon

| Size   | Format | Notes                                    |
|--------|--------|------------------------------------------|
| 16x16  | ICO    | Simplified: may need thicker brackets    |
| 32x32  | ICO    | Standard favicon                         |
| 48x48  | PNG    | Windows taskbar                          |
| 180x180| PNG    | Apple touch icon                         |
| 192x192| PNG    | Android Chrome                           |
| 512x512| PNG    | PWA icon                                 |

### Social & Marketing

| Context        | Size      | Format | Content              |
|----------------|-----------|--------|----------------------|
| GitHub avatar  | 500x500   | PNG    | Icon only, centered  |
| OG image       | 1200x630  | PNG    | Full lockup, centered|
| Twitter card   | 1200x600  | PNG    | Full lockup          |

---

## TUI Design System

This section defines the complete visual design for the terminal user interface.

### Design Principles

1. **Borderless** - No box-drawing characters or heavy borders
2. **Minimal** - Only essential information, generous whitespace
3. **Consistent** - Same layout structure across all screens
4. **Discoverable** - Keyboard hints always visible

---

### Color Palette (lipgloss)

Migrate from purple (`#7D56F4`) to teal (`#22d3a7`) for brand alignment.

```go
var DarkTheme = ColorPalette{
    Primary:    lipgloss.Color("#22d3a7"), // Brand teal - selection, accents
    Secondary:  lipgloss.Color("#e9eaeb"), // Primary text
    Error:      lipgloss.Color("#ff6b6b"), // Error messages
    Success:    lipgloss.Color("#22d3a7"), // Success (same as primary)
    Warning:    lipgloss.Color("#ffd93d"), // Warnings
    Info:       lipgloss.Color("#7c9cbc"), // Info, links
    Muted:      lipgloss.Color("#6b7a8f"), // Secondary text, brackets
    Background: lipgloss.Color("#121417"), // Not used (terminal bg)
    Surface:    lipgloss.Color("#1c1f24"), // Not used (borderless)
}
```

---

### Screen Layout Template

Every screen follows this structure:

```
[●]goingenv v1.1.0                        <- Header (always)
                                          <- Blank line
{Content Area}                            <- Screen-specific content
                                          <- Flexible space
                                          <- Blank line
[key] action  [key] action  [q] quit      <- Footer hints (always)
```

**Header:** Logo + version, colored per brand spec
**Content:** Variable per screen, left-aligned
**Footer:** Context-sensitive keyboard shortcuts, muted color

---

### Header Specification

```
[●]goingenv v1.1.0
```

**Styling:**
- `[` and `]` - Muted color (`#6b7a8f`)
- `●` - Brand teal (`#22d3a7`)
- `goingenv` - Primary text (`#e9eaeb`)
- `v1.1.0` - Muted color (`#6b7a8f`)

**Implementation:**
```go
func renderHeader(version string) string {
    return lipgloss.JoinHorizontal(
        lipgloss.Left,
        mutedStyle.Render("["),
        primaryStyle.Render("●"),
        mutedStyle.Render("]"),
        textStyle.Render("goingenv"),
        mutedStyle.Render(" v"+version),
    )
}
```

---

### Footer Specification

Always visible, shows available actions for current screen.

**Format:**
```
[↑↓] navigate  [enter] select  [q] quit
```

**Styling:**
- Keys in brackets: Muted (`#6b7a8f`)
- Action text: Muted (`#6b7a8f`)
- Separator: Two spaces between each item

**Screen-specific footers:**

| Screen | Footer |
|--------|--------|
| Menu | `[↑↓] navigate  [enter] select  [q] quit` |
| Password | `[enter] confirm  [esc] cancel` |
| File picker | `[↑↓] navigate  [enter] select  [esc] back` |
| Progress | `[esc] cancel` or empty during operation |
| Status | `[p] pack  [u] unpack  [esc] back  [q] quit` |
| List view | `[↑↓] scroll  [esc] back` |
| Help | `[↑↓] scroll  [esc] back` |
| Error | `[enter] continue` |
| Success | `[enter] continue` |

---

### Menu Selection

**Selection indicator:** Teal chevron `>`

```
  [P] Pack Environment Files
> [U] Unpack Archive              <- Selected (teal chevron)
  [L] List Archive Contents
  [S] Status
  [●] Settings
  [?] Help
```

**Styling:**
- Unselected: No prefix, normal text color
- Selected: Teal `>` prefix, text remains normal (no background highlight)
- Two-space indent for unselected items to align with `> ` prefix

**Implementation:**
```go
func renderMenuItem(item string, selected bool) string {
    if selected {
        return primaryStyle.Render("> ") + item
    }
    return "  " + item
}
```

---

### Screen Specifications

#### Main Menu

```
[●]goingenv v1.1.0

> [P] Pack Environment Files
  [U] Unpack Archive
  [L] List Archive Contents
  [S] Status
  [●] Settings
  [?] Help

[↑↓] navigate  [enter] select  [q] quit
```

#### Status Screen

```
[●]goingenv v1.1.0

Directory
  ~/projects/myapp

Environment Files
  .env
  .env.local
  .env.production

Archives
  envs.goingenv    12KB    2 hours ago

[p] pack  [u] unpack  [esc] back  [q] quit
```

**Styling:**
- Section headers (`Directory`, `Environment Files`, `Archives`): Muted color
- Content: Primary text, indented 2 spaces
- File metadata (size, time): Muted color, right-aligned or inline

#### Pack Screen (Password Entry)

```
[●]goingenv v1.1.0

Packing 3 environment files

  .env
  .env.local
  .env.production

Password: ••••••••_

[enter] confirm  [esc] cancel
```

**Styling:**
- File list: Primary text, indented
- Password prompt: Primary text
- Password input: Masked with `•`, cursor shown as `_`
- No border around input field

#### Progress Screen

```
[●]goingenv v1.1.0

Packing environment files...

  .env               done
  .env.local         done
  .env.production    encrypting...

[████████████░░░░░░░░] 65%

```

**Styling:**
- Status `done`: Success/teal color
- Status `encrypting...`: Muted color
- Progress bar: Teal filled (`█`), muted unfilled (`░`)
- Percentage: Muted color

#### Success Screen

```
[●]goingenv v1.1.0

Packed successfully

  Created envs.goingenv (3 files, 2.4KB)
  Encrypted in 0.003s

[enter] continue
```

**Styling:**
- Success message: Teal color
- Details: Primary text, indented

#### Error Screen

```
[●]goingenv v1.1.0

Error

  Archive not found: backup.enc

  The specified file does not exist in
  the current directory.

[enter] continue
```

**Styling:**
- `Error` header: Error color (`#ff6b6b`)
- Error message: Primary text
- Suggestion/context: Muted color

#### List Archive Screen

```
[●]goingenv v1.1.0

Archive: envs.goingenv

  .env                 245 bytes
  .env.local           189 bytes
  .env.production      312 bytes

  Total: 3 files, 746 bytes
  Created: 2025-01-23 14:30

[esc] back
```

#### Help Screen

```
[●]goingenv v1.1.0

Keyboard Shortcuts

  Navigation
    ↑/k     Move up
    ↓/j     Move down
    enter   Select
    esc     Back/Cancel
    q       Quit

  Quick Actions
    p       Pack files
    u       Unpack archive
    s       View status

[esc] back
```

**Styling:**
- Section headers: Muted color
- Keys: Primary text
- Descriptions: Muted color

#### File Picker Screen

```
[●]goingenv v1.1.0

Select archive to unpack

  ~/projects/myapp/.goingenv/

> envs.goingenv           12KB    2 hours ago
  backup-old.goingenv     8KB     3 days ago

[↑↓] navigate  [enter] select  [esc] back
```

**Styling:**
- Current directory: Muted color
- Selected file: Teal chevron prefix
- File metadata: Muted color

#### Settings Screen

```
[●]goingenv v1.1.0

Settings

  Scan Depth        10
  Max File Size     10MB

  Patterns
    Include         \.env.*
    Exclude         \.env\.example$

  Config Location
    ~/.goingenv.json

[esc] back
```

---

### Removed Elements

The following elements from the current TUI should be removed:

1. **TitleStyle with background** - Replace with plain text header
2. **Box borders on all components** - Remove entirely
3. **ListStyle border** - Menu items without container
4. **PasswordInputStyle border** - Inline input without box
5. **FilePickerStyle border** - Simple list without container
6. **StatusCardStyle border** - Section headers with indented content
7. **ArchiveCardStyle border** - Same as above
8. **Purple color usage** - Replace all with teal

---

### Animation & Transitions

**Keep minimal:**
- Cursor blink on text input
- Progress bar updates

**Avoid:**
- Screen transition animations
- Fade effects
- Loading spinners (use text: `loading...`)

---

### Responsive Behavior

**Width handling:**
- Minimum width: 40 characters
- Maximum content width: 80 characters (readable)
- Footer always at bottom of available space

**Height handling:**
- Scrollable content areas for long lists
- Header and footer always visible

---

### Menu Icons Reference

Consistent bracket-icon system matching the `[●]` logo:

```
[P] Pack Environment Files
[U] Unpack Archive
[L] List Archive Contents
[S] Status
[●] Settings
[?] Help
[>] Initialize goingenv (when not initialized)
```

Icons use the same bracket style as the logo, reinforcing brand consistency.

---

## CLI Design System

This section defines the visual design for command-line output (non-TUI mode).

### Design Principles

1. **Minimal** - Only essential information in standard output
2. **Scannable** - Prefix indicators for quick visual parsing
3. **Script-friendly** - Colors auto-disabled when piped
4. **Consistent** - Same patterns across all commands

---

### Color Mode Detection

Colors are enabled only when stdout is a TTY (interactive terminal).

```go
import "github.com/muesli/termenv"

func useColors() bool {
    return termenv.NewOutput(os.Stdout).Profile() != termenv.Ascii
}
```

When piped or redirected, output falls back to plain text with indicators intact.

---

### Color Palette (CLI)

When colors are enabled:

| Element | Color | Hex | ANSI |
|---------|-------|-----|------|
| Brand/Logo `[●]` | Teal | `#22d3a7` | 43 |
| Success `[+]` | Teal | `#22d3a7` | 43 |
| Warning `[!]` | Yellow | `#ffd93d` | 221 |
| Error `[x]` | Red | `#ff6b6b` | 203 |
| Info `[?]` | Blue | `#7c9cbc` | 110 |
| Muted text | Gray | `#6b7a8f` | 103 |
| Primary text | Default | — | — |

---

### Prefix Indicator System

All output lines use bracketed prefixes for visual consistency with the `[●]` logo.

| Indicator | Meaning | Color | Usage |
|-----------|---------|-------|-------|
| `[●]` | Brand/header | Teal | App header only |
| `[+]` | Success | Teal | Completed actions |
| `[!]` | Warning | Yellow | Non-fatal issues |
| `[x]` | Error | Red | Fatal errors |
| `[>]` | Action | Default | Current operation |
| `[?]` | Hint/tip | Blue | Suggestions |
| `[-]` | List item | Default | Items in a list |
| `[ ]` | Unchecked | Default | Pending items |
| `[~]` | Skipped | Gray | Skipped items |

---

### Output Structure

#### Header (all commands)

```
[●] goingenv v1.1.0
```

Shown once at the start of output. Version from build info.

#### Standard Output Template

```
[●] goingenv v1.1.0

[>] {action description}

{content}

[+] {success summary}
```

No separator lines. Blank lines separate logical sections.

---

### Command Output Specifications

#### init

**Standard:**
```
[●] goingenv v1.1.0

[+] Initialized

[?] Run 'goingenv status' to see detected files
```

**Verbose:**
```
[●] goingenv v1.1.0

[>] Initializing goingenv...

[+] Created .goingenv/

[?] Next steps:
    Run 'goingenv status' to see detected files
    Run 'goingenv pack' to create encrypted archive
```

---

#### status

**Standard:**
```
[●] goingenv v1.1.0

Directory
  ~/projects/myapp

Environment Files (3)
  .env
  .env.local
  .env.production

Archives (1)
  envs.goingenv
```

**Verbose:**
```
[●] goingenv v1.1.0

Directory
  ~/projects/myapp

Configuration
  Scan depth: 10
  Max file size: 10MB
  Config: ~/.goingenv.json

Environment Files (3)
  .env                  245 bytes   2025-01-23 10:15
  .env.local            189 bytes   2025-01-23 10:15
  .env.production       312 bytes   2025-01-22 14:30

Archives (1)
  envs.goingenv         2.4 KB      2 hours ago

[?] Run 'goingenv pack' to create a new archive
```

---

#### pack

**Standard:**
```
[●] goingenv v1.1.0

[>] Packing 3 files...

[+] Created envs.goingenv
```

**Verbose:**
```
[●] goingenv v1.1.0

[>] Packing 3 environment files...

[-] .env (245 bytes)
[-] .env.local (189 bytes)
[-] .env.production (312 bytes)

[>] Encrypting with AES-256-GCM...

[+] Created envs.goingenv
    Files: 3
    Size: 2.4 KB
    Time: 0.003s

[?] Store your password securely
```

**With warnings:**
```
[●] goingenv v1.1.0

[>] Packing 3 files...

[!] Large file detected: .env.production (5.2 MB)

[+] Created envs.goingenv
```

---

#### unpack

**Standard:**
```
[●] goingenv v1.1.0

[>] Unpacking envs.goingenv...

[+] Extracted 3 files
```

**With conflicts:**
```
[●] goingenv v1.1.0

[>] Unpacking envs.goingenv...

[!] 2 files already exist:
    .env
    .env.local

[?] Use --force to overwrite
```

**Verbose:**
```
[●] goingenv v1.1.0

[>] Unpacking envs.goingenv...

[+] .env
[+] .env.local
[+] .env.production

[>] Verifying checksums...

[+] Extracted 3 files
    Verified: 3/3
    Time: 0.002s
```

---

#### list

**Standard:**
```
[●] goingenv v1.1.0

envs.goingenv

  .env                  245 bytes
  .env.local            189 bytes
  .env.production       312 bytes

  3 files, 746 bytes total
```

**Verbose:**
```
[●] goingenv v1.1.0

envs.goingenv
  Created: 2025-01-23 14:30:45
  Version: 1

Files
  .env                  245 bytes   a1b2c3d4e5f6...
  .env.local            189 bytes   b2c3d4e5f6a1...
  .env.production       312 bytes   c3d4e5f6a1b2...

Summary
  Files: 3
  Total size: 746 bytes
  Compressed: 512 bytes
```

**JSON format (`--format json`):**
```json
{
  "archive": "envs.goingenv",
  "created": "2025-01-23T14:30:45Z",
  "files": [
    {"name": ".env", "size": 245},
    {"name": ".env.local", "size": 189},
    {"name": ".env.production", "size": 312}
  ],
  "total_size": 746
}
```

---

### Error Output

Errors go to stderr with `[x]` prefix.

**Standard error:**
```
[x] Archive not found: backup.enc
```

**With context:**
```
[x] Failed to decrypt archive

[?] Check your password and try again
```

**Verbose error:**
```
[x] Failed to decrypt archive
    File: backup.enc
    Error: cipher: message authentication failed

[?] This usually means the password is incorrect
```

---

### Warning Output

Warnings are non-fatal, shown with `[!]` prefix.

```
[!] Config file not found, using defaults

[!] 2 files would be overwritten:
    .env
    .env.local
```

---

### Hint/Tip Output

Helpful suggestions shown with `[?]` prefix in blue.

```
[?] Run 'goingenv status' to see detected files

[?] Use --force to overwrite existing files

[?] Store your password securely
```

---

### Progress Output

For long operations, show current action with `[>]`:

```
[>] Scanning for environment files...
[>] Encrypting...
[>] Writing archive...
[+] Done
```

In verbose mode, show per-file progress:

```
[>] Packing files...
[-] .env
[-] .env.local
[-] .env.production
[>] Encrypting...
[+] Created envs.goingenv
```

---

### Interactive Prompts

Confirmation prompts maintain minimal style:

```
[?] Overwrite 2 existing files? [y/N]:
```

Password prompts:

```
[>] Password:
[>] Confirm password:
```

---

### Formatting Rules

#### Indentation
- Section content: 2 spaces
- Sub-items: 4 spaces
- No tabs

#### Alignment
- File names: Left-aligned
- Sizes: Right-aligned
- Timestamps: Right-aligned

#### Numbers
- File sizes: Human-readable (KB, MB)
- Counts: No formatting
- Durations: 3 decimal places (0.003s)

#### Paths
- Use `~` for home directory
- Relative paths when inside project
- Absolute paths for external files

---

### Implementation Notes

#### lipgloss Styles for CLI

```go
var (
    brandStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#22d3a7"))
    successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#22d3a7"))
    warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd93d"))
    errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6b6b"))
    infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#7c9cbc"))
    mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7a8f"))
)

func printHeader(version string) {
    if useColors() {
        fmt.Printf("%s goingenv v%s\n", brandStyle.Render("[●]"), version)
    } else {
        fmt.Printf("[●] goingenv v%s\n", version)
    }
}

func printSuccess(msg string) {
    if useColors() {
        fmt.Printf("%s %s\n", successStyle.Render("[+]"), msg)
    } else {
        fmt.Printf("[+] %s\n", msg)
    }
}
```

#### TTY Detection

```go
import (
    "os"
    "github.com/muesli/termenv"
)

var output = termenv.NewOutput(os.Stdout)

func useColors() bool {
    return output.Profile() != termenv.Ascii
}
```

---

## Website Application

### Navigation Logo

SVG of `[●]goingenv` lockup, linking to home.

### Code Block Styling

- Background: `#1c1f24` (Surface)
- Left border: `#6b7a8f` (Primary) — changes to `#22d3a7` (Signal) on hover
- Text: `#e9eaeb` (Text)

### CTA Styling

- Color: `#22d3a7` (Signal)
- Hover: Subtle glow effect (`text-shadow: 0 0 20px rgba(34, 211, 167, 0.3)`)

---

## Asset Checklist

### Icons to Generate

- [ ] favicon.ico (16x16, 32x32, 48x48 combined)
- [ ] apple-touch-icon.png (180x180)
- [ ] icon-192.png (192x192, for PWA)
- [ ] icon-512.png (512x512, for PWA)
- [ ] github-avatar.png (500x500)
- [ ] og-image.png (1200x630)

### SVG Assets

- [ ] logo-full.svg (full lockup)
- [ ] logo-icon.svg (icon only)
- [ ] logo-full-mono-light.svg (monochrome for light bg)
- [ ] logo-full-mono-dark.svg (monochrome for dark bg)

---

## Image Generation Prompts

The following prompts are designed for AI image generators to create brand assets. Adjust based on the specific tool being used.

### Prompt 1: Icon/Favicon (Square Format)

**Goal:** Generate the `[●]` logomark as a clean icon.

```
Create a minimal, developer-focused icon for a CLI tool called "goingenv".

Design specifications:
- Square format, suitable for favicon and app icon use
- Dark background: #121417 (very dark charcoal, almost black)
- The icon consists of: opening square bracket, solid circle, closing square bracket
- Written as: [●]
- Brackets color: #6b7a8f (muted blue-gray)
- Circle color: #22d3a7 (teal/cyan green)
- Font: Monospace, clean, modern (similar to JetBrains Mono or SF Mono)
- The circle should be the visual focal point, slightly brighter/more prominent
- Generous padding around the symbol (approximately 20% margin on each side)
- No gradients, no shadows, no 3D effects
- Flat design, pixel-perfect for small sizes
- Terminal/code aesthetic
- Professional, minimal, technical appearance

Output sizes needed: 512x512 (will be downscaled to 16x16, 32x32, 48x48, 180x180, 192x192)

Style reference: Similar to the minimal aesthetic of tools like Vercel, Linear, or Raycast icons.
```

### Prompt 2: Full Logo Lockup (Horizontal)

**Goal:** Generate the full `[●]goingenv` wordmark.

```
Create a horizontal logo lockup for a CLI tool called "goingenv".

Design specifications:
- Horizontal format, aspect ratio approximately 4:1
- Dark background: #121417 (very dark charcoal)
- The logo reads: [●]goingenv (no space between ] and g)
- Brackets [ ] color: #6b7a8f (muted blue-gray)
- Circle ● color: #22d3a7 (teal/cyan green)
- Text "goingenv" color: #e9eaeb (off-white)
- Font: Monospace, medium weight, clean (JetBrains Mono style)
- All characters should be the same height, aligned on baseline
- No gradients, shadows, or effects
- Flat, minimal design
- Terminal/developer tool aesthetic
- The circle is the only element with the accent color, making it the focal point

Padding: Comfortable margins, suitable for use in navigation bars or headers.

Style reference: The typography style of terminal emulators and code editors. Clean, technical, professional.
```

### Prompt 3: Social/OG Image (1200x630)

**Goal:** Generate an Open Graph image for social sharing.

```
Create a social media preview image (Open Graph) for a developer tool called "goingenv".

Design specifications:
- Dimensions: 1200x630 pixels
- Background: #121417 (very dark charcoal) with subtle texture
- Optional: Very subtle dot grid pattern (opacity ~4%) for visual interest
- Optional: Subtle gradient glow in top-right corner using #22d3a7 at very low opacity (~5%)

Content centered on image:
- Logo: [●]goingenv
  - Brackets: #6b7a8f
  - Circle: #22d3a7
  - Text: #e9eaeb
  - Font: Monospace, medium weight
  - Size: Prominent, approximately 15-20% of image width

- Below the logo, add tagline: "Share envs the easy way"
  - Color: #8b9099 (muted gray)
  - Font: Same monospace, regular weight
  - Size: Smaller than logo, approximately 40% of logo text size

Layout:
- Logo and tagline vertically centered
- Horizontal center alignment
- Generous whitespace

Style: Minimal, dark mode, developer-focused, professional. Similar to how Vercel, Railway, or Supabase style their OG images.
```

### Prompt 4: GitHub Avatar (Square, Icon Focus)

**Goal:** Generate a GitHub profile/organization avatar.

```
Create a GitHub avatar image for a developer tool called "goingenv".

Design specifications:
- Dimensions: 500x500 pixels (square)
- Background: #121417 (very dark charcoal)
- Center content: The icon [●]
  - Brackets: #6b7a8f (muted blue-gray)
  - Circle: #22d3a7 (teal/cyan green)
- Font: Monospace, clean, bold enough to read at small sizes
- The icon should fill approximately 50-60% of the image width
- Perfectly centered both horizontally and vertically
- No additional elements, text, or decorations
- Flat design, no gradients or shadows
- Must remain legible when scaled down to 40x40px (GitHub comment avatars)

Style: Ultra-minimal, recognizable at small sizes, developer tool aesthetic.
```

### Prompt 5: Light Mode Variant

**Goal:** Generate logo for light backgrounds.

```
Create a logo variant for light backgrounds for a CLI tool called "goingenv".

Design specifications:
- Horizontal format
- Light/white background: #ffffff or #f5f5f5
- The logo reads: [●]goingenv
- Brackets [ ] color: #6b7a8f (muted blue-gray)
- Circle ● color: #22d3a7 (teal/cyan green) — keeps brand color
- Text "goingenv" color: #121417 (dark charcoal)
- Font: Monospace, medium weight
- Flat design, no effects

This is the inverted version for documentation, light-mode websites, or print materials.
```

### Prompt 6: Monochrome Version (Single Color)

**Goal:** Generate single-color logo for limited color contexts.

```
Create a monochrome logo for a CLI tool called "goingenv".

Version A - Teal on dark:
- Background: #121417
- All elements (brackets, circle, text): #22d3a7

Version B - Dark on light:
- Background: #ffffff
- All elements: #121417

Version C - Light on dark:
- Background: #121417
- All elements: #e9eaeb

Font: Monospace, medium weight
Format: Horizontal lockup [●]goingenv
No color differentiation between elements — entire logo is one solid color.
```

---

## Implementation Priority

1. **Phase 1: Core Assets**
   - Generate icon (512x512)
   - Create favicon set from icon
   - Update website with SVG logo

2. **Phase 2: TUI Alignment**
   - Update `internal/tui/styles.go` with new color palette
   - Update header to show `[●]goingenv v{version}`
   - Test across terminal emulators

3. **Phase 3: Marketing Assets**
   - Generate OG image
   - Create GitHub avatar
   - Update repository with new branding

4. **Phase 4: Documentation**
   - Update README with logo
   - Add brand assets to `/assets` or `/brand` directory
   - Create contributing guidelines for brand usage

---

## File Structure

```
/public
  /favicon.ico
  /apple-touch-icon.png
  /icon-192.png
  /icon-512.png
  /og-image.png

/assets (or /brand)
  /logo-full.svg
  /logo-icon.svg
  /logo-full-mono-light.svg
  /logo-full-mono-dark.svg
  /github-avatar.png

/DESIGN.md (this file)
```

---

## Version History

| Version | Date       | Changes                    |
|---------|------------|----------------------------|
| 1.0     | 2025-01-23 | Initial brand system       |
