<!-- START doctoc generated TOC please keep comment here to allow auto update -->
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

The generated configuration loads `acl_xattr catia fruit streams_xattr` for ordinary non-FAT shares as well as eligible Time Machine shares. Samba requires `fruit` to be stacked with `streams_xattr`; `fruit:metadata = stream` delegates Finder metadata to that module. Resource forks continue to use AppleDouble files (`fruit:resource = file`). Share-level `vfs objects` replaces the global list, so enabling the recycle bin must preserve the complete stack and append `recycle` as a separate module.

**FAT limitation:** Shares identified as `exfat`, `vfat`, or `msdos` retain the previous `acl_xattr catia fruit` stack and do not receive `streams_xattr`, because these filesystems cannot store its user xattrs. That existing stack does not provide a complete supported Apple metadata path; this change does not fix Apple metadata handling on FAT shares. Neither `fruit:resource = file` nor switching to `fruit:metadata = netatalk` provides an xattr-free Finder metadata backend. Time Machine remains excluded on these filesystems.

The generator keeps `fruit:aapl = yes` globally and retains `fruit` on every share. Removing `fruit` only from FAT shares would introduce mixed Apple/non-Apple shares: Samba negotiates AAPL on the first tree connection, making support depend on connection order. Disabling AAPL server-wide would also remove the FULLSYNC capability needed by Time Machine. A complete FAT solution therefore needs a separate design and client validation; this bounded fix avoids changing that server-wide behavior.

The existing Time Machine exclusion list is separate from the FAT exception: `f2fs` can support user xattrs, and `fuseblk` identifies a driver rather than a single filesystem capability. Both receive the complete Apple module stack without enabling Time Machine. The generator does not probe mount-level xattr support; these and other drivers still need working user xattrs. Configuration tests do not verify live SMB transfers, especially through another SMB mount, and do not establish a fix for Finder error -43.

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
- [Samba vfs_streams_xattr(8) man page](https://www.samba.org/samba/docs/current/man-html/vfs_streams_xattr.8.html)
- [Linux F2FS mount options](https://docs.kernel.org/filesystems/f2fs.html#mount-options)
- [Samba Wiki: Configure Samba to Work Better with Mac OS X](https://wiki.samba.org/index.php/Configure_Samba_to_Work_Better_with_Mac_OS_X)
- [Apple: About Time Machine](https://support.apple.com/en-us/HT201250)
