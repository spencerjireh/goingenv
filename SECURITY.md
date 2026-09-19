# Security

goingenv encrypts your `.env` files into a single archive so you can commit it
alongside your code. This document describes what that protects, what it does
not, and how to check what you downloaded.

## Status

This project was developed with AI assistance and has not undergone a formal
security audit. The cryptography is Go's `crypto/aes` and `crypto/cipher` with
`golang.org/x/crypto/argon2`, composed by hand. Nobody qualified has reviewed
that composition. Assess it yourself before trusting it with anything you cannot
afford to leak.

Committing an archive publishes its ciphertext. That is the point of the tool,
and it means the password is the only thing between a public repository and your
secrets -- permanently. A leaked password compromises every archive still in the
git history, including ones you have since replaced.

## What it protects

- **File contents at rest.** Env files are encrypted with AES-256-GCM before
  they are written. GCM authenticates as well as encrypts, so tampering with an
  archive causes decryption to fail rather than yield altered plaintext.
- **File names.** Paths live inside the encrypted tar, not in a plaintext
  header. An archive reveals nothing about which files it holds. The optional
  manifest (`pack --manifest`) deliberately publishes paths and key names for
  review; it is off by default for that reason.
- **Integrity of the restored files.** SHA-256 is recorded per file before
  encryption and checked on extraction.

## What it does not protect

- **A committed archive is offline-attackable forever.** Anyone who clones the
  repository can attempt passwords at their own pace, on their own hardware,
  with no rate limit. Password strength is the whole of your security.
- **There is no key rotation.** Re-packing under a new password does not
  invalidate the old archive; it is still in the git history under the old one.
- **One shared password.** Everyone who can decrypt an archive holds the same
  secret, and it has to travel to each of them out of band. Removing a
  teammate means re-packing under a new password and re-sharing it with
  everyone else. Tools that wrap a per-archive key for each recipient's public
  key avoid both problems; goingenv reserves a key-mode byte in its header so
  that can be added later, but today it does not have it.
- **Metadata leaks through git, not the archive.** The archive's existence, its
  size, and the commit timestamps around it are all visible.
- **Passwords in memory.** They are zeroed after use, but Go can move a string
  before that, and a core dump or swap file may retain one.
- **The machine itself.** Physical access, malware with filesystem access, or a
  shoulder-surfed prompt all defeat the tool entirely.
- **Side channels.** No constant-time or timing-attack hardening beyond what the
  standard library provides.

## Crypto parameters

| | |
|---|---|
| Cipher | AES-256-GCM |
| Key derivation | Argon2id, t=3, m=64 MiB, p=4 (RFC 9106 second recommended option) |
| Key size | 32 bytes |
| Salt | 32 bytes, random per archive |
| Nonce | 12 bytes, random per archive |
| Integrity | GCM tag over the archive and its header; SHA-256 per file inside it |
| Randomness | `crypto/rand` |

The Argon2id parameters are written into each archive's header and read back
on decrypt, so a future release can raise them without a format bump. No flag,
config key or environment variable changes the parameters a build writes.
Argon2id is memory-hard, which narrows the gap between an attacker with a
committed archive and one attacking an interactive login, but it does not
close it: password strength still matters.

Because the key must be derived before the GCM tag can be checked, a hostile
header could ask for unbounded work. Decrypt therefore refuses parameters
outside these caps before deriving anything: time 1..32, memory at most 1 GiB
and at least 8 KiB per thread, threads 1..32. The caps are part of the format
contract.

## Archive format

Format v1, big-endian:

```
offset size field
0      4    magic "GENV"
4      1    format version (1)
5      1    KDF id (1 = Argon2id)
6      4    Argon2 time
10     4    Argon2 memory in KiB
14     1    Argon2 threads
15     1    key mode (0 = password; reserved for per-recipient wrapping)
16     32   salt
48     12   GCM nonce
60     ..   AES-256-GCM(tar) followed by the 16-byte tag
```

Bytes 0..59 are the GCM additional data, so a change to the header, salt or
nonce fails authentication rather than altering how the key is derived.

One seal covers the entire tar; files are not encrypted individually. Inside
the tar, the first entry is `metadata.json` -- metadata schema version,
environment name, creation time, description, and per-file path, size,
modification time and SHA-256. The remaining entries are the files themselves
at their paths relative to the project root. Nothing is compressed.

The header identifies an archive as goingenv's and states its parameters
without the password. An archive is therefore not indistinguishable from random
data; the format version and KDF id are what let the tool refuse something it
cannot read with a specific message instead of a generic decryption failure.

### Compatibility

Archives written by goingenv 1.x have no header: they are
`salt || nonce || ciphertext` with PBKDF2-HMAC-SHA256 at 100,000 iterations.
Version 2.0.0 does not read them. Unpack them with goingenv v1.6.0, then
re-pack with this version; the error message says so when it meets one.

## Verifying a download

Every release publishes a `checksums.txt` listing the SHA-256 of each archive.

### Installing with the install script

The install script verifies the archive against `checksums.txt` before
extracting it. A mismatch aborts the install and no binary is written:

```bash
curl -sSL https://raw.githubusercontent.com/spencerjireh/goingenv/main/install.sh | bash
```

Verification is not optional by default. If the checksum cannot be retrieved,
the install fails rather than proceeding unverified. Every published release
includes `checksums.txt`, so a failure here means a corrupted download, a
network problem, or a tampered file. `--skip-checksum` exists as an escape
hatch for constrained environments and prints a warning when used:

```bash
curl -sSL .../install.sh | bash -s -- --skip-checksum
```

The script also requires `sha256sum` or `shasum` to be present, and checks for
one before downloading anything.

### Verifying a manual download

```bash
# Download the archive and the checksum manifest for your platform
curl -sSLO https://github.com/spencerjireh/goingenv/releases/download/<tag>/goingenv-<tag>-<os>-<arch>.tar.gz
curl -sSLO https://github.com/spencerjireh/goingenv/releases/download/<tag>/checksums.txt

# Verify (use shasum -a 256 -c on macOS)
sha256sum -c checksums.txt
```

Releases also carry a build provenance attestation:

```bash
gh attestation verify <archive> --repo spencerjireh/goingenv
```

## Passwords

There is no `--password` flag, deliberately: it would put the password in your
shell history and in the process list, where any other user on the machine can
read it. Two ways in:

```bash
goingenv pack                                  # interactive prompt, not echoed
GOINGENV_PASSWORD=... goingenv pack --password-env GOINGENV_PASSWORD
```

The environment variable is for CI. It is visible to other processes on the same
machine and to anything that dumps the environment, so prefer the prompt when a
human is present. goingenv warns when you use it.

The password is never written anywhere. Losing it means losing the archive --
there is no recovery path, by design.

## Reporting a vulnerability

Report privately, not as a public issue:

- Email: email@spencerjireh.com
- GitHub: [private security advisory](https://github.com/spencerjireh/goingenv/security/advisories/new)

Include what the vulnerability is, how to reproduce it, what an attacker gains,
and a suggested fix if you have one. This is a single-maintainer project, so
expect a reply in days rather than hours. Fixes ship as patch releases on the
current major version, with a GitHub security advisory.

## Supported versions

| Version | Supported |
|---|---|
| 1.x.x | Yes |
| 0.x.x | No |
