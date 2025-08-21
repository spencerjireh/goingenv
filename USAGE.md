# GoingEnv Usage Guide

This guide provides comprehensive examples and workflows for using GoingEnv effectively.

## Table of Contents

- [Getting Started](#getting-started)
- [Interactive Mode (TUI)](#interactive-mode-tui)
- [Command Line Mode](#command-line-mode)
- [Common Workflows](#common-workflows)
- [Advanced Usage](#advanced-usage)
- [Configuration](#configuration)
- [Troubleshooting](#troubleshooting)

## Getting Started

### First Run

After installation, you must initialize GoingEnv in each project directory:

```bash
# Navigate to your project directory
cd /path/to/your/project

# Initialize GoingEnv (required first step)
goingenv init

# Check current directory for environment files
goingenv status

# Verbose output with detailed information
goingenv status --verbose
```

> **⚠️ Important**: You must run `goingenv init` in each project directory before using any other commands. This creates the necessary `.goingenv` directory structure and configuration files.

### Launch Interactive Mode

For beginners, the interactive TUI is the easiest way to get started:

```bash
# Launch interactive terminal interface
goingenv

# Launch with debug logging enabled
goingenv --verbose
```

## Interactive Mode (TUI)

The TUI provides a user-friendly interface for all GoingEnv operations.

### Navigation

- **Arrow Keys** or **j/k**: Navigate menu items
- **Enter**: Select menu option
- **Esc**: Go back to previous screen
- **q** or **Ctrl+C**: Quit application

### Main Menu Options

**When Not Initialized:**
1. **Initialize GoingEnv** - Set up GoingEnv in the current directory (required first step)
2. **Help** - Interactive help and documentation

**After Initialization:**
1. **Pack Environment Files**
   - Scans current directory for env files
   - Shows preview of detected files
   - Prompts for encryption password
   - Creates encrypted archive

2. **Unpack Archive**
   - Browse available `.enc` files
   - Prompts for decryption password
   - Extracts files to current directory

3. **List Archive Contents**
   - Browse and select archive file
   - Shows archive metadata and file list
   - No files are extracted

4. **Status**
   - Shows current directory information
   - Lists available archives
   - Displays detected environment files
   - Shows configuration settings

5. **Settings**
   - View current configuration
   - Shows scan depth, patterns, and limits

6. **Help**
   - Interactive help and documentation
   - Command examples and usage tips

## Command Line Mode

### Initialization

**Initialize GoingEnv (required first step):**
```bash
# Initialize in current directory
goingenv init

# Force re-initialization (if already initialized)
goingenv init --force

# Check initialization status
goingenv status
```

The `init` command creates:
- `.goingenv/` directory in your project
- `.goingenv/.gitignore` file (allows `*.enc` files for safe transfer)
- Adds `.goingenv/` to your project's `.gitignore`
- Default configuration in your home directory

### Password Security

GoingEnv provides two secure methods for password input:

**1. Interactive Prompt (Most Secure)**
```bash
# Password is hidden during input
goingenv pack  # Will prompt for password
```

**2. Environment Variable (For Automation)**
```bash
# Set password in environment variable
export MY_PASSWORD="your-password"

# Use environment variable (shows security warning)
goingenv pack --password-env MY_PASSWORD

# Clear after use
unset MY_PASSWORD
```

**Security Best Practices:**
- **Never use passwords on command line** (visible in shell history and process lists)
- **Interactive prompts** are the most secure for manual operations
- **Environment variables** are visible to other processes - use carefully
- **Clear passwords** from environment variables after use

### Pack Operations

**Basic Packing:**
```bash
# Pack with password prompt
goingenv pack

# Pack with password from environment
goingenv pack --password-env MY_PASSWORD

# Pack from specific directory
goingenv pack -d /path/to/project --password-env MY_PASSWORD

# Pack with custom output name
goingenv pack --password-env MY_PASSWORD -o backup-2024.enc

# Pack with description
goingenv pack --password-env MY_PASSWORD --description "Production backup before deployment"
```

**Advanced Packing:**
```bash
# Dry run (preview what would be packed)
goingenv pack --dry-run

# Verbose output
goingenv pack --password-env MY_PASSWORD --verbose

# Custom scan depth
goingenv pack --password-env MY_PASSWORD --depth 3

# Pack from multiple directories
cd /project1 && goingenv pack -k "pass" -o project1.enc
cd /project2 && goingenv pack -k "pass" -o project2.enc
```

### Unpack Operations

**Basic Unpacking:**
```bash
# Unpack latest archive (prompts for password)
goingenv unpack

# Unpack specific archive
goingenv unpack -f backup.enc --password-env MY_PASSWORD

# Unpack to specific directory
goingenv unpack -f backup.enc --password-env MY_PASSWORD --target-dir /restore/path

# Unpack with overwrite protection
goingenv unpack -f backup.enc --password-env MY_PASSWORD --backup
```

**Advanced Unpacking:**
```bash
# Dry run (preview what would be extracted)
goingenv unpack -f backup.enc --password-env MY_PASSWORD --dry-run

# Force overwrite existing files
goingenv unpack -f backup.enc --password-env MY_PASSWORD --overwrite

# Verbose output with progress
goingenv unpack -f backup.enc --password-env MY_PASSWORD --verbose

# Unpack with file verification
goingenv unpack -f backup.enc --password-env MY_PASSWORD --verify
```

### List Operations

**Archive Inspection:**
```bash
# List contents of latest archive
goingenv list

# List specific archive
goingenv list -f backup.enc --password-env MY_PASSWORD

# List with detailed file information
goingenv list -f backup.enc --password-env MY_PASSWORD --verbose

# List with formatted output
goingenv list -f backup.enc --password-env MY_PASSWORD --format table
```

### Status Operations

**System Information:**
```bash
# Basic status
goingenv status

# Detailed status with file analysis
goingenv status --verbose

# Status with statistics
goingenv status --stats
```

## Common Workflows

### 1. Daily Development Backup

```bash
#!/bin/bash
# daily-backup.sh

DATE=$(date +%Y%m%d)
PROJECT_NAME=$(basename $(pwd))
BACKUP_NAME="${PROJECT_NAME}-${DATE}.enc"

echo "Creating daily backup: $BACKUP_NAME"
goingenv pack -k "$BACKUP_PASSWORD" -o "$BACKUP_NAME" --description "Daily backup"

echo "Backup created: ~/.goingenv/$BACKUP_NAME"
```

### 2. Project Setup from Backup

```bash
# 1. Clone project repository
git clone https://github.com/user/project.git
cd project

# 2. Initialize GoingEnv
goingenv init

# 3. Check for available environment backups
goingenv status

# 4. Restore environment files
goingenv unpack -f project-env.enc --password-env MY_PASSWORD

# 5. Verify restored files
goingenv status --verbose
```

### 3. Environment Migration

```bash
# On source machine (assuming already initialized)
cd /old/project
goingenv pack -k "migration-key" -o migration.enc --description "Migration from server-1"

# Transfer migration.enc to new machine

# On target machine
cd /new/project
goingenv init  # Initialize if not already done
goingenv unpack -f migration.enc -k "migration-key" --backup
goingenv status
```

### 4. Team Environment Sharing

```bash
# Team lead creates template
goingenv pack --password-env TEAM_PASSWORD -o team-template.enc --description "Team environment template"

# Team members restore template
goingenv unpack -f team-template.enc --password-env TEAM_PASSWORD

# Customize local environment
cp .env .env.local
# Edit .env.local with personal settings
```

### 5. Pre-deployment Backup

```bash
#!/bin/bash
# pre-deploy-backup.sh

VERSION=$(git describe --tags --always)
ENVIRONMENT=${1:-production}

echo "Creating pre-deployment backup for $ENVIRONMENT ($VERSION)"

goingenv pack \
  -k "$DEPLOY_BACKUP_PASSWORD" \
  -o "pre-deploy-${ENVIRONMENT}-${VERSION}.enc" \
  --description "Pre-deployment backup for $ENVIRONMENT v$VERSION" \
  --verbose

echo "Backup completed: ~/.goingenv/pre-deploy-${ENVIRONMENT}-${VERSION}.enc"
```

## Advanced Usage

### Environment Variables

Configure GoingEnv behavior with environment variables:

```bash
# Custom configuration directory
export GOINGENV_CONFIG_DIR="$HOME/.config/goingenv"

# Default archive directory
export GOINGENV_ARCHIVE_DIR="$HOME/backups"

# Debug logging
export GOINGENV_DEBUG=1

# Custom scan patterns
export GOINGENV_PATTERNS=".env,.env.*,.environment"
```

### Scripting and Automation

**Automated Backup Script:**
```bash
#!/bin/bash
set -e

# Configuration
PROJECTS_DIR="$HOME/projects"
BACKUP_DIR="$HOME/backups/env-files"
# Read password from environment variable
if [[ -z "$BACKUP_PASSWORD" ]]; then
    echo "Please set BACKUP_PASSWORD environment variable"
    exit 1
fi

PASSWORD="$BACKUP_PASSWORD"

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Backup all projects
for project in "$PROJECTS_DIR"/*; do
    if [[ -d "$project" ]]; then
        project_name=$(basename "$project")
        timestamp=$(date +%Y%m%d_%H%M%S)
        backup_file="$BACKUP_DIR/${project_name}_${timestamp}.enc"
        
        echo "Backing up: $project_name"
        cd "$project"
        
        if goingenv pack -k "$PASSWORD" -o "$backup_file" --description "Automated backup"; then
            echo "✅ $project_name backed up successfully"
        else
            echo "❌ Failed to backup $project_name"
        fi
    fi
done

echo "Backup completed. Files in: $BACKUP_DIR"
```

**CI/CD Integration:**
```yaml
# GitHub Actions example
name: Environment Backup
on:
  push:
    branches: [main]

jobs:
  backup:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install GoingEnv
        run: |
          curl -sSL https://raw.githubusercontent.com/spencerjirehcebrian/goingenv/main/install.sh | bash -s -- --yes
      
      - name: Create Environment Backup
        run: |
          goingenv pack -k "${{ secrets.BACKUP_PASSWORD }}" -o "production-backup.enc"
        env:
          BACKUP_PASSWORD: ${{ secrets.BACKUP_PASSWORD }}
      
      - name: Upload Backup Artifact
        uses: actions/upload-artifact@v3
        with:
          name: environment-backup
          path: ~/.goingenv/production-backup.enc
```

### Custom Configuration

Create a configuration file at `~/.goingenv/config.json`:

```json
{
  "default_depth": 3,
  "env_patterns": [
    ".env",
    ".env.local",
    ".env.production",
    ".env.staging",
    ".env.development",
    ".env.test",
    ".environment",
    "env.json"
  ],
  "exclude_patterns": [
    "node_modules/**",
    ".git/**",
    "vendor/**",
    "*.tmp",
    "*.log"
  ],
  "max_file_size": 1048576
}
```

## Configuration

### Global Settings

View current configuration:
```bash
goingenv status --verbose
```

### Archive Management

List all archives:
```bash
ls -la ~/.goingenv/*.enc
```

Archive naming convention:
- Default: `goingenv_YYYYMMDD_HHMMSS.enc`
- Custom: `your-name.enc`

### Security Considerations

**Password Management:**
- Use strong, unique passwords for each archive
- Consider using a password manager
- Avoid storing passwords in scripts or environment variables
- Use environment variables carefully for passwords

**File Permissions:**
```bash
# Secure archive directory
chmod 700 ~/.goingenv

# Secure environment variable handling
# Avoid storing passwords in shell history
```

## Troubleshooting

### Common Issues

**1. GoingEnv Not Initialized**
```bash
# Error: "GoingEnv is not initialized in this directory"
# Solution: Initialize GoingEnv first
goingenv init

# Check initialization status
goingenv status
```

**2. No Environment Files Found**
```bash
# Check current directory
goingenv status

# Check with verbose output
goingenv status --verbose

# Verify file patterns
ls -la .env*
```

**3. Permission Denied**
```bash
# Check directory permissions
ls -la ~/.goingenv

# Fix permissions
chmod 755 ~/.goingenv
```

**4. Archive Corruption**
```bash
# Verify archive integrity
goingenv list -f backup.enc --password-env MY_PASSWORD

# Check file size
ls -la ~/.goingenv/backup.enc
```

**5. Password Issues**
```bash
# Test password
goingenv list -f backup.enc --password-env MY_PASSWORD

# If password is forgotten, archive cannot be recovered
```

### Debug Mode

Enable verbose logging for troubleshooting:
```bash
# CLI debug mode
goingenv pack --password-env MY_PASSWORD --verbose

# TUI debug mode
goingenv --verbose
```

Debug logs are stored in: `~/.goingenv/debug/`

### Getting Help

1. **Check documentation**: `goingenv help`
2. **View command help**: `goingenv pack --help`
3. **Check GitHub issues**: https://github.com/spencerjirehcebrian/goingenv/issues
4. **Enable debug logging**: `goingenv --verbose`

## Performance Tips

1. **Optimize scan depth**: Use `--depth` flag to limit recursion
2. **Use exclude patterns**: Configure `.goingenv/config.json` to skip large directories
3. **Regular cleanup**: Remove old archives periodically
4. **Archive size**: Monitor archive sizes; large files may indicate issues

## Best Practices

1. **Regular Backups**: Automate daily/weekly backups
2. **Version Control**: Use descriptive archive names and descriptions
3. **Password Security**: Use unique, strong passwords
4. **Testing**: Regularly test restore procedures
5. **Documentation**: Document your backup and restore procedures
6. **Monitoring**: Check backup success in automated scripts