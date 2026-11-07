# Proposal

## Why

Home Assistant 2026.11 (Supervisor 2026.10.1+) can mount local USB/internal disks natively via UDisks2 (usage `backup`/`media`/`share`), reducing the privileges SambaNAS2 needs. SambaNAS2 should stop mounting those disks itself and only manage shares over the HA-mounted paths, while keeping its existing mount path for everything else.

## What Changes

- Detect HA-managed local disks (path under `/share/`, `/media/`, `/backup/` with a block-device source) in `volume.SyncPartitionMountData` and mark them `is_ha_mounted` (`Type: HOST`).
- Add `MountPointData.IsHAMounted` (read-only) and `dto.ErrorHAManagedVolume`.
- Guard `MountVolume`, `UnmountVolume`, `ShouldSkipAutomount`, and the filesystem format/check/label handlers against HA-managed volumes.
- Enrich shares on HA-managed volumes with `Status.IsHAMounted` (live state from the volume cache, with path fallback).
- Skip self-mounting HA-managed shares back into HA in `SupervisorService.NetworkMountAllShares` / `NetworkMountShare`.
- Extend the Supervisor mount client for the `disk` mount type (`uuid`, `device`, `filesystem`, `user_path`).
- UI: show a "Managed by Home Assistant" badge and hide mount/automount/format/label/unmount actions; keep Create Share / Go to Share.

## Capabilities

### New Capabilities

- `ha-native-storage-compat`: compatibility layer for HA 2026.11 local disk mounts.

### Modified Capabilities

- `volumes`: HA-managed mount points carry `is_ha_mounted` and capability flags.
- `shares`: shares on HA-managed volumes are annotated `is_ha_mounted`.

## Impact

- `backend/src/dto/mount_point_data.go`, `backend/src/dto/error_code.go`
- `backend/src/service/volume/udev_handler.go`, `mount_orchestrator.go`, `mount_repository.go`
- `backend/src/service/mount_intelligence.go`, `share_service.go`, `supervisor_service.go`
- `backend/src/api/filesystems.go`, `backend/src/api/volumes.go`
- `backend/src/homeassistant/mount.yaml`, `client.gen.go`
- `frontend/src/pages/volumes/*`, `frontend/src/pages/shares/components/ShareDetailsPanel.tsx`
- OpenAPI regeneration performed.