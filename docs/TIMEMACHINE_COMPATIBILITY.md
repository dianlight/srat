<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Time Machine Compatibility: Samba & macOS (Tahoe and Later)](#time-machine-compatibility-samba--macos-tahoe-and-later)
  - [Overview](#overview)
  - [Required Global smb.conf Options](#required-global-smbconf-options)
  - [Time Machine Share Options](#time-machine-share-options)
  - [Samba Version Compatibility](#samba-version-compatibility)
  - [macOS Version Compatibility](#macos-version-compatibility)
  - [Example Configuration](#example-configuration)
  - [References](#references)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Time Machine Compatibility: Samba & macOS (Tahoe and Later)

This document describes the required Samba configuration for robust Time Machine support on macOS 15+ (Tahoe) and later, including compatibility notes for older macOS versions and Samba releases.

## Overview

macOS 15+ (Tahoe) introduces stricter SMB protocol and signing requirements for Time Machine backups. Samba servers must be configured with the correct `fruit:*` and signing options to ensure reliable operation.

## Required Global smb.conf Options

Set these in the `[global]` section:

- `vfs objects = acl_xattr catia fruit streams_xattr`
- `fruit:aapl = yes`
- `fruit:model = MacSamba`
- `fruit:nfs_aces = no`
- `fruit:copyfile = yes`
- `fruit:resource = file`
- `fruit:metadata = stream`
- `fruit:veto_appledouble = no`
- `fruit:wipe_intentionally_left_blank_rfork = yes`
- `fruit:zero_file_id = yes`
- `fruit:delete_empty_adfiles = yes`
- `server signing = auto` (Samba >= 4.0)
- `ntlm auth = ntlmv2-only` (Samba >= 4.8; use `ntlm auth = yes` for NT1/legacy)
- `min protocol = SMB2_10` (or higher)
- `ea support = yes`

## Time Machine Share Options

Set these in the Time Machine share section (for example, `[TimeMachineBackup]`):

- `vfs objects = catia fruit streams_xattr`
- `fruit:time machine = yes`
- `fruit:time machine max size = <SIZE>` (optional, to limit backup size)

## Samba Version Compatibility

| Option                    | Minimum Samba Version | Notes                                |
| ------------------------- | --------------------- | ------------------------------------ |
| `fruit:posix_rename`      | 4.5 (removed in 4.23) | Needed for TM <4.23, safe to keep    |
| `server signing`          | 4.0                   | Required for macOS 15+               |
| `ntlm auth = ntlmv2-only` | 4.8                   | Use `ntlm auth = yes` for NT1/legacy |

## macOS Version Compatibility

| macOS Version | Notes                                                     |
| ------------- | --------------------------------------------------------- |
| 15+ (Tahoe)   | Requires SMB signing, NTLMv2, and all fruit options above |
| 11–14         | Works with above, but may be less strict                  |
| ≤10.15        | May require additional legacy options (NT1, etc.)         |

## Example Configuration

```ini
[global]
   vfs objects = acl_xattr catia fruit streams_xattr
   fruit:aapl = yes
   fruit:model = MacSamba
   fruit:nfs_aces = no
   fruit:copyfile = yes
   fruit:resource = file
   fruit:metadata = stream
   fruit:veto_appledouble = no
   fruit:wipe_intentionally_left_blank_rfork = yes
   fruit:zero_file_id = yes
   fruit:delete_empty_adfiles = yes
   server signing = auto
   ntlm auth = ntlmv2-only
   min protocol = SMB2_10
   ea support = yes

[TimeMachineBackup]
   vfs objects = catia fruit streams_xattr
   fruit:time machine = yes
   # fruit:time machine max size = 500G
```

## References

- [Samba vfs_fruit(8) man page](https://www.samba.org/samba/docs/current/man-html/vfs_fruit.8.html)
- [Samba Wiki: Configure Samba to Work Better with Mac OS X](https://wiki.samba.org/index.php/Configure_Samba_to_Work_Better_with_Mac_OS_X)
- [Apple: About Time Machine](https://support.apple.com/en-us/HT201250)
