# [FEATURE]: Volume Mount Intelligence — NFS/CIFS Decision, Event Cache Retry

**Target Repo:** `srat`  **Status:** ✅ Done  **Issue Link:** [dianlight/srat#500](https://github.com/dianlight/srat/issues/500)

## 🎯 Objective

Enrich the volume mount structs with partition and filesystem metadata so that services can make an informed decision between NFS and CIFS exports (instead of always defaulting to the share name for NFS path). Additionally, implement proper volume-event handling in `VolumeService`: process add/remove events to retry or invalidate the hardware cache, rather than silently ignoring them.

> _Context for Copilot: Five TODO/FIXME comments across three service files (`supervisor_service.go:249,304`, `server_process_service.go:676`, `volume_service.go:423,447,450`) all converge on this theme. The `dto.DiskMap` is the shared in-memory cache (`*dto.DiskMap` FX singleton). `VolumeService` is the primary updater; it already has `GetPartitionByID` and `GetPartitionDevicePath` helpers._

## 🛠️ Technical Specifications

- **Inputs:**
  - Mount request with `partitionId` or device path
  - `dto.DiskMap` — current in-memory disk/partition cache
  - Kernel udev events forwarded via the volume event loop

- **Outputs:**
  - Mount structs populated with: `FsType`, `DevicePath`, `PartitionID`, `NfsExportPath` (resolved from actual mount point, not share name)
  - Correct NFS vs CIFS selection based on filesystem exportability (`adapter.IsExportable(ctx)`)
  - On add-event: retry mount if partition is in cache; on remove-event: unmount and evict from cache
  - Event-driven dirty-state push (only broadcast when data actually changes)

- **Dependencies:**
  - `backend/src/service/supervisor_service.go` — NFS/CIFS mount decision (lines 249, 304)
  - `backend/src/service/server_process_service.go` — Samba template rendering (line 676)
  - `backend/src/service/volume_service.go` — event handling (lines 423, 447, 450)
  - `backend/src/service/broadcaster_service.go` — dirty data push (line 123)
  - `backend/src/service/filesystem/adapter.go` — `IsExportable(ctx)` interface method
  - `backend/src/dto/` — `DiskMap`, `MountStruct`, `Partition`

## 📝 Task List

- [x] Task 1: Populate `FsType` and `DevicePath` in mount structs by looking up the partition in `dto.DiskMap` before choosing NFS vs CIFS (`supervisor_service.go:249,304` and `server_process_service.go:676`)
- [x] Task 2: Use `adapter.IsExportable(ctx)` to decide NFS vs CIFS; resolve `NfsExportPath` from the actual mount point rather than defaulting to share name
- [x] Task 3: Implement `volume_service.go:423` FIXME — handle udev add/remove events: on add, check if partition is in cache and retry mount; on remove, unmount and invalidate `dto.DiskMap` entry
- [x] Task 4: Implement `volume_service.go:447,450` TODOs — cache presence check before retry; invalidate hardware cache and re-run `GetVolumesData` on unexpected remove
- [x] Task 5: Update `broadcaster_service.go:123` to push dirty state only when data has actually changed (hash-compare or version counter), not on every tick
- [x] Task 6: Unit tests — `supervisor_service` mount decision with NFS-exportable and non-exportable filesystem types
- [x] Task 7: Unit tests — `volume_service` event handler: add-event retry, remove-event cache invalidation
- [x] Task 8: Unit tests — broadcaster dirty-push: verify no broadcast when data unchanged
- [x] Task 9: Integration test — mount a loopback ext4 device, trigger a remove event, verify cache eviction
- [x] Task 10: Documentation — update `docs/SMART_SERVICE.md` or `docs/SHARE_VOLUME_VERIFICATION.md` with the event-driven mount flow
- [x] Task 11: After reboot, ensure symlinks under `/media/` are restored for shares configured for media usage — investigate why they disappear after unclean shutdown and add a recovery step at startup to re-create them (see [hassio-addons#581](https://github.com/dianlight/hassio-addons/issues/581))

## 🧠 Implementation Notes (Copilot Context)

- Approved plan (2026-03-15):
    - Introduce shared mount-protocol decision logic that derives filesystem exportability from `Partition.FsType` using the filesystem adapter contract (`IsExportable(ctx)`), with safe CIFS fallback when metadata is missing.
    - Enrich supervisor/server mount handling with partition-backed metadata (`FsType`, `DevicePath`, `PartitionID`) and resolve NFS export path from the actual mount point instead of share-name defaults.
    - Implement partition udev add/remove processing in `VolumeService`: retry automount when cached startup mount data exists, otherwise invalidate hardware cache and rescan; on remove, unmount cached mount points and evict partition from `DiskMap`.
    - Deduplicate dirty-data push events in `BroadcasterService` so broadcast happens only when tracker state changes.
    - Add targeted unit/integration coverage for protocol selection, udev event handling, and dirty-data broadcast dedupe; then run focused backend tests.
- Progress update (2026-03-15):
    - Completed core service changes for Tasks 1-5 and validated with focused `go test ./service` runs.
    - Added helper-focused unit coverage in `backend/src/service/mount_intelligence_test.go` (adapter-based exportability, path resolution, cache enrichment, dirty-hash stability).
    - Added explicit volume-udev tests in `backend/src/service/volume_service_udev_test.go` and broadcaster dedupe assertion in `backend/src/service/broadcaster_service_internal_test.go`.
    - Updated `docs/SHARE_VOLUME_VERIFICATION.md` with the event-driven mount flow and mount-intelligence behaviour notes.
    - Remaining open items: integration loopback remove-event test and `/media` symlink startup recovery investigation.
    - Remote validation against the Home Assistant test environment (`local_sambanas2`, build `2026.3.0-dev.4`) succeeded: the addon restarted cleanly, the frontend loaded via `bun dev:remote`, and the Dashboard, Volumes, and Shares tabs rendered without visible UI regressions.
    - Filtered addon logs after UI interaction showed the expected automount recovery flow for cached startup shares (`ATA_WD_PC_SN740`, `CAROLA`), successful remounts for both volumes, and `Generated NFS exports configuration exportCount=0`, which matches the new non-exportable-filesystem handling for `ntfs3` and `exfat`.
    - Observed warnings appear pre-existing and unrelated to task 005 logic: HA Core not ready during early welcome-message fetch, SMART USB bridge detection warnings, health ping disk-stats initialization warning, and service-start warnings while `smbd`/`nfsd` were still being brought up.
    - Remote re-validation (2026-03-16, no addon restart per request) passed: addon remained in running state, frontend (`bun dev:remote`) rendered Dashboard/Volumes/Shares correctly, footer process list reported `srat-server` active, and post-UI filtered logs showed only recurring WebSocket connect debug events with no new `panic`, `fatal`, or `permission denied` entries.
    - Task 9 completed (2026-03-16): added `TestHandlePartitionUdevRemoveEvent_LoopbackExt4EvictsCache` in `backend/src/service/volume_service_udev_test.go`; the test binds a real loop device to `backend/test/data/image.dmg`, mounts it as `ext4`, triggers partition remove handling, and verifies both unmount and `DiskMap` partition eviction.
    - Validation for Task 9 passed with focused backend checks: `go test ./service -run TestHandlePartitionUdevRemoveEvent_LoopbackExt4EvictsCache -count=1` and `go test ./service -run 'TestHandlePartitionUdev(Remove|Add)Event' -count=1`.
    - Task 11 completed (2026-03-16): added startup recovery for media-share symlinks in `ServerService` (`recoverMediaUsageSymlinks`), invoked during `OnStart` to restore missing/stale `/media/<share-name>` symlinks for valid, enabled shares with `usage=media`.
    - Task 11 investigation result: symlink restoration was not part of startup initialization flow, so unclean shutdown could leave `/media/<share-name>` missing until a manual share-usage toggle; startup now performs idempotent recovery (create missing links, replace stale symlink targets, skip non-symlink collisions safely).
    - Validation for Task 11 passed with focused backend checks: `go test ./service -run 'TestRecoverMediaUsageSymlinks_(CreatesMissingSymlink|ReplacesStaleSymlink)' -count=1`.

### NFS vs CIFS selection

```go
// Before choosing protocol, resolve partition info from DiskMap
partition, _, found := diskMap.GetPartitionByID(mountRequest.PartitionID)
if found && partition.FsType != nil {
    adapter, err := fsRegistry.Get(*partition.FsType)
    if err == nil && adapter.IsExportable(ctx) {
        // Use NFS; resolve export path from actual mount point
        mountStruct.NfsExportPath = resolveActualMountPoint(partition)
    } else {
        // Fall back to CIFS/Samba
    }
}
```

### Volume event handling

```go
// volume_service.go — in the event loop (currently around line 423)
case EventTypeAdd:
    if p, _, ok := diskMap.GetPartitionByID(event.PartitionID); ok {
        // Partition already tracked — retry mount
        retryMount(ctx, p)
    } else {
        // New partition — invalidate hardware cache and re-scan
        diskMap.InvalidateHardware()
        s.GetVolumesData(ctx)
    }
case EventTypeRemove:
    // Unmount if mounted
    if mounted := diskMap.GetMountedPartition(event.DevicePath); mounted != nil {
        s.unmountPartition(ctx, mounted)
    }
    diskMap.RemovePartition(event.PartitionID)
```

### Broadcaster optimisation

- Compute a SHA-256 or CRC32 of the serialised `DirtyDataTracker` before broadcasting.
- Only call `broker.BroadcastMessage` if the hash has changed since last broadcast.
- Store the last-hash in a field on `BroadcasterService`.

### NfsExportPath resolution

- Query `df --output=target <device>` or read from `/proc/mounts` to find the actual mount point.
- Cache the resolved path in `DiskMap` alongside the partition entry.

## 🔗 Code References & TODOs

- [x] `backend/src/service/supervisor_service.go:249` — `// TODO: populate partition and filesystem info`
- [x] `backend/src/service/supervisor_service.go:304` — same TODO (second call site)
- [x] `backend/src/service/server_process_service.go:676` — same TODO
- [x] `backend/src/service/volume_service.go:423` — `// FIXME: Process Right events here`
- [x] `backend/src/service/volume_service.go:447` — `// TODO: Check if cache contain the partition. If yes retry mount`
- [x] `backend/src/service/volume_service.go:450` — `// TODO: Check if cache contain the partition. if yes umount and remove from cache`
- [x] `backend/src/service/broadcaster_service.go:123` — `// TODO: implement push of dirty data status only`
- [x] `backend/src/service/volume_service_udev_test.go` — integration test coverage for loopback ext4 remove-event cache eviction
- [x] `backend/src/service/server_process_service.go` — startup media symlink recovery (`recoverMediaUsageSymlinks`) executed during service initialization
- [x] `backend/src/service/server_process_service_media_recovery_test.go` — focused tests for missing/stale media symlink restoration
- [x] [hassio-addons#581](https://github.com/dianlight/hassio-addons/issues/581) — Files inside `/media/SERVER` disappear after reboot; startup recovery now re-creates expected media-share symlinks
