# goingenv Brand Assets

This directory contains brand assets for goingenv. See `/docs/design.md` for the complete brand and design system documentation.

## Directory Structure

```
assets/
  logo-full.svg             # Full lockup: [●]goingenv
  logo-full-light.svg       # Full lockup for light backgrounds
  logo-icon.svg             # Icon only: [●]
  logo-full-mono-light.svg  # Monochrome for light backgrounds
  logo-full-mono-dark.svg   # Monochrome for dark backgrounds
  github-avatar.png         # GitHub avatar (500x500)
  icon-192.png              # PWA icon
  icon-512.png              # PWA icon
  og-image.png              # Open Graph image (1200x630)
  og-image.svg              # Open Graph image (SVG source)

public/
  favicon.ico               # Multi-size ICO (16, 32, 48)
  apple-touch-icon.png      # 180x180
  icon-192.png              # PWA icon
  icon-512.png              # PWA icon
```

## Color Palette

Colours are defined in [docs/design.md](../docs/design.md#color-palette). They
are not repeated here so the two cannot drift.

## Logo Usage Guidelines

### Primary Lockup

```
[●]goingenv
```

- No space between `]` and `g`
- Colours per [docs/design.md](../docs/design.md#logo-system)

### Icon Only

```
[●]
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

Assets can be generated using the prompts in `/docs/design.md` with AI image generators.

### Quick Reference

For favicon generation:
1. Generate 512x512 icon using prompt in docs/design.md
2. Resize to required sizes
3. Convert to ICO format combining 16, 32, 48 sizes

For OG image:
1. Generate 1200x630 using prompt in docs/design.md
2. Save as PNG with proper compression
