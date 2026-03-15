# [FEATURE]: Volume Mount Intelligence — NFS/CIFS Decision, Event Cache Retry

**Target Repo:** `srat`  **Status:** 📅 Planned  **Issue Link:** _TBD_

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

- [ ] Task 1: Populate `FsType` and `DevicePath` in mount structs by looking up the partition in `dto.DiskMap` before choosing NFS vs CIFS (`supervisor_service.go:249,304` and `server_process_service.go:676`)
- [ ] Task 2: Use `adapter.IsExportable(ctx)` to decide NFS vs CIFS; resolve `NfsExportPath` from the actual mount point rather than defaulting to share name
- [ ] Task 3: Implement `volume_service.go:423` FIXME — handle udev add/remove events: on add, check if partition is in cache and retry mount; on remove, unmount and invalidate `dto.DiskMap` entry
- [ ] Task 4: Implement `volume_service.go:447,450` TODOs — cache presence check before retry; invalidate hardware cache and re-run `GetVolumesData` on unexpected remove
- [ ] Task 5: Update `broadcaster_service.go:123` to push dirty state only when data has actually changed (hash-compare or version counter), not on every tick
- [ ] Task 6: Unit tests — `supervisor_service` mount decision with NFS-exportable and non-exportable filesystem types
- [ ] Task 7: Unit tests — `volume_service` event handler: add-event retry, remove-event cache invalidation
- [ ] Task 8: Unit tests — broadcaster dirty-push: verify no broadcast when data unchanged
- [ ] Task 9: Integration test — mount a loopback ext4 device, trigger a remove event, verify cache eviction
- [ ] Task 10: Documentation — update `docs/SMART_SERVICE.md` or `docs/SHARE_VOLUME_VERIFICATION.md` with the event-driven mount flow

## 🧠 Implementation Notes (Copilot Context)

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

- [ ] `backend/src/service/supervisor_service.go:249` — `// TODO: populate partition and filesystem info`
- [ ] `backend/src/service/supervisor_service.go:304` — same TODO (second call site)
- [ ] `backend/src/service/server_process_service.go:676` — same TODO
- [ ] `backend/src/service/volume_service.go:423` — `// FIXME: Process Right events here`
- [ ] `backend/src/service/volume_service.go:447` — `// TODO: Check if cache contain the partition. If yes retry mount`
- [ ] `backend/src/service/volume_service.go:450` — `// TODO: Check if cache contain the partition. if yes umount and remove from cache`
- [ ] `backend/src/service/broadcaster_service.go:123` — `// TODO: implement push of dirty data status only`
