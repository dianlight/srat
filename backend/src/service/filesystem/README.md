<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of Contents** *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Filesystem Adapter Pattern](#filesystem-adapter-pattern)
  - [Overview](#overview)
  - [Architecture](#architecture)
    - [Core Components](#core-components)
    - [Supported Filesystems](#supported-filesystems)
  - [Usage](#usage)
    - [Getting a Filesystem Adapter](#getting-a-filesystem-adapter)
    - [Checking Filesystem Support](#checking-filesystem-support)
    - [Formatting a Device](#formatting-a-device)
    - [Checking a Filesystem](#checking-a-filesystem)
    - [Managing Labels](#managing-labels)
    - [Getting Filesystem State](#getting-filesystem-state)
    - [Listing All Supported Filesystems](#listing-all-supported-filesystems)
  - [Command Mapping](#command-mapping)
    - [ext4 (e2fsprogs)](#ext4-e2fsprogs)
    - [vfat (dosfstools)](#vfat-dosfstools)
    - [ntfs (ntfs-3g-progs)](#ntfs-ntfs-3g-progs)
    - [btrfs (btrfs-progs)](#btrfs-btrfs-progs)
    - [xfs (xfsprogs)](#xfs-xfsprogs)
  - [Adding New Filesystem Adapters](#adding-new-filesystem-adapters)
  - [Testing](#testing)
  - [Backward Compatibility](#backward-compatibility)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Filesystem Adapter Pattern

## Overview

The SRAT backend now implements a filesystem adapter pattern that provides a clean, extensible interface for managing different filesystem types. Each supported filesystem has its own adapter that implements common operations like formatting, checking, and label management.

## Architecture

### Core Components

1. **FilesystemAdapter Interface**: Defines the contract for all filesystem operations
2. **Base Adapter**: Provides common functionality shared across all filesystem types
3. **Filesystem-Specific Adapters**: Implement filesystem-specific operations
4. **Registry**: Manages and provides access to all filesystem adapters

### Supported Filesystems

| Filesystem | Alpine Package | Description                |
| ---------- | -------------- | -------------------------- |
| ext4       | e2fsprogs      | Fourth Extended Filesystem |
| vfat       | dosfstools     | FAT32 Filesystem           |
| ntfs       | ntfs-3g-progs  | NTFS Filesystem            |
| btrfs      | btrfs-progs    | B-tree Filesystem          |
| xfs        | xfsprogs       | XFS Filesystem             |

## Usage

### Getting a Filesystem Adapter

```go
import (
    "context"
    "github.com/dianlight/srat/service"
)

// Get the filesystem service
fsService := service.NewFilesystemService(ctx)

// Get a specific filesystem adapter
adapter, err := fsService.GetAdapter("ext4")
if err != nil {
    // Handle error
}
```

### Checking Filesystem Support

```go
// Check if a filesystem is supported on the system
support, err := adapter.IsSupported(ctx)
if err != nil {
    // Handle error
}

fmt.Printf("Can mount: %v\n", support.CanMount)
fmt.Printf("Can format: %v\n", support.CanFormat)
fmt.Printf("Can check: %v\n", support.CanCheck)
fmt.Printf("Can set label: %v\n", support.CanSetLabel)
fmt.Printf("Alpine package: %s\n", support.AlpinePackage)

if !support.CanFormat {
    fmt.Printf("Missing tools: %v\n", support.MissingTools)
}
```

### Formatting a Device

```go
import "github.com/dianlight/srat/service/filesystem"

options := filesystem.FormatOptions{
    Label: "MyDisk",
    Force: true,
}

err := adapter.Format(ctx, "/dev/sdb1", options)
if err != nil {
    // Handle error
}
```

### Checking a Filesystem

```go
checkOptions := filesystem.CheckOptions{
    AutoFix: true,
    Force: false,
    Verbose: true,
}

result, err := adapter.Check(ctx, "/dev/sdb1", checkOptions)
if err != nil {
    // Handle error
}

fmt.Printf("Success: %v\n", result.Success)
fmt.Printf("Errors found: %v\n", result.ErrorsFound)
fmt.Printf("Errors fixed: %v\n", result.ErrorsFixed)
fmt.Printf("Message: %s\n", result.Message)
```

### Managing Labels

```go
// Get current label
label, err := adapter.GetLabel(ctx, "/dev/sdb1")
if err != nil {
    // Handle error
}
fmt.Printf("Current label: %s\n", label)

// Set new label
err = adapter.SetLabel(ctx, "/dev/sdb1", "NewLabel")
if err != nil {
    // Handle error
}
```

### Getting Filesystem State

```go
state, err := adapter.GetState(ctx, "/dev/sdb1")
if err != nil {
    // Handle error
}

fmt.Printf("Is clean: %v\n", state.IsClean)
fmt.Printf("Is mounted: %v\n", state.IsMounted)
fmt.Printf("Has errors: %v\n", state.HasErrors)
fmt.Printf("Description: %s\n", state.StateDescription)
```

### Listing All Supported Filesystems

```go
// Get list of supported filesystem types
types := fsService.ListSupportedTypes()
fmt.Printf("Supported filesystems: %v\n", types)

// Get detailed support information for all filesystems
support, err := fsService.GetSupportedFilesystems(ctx)
if err != nil {
    // Handle error
}

for fsType, info := range support {
    fmt.Printf("%s: Can format=%v, Alpine package=%s\n",
        fsType, info.CanFormat, info.AlpinePackage)
}
```

## Command Mapping

Each filesystem adapter uses specific commands from its Alpine package:

### ext4 (e2fsprogs)

- **Format**: `mkfs.ext4 [-F] [-L label] device`
- **Check**: `fsck.ext4 [-y|-n] [-f] [-v] device`
- **Get Label**: `tune2fs -l device` (parses output)
- **Set Label**: `tune2fs -L label device`

### vfat (dosfstools)

- **Format**: `mkfs.vfat -F 32 [-n label] device`
- **Check**: `fsck.vfat [-a|-n] [-v] device`
- **Get Label**: `fatlabel device`
- **Set Label**: `fatlabel device label`

### ntfs (ntfs-3g-progs)

- **Format**: `mkfs.ntfs -Q [-F] [-L label] device`
- **Check**: `ntfsfix [-n] device`
- **Get Label**: `ntfslabel device`
- **Set Label**: `ntfslabel device label`

### btrfs (btrfs-progs)

- **Format**: `mkfs.btrfs [-f] [-L label] device`
- **Check**: `btrfs check [--repair|--readonly] [--force] device`
- **Get Label**: `btrfs filesystem show device` (parses output)
- **Set Label**: `btrfs filesystem label device label`

### xfs (xfsprogs)

- **Format**: `mkfs.xfs [-f] [-L label] device`
- **Check**: `xfs_repair [-n] [-v] device`
- **Get Label**: `xfs_admin -l device` (parses output)
- **Set Label**: `xfs_admin -L label device`

## Adding New Filesystem Adapters

To add support for a new filesystem:

1. Create a new adapter file (e.g., `newfs_adapter.go`) in `backend/src/service/filesystem/`
2. Implement the `FilesystemAdapter` interface
3. Use the `baseAdapter` struct for common functionality
4. Register the adapter in `registry.go`
5. Add tests for the new adapter

Example:

```go
package filesystem

import (
    "context"
    "github.com/dianlight/srat/dto"
    "gitlab.com/tozd/go/errors"
)

type NewFsAdapter struct {
    baseAdapter
}

func NewNewFsAdapter() FilesystemAdapter {
    return &NewFsAdapter{
        baseAdapter: baseAdapter{
            name:          "newfs",
            description:   "New Filesystem",
            alpinePackage: "newfs-tools",
            mkfsCommand:   "mkfs.newfs",
            fsckCommand:   "fsck.newfs",
            labelCommand:  "newfs-label",
        },
    }
}

func (a *NewFsAdapter) GetMountFlags() []dto.MountFlag {
    return []dto.MountFlag{
        // Define filesystem-specific mount flags
    }
}

// Implement other interface methods...
```

Then register it in `registry.go`:

```go
func NewRegistry() *Registry {
    registry := &Registry{
        adapters: make(map[string]FilesystemAdapter),
    }

    registry.Register(NewExt4Adapter())
    registry.Register(NewVfatAdapter())
    registry.Register(NewNtfsAdapter())
    registry.Register(NewBtrfsAdapter())
    registry.Register(NewXfsAdapter())
    registry.Register(NewNewFsAdapter())  // Add your new adapter

    return registry
}
```

## Testing

Run the filesystem adapter tests:

```bash
cd backend
make test
```

Or test specific adapters:

```bash
cd backend/src
go test ./service/filesystem -v
```

## Backward Compatibility

The new adapter pattern is fully backward compatible with the existing `FilesystemService`. All existing mount flag methods continue to work as before:

- `GetStandardMountFlags()`
- `GetFilesystemSpecificMountFlags(fsType)`
- `MountFlagsToSyscallFlagAndData(flags)`
- `SyscallFlagToMountFlag(syscallFlag)`
- `SyscallDataToMountFlag(data)`
- `FsTypeFromDevice(devicePath)`

New methods have been added without breaking any existing functionality:

- `GetAdapter(fsType)`
- `GetSupportedFilesystems(ctx)`
- `ListSupportedTypes()`
