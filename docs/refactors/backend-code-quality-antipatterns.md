# Refactor: Backend Code Quality — Anti-patterns

<!-- DOCTOC SKIP -->

**Date:** 2026-04-21
**Status:** 🔧 Ready
**Prepare Check:** Yes
**Linked Task:** docs/tasks/028_backend-code-quality-antipatterns.md
**Scope:** Eliminate recurring Go anti-patterns: `interface{}` → `any`, `errors.As` → `errors.AsType[T]`, `wg.Add/Done` → `wg.Go`, extract `drainCommandOutput` helper, context-aware slog, fix FIXME naked goroutine, address empty marker interfaces.

---

## Impacted Functions

| # | Function / Symbol | File | Reason Impacted | Has Test? | Test File |
|---|-------------------|------|-----------------|-----------|-----------|
| 1 | `extractAppSchemaFields` | `service/addons_service.go:277` | `interface{}` → `any` | ✅ | `addons_service_test.go` |
| 2 | `extractSchemaFieldFromNamedItem` | `service/addons_service.go:337` | `interface{}` → `any` | ✅ | `addons_service_test.go` |
| 3 | `extractAppOptionDescriptions` | `service/addons_service.go:393` | `interface{}` → `any` | ✅ | `addons_service_test.go` |
| 4 | `getLastUnmountedState` | `service/filesystem/ntfs_adapter.go:346,348` | `interface{}` → `any` | ✅ | `base_adapter_test.go` |
| 5 | `setLastUnmountedState` | `service/filesystem/ntfs_adapter.go:365,367` | `interface{}` → `any` | ✅ | `base_adapter_test.go` |
| 6 | `AddonConfigWatcherServiceInterface` | `service/addon_config_watcher_service.go:29` | Empty `interface{}` marker | ✅ | `addon_config_watcher_service_test.go` |
| 7 | `ProblemHABridgeInterface` | `service/problem_ha_bridge.go:16` | Empty `interface{}` marker | ✅ | `broadcaster_service_test.go` |
| 8 | `GetSambaUserInfo` | `unixsamba/unixsamba.go:363` | `errors.As` → `errors.AsType[T]` | ❌ | — |
| 9 | `CreateSambaUser` | `unixsamba/unixsamba.go:417` | `errors.As` → `errors.AsType[T]` | ❌ | — |
| 10 | `DeleteSambaUser` | `unixsamba/unixsamba.go:460` | `errors.As` → `errors.AsType[T]` | ❌ | — |
| 11 | `RenameUsername` | `unixsamba/unixsamba.go:538` | `errors.As` → `errors.AsType[T]` | ❌ | — |
| 12 | `CheckSambaUser` | `unixsamba/unixsamba.go:604` | `errors.As` → `errors.AsType[T]` | ❌ | — |
| 13 | `Format` (ext4) | `service/filesystem/ext4_adapter.go` | `wg.Add/Done` → `wg.Go` | ✅ | `ext4_adapter_test.go` |
| 14 | `Check` (ext4) | `service/filesystem/ext4_adapter.go` | `wg.Add/Done` → `wg.Go` | ✅ | `ext4_adapter_test.go` |
| 15 | `Format`/`Check` (btrfs, xfs, f2fs, exfat, gfs2, hfsplus, ntfs, vfat, reiserfs, zfs) | `service/filesystem/*_adapter.go` | `wg.Add/Done` → `wg.Go` | ✅ | per-adapter `*_test.go` |
| 16 | `drainCommandOutput` (new) | `service/filesystem/base_adapter.go` | New helper — no prior test | ❌ | `base_adapter_test.go` (to add) |
| 17 | `ResolveLinuxFsModule` | `service/filesystem_service.go:334` | context-less slog (`s.ctx` available) | ✅ | filesystem service tests |
| 18 | `MountFlagsToSyscallFlagAndData` | `service/filesystem_service.go:377-400` | context-less slog | ✅ | filesystem service tests |
| 19 | `FsTypeFromDevice` | `service/filesystem_service.go:490,494` | context-less slog | ✅ | filesystem service tests |
| 20 | `BroadcastMessage` | `service/broadcaster_service.go:202` | Naked `go` goroutine (FIXME) | ✅ | `broadcaster_service_test.go` |

---

## Pre-Refactor Test Baseline

*To be recorded after running `cd backend/src && go test ./service/... ./unixsamba/...` locally.*

| Test Name | File | Status Before | Notes |
|-----------|------|---------------|-------|
| All `./service/...` tests | — | 🔄 Pending run | — |
| All `./unixsamba/...` tests | — | 🔄 Pending run | — |

---

## Post-Refactor Test Results

| Test Name | File | Status Before | Status After | Result | Notes |
|-----------|------|---------------|--------------|--------|-------|

---

## Decisions & Notes

- **2026-04-21**: User chose to run a prepare check. Branch `refactor/backend-code-quality-antipatterns` created.
- `ResolveInterfaceIPv4s` (server_process_service.go:67,72,100): standalone function — no ctx available, slog calls left unchanged.
- Task 2 (`errors.As` → `errors.AsType`): `unixsamba` functions have no dedicated test file; tests are system-level integration tests. Changes are purely mechanical type-safe refactors with identical semantics.
- Task 6 (FIXME broadcaster): `sendToHomeAssistant` is a short-lived best-effort call that must NOT block the caller. Wrapping it in a goroutine with the service's WaitGroup is the right approach; adding event bus fan-out is a separate task (013). Decision: track in WaitGroup to avoid orphaned goroutine.
- Task 7 (empty interfaces): Keep `AddonConfigWatcherServiceInterface` and `ProblemHABridgeInterface` as `interface{}` marker types but convert to `= any` type aliases with explanatory doc comments; the FX container uses the concrete type via `NewXxx` constructors so the interface name serves only as a nominal marker.

---

## Checklist

- [x] Tracking document created
- [x] Impacted functions identified (direct)
- [x] Impacted functions identified (indirect callers/dependants)
- [ ] All impacted functions have at least one test
- [ ] Missing tests created (drainCommandOutput helper)
- [ ] Pre-refactor baseline run and recorded
- [ ] Refactor implemented
- [ ] Post-refactor tests run
- [ ] All tests pass (or failures accepted by user)
- [ ] Tracking document finalised
