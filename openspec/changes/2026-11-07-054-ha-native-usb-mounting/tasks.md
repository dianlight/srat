# Tasks

## 1. Data Model and Client

- [x] 1.1 Add `MountPointData.IsHAMounted` (read-only, `is_ha_mounted`) and `dto.ErrorHAManagedVolume`.
- [x] 1.2 Extend Supervisor mount client with `disk` type and `uuid`/`device`/`filesystem`/`user_path` fields (mount.yaml + regeneration).

## 2. Backend Detection and Guards

- [x] 2.1 Detect HA-managed mounts in `SyncPartitionMountData` via `IsHAManagedMountPath`.
- [x] 2.2 Re-derive `IsHAMounted` on DB load (`SyncPartitionMountData`, `MountRepository.LoadByPath`).
- [x] 2.3 Guard `MountVolume`, `UnmountVolume`, `ShouldSkipAutomount` against HA-managed volumes.
- [x] 2.4 Guard filesystem format/check/label handlers.
- [x] 2.5 Enrich shares with `Status.IsHAMounted` (cache + path fallback) in `mount_intelligence`/`VerifyShare`.
- [x] 2.6 Skip self-mounting HA-managed shares in `SupervisorService.NetworkMountAllShares`/`NetworkMountShare`.
- [x] 2.7 Fix nil dereference in `NetworkGetAllMounted` for `disk` mounts.

## 3. Frontend

- [x] 3.1 Show "Managed by Home Assistant" badge in `PartitionInformationCard` and `ShareDetailsPanel`.
- [x] 3.2 Hide mount/automount/format/label/unmount actions for HA-managed partitions; keep Create Share / Go to Share.
- [x] 3.3 Hide Mount Settings card for HA-managed partitions.
- [x] 3.4 Regenerate OpenAPI and frontend types.

## 4. Tests

- [x] 4.1 Backend unit tests for `IsHAManagedMountPath`, `ShouldSkipAutomount`, `MountVolume`/`UnmountVolume` guards, `SyncPartitionMountData` marking, `VerifyShare`.
- [x] 4.2 Frontend tests for PartitionActions, VolumeDetailsPanel, ShareDetailsPanel.
- [x] 4.3 OpenAPI diff hygiene (only `is_ha_mounted` added).

## 5. Follow-ups

- [ ] 5.1 Live verification on HA 2026.11 (mount propagation, hardware info correlation).
- [ ] 5.2 Reduced-privilege addon profile in `dianlight/hassio-addons`.
- [ ] 5.3 Version gating and user-configurable mode (see change 055).