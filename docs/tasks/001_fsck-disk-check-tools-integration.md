# [FEATURE]: fsck and Disk Check Tools Integration

**Target Repo:** `srat`  **Status:** 📅 Planned  **Issue Link:** [#185](https://github.com/dianlight/srat/issues/185)

## 🎯 Objective

Improve the integration of `fsck.*` and related disk-check utilities across the SRAT filesystem adapter stack. While the `Check()` interface and basic API endpoints (`/api/filesystem/check`, `/api/filesystem/check/abort`) are already present, the goal is to surface tool availability to users, ensure consistent tool detection across all adapters, and extend the UI to guide users when required tools are missing or checks are not supported.

> _Context for Copilot: The `FilesystemAdapter.Check()` method is already implemented for ext4, xfs, btrfs, ntfs, vfat, f2fs, reiserfs, and exfat adapters. ZFS intentionally omits Check (pool-level). The `FilesystemSupport` struct carries `CanCheck bool` and `MissingTools []string`. The goal is end-to-end coherence: detection → UI feedback → action._

## 🛠️ Technical Specifications

- **Inputs:**
  - Partition/device identifier (`partitionId`)
  - Check options: `autoFix`, `force`, `verbose`
  - Filesystem type (resolved from `dto.DiskMap`)

- **Outputs:**
  - `dto.CheckResult`: `success`, `errorsFound`, `errorsFixed`, `message`, `exitCode`
  - Real-time progress via WebSocket `filesystem_task` events
  - UI feedback: tool availability warnings, install hints, check history

- **Dependencies:**
  - `backend/src/service/filesystem/` — adapter implementations
  - `backend/src/service/filesystem_service.go` — `CheckPartition`, `AbortCheckPartition`
  - `backend/src/api/filesystems.go` — API handler
  - `backend/src/dto/filesystem.go` — `CheckOptions`, `CheckResult`, `FilesystemSupport`
  - `frontend/src/pages/volumes/components/FilesystemCheckDialog.tsx`
  - `frontend/src/pages/volumes/components/VolumeDetailsPanel.tsx`
  - `frontend/src/store/sratApi.ts` — generated RTK Query hooks (do not edit directly)

## 📝 Task List

- [ ] Task 1: Audit `IsSupported().CanCheck` across all adapters — verify tool lookup is correct and handles missing binaries gracefully
- [ ] Task 2: Expose `FilesystemSupport` (especially `CanCheck`, `MissingTools`, `AlpinePackage`) via a dedicated API endpoint or embed it in the existing volume/partition listing response
- [ ] Task 3: Update `FilesystemCheckDialog.tsx` to show a warning/hint when `CanCheck=false`, including the Alpine package name to install
- [ ] Task 4: Investigate and implement scheduled/automatic disk check triggers (e.g., on mount, or periodic background checks)
- [ ] Task 5: Enhance `dto.CheckResult` with additional diagnostics if tools support it (e.g., pass/fail summary, bad block count)
- [ ] Task 6: Unit testing — backend adapter `Check()` unit tests for all adapters, including missing-tool path
- [ ] Task 7: Integration tests — `CheckPartition` service method with mocked adapters
- [ ] Task 8: Frontend component test for `FilesystemCheckDialog` covering missing-tool warning state
- [ ] Task 9: Update OpenAPI spec and regenerate frontend types (`cd frontend && bun run gen`)
- [ ] Task 10: Documentation — update `docs/SHARE_VOLUME_VERIFICATION.md` or create new `docs/FSCK_INTEGRATION.md`

## 🧠 Implementation Notes (Copilot Context)

### Current state

- `FilesystemAdapter.Check(ctx, device, options, progress)` is defined in `service/filesystem/adapter.go` and implemented for: `ext4` (`fsck.ext4`), `xfs` (`xfs_repair`), `btrfs` (`btrfs check`), `ntfs` (`ntfsfix`), `vfat` (`fsck.fat`), `f2fs` (`fsck.f2fs`), `reiserfs` (`fsckfix` or `reiserfsck`), `exfat` (`fsck.exfat`).
- ZFS adapter intentionally has no `Check` implementation (pool-level ops only).
- `FilesystemSupport.CanCheck bool` is returned by `adapter.IsSupported(ctx)` and already checked in `FilesystemService.CheckPartition` before starting the operation.
- `MissingTools []string` and `AlpinePackage string` are available in `FilesystemSupport` for user-facing hints.
- Real-time progress is delivered via WebSocket `filesystem_task` events consumed by `FilesystemCheckDialog`.

### Gaps to address

- The frontend currently shows a generic error if `CanCheck=false`; it should instead show a structured hint (e.g., "Install `e2fsprogs` via `apk add e2fsprogs`").
- There is no lightweight API call to pre-check whether a tool is available before the user opens the dialog.
- No scheduled or post-mount automatic check mechanism exists.
- Some adapters report `progress=999` (unsupported) throughout; consider adding a note in the UI so users understand this is normal.
- `CheckResult` exit codes vary by tool; consider a normalised severity enum (`clean`, `errors_found`, `errors_fixed`, `fatal`).

### Suggested approach

1. Add a `GET /api/filesystem/support?fstype=<type>` endpoint (or extend existing `/api/filesystem/info`) that returns `FilesystemSupport` for a given filesystem type.
2. In `FilesystemCheckDialog`, call this endpoint on open and render a disabled state with install instructions when `canCheck=false`.
3. For scheduled checks, add a background `ticker` in `FilesystemService` gated by a new `AppConfig` boolean `autoFsckOnMount`.

## 🔗 Code References & TODOs

- [ ] `backend/src/service/filesystem/adapter.go` — `Check()` method signature and `FilesystemSupport` struct
- [ ] `backend/src/service/filesystem/ext4_adapter.go` — reference implementation for `Check()` and `IsSupported()`
- [ ] `backend/src/service/filesystem_service.go:677` — `CheckPartition` entry point
- [ ] `backend/src/api/filesystems.go:166` — `CheckPartition` HTTP handler
- [ ] `frontend/src/pages/volumes/components/FilesystemCheckDialog.tsx` — UI component to update
- [ ] `frontend/src/pages/volumes/components/VolumeDetailsPanel.tsx:190` — button that opens the dialog
- [ ] `FIXME: all adapters` — verify `IsSupported()` correctly resolves tool binary paths using `lookPath` (testable via `SetExecOpsForTesting`)
