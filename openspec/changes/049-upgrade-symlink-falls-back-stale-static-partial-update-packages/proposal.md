# Proposal

## Why

Fix `detectBestServerVariant` in `backend/src/service/upgrade_service.go` so a develop-channel update containing only a subset of binaries (e.g. `srat-cli` alone) does not re-point the `srat-server` symlink at a stale `srat-server-static` from a previous installation.

## What Changes

- Reproduce — unit test where `updatePkg` contains only `srat-cli`, target dir has newer musl + older static on a musl-linker system; assert current code returns `srat-server-static` (bug)
- Fix `detectBestServerVariantWithLinkerCheck` to fall back to the newest on-disk variant (compare embedded versions) instead of unconditionally returning static when the package has no server variants
- Run `mise run //backend:test` (full suite green) + per-function coverage ≥70% for touched logic
- Update `docs/SMART_SERVICE.md` remote-deploy notes if behavior changes user-visible flow

## Capabilities

### New Capabilities

- `upgrade-symlink-falls-back-stale-static-partial-update-packages`: Upgrade symlink falls back to stale static on partial update packages

### Modified Capabilities

(none)

## Impact

- **Inputs:** `backend/src/service/upgrade_service.go` (`detectBestServerVariantWithLinkerCheck`, `updateServerSymlink`); `backend/src/service/upgrade_service_internal_test.go`.
- **Outputs:** Symlink selection considers on-disk variant versions (not just update-package membership); regression test with a cli-only package on a system whose musl binary is newer than static.
- **Dependencies:** Go 1.27; `mise run //backend:test`.
- Legacy task: `docs/tasks/049_upgrade-symlink-stale-static-fallback.md`
