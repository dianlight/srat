# [FEATURE]: Disk Check Tools Integration (check, format, label)

**Target Repo:** `srat`  
**Status:** 🔄 In Progress  
**Issue Link:** [#185](https://github.com/dianlight/srat/issues/185)

## 🎯 Objective

Improve the integration of disk-management utilities across the SRAT filesystem adapter stack. The core interfaces and API endpoints for check/format/abort are already present (for example, `/api/filesystem/check`, `/api/filesystem/check/abort`, `/api/filesystem/format`, `/api/filesystem/format/abort`).

The next goal is end-to-end coherence from adapter capability detection to user feedback in the UI:

- Surface tool availability before execution
- Keep tool detection consistent across adapters
- Provide actionable guidance when required tools are missing
- Ensure check/format/label flows behave consistently

> _Context for Copilot: The `FilesystemAdapter.Check()` method is already implemented for ext4, xfs, btrfs, ntfs, vfat, f2fs, reiserfs, and exfat adapters. ZFS intentionally omits `Check` (pool-level operation). The `FilesystemSupport` struct carries `CanCheck bool` and `MissingTools []string`. The goal is end-to-end coherence: detection → UI feedback → action._

## 🛠️ Technical Specifications

- **Inputs:**
  - Partition/device identifier (`partitionId`)
  - Check options: `autoFix`, `force`, `verbose`
  - Filesystem type (resolved from `dto.DiskMap`)
  - Label value for `SetLabel()` operations

- **Outputs:**
  - `dto.CheckResult`: `success`, `errorsFound`, `errorsFixed`, `message`, `exitCode`
  - Real-time progress via WebSocket `filesystem_task` events
  - UI feedback: tool availability warnings, install hints, and operation status/results

- **Dependencies:**
  - `backend/src/service/filesystem/` — adapter implementations
  - `backend/src/service/filesystem_service.go` — `CheckPartition`, `AbortCheckPartition`
  - `backend/src/api/filesystems.go` — API handler
  - `backend/src/dto/filesystem.go` — `CheckOptions`, `CheckResult`, `FilesystemSupport`
  - `frontend/src/pages/volumes/components/FilesystemCheckDialog.tsx`
  - `frontend/src/pages/volumes/components/VolumeDetailsPanel.tsx`
  - `frontend/src/store/sratApi.ts` — generated RTK Query hooks (do not edit directly)

## 📝 Task List

- [x] Task 1: Verify the backend api support for all needed functions (`CheckPartition`, `AbortCheckPartition`, `FormatPartition`, `AbortFormatPartition`, `SetLabelPartition`) and their integration with the adapter layer
- [x] Task 2: Modify `FilesystemCheckDialog.tsx` and `VolumeDetailsPanel.tsx`, trace the flow of the "Check Filesystem" action through the frontend, API call, service layer, and adapter execution to ensure all pieces are wired correctly. Add the visual terminal output display in the dialog to show real-time progress and results from the check operation.
- [x] Task 3: Implement frontend logic to handle real-time progress updates from WebSocket `filesystem_task` events and display them in the `FilesystemCheckDialog`.
- [x] Task 5: Implement error handling and user feedback in the frontend when a check operation fails, including displaying the error message returned from the backend and any relevant diagnostics.
- [x] Task 6: Ensure that the check operation can be aborted by the user and that the backend properly handles the abort request, including cleaning up any ongoing processes and returning an appropriate response.
- [ ] Task 7: Add unit tests for the backend service methods and adapter implementations related to filesystem checking, as well as integration tests for the API endpoints and frontend components.
- [ ] Task 8: Update documentation to reflect the new disk check features, including any user-facing instructions for how to use the check functionality and interpret results.
- [x] Task 9: Verify that the frontend correctly handles cases where the required check tools are not available, showing appropriate warnings and installation hints based on the `MissingTools` and `AlpinePackage` information from the backend.
- [ ] Task 10: Repeat the job done from task 2 to 9 for the related `Format()` and `SetLabel()` functionalities, ensuring a consistent user experience across all disk management operations.
- [ ] Task 11: Clean up any temporary debug code (e.g., console logs) and ensure that all new code adheres to the project's coding standards and best practices.
- [ ] Task 12: Conduct thorough testing across different filesystem types to ensure that the check, format, and label operations work correctly and that the UI feedback is accurate for each type.
- [ ] Task 13: Run `hk check` to ensure that all new code is properly linted and formatted, and that all tests pass successfully
- [ ] Task 14: Do end-to-end testing of the entire flow, from initiating a check operation in the UI to receiving real-time updates and handling results, to ensure a smooth and intuitive user experience.
- [ ] Task 15: Push the changes to the repository and create a pull request for review, ensuring that the PR description clearly outlines the changes made and any relevant context for reviewers.

## 🧠 Implementation Notes (Copilot Context)

### Current state

- `FilesystemAdapter.Check(ctx, device, options, progress)` is defined in `service/filesystem/adapter.go` and implemented for: `ext4` (`fsck.ext4`), `xfs` (`xfs_repair`), `btrfs` (`btrfs check`), `ntfs` (`ntfsfix`), `vfat` (`fsck.fat`), `f2fs` (`fsck.f2fs`), `reiserfs` (`fsckfix` or `reiserfsck`), `exfat` (`fsck.exfat`).
- ZFS adapter intentionally has no `Check` implementation (pool-level operations only).
- `FilesystemSupport.CanCheck` is returned by `adapter.IsSupported(ctx)` and already checked in `FilesystemService.CheckPartition` before starting the operation.
- `MissingTools []string` and `AlpinePackage string` are available in `FilesystemSupport` for user-facing hints.
- Real-time progress is delivered via WebSocket `filesystem_task` events consumed by `FilesystemCheckDialog`.

### Gaps to address

- The frontend currently shows a generic error if `CanCheck = false`; it should instead show a structured hint (for example, "Install `e2fsprogs` via `apk add e2fsprogs`").
- There is no lightweight API call to pre-check whether a tool is available before the user opens the dialog.
- No scheduled or post-mount automatic check mechanism exists.
- Some adapters report `progress = 999` (unsupported) throughout; add a note in the UI so users understand this is expected behavior.
- `CheckResult` exit codes vary by tool; consider a normalized severity enum (`clean`, `errors_found`, `errors_fixed`, `fatal`).

### Suggested approach

1. Add a `GET /api/filesystem/support?fstype=<type>` endpoint (or extend existing `/api/filesystem/info`) that returns `FilesystemSupport` for a given filesystem type.
2. In `FilesystemCheckDialog`, call this endpoint on open and render a disabled state with install instructions when `canCheck = false`.
3. Reuse the same capability-detection and guidance pattern for `Format()` and `SetLabel()` dialogs/actions.
4. Optionally, for scheduled checks, add a background ticker in `FilesystemService` gated by a new `AppConfig` boolean `autoFsckOnMount`.

### Agreed implementation plan (2026-04-03)

- Add a backend capability endpoint for per-filesystem support pre-check and expose actionable metadata (`can*`, `missingTools`, `alpinePackage`).
- Wire frontend check dialog to preflight support on open and disable actions with install guidance when unsupported.
- Improve check dialog UX for realtime progress/log display, including explicit note for indeterminate `progress=999`.
- Align abort/error feedback so users receive clear, actionable status and failure messages.
- Reuse the same support-guidance pattern for format and label actions in follow-up steps.
- Add targeted backend/frontend tests for unsupported tools, abort behavior, and progress rendering.

### Progress update (2026-04-03)

- Added backend preflight endpoint `GET /api/filesystem/support?fstype=<type>` in `api/filesystems.go`.
- Added backend API tests for support endpoint success and validation in `api/filesystems_test.go`.
- Regenerated backend OpenAPI + frontend RTK Query client to expose `useGetApiFilesystemSupportQuery`.
- Updated `FilesystemCheckDialog` to preflight support, disable start when unsupported, and show missing tools + Alpine package install hint.
- Added explicit UI note for indeterminate `progress=999` behavior.
- Added frontend dialog tests for unsupported support-state UX and progress note rendering.
- Replaced `Set Label` and `Format Partition` placeholder console actions with functional dialogs wired to backend mutations.
- Added support preflight + missing-tools/install-hint UX to `FilesystemLabelDialog` and `FilesystemFormatDialog`.
- Removed debug `console.*` action-flow noise from `PartitionActionItems.ts`.

## 🔗 Code References & TODOs

- [ ] `backend/src/service/filesystem/adapter.go` — `Check()` method signature and `FilesystemSupport` struct
- [ ] `backend/src/service/filesystem/ext4_adapter.go` — reference implementation for `Check()` and `IsSupported()`
- [ ] `backend/src/service/filesystem_service.go:677` — `CheckPartition` entry point
- [ ] `backend/src/api/filesystems.go:166` — `CheckPartition` HTTP handler
- [ ] `backend/src/api/filesystems.go` — `GetFilesystemSupport` preflight endpoint (`GET /filesystem/support`)
- [ ] `frontend/src/pages/volumes/components/FilesystemCheckDialog.tsx` — UI component to update
- [ ] `frontend/src/pages/volumes/components/FilesystemLabelDialog.tsx` — set-label UX dialog with support preflight
- [ ] `frontend/src/pages/volumes/components/FilesystemFormatDialog.tsx` — format UX dialog with support preflight
- [ ] `frontend/src/pages/volumes/components/VolumeDetailsPanel.tsx:190` — button that opens the dialog
- [ ] `FIXME: all adapters` — verify `IsSupported()` consistently resolves tool binary paths using `lookPath` (testable via `SetExecOpsForTesting`)
