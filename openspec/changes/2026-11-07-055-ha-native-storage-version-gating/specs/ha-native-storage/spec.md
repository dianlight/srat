# Spec Delta

## Purpose

Defines version-gated, configurable provisioning and use of Home Assistant native local-disk storage: capability detection, the `off`/`on`/`auto` mode setting, the per-disk backend choice in the mount dialog, and the ownership rules for SRAT-provisioned HA mounts. Builds on change 054's `ha-native-storage-compat` detection.

## ADDED Requirements

### Requirement: HA-native storage capability detection

The system SHALL report `ha_native_storage_supported = true` through `GET /api/capabilities` only when the detected Supervisor version is >= 2026.10.1 and the detected Home Assistant Core version is >= 2026.11.0. Until a version is known the capability SHALL default to `false`.

#### Scenario: supported HA version

- **WHEN** the Supervisor reports `supervisor = "2026.10.1"` and `homeassistant = "2026.11.0"`
- **THEN** `ha_native_storage_supported` is `true`

#### Scenario: older Core version

- **WHEN** the Supervisor reports `homeassistant = "2026.10.4"`
- **THEN** `ha_native_storage_supported` is `false`

#### Scenario: version unavailable

- **WHEN** neither the Supervisor `/info` endpoint nor the custom-component handshake provides a version
- **THEN** `ha_native_storage_supported` is `false`

### Requirement: HA-native storage mode setting

The system SHALL persist a `ha_native_storage_mode` setting with values `off`, `on`, `auto` and a default of `auto`, and SHALL expose the effective mode alongside the capability flag.

#### Scenario: default on a fresh install

- **WHEN** no `HaNativeStorageMode` row exists
- **THEN** the loaded setting is `auto`

#### Scenario: invalid/empty value normalized

- **WHEN** the stored value is empty
- **THEN** serialization yields `auto`

### Requirement: Mode semantics

The system SHALL apply the following semantics:

- `off`: only the local backend is offered; HA-path detection is disabled.
- `auto` (default): both backends are offered when the capability is true, HA preselected; the HA-path guard stays on regardless of the resolved version.
- `on`: only the HA backend is offered and requires support; when the capability is false nothing is mountable and the volumes UI reports "not supported on this HA version".

#### Scenario: on without support

- **WHEN** the mode is `on` and `ha_native_storage_supported` is `false`
- **THEN** the mount dialog offers no backend and the volumes page shows the not-supported state

#### Scenario: auto keeps the guard on

- **WHEN** the mode is `auto` and the version cannot be verified
- **THEN** HA-path detection remains enabled

### Requirement: Per-disk backend choice

In `auto` mode the mount dialog SHALL offer a backend selector (local vs. Home Assistant) when the capability is true, and SHALL persist the chosen backend on the mount point.

#### Scenario: HA backend selected

- **WHEN** the user selects the Home Assistant backend and submits
- **THEN** SRAT calls `POST /mounts` with `type = disk` and derives the mount point path from the returned `user_path`

#### Scenario: local backend selected

- **WHEN** the user selects the local backend
- **THEN** SRAT mounts under `/mnt` as before and no Supervisor mount is created

### Requirement: Ownership of SRAT-provisioned HA mounts

The system SHALL persist an ownership marker (`ha_mount_name`) on mount points SRAT provisions through the Supervisor API, and SHALL allow unmount only for those mounts (via `DELETE /mounts/{name}`).

#### Scenario: unmount of an SRAT-provisioned HA mount

- **WHEN** `UnmountVolume` is called for a path whose mount point has `ha_mount_name` set
- **THEN** SRAT calls `DELETE /mounts/{name}` and removes the local row on success

#### Scenario: unmount of a foreign HA mount

- **WHEN** `UnmountVolume` is called for a path whose mount point is HA-backed but has no `ha_mount_name`
- **THEN** it returns `ErrorHAManagedVolume` without calling the unmounter

### Requirement: Destructive operations stay blocked on HA-backed rows

The system SHALL reject format, check and relabel requests for any HA-backed mount point regardless of ownership, requiring the HA mount to be removed first.

#### Scenario: format rejected

- **WHEN** `FormatPartition` is called for a partition whose mount point is HA-backed
- **THEN** it returns `ErrorHAManagedVolume` without formatting

### Requirement: Version source and cache

The system SHALL obtain the Supervisor and Core versions from the existing Supervisor `/info` endpoint (`HaRootService`), falling back to the custom-component handshake version, and SHALL invalidate the cached system info on a Home Assistant STARTED event.

#### Scenario: cache invalidated on HA start

- **WHEN** a `HomeAssistantEvent` of type STARTED is received
- **THEN** the cached system info is flushed so the next read re-fetches the version
