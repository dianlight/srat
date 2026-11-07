# Design

## Context

Fix `detectBestServerVariant` in `backend/src/service/upgrade_service.go` so a develop-channel update containing only a subset of binaries (e.g. `srat-cli` alone) does not re-point the `srat-server` symlink at a stale `srat-server-static` from a previous installation.

## Goals / Non-Goals

**Goals:**

- Deliver: Upgrade symlink falls back to stale static on partial update packages

**Non-Goals:**

- Unrelated refactoring outside the task scope.

## Decisions

- Reproduce — unit test where `updatePkg` contains only `srat-cli`, target dir has newer musl + older static on a musl-linker system; assert current code returns `srat-server-static` (bug)
- Fix `detectBestServerVariantWithLinkerCheck` to fall back to the newest on-disk variant (compare embedded versions) instead of unconditionally returning static when the package has no server variants
- Run `mise run //backend:test` (full suite green) + per-function coverage ≥70% for touched logic
- Update `docs/SMART_SERVICE.md` remote-deploy notes if behavior changes user-visible flow

## Risks / Trade-offs

- Behaviour changes are covered by targeted tests before handoff.
- Legacy task content remains the reference until the change is archived.
