# [FIX]: Home Assistant internal shares not auto-created on startup

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** https://github.com/dianlight/hassio-addons/issues/689

## 🎯 Objective

Fix the bug where the 7 standard Home Assistant internal shares (`config`, `addons`, `ssl`, `share`, `backup`, `media`, `addon_configs`) are not auto-created when the SRAT backend starts and the corresponding HA volumes are already mounted. The user reports that after installing/restarting Samba NAS2, the shares UI shows no default HA shares.

## 🔍 Root Cause Analysis

The share auto-creation logic in `backend/src/service/share_service.go` is **entirely event-driven**. It subscribes to `MountPointEvent` in `NewShareService` (line 113) and only creates internal shares when a mount event fires for a path matching the `internalShares` map (lines 20-49).

**The gap:** When the backend starts, `VolumeService.getVolumesData()` (line 625) discovers hardware disks and partitions via the hardware client and emits `PartitionEvent` for each partition. The `handlePartitionEvent` handler (line 765) reads procfs mount information and emits `MountPointEvent` **only for partitions that have a matching device path in procfs**.

However, the Home Assistant internal paths (`/config`, `/backup`, `/media`, etc.) are **not physical partitions** — they are bind-mounted directories provided by the HA Supervisor container. They do **not** appear in procfs as partition mounts with a matching `Source == DevicePath`. This means:

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
- [ ] Task 1: Confirm the root cause by verifying that HA internal paths (`/config`, `/backup`, etc.) do not appear in procfs mount info and are not associated with any hardware partition

### Implementation
- [ ] Task 2: Add startup-time internal share seeding in `NewShareService` `OnStart` hook — iterate `internalShares`, check DB for existence, create missing ones with admin user
- [ ] Task 3: Extract the share auto-creation logic into a reusable private method (e.g., `ensureInternalShare(path string)`) to avoid duplicating the logic between the event handler and the startup hook

### Testing
- [ ] Task 4: Add unit test in `share_service_test.go` that verifies internal shares are created on startup when they don't exist
- [ ] Task 5: Add unit test that verifies no duplicate shares are created when internal shares already exist in the database
- [ ] Task 6: Run backend test suite (`mise run //backend:test`) — all must pass

### Verification
- [ ] Task 7: Run `mise run //backend:format` and `go vet ./...` from `backend/src`
- [ ] Task 8: Verify the fix addresses the scenario: fresh install → backend starts → HA paths already mounted → internal shares auto-created without requiring a mount event
