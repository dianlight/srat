<!-- DOCTOC SKIP -->

# [FIX]: Upgrade symlink falls back to stale static on partial update packages

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** _to be created_

## 🎯 Objective

Fix `detectBestServerVariant` in `backend/src/service/upgrade_service.go` so a develop-channel update containing only a subset of binaries (e.g. `srat-cli` alone) does not re-point the `srat-server` symlink at a stale `srat-server-static` from a previous installation.

## 🛠️ Technical Specifications

- **Inputs:** `backend/src/service/upgrade_service.go` (`detectBestServerVariantWithLinkerCheck`, `updateServerSymlink`); `backend/src/service/upgrade_service_internal_test.go`.
- **Outputs:** Symlink selection considers on-disk variant versions (not just update-package membership); regression test with a cli-only package on a system whose musl binary is newer than static.
- **Dependencies:** Go 1.27; `mise run //backend:test`.

## 📝 Task List

- [ ] Task 1: Reproduce — unit test where `updatePkg` contains only `srat-cli`, target dir has newer musl + older static on a musl-linker system; assert current code returns `srat-server-static` (bug)
- [ ] Task 2: Fix `detectBestServerVariantWithLinkerCheck` to fall back to the newest on-disk variant (compare embedded versions) instead of unconditionally returning static when the package has no server variants
- [ ] Task 3: Run `mise run //backend:test` (full suite green) + per-function coverage ≥70% for touched logic
- [ ] Task 4: Update `docs/SMART_SERVICE.md` remote-deploy notes if behavior changes user-visible flow

## 🧠 Implementation Notes (Copilot Context)

- **Observed 2026-09-11 on 192.168.0.68:** transferred `srat-server-musl` (dev.15) then `srat-cli` (dev.15) to `/config/upgrade/`. First trigger installed musl → symlink `srat-server-musl` → restart OK. Second trigger (files=`[srat-cli]`) recomputed the symlink with an empty `packagedVariants` set → fell through to `srat-server-static` (stale dev.14 on disk). Symlink fixed manually via `docker exec ... ln -sf srat-server-musl`.
- **Why no outage:** s6 runs `/usr/local/bin/srat-server-musl` directly (confirmed via `ps`), not the symlink — but the next container reboot would have booted the stale static (dev.14, pre-migration-19 schema), likely failing goose startup on a DB already at version 19.
- **Suggested fix direction:** when `packagedVariants` has no server variant, keep the existing symlink target if its binary is still present, or pick the newest on-disk variant by embedded version; only default to static when nothing else is runnable.
- **Version extraction:** `scripts/extract-binary-version.sh` shows how the build reads the embedded version; reuse the same metadata source for on-disk comparison.
