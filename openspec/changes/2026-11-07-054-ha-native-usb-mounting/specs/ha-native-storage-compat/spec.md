# Spec Delta

## Purpose

Defines how SambaNAS2 detects Home Assistant 2026.11+ native local-disk mounts, exposes them as read-only HA-managed volumes, and prevents SRAT from mounting, formatting, checking, labeling or unmounting them, while still allowing shares to be created on top.

## ADDED Requirements

### Requirement: HA-managed mount detection

The system SHALL classify a mount point as Home Assistant-managed when its path is under `/share/`, `/media/` or `/backup/` and its source is a block device, and SHALL set `is_ha_mounted = true` and `type = HOST` on the in-memory mount point.

#### Scenario: mount under /share is HA-managed
- **WHEN** `SyncPartitionMountData` processes a procfs mount whose source equals the partition device path and whose mount point is `/share/usb`
- **THEN** the resulting `MountPointData` has `is_ha_mounted = true` and `type = "HOST"`

#### Scenario: mount under /mnt is not HA-managed
- **WHEN** `SyncPartitionMountData` processes a procfs mount whose mount point is `/mnt/usb`
- **THEN** the resulting `MountPointData` has `is_ha_mounted = false`

### Requirement: HA-managed mount points are not persisted as owned

The system SHALL re-derive `is_ha_mounted` at runtime from the path whenever a mount point is loaded from the database, because the flag is read-only and never persisted.

#### Scenario: DB-loaded mount at /share/usb
- **WHEN** `SyncPartitionMountData` or `MountRepository.LoadByPath` loads a persisted mount point whose path is `/share/usb`
- **THEN** the loaded `MountPointData` has `is_ha_mounted = true` and `type = "HOST"`

### Requirement: SRAT must not mount or unmount HA-managed volumes

The system SHALL reject mount and unmount requests for HA-managed mount points with `ErrorHAManagedVolume`, and SHALL skip automount for them.

#### Scenario: mount rejected
- **WHEN** `MountVolume` is called with a `MountPointData` whose `is_ha_mounted` is true, or whose path is HA-managed, or whose cached mount point is HA-managed
- **THEN** it returns `ErrorHAManagedVolume` without calling the mounter

#### Scenario: unmount rejected
- **WHEN** `UnmountVolume` is called for a path whose cached mount point is HA-managed
- **THEN** it returns `ErrorHAManagedVolume` without calling the unmounter

#### Scenario: automount skipped
- **WHEN** `ShouldSkipAutomount` is called for a mount point with `is_ha_mounted = true` or an HA-managed path
- **THEN** it returns `true`

### Requirement: SRAT must not format, check or relabel HA-managed volumes

The system SHALL reject filesystem format, check and label operations on partitions that have any HA-managed mount point.

#### Scenario: format rejected
- **WHEN** `FormatPartition` is called for a partition whose `MountPointData` contains an HA-managed entry
- **THEN** it returns HTTP 409 Conflict

#### Scenario: check rejected
- **WHEN** `CheckPartition` is called for a partition whose `MountPointData` contains an HA-managed entry
- **THEN** it returns HTTP 409 Conflict

#### Scenario: label rejected
- **WHEN** `SetPartitionLabel` is called for a partition whose `MountPointData` contains an HA-managed entry
- **THEN** it returns HTTP 409 Conflict

### Requirement: Shares on HA-managed volumes are annotated

The system SHALL mark a share's `Status.IsHAMounted` as true when its mount point is HA-managed, using the live volume cache with a path-based fallback when the cache misses.

#### Scenario: share on /share/usb
- **WHEN** `VerifyShare` is called for a share whose mount point path is `/share/usb`
- **THEN** `share.Status.IsHAMounted` is `true` and `share.MountPointData.IsHAMounted` is `true`

### Requirement: SRAT must not mount HA-managed shares back into Home Assistant

The system SHALL skip `NetworkMountShare` / `NetworkMountAllShares` for shares whose mount point is HA-managed, so HA does not receive a mount pointing at its own disk.

#### Scenario: network mount skipped
- **WHEN** `NetworkMountAllShares` iterates a share whose `MountPointData.IsHAMounted` is true (or path is HA-managed)
- **THEN** it does not call `NetworkMountShare` for that share

### Requirement: Supervisor mount client models disk mounts

The system SHALL model the `disk` mount type in the Supervisor mount client, including `uuid`, `device`, `filesystem` and `user_path` fields.

#### Scenario: disk mount type parsed
- **WHEN** the Supervisor returns a mount with `type: disk`
- **THEN** `MountType` equals `disk` and `UserPath`, `Uuid`, `Filesystem` and `Device` are populated as present

### Requirement: Supervisor mount list tolerates nil server

The system SHALL not dereference a nil `Server` field when listing Supervisor mounts, so disk mounts (which have no server) do not cause a panic.

#### Scenario: disk mount with nil server
- **WHEN** `NetworkGetAllMounted` processes a mount with `Server: nil`
- **THEN** it does not panic and excludes that mount from the addon's own-mount set