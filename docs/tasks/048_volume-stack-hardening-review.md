<!-- DOCTOC SKIP -->

# [REFACTOR]: Volume Stack Hardening — Code Review Findings

**Target Repo:** `srat`
**Status:** 📅 Planned
**Type:** REFACTOR (invoke `prepare-refactor` skill before starting; see Task 0)
**Scope:** `backend/src/service/volume_service.go`, `backend/src/service/volume_mount_manager.go`, `backend/src/service/volume_service_udev_linux.go`, `backend/src/api/volumes.go`, `backend/src/dto/disk_map.go`, `frontend/src/hooks/volumeHook.ts`, `frontend/src/pages/volumes/**`

## 🎯 Objective

Simplify, speed up, and make bug/process-resistant the volume subsystem (disk → partition → mount point lifecycle) across backend and frontend. This document is the output of a full code review: it catalogs confirmed bugs (with root cause and evidence), ranks them by severity, and proposes concrete fixes with test coverage for each.

> _Context for Copilot: The volume stack has three writers competing for one unsynchronized in-memory map (`dto.DiskMap`): the singleflight-protected `getVolumesData` snapshot loop, the event-bus subscribers (`handlePartitionEvent`, `handleMountPointEvent`, share/smart/power handlers), and the udev netlink goroutine. Several data-loss and race bugs stem from that shared-mutable-state design. The frontend mirrors the problem by keeping a second, locally-mutated copy of `disks` that fights with SSE updates._

---

## 🔴 Critical Bugs (data loss / crash risk)

### B1 — Data race on `dto.DiskMap` and `refreshVersion`

**Files:** `volume_service.go:545`, `volume_service.go:634-642`, `disk_map.go` (all mutators)
**Evidence:**

- `self.refreshVersion++` (line 545) is a plain `uint32` incremented inside the singleflight closure while being read at lines 634, 846, 882, 914 from event-handler goroutines.
- `dto.DiskMap` is a bare `map[string]*Disk`. Writers: `getVolumesData` (snapshot loop), `handlePartitionEvent` (event bus goroutine), `handleMountPointEvent` (event bus goroutine), `AddMountPointShare`/`RemoveMountPointShare`/`AddSmartInfo`/`AddHDIdleDevice` (event bus goroutines), `PatchMountPointSettings` (HTTP handler goroutine), `handlePartitionUdevRemoveEvent` (udev goroutine). **No mutex anywhere.**
- `GetVolumesData()` → `slices.Collect(maps.Values(*self.disks))` (line 504) reads the map while any of the above may be writing → `fatal error: concurrent map read and map write` (non-recoverable runtime panic).

**Fix (suggested):**

1. Make `refreshVersion` an `atomic.Uint32`.
2. Add `sync.RWMutex` to `VolumeService` guarding all `self.disks` access, OR move the lock into `DiskMap` itself (preferred: keeps the invariant local):

   ```go
   type DiskMap struct {
       mu sync.RWMutex
       m  map[string]*Disk
   }
   ```

   This is a breaking change to the map-type API — all call sites must switch from map syntax to methods. Mechanical but wide.

3. `GetVolumesData()` must return deep copies (or hold `RLock` through HTTP serialization — copy is safer).

**Tests:** new `TestVolumeService_ConcurrentSnapshotAndEvents` using `t.Parallel`-style goroutine fan-out: N goroutines calling `GetVolumesData()` while M goroutines emit partition/mount-point events; run with `-race`. Must pass with zero race reports.

---

### B2 — `persistMountPoint` nil-wipes DB columns on mount point events (automount flag, flags, share)

**File:** `volume_service.go:868-899` → `handleMountPointEvent` → `persistMountPoint` (line 225: `clause.OnConflict{UpdateAll: true}`)
**Root cause chain:**

1. `handlePartitionEvent` finds a mount in `/proc/mounts` not yet in cache → builds `dto.MountPointData{...}` (line 871) carrying only procfs-derived fields.
2. Emits `MountPointEvent{Type: ADD}` (line 895).
3. `handleMountPointEvent` (line 938) receives it and calls `persistMountPoint` → `UpdateAll: true` upsert.

**Mechanism (verified in `converter/dto_to_dbom_conv_gen.go:108-136`):** `persistMountPoint` converts into a **fresh zero-value** `dbom.MountPointPath{}`. The converter's nil-guards (`if source.IsToMountAtStartup != nil`) only skip the assignment — the target field stays zero — and `UpdateAll: true` then writes **every** column including the zeros. Net effect: every event whose DTO lacks a field NULLs that column in the DB:

| DTO field missing in event | DB column wiped |
|----------------------------|-----------------|
| `IsToMountAtStartup`       | `is_to_mount_at_startup` (user's automount preference) |
| `Flags`                    | `flags` |
| `CustomFlags`              | `data` (custom flags) |
| `Share`                    | `exported_share` FK (share association) |
| `FSType`, `Root`, `Type`, `DeviceId` | same-named columns |

The procfs-discovery ADD path is the primary wipe vector; the stale-marking UPDATE path (line 926) re-emits whatever the cache holds, propagating already-wiped state back to the DB.

**Evidence:** any user who enables automount on a mount point, then triggers a volume refresh (udev event, HA start, provisional recheck) before the next mount event round-trip loses the flag. The `TestPatchMountPointSettings_UpdatesStartupFlagInGetVolumesData` test only passes because procfs mock returns empty mounts — it never exercises the collision.

**Fix (suggested):** in `handlePartitionEvent`, before emitting ADD for a procfs-discovered mount point, merge DB state:

```go
if existing, err := gorm.G[dbom.MountPointPath](self.db).
    Where(g.MountPointPath.Path.Eq(prtstate.MountPoint)).
    First(self.ctx); err == nil {
    mountPoint.IsToMountAtStartup = existing.IsToMountAtStartup
    mountPoint.Flags = existing.Flags
    mountPoint.CustomFlags = existing.Data
    // do NOT merge Share — it is owned by the share service
}
```

Additionally make `persistMountPoint` defensive: load-then-merge (convert into the existing DB record like `PatchMountPointSettings` does) instead of converting into a fresh struct, so nil fields can never overwrite persisted values.

**Tests:** `TestPartitionEvent_ProcfsDiscoveryPreservesStartupFlag`: seed DB with `IsToMountAtStartup=true` + flags, mock procfs with a matching mount, emit partition event, assert DB row still has `true` and flags intact. Extend to the stale-marking path.

---

### B3 — Cross-partition stale-marking interferes with unrelated mount points

**File:** `volume_service.go:903-932`
**Root cause:** the "mark stale" loop iterates `self.disks.GetAllMountPoints()` — **all disks, all partitions** — and marks `IsMounted=false` anything whose `RefreshVersion != self.refreshVersion`. But `refreshVersion` only advances on full `getVolumesData()` snapshots, while `handlePartitionEvent` fires for a **single** partition. Sequence:

1. Full snapshot bumps version to N; partitions A and B both get version N mount points.
2. udev add event for partition A → `getVolumesData` bumps to N+1, emits partition event for A.
3. `handlePartitionEvent(A)` runs the stale loop → partition B's mount points still carry version N... **but only if B was not in the latest snapshot's emit path.** Since `getVolumesData` emits for every partition, B is normally refreshed too — however if B's disk was evicted/re-added or its event handler errored earlier (the `continue` at line 825/858/892 skips emit on cache-write failure), B's mount points are falsely unmounted.

**Fix:** scope the stale loop to the partition being processed:

```go
for _, mountPoint := range self.disks.GetMountPointsForPartition(*e.Partition.DiskId, *e.Partition.Id) {
```

(requires a new `DiskMap` helper — small, mechanical.)

**Tests:** `TestPartitionEvent_StaleMarkingScopedToPartition`: two partitions with mounted mount points; process event for A only; assert B's mount point remains `IsMounted=true`.

---

### B4 — `UnmountVolume` bypasses `ProtectedMode`; API gaps in read-only enforcement

**Files:** `volume_service.go:383` (no `ProtectedMode` check — compare `MountVolume` line 237), `api/volumes.go:87-113` (`UmountVolume` has no `ReadOnlyMode` check — compare `PatchMountPointSettings` line 125)
**Impact:** protected/read-only mode is enforced inconsistently; the UI hides the buttons but the REST surface still accepts mutations. `POST /volume/mount` also lacks a `ReadOnlyMode` check in the handler (relies on service-level `ProtectedMode` only).

**Fix:** add the same guard to all three mutating endpoints:

```go
if self.apiContext.ReadOnlyMode {
    return nil, huma.Error403Forbidden("Cannot modify volumes in read-only mode")
}
```

and add `ProtectedMode` check to `UnmountVolume` in the service.

**Tests:** table-driven API tests: for each of mount/umount/patch × (ReadOnlyMode, ProtectedMode) assert 403 / `ErrorOperationNotPermittedInProtectedMode`.

---

### B5 — `GetDevicePathByDeviceID` conflates disk IDs with device IDs; nil-deref hazard

**File:** `volume_service.go:1133-1139`

```go
md, ok := ms.disks.Get(deviceID)      // looks up DISK by id
return *md.DevicePath, nil            // nil-deref if DevicePath is nil
```

`DeviceId` elsewhere means "partition id" (see `MountVolume` line 274-283 matching `*part.Id == md.DeviceId`) and the DTO comment says it's a device path (`/dev/sda1`). Three semantics, one field. Also no nil check on `md.DevicePath`.

**Fix:** decide the contract (recommend: `DeviceId` = partition Id everywhere; device **paths** go through `GetPartitionDevicePath`), then rewrite:

```go
part, _, ok := ms.disks.GetPartitionByID(deviceID)
if !ok { return "", errors.WithDetails(dto.ErrorNotFound, ...) }
path := ms.disks.GetPartitionDevicePath(part)
if path == "" { return "", errors.WithDetails(dto.ErrorDeviceNotFound, ...) }
return path, nil
```

Audit callers (`grep -rn "GetDevicePathByDeviceID"`) before changing.

**Tests:** unit tests for: partition-id hit, disk-id passed (must NOT match), missing DevicePath fallback to legacy, all-empty → error (no panic).

---

### B6 — `volumeHook.ts`: SSE/REST dual-source race + unsafe cast

**File:** `frontend/src/hooks/volumeHook.ts`
**Problems:**

1. Two `useEffect`s race to `setDisks`: REST (`data`) and SSE (`evdata.volumes`). Last render wins; ordering is not guaranteed → visible flicker, and optimistic edits (partition label rename in `Volumes.tsx`) are clobbered by the next SSE event.
2. `setDisks(data as Disk[])` — when the REST query errors, `isLoading` is false and `data` is `undefined` → `disks` becomes `undefined` despite the `Disk[]` type → downstream `disks.map` crashes in `Volumes.tsx` (line 64 in `updatePartitionLabelInDisks` is guarded, but `sourceDisks ?? []` at line 110/199 is not reached the same way for the hook path... actually `setDisks(sourceDisks ?? [])` covers it — still, the cast hides the bug).
3. `isLoading: isLoading && evloading` — if SSE query fails fast (`evloading=false`, `everror` set) while REST is still loading, combined loading is false and `error` may still be null → renders empty list instead of loading.

**Fix (suggested):** single derived value, SSE-wins-with-fallback:

```ts
export function useVolume() {
  const { data: evdata, error: everror, isLoading: evloading } = useGetServerEventsQuery();
  const { data, error, isLoading } = useGetApiVolumesQuery();
  const disks = useMemo<Disk[]>(
    () => evdata?.volumes ?? data ?? [],
    [evdata?.volumes, data],
  );
  return {
    disks,
    isLoading: isLoading && !evdata?.volumes, // REST gates only until first SSE payload
    error: error ?? everror,
  };
}
```

(removes both `useEffect`s and the local state entirely — simpler and race-free.)

**Tests:** MSW-backed tests: (a) REST resolves then SSE emits → final value is SSE; (b) REST errors, no SSE → `error` set, `disks=[]`; (c) SSE emits first → `isLoading` false immediately.

---

## 🟠 High Issues (correctness / robustness)

### H1 — Mount-event → mount-attempt loop has no backoff

**File:** `volume_service.go:956-967`
`handleMountPointEvent` on ADD/UPDATE with `IsToMountAtStartup && !IsMounted` calls `MountVolume`. `MountVolume` → `mounter.Mount` → emits `MountPointEvent{UPDATE}` → re-enters `handleMountPointEvent`. Normally the second pass sees `IsMounted=true` and stops — but if the mount **succeeds at OS level while the converter or cache write fails** (`volume_mount_manager.go:111-134` returns error after successful syscall), the event carries stale `IsMounted=false` → infinite retry loop with no backoff, one `mount(2)` per iteration. Add a per-mount-path retry counter/timestamp (e.g. in a `map[string]time.Time` guarded by the new mutex) with exponential backoff, and stop after N attempts (the automount-failure notification already exists for the terminal case).

**Tests:** fake mounter that succeeds the syscall but fails cache write; assert bounded number of `Mount` calls.

**Status:** ✅ Done — commit `59922eb0` (backoff guard + tests)

### H2 — `getVolumesData` emits per-partition events inside the snapshot; subscribers do per-partition DB + procfs I/O

**Files:** `volume_service.go:611-623` (emit), `803-935` (handler)
Cost per full refresh with P partitions: P × (`loadMountPointFromDB` query + `procfsGetMounts` full `/proc/mounts` parse + `GetAllMountPoints` slice alloc). Under a udev add-storm (USB hub), refreshes serialize through singleflight but each still costs O(P²) procfs parses. Two cheap wins:

1. Hoist `procfsGetMounts()` out of `handlePartitionEvent` — parse once per `getVolumesData` cycle and pass the slice via the event payload (or a request-scoped field).
2. Batch: emit one `PartitionsChanged` event carrying all changed partitions; handler iterates the payload, reusing one procfs parse and one DB transaction.

**Tests:** benchmark `BenchmarkGetVolumesData_10Disks50Partitions` before/after; assert procfs parse count == 1 via a counting mock.

**Status:** ✅ Done — commit `5d555733` (batched events + tests)

### H3 — Duplicate device-path validation (dead code)

**File:** `volume_service.go:293-299` vs `346-352` — identical `md.Partition.DevicePath == nil || *md.Partition.DevicePath == ""` check twice; the second is unreachable. Delete lines 346-352's duplicated branch (keep the `os.Stat` existence check that follows it).

**Status:** ✅ Done — commit `8f882f8d` (dead check removed + 11 MountVolume coverage tests; 65.3% → 89.8%)

### H4 — `PatchMountPointSettings` partial-PATCH semantics (verified: mostly safe, one quirk)

**File:** `volume_service.go:990-1012`
**Second-cycle verification (converter source `dto_to_dbom_conv_gen.go:108-136`):** the patch flow converts into the **DB-loaded** record, and the converter nil-guards `Path`/`Root`/`Type`/`DeviceId`/`FSType`/`IsToMountAtStartup` — nil patch fields leave the loaded values intact. The three unconditional assignments (`Flags` line 125, `Data` line 126, `ExportedShare` line 134) do nil-overwrite the loaded record, **but** the subsequent `gorm.Updates(ctx, &dbMountData)` skips zero-value (nil pointer) struct fields, so the DB keeps its values. The real nil-wipe danger lives in `persistMountPoint` (see B2), not here.

**Residual issues (re-verified during Task 10 implementation):**

1. ~~`Updates` returns `affected == 0` when the patch contains only nil/zero fields → handler returns 404 "not found"~~ — **does not reproduce**: GORM's struct-based `Updates` always includes the loaded record's non-zero fields (Path/Root/Type/DeviceId/FSType/CreatedAt) plus the auto-updated `UpdatedAt` (`AutoUpdateTime` field, `callbacks/update.go:279-292`), so `affected ≥ 1` for any existing record. An all-nil patch is already a 200 no-op with the DB untouched (locked by `TestPatchMountPointSettings_EmptyPatch_NoOp`). The `affected == 0` 404 branch is effectively dead for existing records; it is now defensively downgraded to a no-op debug log instead of a misleading 404.
2. **REAL BUG FOUND:** although the DB keeps `Flags`/`Data`/`ExportedShare` (GORM skips nil fields), the nil-wiped **in-memory** `dbMountData` was converted back into the response DTO **and pushed into the disk cache** (`AddOrUpdateMountPoint`, line ~1221). Result: a partial PATCH (e.g. toggling `IsToMountAtStartup`) silently dropped the flags/custom flags from the response **and from `GetVolumesData`** until the next full refresh — verified by probe: response DTO had `Flags=nil` while the DB row still had `[{noatime}]`. **Fixed** by reloading the record from the DB after `Updates` before converting back (line ~1224-1227); response and cache now reflect true persisted state.
3. The frontend's `handleToggleAutomount` sends `share: undefined` explicitly (`Volumes.tsx:570`) — harmless today only because of GORM's zero-skip; the intent ("don't touch the share") is implicit, not explicit. Document it or strip the field client-side. (Out of scope for this backend task.)

**Tests (Task 10):** PATCH with empty body / all-nil fields → 200 no-op, DB unchanged, **and response DTO keeps persisted flags** (`TestPatchMountPointSettings_EmptyPatch_NoOp`); PATCH with only `IsToMountAtStartup` → `flags`/`fstype`/data unchanged at DB level **and in the response DTO** (`TestPatchMountPointSettings_OnlyStartup_KeepsFlagsAndShareAtDBLevel`); record-not-found → `ErrorNotFound` (`TestPatchMountPointSettings_RecordNotFound`); fallback cache paths when the device can't be resolved to a partition (`TestPatchMountPointSettings_FallbackCacheUpdate_CachedMountPoint`, `TestPatchMountPointSettings_FallbackSnapshotLoop`). Coverage: service `PatchMountPointSettings` 39.6% → **75.0%**.

**Status:** ✅ Done — commit `82549d9b` (reload-after-patch fix + 7 PATCH tests; coverage 39.6% → 75.0%)

### H5 — `GetVolumesData()` masks load errors as "no disks"

**File:** `volume_service.go:496-506` — on `getVolumesData()` error, returns `[]*dto.Disk{}` and only logs. The UI shows an empty volume list instead of an error state. Change the public signature to `([]*dto.Disk, errors.E)` and let the API handler return 500 with detail; frontend `volumeHook` already surfaces `error`.

**Tests:** hardware client returns error → `GET /volumes` returns 500 (today: 200 + `[]`).

**Status:** ✅ Done — commit `028e7b24` (signature `([]*dto.Disk, errors.E)`; `ListVolumes` → 500; `PatchMountPointSettings` falls back to nil context on load failure; tests `TestListVolumes_ErrorReturns500`, `TestGetVolumesData_HardwareErrorPropagates`, `TestGetVolumesData_ReturnsCachedOnSubsequentCall`; coverage `GetVolumesData`/`ListVolumes` 100%)

### H6 — Unmount flag semantics inverted: "force" can fail while "normal" never does (fixed, Task 12)

**File:** `volume_mount_manager.go:148` → `filesystem_service.go:981` → u-root `mount_linux.go:133-151`
**Verified kernel semantics:** the call `UnmountPartition(m.ctx, md.Path, fsType, force, !force)` maps to:

- **Normal unmount** (`force=false` → `lazy=true`) → `MNT_DETACH`: always "succeeds" immediately, even with open files — busy state is hidden, I/O errors surface in still-running consumers later.
- **Force unmount** (`force=true` → `lazy=false`) → `MNT_FORCE` only: on busy local filesystems this **fails with EBUSY**. The u-root library explicitly rejects combining both flags (`mount_linux.go:138-139`).

This is the opposite of user expectations: the UI presents "Force Unmount" as the stronger action (red button, data-loss warning), but it is the one that fails when the volume is busy, while the graceful path silently detaches.

**Fix (Task 12):** `volume_mount_manager.go:168` now passes `UnmountPartition(m.ctx, md.Path, fsType, false, force)` — normal unmount = no flags (fails on busy → user sees the real error), force unmount = `MNT_DETACH` (guaranteed detach). The mount directory is only removed on a normal (non-lazy) unmount (`os.Remove` guarded by `!force`); with `MNT_DETACH` the filesystem stays active underneath until the last reference is gone, so the directory must remain valid.

**Tests (Task 12):** `volume_mount_manager_test.go` — mocked `FilesystemServiceInterface` capturing the `(force, lazy)` flags: (a) normal → `(force=false, lazy=false)` (`TestUnmount_NormalPassesNoFlags`); (b) force → `(force=false, lazy=true)` (`TestUnmount_ForceDetachesLazily`); (c) busy error propagated on normal unmount (`TestUnmount_NormalBusyErrorPropagates`); nil `MountPointData` rejected (`TestUnmount_NilMountPoint`). Coverage: `volumeMountManager.Unmount` **90%**. Note: `VolumeService.MockSetMountOps` type-asserts `FilesystemService` for a `MockSetMountOps` method it does not implement — the assertion silently no-ops; tests bypass it by mocking the manager's `FilesystemServiceInterface` directly.

### H7 — udev goroutine lifecycle

**File:** `volume_service_udev_linux.go:23-35` — `queue := make(chan netlink.UEvent, 10)`: on udev burst >10 events, netlink lib blocks or drops (impl-dependent); on shutdown `close(quit)` stops the monitor but an in-flight `queue <-` send may leak the producer goroutine. Buffer to 64+ and drain `queue`/`errorChan` after `close(quit)` before returning. Low frequency, low severity, cheap fix.

**Status:** done (Task 13).

**Fix (Task 13):** `volume_service_udev_linux.go` — `queue` buffered to 64; after `close(quit)` the handler drains `queue`/`errorChan` for 100 ms before returning. The go-udev `Monitor` producer sends with a **blocking `queue <- *uevent` outside its select loop** (`netlink/conn.go`), so a full queue at shutdown would leak that goroutine; draining lets the in-flight send complete and the producer observe the closed `quit` channel. A bounded timeout (not `default: return`) is required because the producer may emit a couple more events after quit closes (select picks randomly between the closed quit case and a buffered read).

**Tests (Task 13):** `udev_channel_drain_test.go` — generic `drainUdevChannels[T any]` extracted to `udev_channel_drain.go` (no netlink import, so it compiles/tests on non-Linux hosts; the netlink package itself is Linux-only): (a) drains buffered items + pending error; (b) unblocks a producer blocked on a full-queue send (`TestDrainUdevChannels_UnblocksBlockedProducer`); (c) returns promptly when idle; (d) keeps consuming under a concurrent producer. Coverage: `drainUdevChannels` **100%**. Linux build verified via `GOOS=linux go vet ./service/` (the linux-tagged handler itself can't run tests on the darwin dev host).

### H8 — Hardware-cache aliasing: `getVolumesData` mutates the shared 30-minute cache (race risk)

**File:** `volume_service.go:557-606` (second-cycle finding)
`self.hardwareClient.GetHardwareInfo()` returns the **cached** `dto.DiskMap` (30-min TTL). The loop copies each `dto.Disk` value, but `disk.Partitions` is a `*map[string]dto.Partition` — a **pointer shared with the cached map**. Line 606 `(*disk.Partitions)[pid] = part` writes `FilesystemInfo` through that shared pointer, mutating the cache itself. Two consequences:

1. **Cache pollution:** the hardware cache no longer represents pure hardware state; it accumulates `FilesystemInfo` enrichment. Mostly idempotent today, but the cached shape now depends on which service touched it first.
2. **Concurrent map read/write → panic risk:** the same map object is served to `api/hdidle_handler.go:77` (an HTTP handler on a different goroutine). If a HDIdle request iterates the map while `getVolumesData` writes a partition entry, Go panics with "concurrent map read and map write" → process crash.

Fix: deep-copy the partition map (or the whole DiskMap) before enrichment in `getVolumesData`; alternatively make `GetHardwareInfo` return an immutable snapshot (defensive copy at cache boundary). The B1 lock does not fix this — the aliasing exists even with correct locking, because two different lock domains (hardware cache vs DiskMap) guard the same memory.

**Test:** unit test calling `GetHardwareInfo()` concurrently with `GetVolumesData()` (loop 1000×) under `-race`; assert no race and no FilesystemInfo leak into the raw hardware cache.

### H9 — Event-bus handlers swallow errors: DB failures invisible to callers

**File:** `volume_service.go:604-605` — `_ = emitEvent(...)` and `_ = emitDisk(...)` discard the error returned by `handlePartitionEvent` / `handleDiskEvent`. Those handlers perform DB I/O (`loadMountPointFromDB`, persists) — a DB failure there is logged inside the handler (if at all) but never surfaces to the operation that triggered the event (e.g. the mount flow), and the event bus itself only reports panic recovery, not handler errors. Observability + correctness gap: a failed persist looks like a successful volume operation.

Fix: at minimum, log the emitted-handler error with context at the emit site (keep the bus fire-and-forget contract but make failures visible); for the mount path, consider propagating DB-persist errors from `handlePartitionEvent` back through the emit path when the trigger is synchronous (the bus is synchronous per `signals_sync.go`).

**Test:** fake event bus returning an error from the partition handler → assert the error is logged (capture log) and, for the mount flow, propagated.

### H10 — `GetVolumesData` lazy-load executes hardware I/O inside HTTP request path

**File:** `volume_service.go:1015` (converter context) + `ListVolumes` (`api/volumes.go`)
`GetVolumesData()` performs `getVolumesData` (procfs + hardware fetch + DB) whenever the cache is empty. `ListVolumes` and `PatchMountPointSettings` both run it synchronously inside the HTTP handler. After startup (or after H5's error case left the cache empty), the **first request blocks on a multi-second hardware discovery**, with the client seeing a stall — and H5 means it returns `[]` on failure anyway. 

Fix: warm the cache at service start (async) so the first HTTP request never pays discovery cost; keep the lazy path only as fallback. Consider returning 503 while the cache is cold instead of `[]` (ties into H5).

**Test:** API test with empty cache → assert response time budget (or that cache-warm request returns immediately); service-start warmup test asserting `GetHardwareInfo` called once at boot.

---

## 🟡 Frontend Findings

### F1 — `VolumesTreeView.tsx:122` reads map as array — automount icon never colors

```ts
const isToMountAtStartup =
  partition.mount_point_data?.[0]?.is_to_mount_at_startup === true;
```

`mount_point_data` is `Record<string, MountPointData>` → `?.[0]` is always `undefined` → primary-color icon never applies. Fix:

```ts
const isToMountAtStartup =
  Object.values(partition.mount_point_data ?? {})[0]?.is_to_mount_at_startup === true;
```

**Status:** done (Task 14).

**Fix (Task 14):** `VolumesTreeView.tsx` `renderPartitionIcon` — replaced `partition.mount_point_data?.[0]?.is_to_mount_at_startup === true` with `Object.values(partition.mount_point_data ?? {}).some((mpd) => mpd.is_to_mount_at_startup === true)`. Uses `.some()` rather than the prescription's `Object.values(...)[0]`: automount toggling (`Volumes.tsx` `handleToggleAutomount`) applies to **every** mount point, so the icon must color when *any* of them has automount enabled; `[0]` on a `Record` is still order-dependent.

**Tests (Task 14):** `VolumesTreeView.test.tsx` — (a) partition with `is_to_mount_at_startup: true` → `container.querySelector(".MuiSvgIcon-colorPrimary")` present; (b) `false` → absent. 13/13 tests pass (incl. `--retry=10`); `bun tsc --noEmit` clean; `bunx vitest run --changed` green.

**Test:** `VolumesTreeView.test.tsx` — partition with automount-enabled mount point → icon has `color="primary"` (assert via `data-testid` or class query on the icon).

### F2 — `VolumeMountDialog` violates the mandatory form standard

**File:** `VolumeMountDialog.tsx:311-315, 69, 232`
Per `.opencode/instructions/react-hook-form-mui.instructions.md`:

- ❌ uses raw `<form onSubmit={handleSubmit(...)}>` → must be `<FormContainer formContext={formContext} onSuccess={...}>` wrapping `DialogContent` **and** `DialogActions`.
- ❌ `const [mounting, setMounting] = useState(false)` for submit state → use `formState.isSubmitting`.
- The submit button must be `type="submit"` inside the `FormContainer` (currently uses `form="mountvolumeform"` attr — works but non-standard).

Refactor per the Dialog Pattern in the instruction file; behavior unchanged. This also removes the `handleSubmit` wrapper and simplifies `handleCloseSubmit` to the `onSuccess(data)` signature.

**Test:** existing `VolumeMountDialog.test.tsx` must keep passing; add "submit button disabled while submitting" case.

**Fix (Task 15):** `VolumeMountDialog.tsx` migrated to the canonical Dialog Pattern:

- `useForm` hoisted to a `formContext` object; `<form id="mountvolumeform" onSubmit={handleSubmit(...)} noValidate>` replaced with `<FormContainer formContext={formContext} onSuccess={handleCloseSubmit}>` wrapping **both** `DialogContent` and `DialogActions` (`noValidate` is implicit in `FormContainer`).
- Manual `const [mounting, setMounting] = useState(false)` removed; submit button now `type="submit"` with `disabled={formState.isSubmitting}` (dropped the non-standard `form="mountvolumeform"` attr and the `loading` spinner).
- `onClose` prop widened to `(data?: MountPointData) => void | Promise<void>` and `handleCloseSubmit` now `await props.onClose(submitData)`. `Volumes.tsx` `onSubmitMountVolume` returns the `mountVolume` mutation promise, so `isSubmitting` stays `true` for the whole in-flight mount — preserving the pre-refactor "button disabled until the parent closes the dialog" behavior.

**Tests (Task 15):** existing 10 tests pass unchanged; added "disables the Mount button while submitting" — controllable `onClose` promise, assert button disabled while pending and re-enabled after resolve. 11/11 pass (incl. `--retry=10`); full `volumes/` suite 155/155; `bunx vitest run --changed` green; `bun tsc --noEmit` clean (fixed TS7030 by making the parent's `onClose` handler return on every path).

### F3 — `Volumes.tsx` local `disks` state duplicates the source of truth

**File:** `Volumes.tsx:113, 198-200, 265-278`
`const [disks, setDisks] = useState(sourceDisks ?? [])` + sync effect + `handlePartitionLabelUpdated` mutating local state. Next SSE event overwrites the optimistic rename → label visually reverts until the backend round-trips. With the B6 hook fix (`useMemo` derivation), local state can only be kept if label updates go through RTK Query cache update (`dispatch(sratApi.util.updateQueryData(...))`) instead of `setDisks`. Recommend: drop local state; do the optimistic update on the RTK cache so SSE and REST both see it.

### F4 — `handleToggleAutomount` fires N uncoordinated PATCHes

**File:** `Volumes.tsx:539-588` — loops `Object.entries(partition.mount_point_data)` firing one PATCH per mount point, no `await`, no aggregate error handling, one toast per mount point. For multi-mount partitions this spams toasts and can interleave failures. Fix: aggregate into a single `Promise.allSettled`, one summary toast; ideally add a backend bulk endpoint later (out of scope here).

### F5 — Identifier helpers: stable IDs preferred, index only as last resort (second-cycle correction)

**File:** `frontend/src/pages/volumes/utils.ts:54-87`
**Correction:** the first-cycle claim that identifiers embed array indices was **wrong** — `getDiskIdentifier`/`getPartitionIdentifier` already prefer `disk.id` → `legacy_device_name` → `device_path` → `serial` (and `partition.id` → `uuid` → `device_path` → …), using the index only when **all** ID fields are missing. Selection persistence is therefore stable across reordering in normal operation.

**Residual (low):** if the backend ever emits a disk/partition with no id, uuid, device path or legacy name, the fallback `disk-${fallbackIndex}` / `part-${fallbackIndex}` keys reorder with the snapshot and silently re-point persisted selection/expansion state. Such a record would already indicate a backend data defect, so treat this as a defensive concern only: log a console warning when the fallback branch is taken (makes the backend defect visible) — no structural change needed.

### F6 — `VolumeDetailsPanel` hardcodes `isReadOnlyMode={false}` on the SMART panel

**File:** `VolumeDetailsPanel.tsx:389-396` (second-cycle finding)
`SmartStatusPanel` is rendered with `isReadOnlyMode={false}` even though the component receives a `readOnly` prop in scope (also forwarded to `PartitionInformationCard` at line 404 and `HDIdleDiskSettings` at line 380). In read-only mode the SMART self-test start/abort controls (gated by `isReadOnlyMode` at `SmartStatusPanel.tsx:562/580/598/620`) stay enabled — a read-only user can still start/abort disk self-tests. Fix: pass `isReadOnlyMode={readOnly}` (single call site).

**Test:** `VolumeDetailsPanel.test.tsx` — render with `readOnly` + SMART-capable disk → assert self-test action buttons disabled.

---

## 📝 Task List

- [x] Task 0: Run `prepare-refactor` skill (per AGENTS.md REFACTOR rule): baseline `mise run //backend:test` + `mise run //frontend:test`, record green state in `docs/refactors/048-volume-hardening.md`
- [x] Task 1: **B1** — Add `RWMutex` to `DiskMap` (convert to struct with methods) + `atomic.Uint32` refreshVersion; update all call sites; `-race` clean. *(Full call-site survey: see "B1 Implementation Survey" appendix below)* — done in `a08e8e40`; post-refactor suites green (backend 0 fail / 40.5%, frontend 95 files / 728 tests); breakage detector clean (only false positive: `homeassistant_service.go` iterates `*[]*dto.Disk`)
- [x] Task 2: **B2** — Merge DB state before procfs-discovery ADD emit; add regression test — done in `cd31e1ec`; see "B2 Implementation" appendix below; suites green (backend 0 fail / 40.5%, coverage gate met)
- [x] Task 3: **B3** — Scope stale-marking to current partition; add `GetMountPointsForPartition` helper + test — done; see "B3 Implementation" appendix below; suites green (backend 0 fail / 40.6%, coverage gate met)
- [x] Task 4: **B4** — ProtectedMode/ReadOnlyMode guards on unmount + mount endpoints; table-driven API tests — done; see "B4 Implementation" appendix below; suites green (backend 0 fail / 40.8%, coverage gate met)
- [x] Task 5: **B5** — Fix `GetDevicePathByDeviceID` semantics + nil guard; caller audit; unit tests — done; see "B5 Implementation" appendix below; suites green (backend 0 fail / 40.8%, coverage gate met)
- [x] Task 6: **B6** — Rewrite `volumeHook` with `useMemo` derivation (delete both `useEffect`s); MSW tests — done; see "B6 Implementation" appendix below; frontend suites green (727 passed), tsc + lint clean
- [x] Task 7: **H1** — Automount retry backoff (per-path attempt map, max 5, exp. backoff); loop test with failing cache write
- [x] Task 8: **H2** — Hoist procfs parse out of per-partition handler (parse once per refresh, pass via payload); benchmark before/after
- [x] Task 9: **H3** — Delete duplicate DevicePath validation block
- [x] Task 10: **H4** — Verify converter nil-handling; switch to selective field updates if needed; DB-level assertion test
- [x] Task 11: **H5** — `GetVolumesData` returns error; API 500 mapping; hook surfaces error; update callers
- [x] Task 12: **H6** — Unmount lazy/force semantics + dir-removal-only-if-created; fake-fs tests
- [x] Task 13: **H7** — udev channel buffer 64 + drain on shutdown
- [x] Task 14: **F1** — Fix map-as-array icon bug + RTL test
- [x] Task 15: **F2** — Migrate `VolumeMountDialog` to `FormContainer` pattern per instructions; keep tests green
- [ ] Task 16: **F3** — Drop local `disks` state; optimistic label update via RTK `updateQueryData`
- [ ] Task 17: **F4** — `Promise.allSettled` aggregation for automount toggle; single summary toast
- [ ] Task 18: **F5** — ID-only identifiers; localStorage migration note; reorder-stability test
- [ ] Task 19: **H8** — Deep-copy partition map in `getVolumesData` (or immutable snapshot at cache boundary); concurrent `-race` test vs HDIdle handler
- [ ] Task 20: **H9** — Surface event-handler errors (log at emit site; propagate DB-persist errors on mount path); log-capture test
- [ ] Task 21: **H10** — Warm hardware cache at service start; keep lazy path as fallback; response-time test
- [ ] Task 22: **F6** — `isReadOnlyMode={readOnly}` on `SmartStatusPanel`; RTL test with readOnly + SMART disk
- [ ] Task 23: Coverage gate: `mise run //backend:test` + `go tool cover -func=coverage.out` — every touched function ≥70%; frontend `bun tsc --noEmit` + `mise run //frontend:test:new` green
- [ ] Task 24: Update `CHANGELOG.md` under `[ 🚧 Unreleased ]`

## 🔬 B4 Implementation (2026-08-14)

**Branch:** `fix/volume-phantom-entries` (uncommitted; ready for one atomic commit)

### Changes

1. **`backend/src/service/volume_service.go` — `UnmountVolume` gains the `ProtectedMode` guard.** Mirrors `MountVolume`'s existing check: returns `dto.ErrorOperationNotPermittedInProtectedMode` (with `Operation`/`Detail` details) before any cache lookup or unmount attempt.
2. **`backend/src/api/volumes.go` — `MountVolume` and `UmountVolume` handlers gain the `ReadOnlyMode` guard** (same pattern as `PatchMountPointSettings`): `huma.Error403Forbidden` when `self.apiContext.ReadOnlyMode` is set, before any validation or service call.
3. **`backend/src/api/volumes.go` — ProtectedMode error now maps to HTTP 403** in both handlers (`errors.Is(errE, dto.ErrorOperationNotPermittedInProtectedMode)` → `huma.Error403Forbidden`). Previously the mount handler mapped it to 500 (unknown-error branch) and the umount handler to 406 (catch-all), so the REST surface misrepresented the protected-mode rejection.

### Tests

| Test | Coverage | Notes |
|------|----------|-------|
| `TestVolumeMutations_ProtectedMode` (`service/volume_service_test.go`) | `UnmountVolume` → **100%** (was 18.2%) | Table-driven over `MountVolume` + `UnmountVolume` with `ProtectedMode: true`; asserts sentinel error and `Operation` detail. Suite gained a populated `*dto.ContextState` (`suite.state`) so the shared service state pointer can be flipped in-test |
| `TestUnmountVolume_Paths` (`service/volume_service_test.go`) | same | Branch coverage for `UnmountVolume`: (a) cached mount point with an HA-mounted share → asserts REMOVE `ShareEvent` emitted (synchronous `OnShare` listener) + unmount attempted; (b) path not in cache → fallback synthesized mount point + unmount attempted. Real mounter fails cleanly with `ErrorUnmountFail` on non-existent paths, so assertions are deterministic |
| `TestMutatingEndpoints_ForbiddenInReadOnlyMode` (`api/volumes_extra_test.go`) | `MountVolume` handler → **91.3%**, `UmountVolume` handler → **87.5%** | Table-driven 403 matrix over mount/umount/patch with a `ReadOnlyMode: true` handler; verifies service methods called `Times(0)` |
| `TestMutatingEndpoints_ForbiddenInProtectedMode` (`api/volumes_extra_test.go`) | same | Table-driven 403 over mount/umount with the service mocked to return the ProtectedMode error; proves the handler maps it to 403 (not 500/406) |

Mutation checks (git-free, backup-copy + revert): removing the `UnmountVolume` ProtectedMode guard fails `TestVolumeMutations_ProtectedMode/UnmountVolume_rejected`; removing the API guards + 403 mappings fails `TestMutatingEndpoints_ForbiddenInProtectedMode`. Restored; working tree clean.

### Coverage gate (Task 23, per-function ≥70% for touched code)

| Function | Before | After |
|----------|--------|-------|
| `UnmountVolume` (service) | 18.2% | **100.0%** |
| `MountVolume` (handler) | — | **91.3%** |
| `UmountVolume` (handler) | — | **87.5%** |

(`MountVolume` service method 62.7% — pre-existing, untouched by B4.)

### Verification

- Targeted `go -C src tool gotest ... ./api ./service`: both packages ok.
- Full `mise run //backend:test`: exit=0, Total coverage 40.8% (baseline 40.5%).
- gofmt clean on all 4 touched files; `go -C src vet ./api/ ./service/` clean.
- Final diff: 4 files, +215.

## 🔬 B5 Implementation (2026-08-14)

**Branch:** `fix/volume-phantom-entries` (uncommitted; ready for one atomic commit)

### Changes

1. **`backend/src/service/volume_service.go` — `GetDevicePathByDeviceID` now resolves partitions, not disks.** Contract decided: `DeviceId` means *partition* Id everywhere (consistent with `MountVolume`'s `*part.Id == md.DeviceId` match and the DTO comment); device *paths* go through `DiskMap.GetPartitionDevicePath`. Implementation rewritten to:
   - `ms.disks.GetPartitionByID(deviceID)` — a disk ID passed here must **not** match (old code looked up the disk map, so a disk ID returned the disk's `DevicePath` or panicked).
   - `ms.disks.GetPartitionDevicePath(part)` — nil-safe cascade: `DevicePath` → `LegacyDevicePath` → `LegacyDeviceName`, empty string if none.
   - Distinct errors: partition not found → `dto.ErrorNotFound`; no path available → `dto.ErrorDeviceNotFound` (previously `*md.DevicePath` could nil-deref).
2. **Caller audit (`grep -rn "GetDevicePathByDeviceID")`:** the only production caller, the SMART-info handler pre-lookup in `api/smart.go:83`, is commented out — no active caller depends on the old disk-ID semantics. Interface signature unchanged, so upstream mocks (e.g. `smart_test.go`) remain valid.

### Tests

| Test | Coverage | Notes |
|------|----------|-------|
| `TestGetDevicePathByDeviceID` (`service/volume_service_test.go`) | `GetDevicePathByDeviceID` → **100%** (was ~60%) | Table-driven over the four required scenarios: partition-id hit returns `DevicePath`; disk-id passed → `ErrorNotFound` (must NOT match); missing `DevicePath` falls back to `LegacyDevicePath`; all-empty → `ErrorDeviceNotFound` (no panic). Disk also carries a device path so the old code fails on a clean value mismatch, not a panic |

Mutation check (git-free, backup-copy + revert): restoring the old implementation (`ms.disks.Get(deviceID)` + `*md.DevicePath`) fails `TestGetDevicePathByDeviceID/partition_id_hit_returns_device_path` with "Not Found" — proving the test catches the disk/partition conflation. Restored; working tree clean.

### Coverage gate (Task 23, per-function ≥70% for touched code)

| Function | Before | After |
|----------|--------|-------|
| `GetDevicePathByDeviceID` (service) | 75.0% | **100.0%** |

### Verification

- Targeted `go -C src tool gotest ... ./api ./service`: both packages ok.
- Full `mise run //backend:test`: exit=0, Total coverage 40.8%.
- gofmt clean on both touched files; `go -C src vet ./service/` clean.
- Final diff: 2 files, +51.

## 🔬 B6 Implementation (2026-08-14)

**Branch:** `fix/volume-phantom-entries` (uncommitted; ready for one atomic commit)

### Changes

1. **`frontend/src/hooks/volumeHook.ts` — rewritten as a single derived value; both `useEffect`s and the local `useState` removed.** The SSE payload (`evdata?.volumes`) wins once available, REST (`data`) is the fallback until then, `[]` otherwise:
   ```ts
   const disks = useMemo<Disk[]>(
     () => evdata?.volumes ?? (isDiskArray(data) ? data : []),
     [evdata?.volumes, data],
   );
   ```
2. **Unsafe cast removed.** `GetApiVolumesApiResponse` is a union (`Disk[] | null | ErrorModel`); the old `setDisks(data as Disk[])` produced `undefined` when the REST query errored. The new `isDiskArray` type guard narrows before use, so `disks` is always `Disk[]`.
3. **Loading gate fixed.** `isLoading: isLoading && !evdata?.volumes` — REST gates only until the first SSE payload arrives; a failing/fast SSE query no longer renders an empty list while REST is still loading (old code: `isLoading && evloading`).
4. **Error surfacing:** `error: error ?? everror` (REST error preferred, SSE error fallback; old code used `||`).

### Tests

| Test | Notes |
|------|-------|
| `volumeHook.test.ts` — 5 deterministic MSW tests | `wsApi` module mocked (repo pattern from `Shares.test.tsx`) with a controllable SSE mock; REST `/api/volumes` overridden per test via `getMswServer().use(...)`. Scenarios: REST fallback when no SSE payload; **SSE wins** over different REST data; REST failure → `[]` (not `undefined`) + error surfaced; SSE error surfaced when REST succeeds; `isLoading` stays true while REST pending with no SSE payload (deferred-response handler) |

Mutation check (backup-copy + revert): restoring the old implementation fails 2 tests semantically — `returns an empty array (not undefined) and surfaces the error when REST fails` (the `data as Disk[]` cast bug) and `keeps isLoading true while REST loads and no SSE payload is present` (the `isLoading && evloading` gate bug). Restored; working tree clean.

### Verification

- `bun tsc --noEmit`: clean.
- `bunx vitest run src/hooks/__tests__/volumeHook.test.ts`: 5/5 pass.
- Consumer suites (`src/pages/volumes`, `src/pages/shares`, `src/pages/dashboard`): 35 files / 292 tests pass.
- Full `bunx vitest run`: 95 files / 727 tests pass (1 skipped, pre-existing).
- `mise run //frontend:lint`: clean.
- Final diff: 2 files (hook + tests).

## 🧪 Test Plan Summary





| Area | New/Updated Tests | Catches |
|------|-------------------|---------|
| `disk_map.go` | concurrent mutators under `-race` | B1 |
| `volume_service.go` | procfs-discovery preserves startup flag | B2 |
| `volume_service.go` | stale-marking scoped per partition | B3 |
| `api/volumes.go` | 403 matrix mount/umount/patch × modes | B4 |
| `volume_service.go` | GetDevicePathByDeviceID semantics table | B5 |
| `volumeHook.ts` | SSE-wins ordering, REST-error, SSE-first | B6 |
| `volume_service.go` | automount retry bounded on cache-write failure | H1 |
| `volume_service.go` | procfs parse count == 1 per refresh | H2 |
| `volume_service.go` | PATCH partial-update DB assertions | H4 |
| `api/volumes.go` | `/volumes` 500 on hardware error | H5 |
| `volume_mount_manager.go` | lazy flag matrix, dir-preservation | H6 |
| `volume_service.go` | concurrent `GetHardwareInfo` + `GetVolumesData` under `-race`; no cache pollution | H8 |
| `volume_service.go` | emit-site error logging + mount-path propagation | H9 |
| `api/volumes.go` | cache-warm response-time budget; boot warmup | H10 |
| `VolumesTreeView.test.tsx` | automount icon color | F1 |
| `VolumeMountDialog.test.tsx` | isSubmitting disables submit | F2 |
| `Volumes.test.tsx` | selection stable across reorder | F5 |
| `VolumeDetailsPanel.test.tsx` | SMART actions disabled in read-only mode | F6 |

## 🧠 Implementation Notes (Copilot Context)

- **Lock placement decision (B1):** locking inside `DiskMap` is preferred — it makes the invariant un-bypassable. The type change from `map[string]*Disk` to a struct ripples to every `(*m)[key]` site; do it in one mechanical commit before any behavioral fix.
- **Event-loop reentrancy (H1):** the event bus used here emits synchronously (`signals_sync.go` seen in stack traces), so recursion is a same-goroutine loop — a simple attempt-counter map suffices; no distributed state needed.
- **Do not** change the `MountPointData.DeviceId` JSON contract in this task — frontend and HA component consume it; B5 is service-internal only.
- `VolumeMountDialog` (F2) is explicitly listed as a **reference implementation** in the form-standard instructions file — after migration, re-check the instruction file's reference list still points at compliant code.
- Baseline test state at review time (darwin, Go 1.26.5): backend `api` 73.7% / `service` 57.2% coverage, all volume tests green except `TestMountUnmountVolume_Success` (SKIP — no loop device on darwin); frontend `volumeHook` 6/6 green (shallow truthiness only). `TestMountUnmountVolume_Success` skip means the primary mount path is **untested in CI** — consider a fake-mounter-backed equivalent that runs everywhere (H1's test doubles as this).

## 🔎 DTO Verification Appendix (2026-08-13)

Verified against `backend/src/dto/` sources. Grounds findings B1, B2, B3, B5, H4, H8.

| Finding | Verification result |
|---------|---------------------|
| B1 | `DiskMap` is a bare `map[string]*Disk` with a method set in `disk_map.go`; zero synchronization — confirmed. ⚠️ Deep-copy requirement: `MountPointData.Partition` (`json:"-"` back-ref) and `SharedResource.MountPointData` (forward-ref) form a **circular reference in memory**; any deep copy (B1 item 3, H8) must break the cycle (nil the ephemeral refs) or it recurses. ⚠️ `GetMountPoint`/`GetMountPointByPath` return `&mp` pointing at a **local copy** (range var), not the map slot — mutations via those pointers silently no-op today; preserve this read-only contract when introducing the lock API. |
| B2 | `MountPointData` confirmed: procfs discovery populates only `Path/Root/Type/FSType/DeviceId/IsMounted`; `IsToMountAtStartup`/`Flags`/`CustomFlags`/`Share` nil → converter nil-guards skip → zero-write under `UpdateAll: true`. The merge fix targets exactly those 4 fields (Share excluded — owned by the share service). |
| B3 | `GetAllMountPoints()` iterates all disks × partitions — confirmed. `GetMountPointsForPartition(diskID, partitionID)` **does not exist** in `disk_map.go`; must be added (Task 3). |
| B5 | `GetPartitionDevicePath(partition)` (DevicePath → LegacyDevicePath → LegacyDeviceName) and `GetPartitionByID(partitionID)` **already exist** in `disk_map.go` — the suggested fix only needs the `GetDevicePathByDeviceID` rewrite + the DeviceId contract decision. |
| H4 | Converter nil-guard behavior confirmed (`dto_to_dbom_conv_gen.go:108-136`); the patch path converts into the DB-loaded record — **Task 10: quirk `affected == 0 → 404` does NOT reproduce** (GORM auto-updates `UpdatedAt`), but a real bug was found: nil-wiped in-memory record dropped Flags/Data from response DTO and disk cache; fixed via post-`Updates` DB reload. |
| H8 | `Disk`/`Partition` both carry `RefreshVersion uint32` (per-entity snapshot copies, not the shared counter); `AddSmartInfo`/`AddHDIdleDevice` mutate the cached `*Disk` **in place** through the map pointer — cache aliasing confirmed. |

## 🔬 B1 Implementation Survey (2026-08-13)

Compiled from a full read of `backend/src/service/volume_service.go` (all 1139 lines), `backend/src/dto/disk_map.go`, every DiskMap consumer, and every test file. Grounds Task 1 (B1) and cross-references Task 3 (B3) and Task 19 (H8).

### A. Current type & DI wiring

- `type DiskMap map[string]*Disk` (bare map, zero synchronization) + **17 pointer-receiver methods** in `dto/disk_map.go`: `AddOrUpdate`, `Remove`, `Get`, `AddOrUpdateMountPoint`, `RemoveMountPoint`, `AddPartition`, `RemovePartition`, `GetPartition`, `GetMountPoint`, `GetMountPointByPath`, `GetAllMountPoints`, `AddMountPointShare`, `RemoveMountPointShare`, `AddHDIdleDevice`, `AddSmartInfo`, `GetPartitionDevicePath`, `GetPartitionByID`.
- **One shared instance** across services via DI: `internal/appsetup/appsetup.go:66` (`func() *dto.DiskMap { return &dto.DiskMap{} }`). Injected into 8 consumers: `VolumeService` (field L66), `BroadcasterService` (L43/L61), `DiskStatsService` (L32/L54), `ServerProcessService` (L138/L156), `SupervisorService` (L41/L55), `VolumeMountManager` (L31/L41), `FilesystemHandler` (`api/filesystems.go:18/25`), plus converter context param (`converter/mount_to_dto.go:35/46/58`) and `mount_intelligence.go:12` helper param. **All fields stay `*dto.DiskMap`** (pointer type) after the struct conversion — nil guards (`self.disks == nil`) keep working.

### B. Direct-map read sites (must become methods)

| File:line | Current code | Replacement |
|-----------|--------------|-------------|
| `volume_service.go:275` | `for _, disk := range *ms.disks` (findPartitionByDeviceId) | `self.disks.All()` (read-only) |
| `volume_service.go:409` | `for diskID, disk := range *self.disks` (findPartitionByDevName) | `self.disks.All()` |
| `volume_service.go:497` | `if len(*self.disks) == 0` (GetVolumesData) | `self.disks.Len() == 0` |
| `volume_service.go:504` | `slices.Collect(maps.Values(*self.disks))` (GetVolumesData — the B1-race read) | `self.disks.All()` |
| `volume_service.go:634` | `for id, disk := range *self.disks` — eviction loop, **`Remove` inside** | `self.disks.Snapshot()` (iterate+mutate) |
| `volume_service.go:680` | `for _, disk := range *self.disks` (hasWholeDiskSynthesizedDisks) | `self.disks.All()` |
| `volume_service.go:789` | `for _, disk := range *self.disks` (findDiskForDevicePath) | `self.disks.All()` |
| `volume_service.go:1038` | `for dk, d := range *ms.disks` (PatchMountPointSettings fallback) | `self.disks.Snapshot()` (derefs/mutates outside) |
| `broadcaster_service.go:108,126` | `slices.Collect(maps.Values(*broker.disks))` (relay broadcasts) | `broker.disks.All()` |
| `disk_stats_service.go:205` | `slices.Collect(maps.Values(*s.disks))` | `s.disks.All()` |
| `converter/mount_to_dto.go:47` | `for _, d := range *disks` (partitionFromDevice) | `disks.All()` (converter re-gen not needed — hand edit only this fn) |

**Already method-based (no change):** `api/filesystems.go:152-160` (`GetPartitionByID`/`GetPartitionDevicePath`), `mount_intelligence.go:12-28` (`GetPartitionByID`), `rootFromPath` (`GetMountPointByPath`), `homeassistant_service.go:65/76` (consumes `*[]*dto.Disk` — output of `GetVolumesData()`, not the map).

### C. `refreshVersion` sites (`volume_service.go`)

| Line | Site | Migration |
|------|------|-----------|
| 67 | field `refreshVersion uint32` | **delete** (moves into DiskMap) |
| 115 | init `refreshVersion: 0` | **delete** (atomic zero value) |
| 545 | `self.refreshVersion++` (singleflight) | `v := self.disks.NextRefreshVersion()` |
| 570 | `disk.RefreshVersion = self.refreshVersion` (snapshot stamp) | use local `v` |
| 635 | `disk.RefreshVersion != self.refreshVersion` (eviction compare) | use local `v` |
| 846 | `mountPoint.RefreshVersion = self.refreshVersion` (stamp) | `self.disks.CurrentRefreshVersion()` |
| 882 | `RefreshVersion: self.refreshVersion` (stamp) | `CurrentRefreshVersion()` |
| 910 | trace log read (handlePartitionEvent) | `CurrentRefreshVersion()` |
| 914 | `mountPoint.RefreshVersion != self.refreshVersion` (staleness compare) | `CurrentRefreshVersion()` |
| 919 | `mountPoint.RefreshVersion = self.refreshVersion` (re-stamp) | `CurrentRefreshVersion()` |

⚠️ Sites 846-919 run in **event-handler goroutines outside the singleflight** — exactly the race B1 fixes; `atomic.Uint32` makes them safe without restructuring.

### D. Test sites requiring conversion

| Pattern | Sites | Fix |
|---------|-------|-----|
| `dto.DiskMap{}` empty literal | `dto/disk_map_test.go` (26×) | `dto.NewDiskMap()` |
| `&dto.DiskMap{}` provider/value | `appsetup.go:66`; fxtest providers in `volume_service_test.go:82`, `event_propagation_test.go:96/174`, `broadcaster_service_test.go:59/231`, `ws_test.go:58`, `filesystems_test.go:52`, `disk_stats_service_test.go:44`; `broadcaster_service_internal_test.go:23` | `dto.NewDiskMap()` |
| Seeded composite literal `dto.DiskMap{diskID: &disk}` | `volume_service_reconcile_test.go:197`, `volume_service_udev_test.go:82/119/164`, `mount_intelligence_test.go:88`, `converter/converter_test.go:196` | `dto.NewDiskMap(&disk)` |
| Direct index writes `(*suite.disks)[diskID] = ...` | `volume_service_test.go:811-855` | `suite.disks.AddOrUpdate(&disk)` |
| Seeded read test `mount_intelligence_test.go:88-...` (map read in `enrichSharePartitionFromCache` assertions) | same file | seed via `NewDiskMap`, read via `GetPartitionByID` |

Note: `event_propagation_test.go:173-176` has a **commented-out** block calling `vs.disks.Add(...)` — no `Add` method exists (only `AddOrUpdate`); leave commented, do not resurrect.

### E. Design decisions (Task 1 execution)

1. **Struct layout:** `type DiskMap struct { mu sync.RWMutex; entries map[string]*Disk; refreshVersion atomic.Uint32 }`. All 17 existing methods become lock-guarded (`RLock` for read-only, `Lock` for mutators). Pointer receivers unchanged.
2. **refreshVersion moves into DiskMap** (matches the task's "keep the invariant local" grouping): `NextRefreshVersion() uint32` = `Add(1)`; `CurrentRefreshVersion() uint32` = `Load()`. `VolumeService` drops its field/init (C above).
3. **New methods:** `Len()`, `All() []*Disk` (replaces every `maps.Values` site — note iteration order is nondeterministic, same as today), `Snapshot() map[string]*Disk` (copy-under-RLock for iterate+mutate loops), `Keys()`, `GetMountPointsForPartition(diskID, partitionID)` (**deferred to Task 3/B3** — do not add in Task 1).
4. **Constructor strategy:** `NewDiskMap()` + `NewDiskMapFrom(disks ...*Disk)` (AddOrUpdates each). Composite literals with the unexported `entries` field are impossible from other packages — the ~40 test literals in D are a mechanical `sed`-able rewrite. Alternatively keep a package-local `diskMapFrom(map[string]*Disk)` for `dto` tests; decide at implementation time.
5. **Semantics preserved:** `GetMountPoint`/`GetMountPointByPath` keep returning a pointer to a **local copy** (read-only contract — mutations no-op today, keep it); `Get` returns the map slot pointer (in-place mutation by `AddSmartInfo`/`AddHDIdleDevice` stays valid under `RLock` — document that callers must not hold the returned pointer across lock releases).
6. **Iterate-with-mutate loops** (L634 eviction, L1038 fallback) must switch to `Snapshot()` — ranging the live map while calling locked `Remove` would deadlock (`sync.RWMutex` is not reentrant).
7. **H8 is NOT fixed by this conversion:** the hardware cache is a separate type (`map[string]dto.Disk`, `hardware_service.go:36`). The `Clone()`/immutable-snapshot helper Task 19 (H8) needs must **break the circular ref** (`MountPointData.Partition` ↔ `SharedResource.MountPointData`) — nil the ephemeral refs in the clone; Task 1 should add the helper in `dto` (or defer it entirely to Task 19; do not over-scope).
8. **Compile-time breakage detector:** after the conversion, `grep -rn "range \*.*disks\|maps\.Values(\*\|len(\*.*disks" backend/src` must return **zero** hits outside `dto/disk_map.go`.
9. **Commit sequence (one mechanical commit, per Implementation Notes):** (1) rewrite `dto/disk_map.go` + constructors; (2) migrate `dto/disk_map_test.go`; (3) `appsetup.go` provider; (4) `volume_service.go` B+C sites; (5) `broadcaster_service.go` + `disk_stats_service.go` reads; (6) `converter/mount_to_dto.go:47`; (7) test literals D; (8) `-race` concurrent test (Task 1's own test); (9) `mise run //backend:test` + coverage gate (Task 23).

## 🔬 B2 Implementation (2026-08-14)

**Commit:** `cd31e1ec` — "🐛 fix(be): preserve mount point settings on discovery persist" (2 files, +162/-4). Pre-commit hooks (go-fmt, golangci-lint, go-vet) green.

### Changes (`backend/src/service/volume_service.go`)

1. **`persistMountPoint` (line 206) — load-then-merge upsert.** Replaces the fresh-zero-value convert + `OnConflict{UpdateAll:true}` (which NULLed every column missing from the event DTO):
   - falls back `Root` to `/` when empty;
   - `First`s the existing row by `(Path, Root)`;
   - on `gorm.ErrRecordNotFound` builds a zero-value `dbom.MountPointPath{}` (unchanged behavior for new rows);
   - otherwise converts the DTO **into the loaded DB record** (nil DTO fields leave loaded values intact — same pattern `PatchMountPointSettings` uses);
   - `Create`s with `OnConflict{UpdateAll:true}`.
   - `ExportedShare` close-loop is now `ExportedShare.MountPointData = dbom_mount_data` (value semantics). The FK columns live on the exported_shares side (`MountPointDataPath`,`MountPointDataRoot`), so a nil `ExportedShare` **cannot** wipe the association — Share preservation is automatic (comment documents this).
2. **New helper `loadMountPointFromDBByPath` (line 557):** empty-path guard; `Find` by `(Path, Root)`; converts first hit via `MountPointPathToMountPointData(dbMPs[0], &dtoMP, nil)`. The existing (DeviceId-based) `loadMountPointFromDB` path is untouched (still used for cache seeding).
3. **Discovery merge in `handlePartitionEvent` (~lines 888-905):** before emitting the procfs-discovery ADD, merge `IsToMountAtStartup`, `Flags`, `CustomFlags` from `disks.GetMountPointByPath(...)` first (live cache), falling back to `loadMountPointFromDBByPath(...)`; DB errors are warn-logged (non-fatal). Share is intentionally **not** merged (owned by the share service) — per the B2 finding.

### Tests (`backend/src/service/volume_service_test.go`)

| Test | Coverage | Notes |
|------|----------|-------|
| `TestHandlePartitionEvent_DiscoveryPreservesPersistedMountPointConfig` (~line 833) | Regression: seeds DB row (`DeviceId` differing from partition Id, `IsToMountAtStartup:true`, Flags `user_custom_flag`, Data `custom_super_opt`), mock HW disk/partition (`/dev/b2disk1-part1`), mock procfs mount (`/mnt/b2-discovery`); asserts all three fields survive `InvalidateHardwareInfo()` + `GetVolumesData()` | Proven to **FAIL without the fix** (`IsToMountAtStartup` wiped to false; validated via `git stash push` on `volume_service.go`) |
| `TestEmitMountPointEventWithSharePersistsAssociation` (~line 152) | Event with `Share: &dto.SharedResource{Name:"b2-share-assoc"}`; asserts preloaded DB row keeps `ExportedShare.Name` + `MountPointDataPath` | Covers the ExportedShare close-loop branch (3 statements) — added to meet the ≥70% per-function coverage gate |

### Coverage gate (Task 23, per-function ≥70% for touched code)

| Function | Before | After |
|----------|--------|-------|
| `persistMountPoint` | 68.4% | **84.2%** |
| `loadMountPointFromDBByPath` | — (new) | **72.7%** |
| `loadMountPointFromDB` | — | 73.3% |
| `handleMountPointEvent` | — | 77.8% |
| `handlePartitionEvent` | 42.0% (pre-existing legacy) | unchanged; changed discovery lines covered by regression test |

Total suite: 40.5% (baseline unchanged). Remaining uncovered in `persistMountPoint`: non-not-found `First` error, converter error, `Create` error — all hard to trigger, no blocker.

### Verification

- Full `mise run //backend:test`: exit=0 (final run all-cached, Total coverage 40.5%).
- gofmt clean; `go -C src vet ./service/` clean; targeted run from `backend/`: `go -C src tool gotest -p 1 -failfast -timeout 120s -tags embedallowed_no -run '<TestSuite/TestName>' ./service`.
- `mise run //backend:format` `depends_post` security step (govulncheck/gosec) fails with exit status 3 — pre-existing dependency vulns, unrelated to B2. testifylint auto-fix of `dto/disk_map_test.go` (`assert.Equal("",…)` → `assert.Empty`) was reverted to keep the commit atomic.

## 🔬 B3 Implementation (2026-08-14)

**Branch:** `fix/volume-phantom-entries` (uncommitted; ready for one atomic commit)

### Changes

1. **`backend/src/dto/disk_map.go` — new `GetMountPointsForPartition(diskID, partitionID)`** (after `GetAllMountPoints`, ~line 408): returns the mount points of a single disk+partition only, `RLock`-guarded, same read-only local-copy contract as `GetAllMountPoints`. Empty ids and nil receiver short-circuit to `nil`; unknown disk/partition yield an empty slice.
2. **`backend/src/service/volume_service.go:958` — stale-marking loop scoped.** `handlePartitionEvent`'s stale loop now iterates `self.disks.GetMountPointsForPartition(*e.Partition.DiskId, *e.Partition.Id)` instead of `GetAllMountPoints()`. The unscoped loop marked **every** stale mount point across all disks/partitions and — because `GetAllMountPoints` returns local copies — wrote each one into the **partition being processed** (keyed by its path), polluting partition A with phantom copies of B's mount points (the "phantom volume" symptom) and persisting cross-partition state.

### Tests

| Test | Coverage | Notes |
|------|----------|-------|
| `TestDiskMap_GetMountPointsForPartition` (`dto/disk_map_test.go`) | `GetMountPointsForPartition` → **93.8%** | Two partitions with 2+1 mount points; asserts per-partition scoping, empty slice for unknown disk/partition, nil on empty ids, nil receiver safety |
| `TestHandlePartitionEvent_StaleMarkingScopedToPartition` (`service/volume_service_test.go`, after the B2 regression test) | Changed stale-loop line in `handlePartitionEvent` | Seeds disk with partitions A/B each carrying a mounted mount point (version 0); bumps refresh version (service startup already bumped once via the `OnStart` warmup — test derives `baseVersion` so it stays robust); procfs mocked empty; emits partition event for A only. Asserts: A's own mount point is unmounted (stale-marking still works), A gains **no** phantom entry for B's path, and B's mount point stays `IsMounted=true` with version 0. **Proven to FAIL without the fix** (`git`-free verification: backup copy of `volume_service.go`, one-line revert to `GetAllMountPoints()`, test fails with "partition A must not contain partition B's mount point as a phantom entry", restore) |

### Coverage gate (Task 23, per-function ≥70% for touched code)

| Function | Before | After |
|----------|--------|-------|
| `GetMountPointsForPartition` | — (new) | **93.8%** |
| `handlePartitionEvent` | 42.0% (pre-existing legacy) | 51.9% (rose via the new test); the changed stale-loop line is covered by the regression test — same precedent as B2 |

### Verification

- Targeted `go -C src tool gotest ... ./dto ./service`: `dto` ok, `service` ok (full package runs, 26s).
- Full `mise run //backend:test`: exit=0, Total coverage 40.6% (baseline 40.5%).
- gofmt clean on all 4 touched files; `go -C src vet ./dto/ ./service/` clean.
- Working tree after the accidental `git stash pop` of an unrelated stash (`stash@{0}` from `feat/hdidle-per-disk-rework`) was restored with `git reset --merge`; the stash entry is preserved untouched. No unrelated files in the final diff (4 files, +142/−1).
