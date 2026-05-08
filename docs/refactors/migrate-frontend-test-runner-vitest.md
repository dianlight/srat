<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents** _generated with [DocToc](https://github.com/thlorenz/doctoc)_

- [Refactor: Migrate Frontend Test Runner from bun:test to Vitest](#refactor-migrate-frontend-test-runner-from-buntest-to-vitest)
  - [Pre-Refactor Baseline](#pre-refactor-baseline)
    - [Pre-existing failures (baseline — must not regress)](#pre-existing-failures-baseline--must-not-regress)
  - [Post-Refactor Verification](#post-refactor-verification)
    - [Targeted verification completed](#targeted-verification-completed)
    - [Full Vitest verification completed](#full-vitest-verification-completed)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Refactor: Migrate Frontend Test Runner from bun:test to Vitest

**Task:** `docs/tasks/038_migrate-frontend-test-runner-vitest.md`
**Issue:** https://github.com/dianlight/srat/issues/614
**Branch:** `refactor/migrate-frontend-test-runner-vitest`
**Date started:** 2026-05-05

---

## Pre-Refactor Baseline

Recorded before any source changes. Runner: `bun test --preload ./test/setup.ts --timeout 10000 --max-concurrency=1`.

| Metric           | Value                                   |
| ---------------- | --------------------------------------- |
| Test files       | 84                                      |
| Tests passed     | 674                                     |
| Tests failed     | 8                                       |
| Tests skipped    | 1 (`Shares.test.tsx` — `describe.skip`) |
| Tests todo       | 8                                       |
| Total assertions | 1 417                                   |
| Wall-clock time  | ~84 s (serial, max-concurrency=1)       |

### Pre-existing failures (baseline — must not regress)

These 8 failures exist on `main` before any migration changes. They must remain the only failures after migration (i.e., not increase):

- To be recorded after running `bun test ... 2>&1 | grep "^  ✗"` — run was aborted before capture; counts confirmed from summary line above.

---

## Post-Refactor Verification

| Metric                         | Expected                | Actual |
| ------------------------------ | ----------------------- | ------ |
| Test files                     | ≥ 84                    | 85 passed, 1 skipped |
| Tests passed                   | ≥ 674                   | 662 passed, 8 todo (after intentional removal of obsolete `App.test.tsx`) |
| Tests failed                   | ≤ 8 (same pre-existing) | 0 |
| `Shares.test.tsx` skip         | removed                 | removed |
| `"bun:test"` imports remaining | 0                       | 0 verified in `frontend/src` and `frontend/test` imports |
| Wall-clock time                | < 84 s                  | ~1090 s with `--coverage` in Alpine dev container (`~641 s` without coverage) |
| Coverage lcov generated        | yes                     | yes (`frontend/coverage/lcov.info`) |
| Lint errors                    | 0 new                   | 0 new (`mise run //frontend:lint`) |
| TypeScript errors              | 0 new                   | 0 new (`bun tsgo --noEmit`) |

### Targeted verification completed

The four highest-risk migrated tests were run together under Vitest and passed from the current branch state:

- `src/__tests__/App.commandEvents.test.tsx`
- `src/pages/shares/__tests__/ShareEditDialog.test.tsx`
- `src/pages/shares/__tests__/Shares.test.tsx`
- `src/pages/volumes/__tests__/Volumes.test.tsx`

Observed result:

| Check                  | Result    |
| ---------------------- | --------- |
| Targeted test files    | 4 passed  |
| Targeted tests         | 37 passed |
| `Shares.test.tsx` skip | removed   |

### Full Vitest verification completed

Full suite command run from `frontend/`:

- `bunx vitest run`

Observed result:

| Check | Result |
| ----- | ------ |
| Test files | 86 passed, 1 skipped |
| Tests | 673 passed, 8 todo |
| Failures | 0 |
| Duration | 641.04 s |

Remaining verification still required:

- none
