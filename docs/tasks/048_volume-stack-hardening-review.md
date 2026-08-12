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

### B2 — `IsToMountAtStartup` silently wiped when a mount is discovered from procfs

**File:** `volume_service.go:868-899` → `handleMountPointEvent` → `persistMountPoint` (line 225: `clause.OnConflict{UpdateAll: true}`)
**Root cause chain:**

1. `handlePartitionEvent` finds a mount in `/proc/mounts` not yet in cache → builds `dto.MountPointData{...}` (line 871) **without setting `IsToMountAtStartup`** (nil).
2. Emits `MountPointEvent{Type: ADD}` (line 895).
3. `handleMountPointEvent` (line 938) receives it and calls `persistMountPoint` → `UpdateAll: true` upsert → **NULLs the `is_to_mount_at_startup` column**, erasing the user's automount preference.

**Evidence:** any user who enables automount on a mount point, then triggers a volume refresh (udev event, HA start, provisional recheck) before the next mount event round-trip loses the flag. The `TestPatchMountPointSettings_UpdatesStartupFlagInGetVolumesData` test only passes because procfs mock returns empty mounts — it never exercises the collision.

**Fix (suggested):** in `handlePartitionEvent`, before emitting ADD for a procfs-discovered mount point, merge DB state:

```go
if existing, err := gorm.G[dbom.MountPointPath](self.db).
    Where(g.MountPointPath.Path.Eq(prtstate.MountPoint)).
    First(self.ctx); err == nil {
    mountPoint.IsToMountAtStartup = existing.IsToMountAtStartup
}
```

Alternatively: make `persistMountPoint` use a selective column update (`clause.OnConflict{DoUpdates: ...}` on non-nil fields only). The merge-approach is safer and explicit.

**Tests:** `TestPartitionEvent_ProcfsDiscoveryPreservesStartupFlag`: seed DB with `IsToMountAtStartup=true`, mock procfs with a matching mount, emit partition event, assert DB row still has `true`.

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

### H2 — `getVolumesData` emits per-partition events inside the snapshot; subscribers do per-partition DB + procfs I/O

**Files:** `volume_service.go:611-623` (emit), `803-935` (handler)
Cost per full refresh with P partitions: P × (`loadMountPointFromDB` query + `procfsGetMounts` full `/proc/mounts` parse + `GetAllMountPoints` slice alloc). Under a udev add-storm (USB hub), refreshes serialize through singleflight but each still costs O(P²) procfs parses. Two cheap wins:

1. Hoist `procfsGetMounts()` out of `handlePartitionEvent` — parse once per `getVolumesData` cycle and pass the slice via the event payload (or a request-scoped field).
2. Batch: emit one `PartitionsChanged` event carrying all changed partitions; handler iterates the payload, reusing one procfs parse and one DB transaction.

**Tests:** benchmark `BenchmarkGetVolumesData_10Disks50Partitions` before/after; assert procfs parse count == 1 via a counting mock.

### H3 — Duplicate device-path validation (dead code)

**File:** `volume_service.go:293-299` vs `346-352` — identical `md.Partition.DevicePath == nil || *md.Partition.DevicePath == ""` check twice; the second is unreachable. Delete lines 346-352's duplicated branch (keep the `os.Stat` existence check that follows it).

### H4 — `PatchMountPointSettings` full-converter overwrite risk

**File:** `volume_service.go:996` — `convDto.MountPointDataToMountPointPath(patchData, &dbMountData)` copies the patch over the loaded DB row. If the generated converter copies nil pointer fields as nil, a partial PATCH (e.g. only `is_to_mount_at_startup`) nil-wipes `flags`/`custom_flags`/`fstype`. The current frontend always spreads the full object so this is latent, not live — but the API contract says PATCH. Verify converter nil-handling; if it nil-wipes, switch to explicit field assignment:

```go
if patchData.IsToMountAtStartup != nil {
    dbMountData.IsToMountAtStartup = patchData.IsToMountAtStartup
}
// ... repeat per patchable field
```

**Tests:** PATCH with only `IsToMountAtStartup` set → assert `fstype`/`flags` unchanged in DB (extend `TestPatchMountPointSettings_Success_OnlyStartup` — it already asserts fstype preserved, but via the service's own converter path; add a DB-level assertion).

### H5 — `GetVolumesData()` masks load errors as "no disks"

**File:** `volume_service.go:496-506` — on `getVolumesData()` error, returns `[]*dto.Disk{}` and only logs. The UI shows an empty volume list instead of an error state. Change the public signature to `([]*dto.Disk, errors.E)` and let the API handler return 500 with detail; frontend `volumeHook` already surfaces `error`.

**Tests:** hardware client returns error → `GET /volumes` returns 500 (today: 200 + `[]`).

### H6 — `volumeMountManager.Unmount` directory removal semantics

**File:** `volume_mount_manager.go:162-174`

- `UnmountPartition(ctx, path, fsType, force, !force)` — lazy flag is `!force`, so a **normal** unmount is lazy. Lazy + immediate `os.Remove(md.Path)` races in-flight I/O; prefer `lazy=false` for normal unmount and only lazy for force, or drop the directory removal when lazy.
- `os.Remove` on a non-empty pre-existing directory fails (Warn only) → stale empty-ish dirs accumulate under `/mnt`. Document or switch to "remove only if we created it" (track creation in `Mount`).

**Tests:** fake fs service: (a) normal unmount asserts `lazy=false`; (b) unmount with pre-existing non-empty dir → no error returned, dir preserved.

### H7 — udev goroutine lifecycle

**File:** `volume_service_udev_linux.go:23-35` — `queue := make(chan netlink.UEvent, 10)`: on udev burst >10 events, netlink lib blocks or drops (impl-dependent); on shutdown `close(quit)` stops the monitor but an in-flight `queue <-` send may leak the producer goroutine. Buffer to 64+ and drain `queue`/`errorChan` after `close(quit)` before returning. Low frequency, low severity, cheap fix.

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

**Test:** `VolumesTreeView.test.tsx` — partition with automount-enabled mount point → icon has `color="primary"` (assert via `data-testid` or class query on the icon).

### F2 — `VolumeMountDialog` violates the mandatory form standard

**File:** `VolumeMountDialog.tsx:311-315, 69, 232`
Per `.opencode/instructions/react-hook-form-mui.instructions.md`:

- ❌ uses raw `<form onSubmit={handleSubmit(...)}>` → must be `<FormContainer formContext={formContext} onSuccess={...}>` wrapping `DialogContent` **and** `DialogActions`.
- ❌ `const [mounting, setMounting] = useState(false)` for submit state → use `formState.isSubmitting`.
- The submit button must be `type="submit"` inside the `FormContainer` (currently uses `form="mountvolumeform"` attr — works but non-standard).

Refactor per the Dialog Pattern in the instruction file; behavior unchanged. This also removes the `handleSubmit` wrapper and simplifies `handleCloseSubmit` to the `onSuccess(data)` signature.

**Test:** existing `VolumeMountDialog.test.tsx` must keep passing; add "submit button disabled while submitting" case.

### F3 — `Volumes.tsx` local `disks` state duplicates the source of truth

**File:** `Volumes.tsx:113, 198-200, 265-278`
`const [disks, setDisks] = useState(sourceDisks ?? [])` + sync effect + `handlePartitionLabelUpdated` mutating local state. Next SSE event overwrites the optimistic rename → label visually reverts until the backend round-trips. With the B6 hook fix (`useMemo` derivation), local state can only be kept if label updates go through RTK Query cache update (`dispatch(sratApi.util.updateQueryData(...))`) instead of `setDisks`. Recommend: drop local state; do the optimistic update on the RTK cache so SSE and REST both see it.

### F4 — `handleToggleAutomount` fires N uncoordinated PATCHes

**File:** `Volumes.tsx:539-588` — loops `Object.entries(partition.mount_point_data)` firing one PATCH per mount point, no `await`, no aggregate error handling, one toast per mount point. For multi-mount partitions this spams toasts and can interleave failures. Fix: aggregate into a single `Promise.allSettled`, one summary toast; ideally add a backend bulk endpoint later (out of scope here).

### F5 — Identifier helpers use array indices

**File:** `Volumes.tsx` + `volumes/utils.ts` (`getDiskIdentifier(disk, diskIdx)`, `getPartitionIdentifier(..., partIdx)`)
Selection/expansion state persisted in `localStorage` embeds `diskIdx`/`partIdx`. Disk/partition order comes from Go map iteration → **not stable across refreshes** → restored selection can point at the wrong partition after any event. Fix: make identifiers purely ID-based (`disk.id`, `partition.id`), keep index only as a display fallback. Verify `utils.ts` implementations when editing.

**Test:** render with disks in order [A, B], select B's partition, re-render with order [B, A] → selection still on B's partition.

---

## 📝 Task List

- [ ] Task 0: Run `prepare-refactor` skill (per AGENTS.md REFACTOR rule): baseline `mise run //backend:test` + `mise run //frontend:test`, record green state in `docs/refactors/048-volume-hardening.md`
- [ ] Task 1: **B1** — Add `RWMutex` to `DiskMap` (convert to struct with methods) + `atomic.Uint32` refreshVersion; update all call sites; `-race` clean
- [ ] Task 2: **B2** — Merge DB state before procfs-discovery ADD emit; add regression test
- [ ] Task 3: **B3** — Scope stale-marking to current partition; add `GetMountPointsForPartition` helper + test
- [ ] Task 4: **B4** — ProtectedMode/ReadOnlyMode guards on unmount + mount endpoints; table-driven API tests
- [ ] Task 5: **B5** — Fix `GetDevicePathByDeviceID` semantics + nil guard; caller audit; unit tests
- [ ] Task 6: **B6** — Rewrite `volumeHook` with `useMemo` derivation (delete both `useEffect`s); MSW tests
- [ ] Task 7: **H1** — Automount retry backoff (per-path attempt map, max 5, exp. backoff); loop test with failing cache write
- [ ] Task 8: **H2** — Hoist procfs parse out of per-partition handler (parse once per refresh, pass via payload); benchmark before/after
- [ ] Task 9: **H3** — Delete duplicate DevicePath validation block
- [ ] Task 10: **H4** — Verify converter nil-handling; switch to selective field updates if needed; DB-level assertion test
- [ ] Task 11: **H5** — `GetVolumesData` returns error; API 500 mapping; hook surfaces error; update callers
- [ ] Task 12: **H6** — Unmount lazy/force semantics + dir-removal-only-if-created; fake-fs tests
- [ ] Task 13: **H7** — udev channel buffer 64 + drain on shutdown
- [ ] Task 14: **F1** — Fix map-as-array icon bug + RTL test
- [ ] Task 15: **F2** — Migrate `VolumeMountDialog` to `FormContainer` pattern per instructions; keep tests green
- [ ] Task 16: **F3** — Drop local `disks` state; optimistic label update via RTK `updateQueryData`
- [ ] Task 17: **F4** — `Promise.allSettled` aggregation for automount toggle; single summary toast
- [ ] Task 18: **F5** — ID-only identifiers; localStorage migration note; reorder-stability test
- [ ] Task 19: Coverage gate: `mise run //backend:test` + `go tool cover -func=coverage.out` — every touched function ≥70%; frontend `bun tsc --noEmit` + `mise run //frontend:test:new` green
- [ ] Task 20: Update `CHANGELOG.md` under `[ 🚧 Unreleased ]`

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
| `VolumesTreeView.test.tsx` | automount icon color | F1 |
| `VolumeMountDialog.test.tsx` | isSubmitting disables submit | F2 |
| `Volumes.test.tsx` | selection stable across reorder | F5 |

## 🧠 Implementation Notes (Copilot Context)

- **Lock placement decision (B1):** locking inside `DiskMap` is preferred — it makes the invariant un-bypassable. The type change from `map[string]*Disk` to a struct ripples to every `(*m)[key]` site; do it in one mechanical commit before any behavioral fix.
- **Event-loop reentrancy (H1):** the event bus used here emits synchronously (`signals_sync.go` seen in stack traces), so recursion is a same-goroutine loop — a simple attempt-counter map suffices; no distributed state needed.
- **Do not** change the `MountPointData.DeviceId` JSON contract in this task — frontend and HA component consume it; B5 is service-internal only.
- `VolumeMountDialog` (F2) is explicitly listed as a **reference implementation** in the form-standard instructions file — after migration, re-check the instruction file's reference list still points at compliant code.
- Baseline test state at review time (darwin, Go 1.26.5): backend `api` 73.7% / `service` 57.2% coverage, all volume tests green except `TestMountUnmountVolume_Success` (SKIP — no loop device on darwin); frontend `volumeHook` 6/6 green (shallow truthiness only). `TestMountUnmountVolume_Success` skip means the primary mount path is **untested in CI** — consider a fake-mounter-backed equivalent that runs everywhere (H1's test doubles as this).
