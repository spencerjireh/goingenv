# goingenv Brand Assets

This directory contains brand assets for goingenv. See `/DESIGN.md` for the complete brand and design system documentation.

## Directory Structure

```
assets/
  logo-full.svg           # Full lockup: [*]goingenv
  logo-icon.svg           # Icon only: [*]
  logo-full-mono-light.svg # Monochrome for light backgrounds
  logo-full-mono-dark.svg  # Monochrome for dark backgrounds
  github-avatar.png       # GitHub avatar (500x500)
  og-image.png            # Open Graph image (1200x630)
  favicon/                # Favicon files
    favicon.ico           # Multi-size ICO (16, 32, 48)
    favicon-16x16.png
    favicon-32x32.png
    apple-touch-icon.png  # 180x180
    icon-192.png          # PWA icon
    icon-512.png          # PWA icon
```

## Color Palette

| Name       | Hex       | Usage                           |
|------------|-----------|----------------------------------|
| Base       | `#121417` | Primary background               |
| Surface    | `#1c1f24` | Elevated surfaces                |
| Signal     | `#22d3a7` | Brand color, CTAs, success       |
| Text       | `#e9eaeb` | Primary text on dark             |
| Text Muted | `#8b9099` | Secondary text                   |
| Primary    | `#6b7a8f` | Borders, brackets                |
| Error      | `#ff6b6b` | Error messages                   |
| Warning    | `#ffd93d` | Warning messages                 |
| Info       | `#7c9cbc` | Informational, links             |

## Logo Usage Guidelines

### Primary Lockup

```
[*]goingenv
```

- No space between `]` and `g`
- Brackets: `#6b7a8f`
- Asterisk: `#22d3a7`
- Wordmark: `#e9eaeb` (dark bg) or `#121417` (light bg)

### Icon Only

```
[*]
```

Use for favicon, app icons, small contexts.

### Minimum Sizes

- Full lockup: 120px width minimum
- Icon only: 16px minimum (simplified if needed)

## File Naming Conventions

- Descriptive names: `logo-full.svg`, `logo-icon.svg`
- Size in filename for PNGs: `icon-192.png`
- Context suffix: `logo-full-mono-light.svg`

## Generating Assets

Assets can be generated using the prompts in `/DESIGN.md` with AI image generators.

### Quick Reference

For favicon generation:
1. Generate 512x512 icon using prompt in DESIGN.md
2. Resize to required sizes
3. Convert to ICO format combining 16, 32, 48 sizes

For OG image:
1. Generate 1200x630 using prompt in DESIGN.md
2. Save as PNG with proper compression
