<!-- DOCTOC SKIP -->

# [FIX]: CRITICAL — Migration 14 Wrong Function Registration

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** _None — discovered in reliability review 2026-04-28_

## 🎯 Objective

Fix a **critical data-correctness bug** where migration `00014_sanitize_empty_HASmbPassword.go` registers the wrong callback functions in its `init()`. The migration's `init()` calls `goose.AddMigrationNoTxContext(Up00008, Down00008)` instead of `goose.AddMigrationNoTxContext(Up00014, Down00014)`. As a result:

- When goose runs migration 14, it executes migration 8's `INSERT OR IGNORE` logic a second time (no-op on existing rows, but wrong intent).
- Migration 14's actual `UPDATE` logic — which sanitises empty `HASmbPassword` values by replacing them with a freshly generated secure password — **is never executed**.
- Any database that has an empty `HASmbPassword` (created before migration 14 was introduced, or after a failed `GenerateSecurePassword` call) remains broken indefinitely.

This is a **one-line fix** that must be prioritised before any other migration or password-related work.

## 🛠️ Technical Specifications

- **Inputs:** `backend/src/dbom/migrations/00014_sanitize_empty_HASmbPassword.go`
- **Outputs:** Migration 14 correctly runs `Up00014`/`Down00014`
- **Risk:** Existing databases that have already run migration 14 (as recorded in the `goose_db_version` table) will **not** re-run it automatically. A compensating migration 15 (or a startup-time check) may be needed to repair affected databases.

## 📝 Task List

- [ ] Task 1: Change `goose.AddMigrationNoTxContext(Up00008, Down00008)` to `goose.AddMigrationNoTxContext(Up00014, Down00014)` in the `init()` function
- [ ] Task 2: Assess whether existing deployed databases have already executed the broken migration 14 (check the `goose_db_version` table). If yes:
  - [ ] Task 2a: Create migration `00015_repair_empty_ha_password.go` that runs the same `UPDATE` logic as `Up00014`, guarded by `WHERE HASmbPassword = '' OR HASmbPassword IS NULL`
  - [ ] Task 2b: Set `Up00015` and `Down00015` correctly in its `init()`
- [ ] Task 3: Add a unit test that verifies the `init()` in every migration file registers the correct function pair (function name should contain the migration number)
- [ ] Task 4: Add a test that running the full migration suite from a clean database results in no empty `HASmbPassword` values
- [ ] Task 5: Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark B-REL-01 resolved

## 🧠 Implementation Notes

```go
// 00014_sanitize_empty_HASmbPassword.go — THE FIX
func init() {
    // BUG WAS: goose.AddMigrationNoTxContext(Up00008, Down00008)
    goose.AddMigrationNoTxContext(Up00014, Down00014) // CORRECT
}
```

To detect which databases need the compensating migration: query `goose_db_version` for version 14. If the row exists with `is_applied = true`, the broken version was run. The `Up00015` logic should be safe to run idempotently even on databases where `Up00014` ran correctly (use `WHERE HASmbPassword = ''` to avoid touching valid rows).

**Naming convention check:** All other migrations follow the pattern of function names containing the migration number (e.g., `Up00004`, `Down00008`). A simple `go test` that asserts `strings.Contains(fnName, fmt.Sprintf("%05d", migrationNumber))` would catch this class of bug in the future.

## 🔗 Code References & TODOs

- [ ] `FIXME: backend/src/dbom/migrations/00014_sanitize_empty_HASmbPassword.go:15` — wrong function registration
- [ ] Related: B-REL-01 (Critical) in `docs/SECURITY_OPTIMIZATION_REVIEW.md`
- [ ] Related: B-SEC-07 (hardcoded fallback password in same migration)
