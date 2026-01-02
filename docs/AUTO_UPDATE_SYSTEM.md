<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of Contents** *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [SRAT Auto-Update System](#srat-auto-update-system)
  - [Overview](#overview)
  - [Features](#features)
  - [How It Works](#how-it-works)
    - [Update Check Process](#update-check-process)
    - [Update Application Process](#update-application-process)
    - [Signature Verification](#signature-verification)
  - [Configuration](#configuration)
    - [Command Line Flags](#command-line-flags)
    - [Update Channels](#update-channels)
  - [Security](#security)
    - [Key Management](#key-management)
    - [Signature Verification Process](#signature-verification-process)
    - [Threat Model](#threat-model)
  - [API Endpoints](#api-endpoints)
    - [Check for Updates](#check-for-updates)
    - [Apply Update](#apply-update)
    - [Get Update Channels](#get-update-channels)
  - [Build Process Integration](#build-process-integration)
    - [Build Workflow Steps](#build-workflow-steps)
  - [S6 Integration](#s6-integration)
  - [Troubleshooting](#troubleshooting)
    - [Update fails with signature verification error](#update-fails-with-signature-verification-error)
    - [Update downloads but doesn't apply](#update-downloads-but-doesnt-apply)
    - [Service doesn't restart after update](#service-doesnt-restart-after-update)
    - [Development/Testing](#developmenttesting)
  - [Maintenance](#maintenance)
    - [Rotating Keys](#rotating-keys)
    - [Monitoring Updates](#monitoring-updates)
  - [Architecture Decisions](#architecture-decisions)
    - [Why minio/selfupdate?](#why-minioselfupdate)
    - [Why minisign over other signature schemes?](#why-minisign-over-other-signature-schemes)
    - [Why embed the public key?](#why-embed-the-public-key)
  - [Related Files](#related-files)
  - [References](#references)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# SRAT Auto-Update System

## Overview

SRAT uses a secure auto-update mechanism based on `minio/selfupdate` with cryptographic signature verification using minisign (Ed25519). This ensures that updates are authentic and haven't been tampered with.

## Features

- **Automatic Updates**: Optional `--auto-update` flag to automatically download and apply updates without user intervention
- **Signature Verification**: All release binaries are signed with minisign for cryptographic verification
- **Smart Restart**: Automatically restarts the service when running under s6 supervision
- **Multiple Update Channels**: Support for release, prerelease, and development channels
- **Rollback Support**: Automatic rollback if update fails to apply

## How It Works

### Update Check Process

1. SRAT periodically checks GitHub releases for newer versions (every 30 minutes)
2. Filters releases based on the configured update channel (release/prerelease/develop)
3. Compares available versions with the current binary version
4. Notifies users when an update is available

### Update Application Process

1. **Download**: Downloads the appropriate architecture-specific binary package (.zip file)
2. **Extract**: Extracts the package to a temporary directory
3. **Verify**: Verifies the binary signature using the embedded public key (if signature file present)
4. **Apply**: Uses `selfupdate.Apply()` to atomically replace the current binary
5. **Restart**: If running under s6, exits with code 0 to trigger s6 restart; otherwise requires manual restart

### Signature Verification

- All release binaries are signed during the build process using minisign
- The public key is embedded in the SRAT binary at build time
- Signature files (`.minisig`) are included in the release archives
- If signature verification fails, the update is rejected and rolled back
- Development builds without signatures will proceed without verification (with a warning)

## Configuration

### Command Line Flags

```bash
# Enable automatic updates (no user confirmation required)
srat-server --auto-update

# Set update channel
srat-server --update-channel=release      # Stable releases only (default)
srat-server --update-channel=prerelease   # Include pre-releases
srat-server --update-channel=develop      # Development builds (not recommended for production)
```

### Update Channels

- **release**: Stable releases only (recommended for production)
- **prerelease**: Includes beta/RC releases for early access to features
- **develop**: Development builds (for testing, may be unstable)
- **none**: Disables update checking

## Security

### Key Management

- **Public Key**: Embedded in the SRAT binary at build time (`backend/src/internal/updatekey/update-public-key.pub`)
- **Private Key**: Stored securely in GitHub secrets (`UPDATE_SIGNING_KEY`)
- **Key Format**: Minisign format (Ed25519)
- **Key Generation**: Use `scripts/generate-update-keys.sh` to generate a new keypair

### Signature Verification Process

1. During update, SRAT looks for a `.minisig` file alongside the binary
2. Loads the embedded public key
3. Uses minio/selfupdate's Verifier to check the signature
4. Only proceeds if signature is valid or if running in development mode without signatures

### Threat Model

- **Prevents**: Man-in-the-middle attacks, binary tampering, unauthorized updates
- **Requires**: Attacker would need access to the private key (stored in GitHub secrets)
- **Fallback**: Development/unsigned builds skip verification with a warning log

## API Endpoints

### Check for Updates

```http
GET /update
```

Returns information about available updates.

**Response**:

```json
{
  "LastRelease": "2025.1.0",
  "ArchAsset": {
    "Name": "srat_x86_64.zip",
    "BrowserDownloadURL": "https://github.com/dianlight/srat/releases/download/2025.1.0/srat_x86_64.zip",
    "Size": 12345678
  }
}
```

### Apply Update

```http
PUT /update
```

Triggers the update process. Returns immediately and updates in the background.

**Response**:

```json
{
  "ProgressStatus": "DOWNLOADING",
  "Progress": 0,
  "LastRelease": "2025.1.0"
}
```

### Get Update Channels

```http
GET /update_channels
```

Returns available update channels.

## Build Process Integration

The GitHub Actions workflow automatically:

1. Builds binaries for all supported architectures (amd64, aarch64)
2. Installs minisign
3. Signs each binary using the private key from secrets
4. Creates `.minisig` signature files
5. Packages binaries and signatures into architecture-specific ZIP files
6. Uploads as release assets

### Build Workflow Steps

```yaml
- name: Sign binaries with Minisign
  env:
    UPDATE_SIGNING_KEY: ${{ secrets.UPDATE_SIGNING_KEY }}
  run: |
    # Install minisign
    # Sign each binary
    minisign -S -s $PRIVATE_KEY_FILE -m $binary_path -x ${binary_path}.minisig
```

## S6 Integration

When SRAT detects it's running under s6 supervision:

1. Checks for `S6_VERSION` environment variable
2. Checks if parent process is `s6-supervise` (via `/proc/[ppid]/cmdline`)
3. If detected, exits with code 0 after successful update
4. S6 automatically restarts the service with the new binary

## Troubleshooting

### Update fails with signature verification error

- Check that the public key in the binary matches the key used to sign the release
- Verify the `.minisig` file is present in the update package
- Check logs for detailed error messages

### Update downloads but doesn't apply

- Check file permissions on the binary directory
- Verify sufficient disk space
- Check logs for rollback messages

### Service doesn't restart after update

- Verify s6 is running and configured correctly
- Check `S6_VERSION` environment variable
- Manually restart if not running under s6

### Development/Testing

For development builds without signatures:

1. Update proceeds with a warning log message
2. Signature verification is skipped
3. Update applies normally

## Maintenance

### Rotating Keys

If you need to rotate the signing keys:

1. Generate new keypair: `./scripts/generate-update-keys.sh --add-secret`
2. Update `UPDATE_SIGNING_KEY` secret in GitHub
3. Commit new public key files
4. Next release will use the new keys
5. **Important**: Old binaries can't verify updates signed with new keys

### Monitoring Updates

- Check logs for update check activity (every 30 minutes)
- Monitor GitHub API rate limits
- Track update success/failure rates via telemetry (if enabled)

## Architecture Decisions

### Why minio/selfupdate?

- Battle-tested library used by MinIO and other projects
- Built-in signature verification with minisign
- Atomic updates with automatic rollback
- Cross-platform support
- Active maintenance

### Why minisign over other signature schemes?

- Simpler than GPG (single-purpose tool)
- Small key sizes (Ed25519)
- Fast signature generation and verification
- Good compatibility with minio/selfupdate
- Widely adopted for binary signing

### Why embed the public key?

- Ensures the binary can always verify updates
- No external dependencies or key distribution needed
- Simplifies deployment
- Public key is not secret, so embedding is safe

## Related Files

- `backend/src/service/upgrade_service.go` - Update service implementation
- `backend/src/api/upgrade.go` - Update API handlers
- `backend/src/internal/updatekey/public_key.go` - Embedded public key
- `backend/src/internal/updatekey/update-public-key.pub` - Public key file
- `scripts/generate-update-keys.sh` - Key generation script
- `.github/workflows/build.yaml` - Build and signing workflow

## References

- [minio/selfupdate](https://github.com/minio/selfupdate) - Update library
- [minisign](https://jedisct1.github.io/minisign/) - Signature tool
- [s6](https://skarnet.org/software/s6/) - Process supervision
