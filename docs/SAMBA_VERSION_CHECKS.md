<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Samba Version Checks in SRAT](#samba-version-checks-in-srat)
  - [Overview](#overview)
  - [Architecture](#architecture)
    - [Backend Components](#backend-components)
      - [1. Version Detection (`backend/src/internal/osutil/osutil.go`)](#1-version-detection-backendsrcinternalosutilosutilgo)
      - [2. Template Context Enhancement (`backend/src/service/samba_service.go`)](#2-template-context-enhancement-backendsrcservicesamba_servicego)
      - [3. Template Functions (`backend/src/tempio/template.go`)](#3-template-functions-backendsrctempiotemplatego)
    - [Template Configuration (`backend/src/templates/smb.gtpl`)](#template-configuration-backendsrctemplatessmbgtpl)
  - [Version Feature Mapping](#version-feature-mapping)
    - [Samba 4.21.0](#samba-4210)
    - [Samba 4.22.0](#samba-4220)
    - [Samba 4.23.0](#samba-4230)
  - [SRAT Template Updates](#srat-template-updates)
    - [Critical Changes in smb.gtpl](#critical-changes-in-smbgtpl)
  - [Configuration Behavior by Samba Version](#configuration-behavior-by-samba-version)
    - [Samba 4.20.x (Not Explicitly Supported)](#samba-420x-not-explicitly-supported)
    - [Samba 4.21.x](#samba-421x)
    - [Samba 4.22.x](#samba-422x)
    - [Samba 4.23.x+ (Current Stable)](#samba-423x-current-stable)
  - [Error Handling](#error-handling)
    - [Template Rendering Errors](#template-rendering-errors)
    - [Safe Defaults](#safe-defaults)
  - [Debugging Version Issues](#debugging-version-issues)
    - [Check Detected Samba Version](#check-detected-samba-version)
    - [Manual Version Check](#manual-version-check)
    - [Enable Debug Logging](#enable-debug-logging)
    - [Test Configuration](#test-configuration)
  - [Maintenance Guide](#maintenance-guide)
    - [Adding New Version Checks](#adding-new-version-checks)
    - [Testing Across Versions](#testing-across-versions)
  - [Related Documentation](#related-documentation)
  - [References](#references)
    - [Samba Feature Timeline](#samba-feature-timeline)
    - [Configuration Files](#configuration-files)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Samba Version Checks in SRAT

## Overview

This document describes the Samba version checking infrastructure implemented in SRAT to ensure smb.conf configuration options are only used with compatible Samba versions. This prevents configuration errors when deploying SRAT across different Samba versions.

## Architecture

### Backend Components

#### 1. Version Detection (`backend/src/internal/osutil/osutil.go`)

The `osutil` package provides functions to detect and compare Samba versions:

- **`GetSambaVersion()`**: Executes `smbd --version` and parses the version string (for example, "4.23.0")
- **`IsSambaVersionSufficient()`**: Returns true if version >= 4.23.0 (minimum QUIC support)
- **`IsSambaVersionAtLeast(majorRequired, minorRequired)`**: Generic version comparison function

```go
// Example usage:
version, _ := osutil.GetSambaVersion()              // Returns "4.23.0"
sufficient, _ := osutil.IsSambaVersionSufficient() // Returns true if >= 4.23.0
atLeast, _ := osutil.IsSambaVersionAtLeast(4, 21)  // Returns true if >= 4.21.0
```

#### 2. Template Context Enhancement (`backend/src/service/samba_service.go`)

The `CreateConfigStream()` method in `SambaService` now enriches the template context with Samba version information:

```go
sambaVersion, _ := osutil.GetSambaVersion()
isSambaVersionSufficient, _ := osutil.IsSambaVersionSufficient()
(*config_2)["samba_version"] = sambaVersion
(*config_2)["samba_version_sufficient"] = isSambaVersionSufficient
```

This makes version information available as template variables:

- `{{ .samba_version }}` - The detected Samba version string
- `{{ .samba_version_sufficient }}` - Boolean flag for Samba >= 4.23.0

#### 3. Template Functions (`backend/src/tempio/template.go`)

Custom template functions for version comparison:

**`versionAtLeast(versionStr, majorRequired, minorRequired)`**

- Checks if a version string meets or exceeds the required major.minor version
- Returns: `bool`
- Usage: `{{ if versionAtLeast .samba_version 4 23 }}...{{ end }}`

**`versionBetween(versionStr, minMajor, minMinor, maxMajor, maxMinor)`**

- Checks if a version is within a specified range (inclusive)
- Returns: `bool`
- Usage: `{{ if versionBetween .samba_version 4 21 4 22 }}...{{ end }}`

### Template Configuration (`backend/src/templates/smb.gtpl`)

The Samba configuration template uses version checks to conditionally include options:

```go
{{if versionAtLeast .samba_version 4 23 -}}
server smb transports = tcp{{if .smb_over_quic -}}, quic{{- end }}
{{- end }}
```

## Version Feature Mapping

### Samba 4.21.0

**New Parameters:**

- `client ldap sasl wrapping` (new values: `starttls`, `ldaps`)
- `dns hostname`
- `sync machine password to keytab`
- `sync machine password script`
- `tls trust system cas` (TLS option)
- `tls ca directories` (TLS option)

**Modified Parameters:**

- `valid users` - Hardening: non-existing users now cause errors on communication failure
- `invalid users` - Hardening: non-existing users now cause errors on communication failure
- `read list` - Hardening: non-existing users now cause errors on communication failure
- `write list` - Hardening: non-existing users now cause errors on communication failure
- `veto files` - Now supports per-user and per-group specifications
- `hide files` - Now supports per-user and per-group specifications

**Removed Parameters:**

- `client use spnego principal` (removed)

### Samba 4.22.0

**New Parameters:**

- `smb3 directory leases` - Enable/disable SMB3 directory leases (default: auto)
- `client netlogon ping protocol` - Protocol for netlogon pings (default: cldap)
- `vfs mkdir use tmp name` - Use temporary names for mkdir operations
- `himmelblaud_hello_enabled` - Enable Himmelblaud authentication
- `himmelblaud_hsm_pin_path` - HSM pin path for Himmelblaud
- `himmelblaud_sfa_fallback` - SFA fallback for Himmelblaud
- `client use krb5 netlogon` (Experimental)
- `reject aes netlogon servers` (Experimental)
- `server reject aes schannel` (Experimental)
- `server support krb5 netlogon` (Experimental)

**Removed Parameters:**

- `fruit:posix_rename` - Removed (causes issues with Windows clients)
- `cldap port` - Removed (runs on fixed UDP 389)

### Samba 4.23.0

**New Parameters:**

- `server smb transports` - Specifies SMB transports (tcp, nbt, quic)
- `client smb transports` - Client-side SMB transports
- `smbd profiling share` - Per-share profiling statistics
- `winbind varlink service` - Varlink service for winbind

**Behavioral Changes:**

- SMB3 Unix Extensions enabled by default
- Modern write time update logic (immediate updates instead of delayed)
- Directory leases auto-tuned based on clustering config

## SRAT Template Updates

### Critical Changes in smb.gtpl

**1. SMB Transports (Samba 4.23+)**

```go
{{if versionAtLeast .samba_version 4 23 -}}
server smb transports = tcp{{if .smb_over_quic -}}, quic{{- end }}
{{- end }}
```

Reason: The `server smb transports` option was introduced in Samba 4.23. Earlier versions would reject this configuration, and SMB over QUIC is not supported below 4.23.

**2. Fruit posix_rename (Samba < 4.22)**

```go
{{if versionAtLeast .samba_version 4 22 -}}
{{- else -}}
fruit:posix_rename = yes
{{- end }}
```

Reason: This option was removed in Samba 4.22 due to issues with Windows clients. If you're on 4.22+, this line is excluded.

**3. QUIC Configuration (Samba 4.23+)**

```go
{{if .smb_over_quic -}}
{{if versionAtLeast .samba_version 4 23 -}}
server smb3 encryption = mandatory
smb3 unix extensions = yes
tls enable = yes
...
{{- else -}}
# WARNING: SMB over QUIC requires Samba 4.23.0+
{{- end }}
{{- end }}
```

Reason: SMB over QUIC with TLS configuration is only available in Samba 4.23+. The template will warn if this is attempted on earlier versions.

## Configuration Behavior by Samba Version

### Samba 4.20.x (Not Explicitly Supported)

- Uses legacy configuration style
- No version checks apply
- Limited to standard SMB3 features

### Samba 4.21.x

- Enhanced user/group validation
- TLS channel binding support for LDAP
- Keytab synchronization available
- Still uses standard SMB transports

### Samba 4.22.x

- Directory leases support
- Himmelblaud authentication available (experimental)
- `fruit:posix_rename` removed - **template excludes this option**
- Enhanced VFS options

### Samba 4.23.x+ (Current Stable)

- Full SMB3 Unix Extensions by default
- SMB over QUIC support
- New transport configuration options
- Per-share profiling available
- Modern write time update logic
- **All SRAT features fully supported**

## Error Handling

### Template Rendering Errors

If version information cannot be determined:

- `{{ .samba_version }}` will be empty or "0.0.0"
- `versionAtLeast()` will return `false` for any comparison
- Configuration will fall back to conservative defaults

### Safe Defaults

SRAT uses a "safe by default" approach:

- If version cannot be determined, version-specific options are excluded
- Fallback to widely-compatible options
- Comments in template warn about version mismatches

Example:

```txt
# WARNING: SMB over QUIC requires Samba 4.23.0+. Current version: <unknown>
# Falling back to standard SMB3 configuration
```

## Debugging Version Issues

### Check Detected Samba Version

The `/health` API endpoint includes version information:

```bash
curl http://localhost:8000/health
```

Response:

```json
{
  "samba_version": "4.23.0",
  "samba_version_sufficient": true,
  "supports_quic": true,
  ...
}
```

### Manual Version Check

```bash
smbd --version
```

Output:

```txt
Version 4.23.0
```

### Enable Debug Logging

Set `log_level: "debug"` in configuration to see template rendering details.

### Test Configuration

```bash
testparm -s /etc/samba/smb.conf
```

## Maintenance Guide

### Adding New Version Checks

When updating SRAT for a new Samba release:

1. **Identify new/removed/changed parameters** from Samba release notes
2. **Update smb.gtpl** with version-conditional blocks:

   ```go
   {{if versionAtLeast .samba_version 4 24 -}}
   new_option = value
   {{- end }}
   ```

3. **Update osutil.go** if new comparison functions are needed
4. **Document in CHANGELOG.md** which Samba versions are supported
5. **Test with all supported Samba versions** (4.21, 4.22, 4.23+)

### Testing Across Versions

To test SRAT configuration generation with different Samba versions:

```bash
# Temporarily mock version for testing
# (Edit CreateConfigStream() to use a test version)
sambaVersion := "4.21.0"  // Test value

# Regenerate config
curl -X POST http://localhost:8000/samba/config
```

## Related Documentation

- [Samba Release Notes](<https://wiki.samba.org/index.php/Samba_Features_added/changed_(by_release)>)
- [SMB over QUIC Implementation](./SMB_OVER_QUIC_IMPLEMENTATION.md)
- [SRAT Backend Architecture](../README.md)

## References

### Samba Feature Timeline

| Version | Release Date | Key Features                              |
| ------- | ------------ | ----------------------------------------- |
| 4.21.0  | Sept 2024    | Enhanced validation, TLS improvements     |
| 4.22.0  | March 2025   | Directory leases, Himmelblaud support     |
| 4.23.0  | Sept 2025    | SMB over QUIC, Unix Extensions by default |

### Configuration Files

- Template: `backend/src/templates/smb.gtpl`
- Version Logic: `backend/src/internal/osutil/osutil.go`
- Template Functions: `backend/src/tempio/template.go`
- Template Rendering: `backend/src/service/samba_service.go`
