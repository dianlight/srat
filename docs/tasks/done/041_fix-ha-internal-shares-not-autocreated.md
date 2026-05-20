# [FIX]: Home Assistant internal shares not auto-created on startup

**Target Repo:** `srat`
**Status:** ✅ Complete
**Issue Link:** https://github.com/dianlight/srat/issues/650

## 🎯 Objective

Fix the bug where the 7 standard Home Assistant internal shares (`config`, `addons`, `ssl`, `share`, `backup`, `media`, `addon_configs`) are not auto-created when the SRAT backend starts and the corresponding HA volumes are already mounted. The user reports that after installing/restarting Samba NAS2, the shares UI shows no default HA shares.

## 🔍 Root Cause Analysis

The share auto-creation logic in `backend/src/service/share_service.go` is **entirely event-driven**. It subscribes to `MountPointEvent` in `NewShareService` (line 113) and only creates internal shares when a mount event fires for a path matching the `internalShares` map (lines 20-49).

**The gap:** When the back-end starts, `VolumeService.getVolumesData()` (line 625) discovers hardware disks and partitions via the hardware client and emits `PartitionEvent` for each partition. The `handlePartitionEvent` handler (line 765) reads Linux mount information and emits `MountPointEvent` **only for partitions that have a matching device path in the mount info**.

However, the Home Assistant internal paths (`/config`, `/backup`, `/media`, etc.) are **not physical partitions** — they are bind-mounted directories provided by the HA Supervisor container. They do **not** appear in Linux mount information as partition mounts with a matching `Source == DevicePath`. This means:

1. `handlePartitionEvent` never emits `MountPointEvent` for these paths because they don't belong to any hardware partition
2. No `MountPointEvent` → the `OnMountPoint` subscriber in `ShareService` never fires
3. No subscriber fire → no auto-creation of internal shares

**Evidence from the issue logs:** The logs show `Autocreating user` messages (user auto-creation works) but **no** `Creating default share` debug messages. The logs also show warnings like `Share volume does not exist share=Mario path=/mnt/Mario` for user-created shares, confirming the share service is running but internal shares were never seeded.

**Secondary issue:** The `OnStart` hook in `NewShareService` (lines 149-155) is a no-op — it does not perform any startup-time scan for missing internal shares.

## 🛠️ Technical Specifications

- **Files to modify:**
  - `backend/src/service/share_service.go` — add startup-time internal share seeding
  - `backend/src/service/share_service_test.go` — add test for startup seeding behavior
- **Approach:** Add a startup scan in the `OnStart` hook of `NewShareService` that:
  1. Iterates all entries in `internalShares` map
  2. For each, checks if a share already exists in the database (`GetShareFromPath`)
  3. If not found, creates the share with admin user (same logic as the event handler)
  4. This is idempotent — if shares already exist, they are skipped
- **Dependencies:** None — uses existing `GetShareFromPath`, `CreateShare`, `user_service.GetAdmin`

## 📝 Task List

### Analysis & Design
- [x] Task 1: Confirm the root cause by verifying that HA internal paths (`/config`, `/backup`, etc.) do not appear in procfs mount info and are not associated with any hardware partition

### Implementation
- [x] Task 2: Add startup-time internal share seeding in `NewShareService` `OnStart` hook — iterate `internalShares`, check DB for existence, create missing ones with admin user
- [x] Task 3: Extract the share auto-creation logic into a reusable private method (e.g., `ensureInternalShare(path string)`) to avoid duplicating the logic between the event handler and the startup hook

### Testing
- [x] Task 4: Add unit test in `share_service_test.go` that verifies internal shares are created on startup when they don't exist
- [x] Task 5: Add unit test that verifies no duplicate shares are created when internal shares already exist in the database
- [x] Task 6: Run backend test suite (`mise run //backend:test`) — all must pass

### Verification
- [x] Task 7: Run `mise run //backend:format` and `go vet ./...` from `backend/src`
- [x] Task 8: Verify the fix addresses the scenario: fresh install → backend starts → HA paths already mounted → internal shares auto-created without requiring a mount event

## 🧠 Implementation Notes

- Pre-implementation inspection confirms `backend/src/service/share_service.go` only auto creates internal shares from the `OnMountPoint` subscription; `NewShareService`'s current `OnStart` hook is a no-op outside `SRAT_MOCK=true` test mode.
- `backend/src/service/volume_service.go` emits `MountPointEvent` only when Linux mount info entries match partition device paths or cached partition mount points, which does not cover Home Assistant bind-mounted internal paths like `/config` or `/backup`.
- Created and linked `dianlight/srat` issue: https://github.com/dianlight/srat/issues/650.
- Branch safety gate completed: `fix/ha-internal-shares-not-autocreated` created from `main`.
- Agreed implementation plan:
  - Add startup seeding in `NewShareService` `OnStart` for all `internalShares` entries.
  - Extract reusable private helper for internal share ensure/create logic shared by startup and mount-event path.
  - Add tests for startup seeding creation and duplicate prevention.
  - Validate with targeted tests first, then full back-end validation commands from the task.
- Implementation completed:
  - Added `ensureInternalShare` helper in `backend/src/service/share_service.go` and reused it in both mount-event handling and startup seeding.
  - Startup seeding now creates missing internal shares with a synthetic internal `DeviceId` for DB mount-point constraint compatibility.
  - Added `ShareServiceStartupSeedingSuite` in `backend/src/service/share_service_test.go` with coverage for startup creation and restart duplicate prevention.
- Validation completed:
  - `go test ./service -run TestShareServiceStartupSeedingSuite -count=1` (initially failing before fix, now passing).
  - `go test ./service -run 'TestShareServiceSuite|TestShareServiceStartupSeedingSuite' -count=1` passing.
  - `mise run //backend:test` passing.
  - `mise run //backend:format` and `go vet ./...` from `backend/src` passing.

## 🔗 Code References & TODOs

- `backend/src/service/share_service.go`: current internal share seeding lives inside the `OnMountPoint` subscription; `OnStart` is the likely insertion point for startup seeding.
- `back-end/src/service/volume_service.go`: `getVolumesData()` emits `PartitionEvent`, while `handlePartitionEvent()` emits `MountPointEvent` only for partition-backed mounts discovered via Linux mount info.
- `backend/src/service/share_service_test.go`: current suite sets `SRAT_MOCK=true`, so startup seeding tests will need a targeted way to exercise `OnStart` without touching real host paths.
