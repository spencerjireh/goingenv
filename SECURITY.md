# Security Guide

This document outlines security considerations, best practices, and implementation details for goingenv.

## ⚠️ Security Notice

> **WARNING**: goingenv has not been security audited but has been tested in production environments. Use at your own risk and avoid use in public repositories. Always review the code and consult with security professionals before using in production systems.

## Table of Contents

- [Security Model](#security-model)
- [Encryption Details](#encryption-details)
- [Best Practices](#best-practices)
- [Threat Model](#threat-model)
- [Reporting Security Issues](#reporting-security-issues)

## Security Model

### What goingenv Protects

- **Data at Rest**: Environment files are encrypted using AES-256-GCM
- **File Integrity**: SHA-256 checksums detect tampering
- **Password-Based Security**: Strong key derivation using PBKDF2
- **Local Storage**: Encrypted archives stored locally

### What goingenv Does NOT Protect

- **Data in Transit**: No network transmission (local tool only)
- **Memory Protection**: Passwords/keys may be visible in memory during operation
- **Physical Access**: Cannot protect against physical compromise of the system
- **Weak Passwords**: Security depends on password strength
- **Side-Channel Attacks**: No protection against timing attacks, etc.

## Encryption Details

### Encryption Algorithm

**Primary Encryption**: AES-256-GCM
- **Key Size**: 256 bits (32 bytes)
- **Mode**: Galois/Counter Mode (GCM)
- **Authentication**: Built-in authenticated encryption
- **Nonce**: 96-bit random nonce per encryption

**Key Derivation**: PBKDF2-SHA256
- **Iterations**: 100,000 (configurable)
- **Salt**: 128-bit (16 bytes) random salt per archive
- **Hash Function**: SHA-256
- **Output**: 256-bit encryption key

### File Integrity

**Checksum Algorithm**: SHA-256
- Applied to original file content before encryption
- Stored in archive metadata
- Verified during extraction

### Random Number Generation

- Uses Go's `crypto/rand` package
- Backed by OS entropy sources
- Used for salts, nonces, and IV generation

### Archive Format

```
Archive Structure:
├── Header
│   ├── Version (4 bytes)
│   ├── Salt (16 bytes)
│   └── Metadata Length (4 bytes)
├── Encrypted Metadata
│   ├── Archive Info (JSON)
│   └── File List with Checksums
└── Encrypted File Data
    ├── File 1 (AES-256-GCM)
    ├── File 2 (AES-256-GCM)
    └── ...
```

## Best Practices

### Password Security

**Secure Password Input Methods:**
```bash
# 1. Interactive prompt (most secure)
goingenv pack  # Will prompt securely for password

# 2. Environment variable (for automation)
export MY_PASSWORD="coffee-mountain-bicycle-sunshine-42"
goingenv pack --password-env MY_PASSWORD
unset MY_PASSWORD  # Clear after use
```

**Password Management:**
- **Never use command-line passwords** - would be visible in shell history and process lists
- Use environment variables carefully - visible to other processes
- Interactive prompts are most secure for manual operations
- Store passwords in secure password managers
- Rotate passwords regularly
- Clear environment variables after use

### Secure File Handling

**File Permissions:**
```bash
# Secure archive directory
chmod 700 ~/.goingenv

# Secure individual archives
chmod 600 ~/.goingenv/*.enc
```

**Temporary Files:**
- goingenv uses secure temporary directories
- Temporary files are cleaned up automatically
- Avoid interrupting operations to prevent temp file leaks

### Operational Security

**Environment Isolation:**
```bash
# Use dedicated environment for sensitive operations
unset HISTFILE  # Disable shell history

# Use environment variables for automation
export BACKUP_PASSWORD="secure-password"
goingenv pack --password-env BACKUP_PASSWORD
unset BACKUP_PASSWORD
```

**Regular Cleanup:**
```bash
# Remove old archives
find ~/.goingenv -name "*.enc" -mtime +30 -delete

# Clear debug logs
rm -rf ~/.goingenv/debug/*

# Clear shell history if needed
history -c
```

**Secure Deletion:**
```bash
# Use secure deletion tools for sensitive files
shred -vfz -n 3 old-archive.enc

# Or on macOS
rm -P old-archive.enc
```

### Backup Security

**Secure Backup Storage:**
- Store backups on encrypted filesystems
- Use additional encryption layers for cloud storage
- Consider offline storage for critical backups

**Access Control:**
- Limit access to backup locations
- Use separate passwords for different environments
- Implement backup retention policies

## Threat Model

### Assets

1. **Environment Files**: Sensitive configuration data (.env files)
2. **Passwords**: Encryption keys for archives
3. **Archives**: Encrypted backup files
4. **Metadata**: File paths, timestamps, checksums

### Threats

**High Risk:**
- Password compromise (weak passwords, credential theft)
- Physical access to unlocked system
- Memory dumps containing passwords/keys
- Malicious software with file system access

**Medium Risk:**
- Archive file theft (without password)
- Metadata leakage (file paths, timestamps)
- Side-channel attacks during encryption/decryption
- Brute force attacks on archives

**Low Risk:**
- Network interception (local tool only)
- Timing attacks on password verification
- Archive format analysis

### Mitigations

**Password Protection:**
- Enforce strong password policies
- Use secure password entry (hidden input)
- Clear password variables after use
- Avoid command-line password exposure
- Monitor environment variable usage
- Use interactive prompts for maximum security

**File System Security:**
- Use appropriate file permissions
- Store archives in secure locations
- Implement secure deletion
- Regular security audits

**Operational Security:**
- Security awareness training
- Regular password rotation
- Incident response procedures
- Secure development practices

## Security Implementation

### Password Security Enhancements

goingenv has been enhanced with secure password handling to eliminate command-line password exposure:

**Secure Password Input Methods:**
```go
// Password options with validation
type Options struct {
    PasswordEnv string // Environment variable (with warnings)
}

// Secure memory clearing
func ClearPassword(password *string) {
    bytes := []byte(*password)
    for i := range bytes {
        bytes[i] = 0  // Zero out memory
    }
    *password = ""
}

// Environment variable validation
if strings.TrimSpace(opts.PasswordEnv) == "" {
    return fmt.Errorf("environment variable name cannot be empty")
}
```

**Security Improvements:**
- **No command-line password exposure** - passwords never visible in shell history or process lists
- **Memory clearing** - ensures passwords are zeroed after use
- **Environment variable warnings** - alerts users to potential security risks
- **Input priority system** - environment variable → interactive prompt
- **Simplified attack surface** - minimal password input methods for reduced risk

### Code Security

**Memory Safety:**
```go
// Clear sensitive data from memory
defer func() {
    for i := range password {
        password[i] = 0
    }
}()

// Use secure random generation
salt := make([]byte, 16)
if _, err := rand.Read(salt); err != nil {
    return err
}
```

**Input Validation:**
```go
// Validate file paths
if !filepath.IsAbs(archivePath) {
    return errors.New("archive path must be absolute")
}

// Sanitize user input
filename = filepath.Clean(filename)
```

**Error Handling:**
```go
// Don't leak sensitive information in errors
if err := decrypt(data, key); err != nil {
    return errors.New("decryption failed")  // Generic error
}
```

### Dependencies

**Security Scanning:**
```bash
# Scan for vulnerabilities
go list -json -deps ./... | nancy sleuth

# Check for security updates
go list -u -m all
```

**Minimal Dependencies:**
- Use only necessary dependencies
- Regularly update dependencies
- Review dependency security advisories

## Vulnerability Disclosure

### Reporting Security Issues

**Contact Information:**
- **Email**: security@[your-domain].com
- **GPG Key**: [Public key fingerprint]
- **GitHub**: Private security advisory

**What to Include:**
1. Description of the vulnerability
2. Steps to reproduce
3. Potential impact assessment
4. Suggested fixes (if any)
5. Your contact information

**Response Timeline:**
- **Initial Response**: Within 48 hours
- **Assessment**: Within 1 week
- **Fix Development**: Within 2 weeks
- **Public Disclosure**: After fix is released

### Supported Versions

| Version | Supported |
|---------|-----------|
| 1.x.x   | ✅ Yes    |
| 0.x.x   | ❌ No     |

### Security Updates

Security updates are published as:
- Patch releases for current major version
- Security advisories on GitHub
- Release notes with security details

## Security Checklist

### For Users

- [ ] Use strong, unique passwords
- [ ] Store archives in secure locations
- [ ] Set proper file permissions
- [ ] Regularly update goingenv
- [ ] Monitor for security advisories
- [ ] Use secure backup practices
- [ ] Avoid password reuse
- [ ] Clear sensitive data from shell history

### For Developers

- [ ] Follow secure coding practices
- [ ] Review all dependencies
- [ ] Implement input validation
- [ ] Use secure random generation
- [ ] Clear sensitive data from memory
- [ ] Handle errors securely
- [ ] Regular security testing
- [ ] Keep dependencies updated

### For System Administrators

- [ ] Implement file system encryption
- [ ] Configure proper access controls
- [ ] Monitor for unusual activity
- [ ] Establish incident response procedures
- [ ] Regular security audits
- [ ] Backup security validation
- [ ] User security training
- [ ] Network security controls

## Security Resources

### Documentation
- [Go Security Checklist](https://github.com/securego/gosec)
- [OWASP Cryptographic Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)

### Tools
- **gosec**: Go security analyzer
- **nancy**: Dependency vulnerability scanner
- **git-secrets**: Prevent secrets in git
- **truffleHog**: Find secrets in repositories

### Standards
- **FIPS 140-2**: Cryptographic module standards
- **NIST SP 800-132**: Password-based key derivation
- **RFC 3394**: AES key wrap specification
- **RFC 5652**: Cryptographic message syntax

## Disclaimer

This security guide is provided for informational purposes only. Security is a complex topic and this guide does not guarantee complete security. Users should perform their own security assessments and consult with security professionals for sensitive applications.

The goingenv project makes no warranties about the security of the software and users assume all risks associated with its use.