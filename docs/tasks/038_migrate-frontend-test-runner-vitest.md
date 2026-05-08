<!-- DOCTOC SKIP -->

# [REFACTOR]: Migrate Frontend Test Runner from bun:test to Vitest

**Target Repo:** `srat`
**Status:** ✅ Complete
**Issue Link:** https://github.com/dianlight/srat/issues/614

## ⚠️ Compliance & Pre-reqs

- This task is a `[REFACTOR]`. Per `.github/copilot-instructions.md` you MUST ask whether to run the `prepare-refactor` check before editing source files. If the user opts in, create `docs/refactors/migrate-frontend-test-runner-vitest.md` and follow the prepare-refactor phases (record baseline, run post-refactor verification).
- The **short-term fixes** listed below must be applied (or verified already applied) before starting:
  - `wsApi.test.tsx` `waitForCondition` default timeout raised to 2000 ms
  - `handlers.ts` exports `resetApiCounters()`, called in `bun-setup.ts` `afterEach`
  - `createTestStore()` silent `catch {}` replaced with `console.warn`
  - `bunfig.toml` `bail = 0`
- Read the header of any file you modify before changing it.
- Do NOT run `git add/commit/push` unless explicitly asked.
- After migration, update `.github/instructions/fontend_test.instructions.md` and `docs/test-setup-patterns.md` to reference Vitest instead of `bun:test`.
- Do NOT edit generated files: `frontend/src/store/sratApi.ts`, `backend/docs/openapi.*`.
- Branch naming: `refactor/migrate-frontend-test-runner-vitest`.

## 🎯 Objective

Replace `bun test` with Vitest as the frontend test runner while keeping Bun as the runtime for all other tasks (build, dev, package management).

**Root problem being solved:** Bun's module registry is process-wide. `mock.module()` patches it globally; `mock.restore()` cannot perfectly reverse RTK Query singleton state. This forces `--max-concurrency=1` (serial execution of all 75 test files), causes `Shares.test.tsx` to be permanently skipped, and makes any test file using `mock.module()` a source of ordering-dependent failures for the tests that run after it.

**Why Vitest fixes this architecturally:** Vitest runs each test file in its own `vm.Worker` subprocess. `vi.mock()` is scoped to that worker and automatically torn down when the file finishes. No global module registry is shared across files. Parallel file execution is enabled by default.

**Expected outcome:** ~3–4× faster wall-clock test time (serial → parallel), `Shares.test.tsx` restored to active tests, `mock.module()` contamination eliminated by design, `--max-concurrency=1` removed.

## 🛠️ Technical Specifications

- **Inputs:**
  - `frontend/bunfig.toml` (test runner config)
  - `frontend/package.json` (scripts)
  - `frontend/.mise.toml` (task runner)
  - `frontend/test/setup.ts`, `frontend/test/bun-setup.ts`
  - All 75 `*.test.{ts,tsx}` files under `frontend/src/` and `frontend/test/`
  - `.github/instructions/fontend_test.instructions.md`
  - `docs/test-setup-patterns.md`
- **Outputs:** Same tests passing under `bunx vitest run`, parallel execution, `Shares.test.tsx` unskipped
- **Dependencies:** `vitest`, `@vitest/coverage-v8` (new dev deps); Bun runtime unchanged
- **Acceptance criteria:**
  - `bunx vitest run` exits 0 with all previously-passing tests still passing
  - `Shares.test.tsx` unskipped and passing
  - No `"bun:test"` imports remain in `src/` or `test/`
  - `--max-concurrency` flag is gone from all scripts
  - `mise run //frontend:test` uses Vitest
  - Coverage report generates under `frontend/coverage/`
  - `mise run //frontend:lint` passes with zero new errors

## 📝 Task List

Preflight:
- [x] PREP: Ask user whether to run a prepare-refactor check (recommended). If yes, create `docs/refactors/migrate-frontend-test-runner-vitest.md`, record baseline test count and timing before any changes.

Installation and config:
- [x] Task 1: Install Vitest — `cd frontend && bun add -d vitest @vitest/coverage-v8`
- [x] Task 2: Create `frontend/vitest.config.ts` (see Implementation Notes §A)
- [x] Task 3: Update `frontend/package.json` scripts — replace `bun test ...` with `bunx vitest run ...`, remove `--max-concurrency=1` and `--preload` flags (see Implementation Notes §B)
- [x] Task 4: Update `frontend/.mise.toml` tasks `test`, `test:new`, `test:ci` to invoke Vitest (see Implementation Notes §B)

Setup file adaptation:
- [x] Task 5: In `frontend/test/bun-setup.ts`, replace `import { afterAll, afterEach, beforeAll } from "bun:test"` with `from "vitest"`
- [x] Task 6: Verify `frontend/test/setup.ts` has no `"bun:test"` imports (it currently imports from `"./bun-setup"` only — confirm no direct runner imports)
- [x] Task 7: Confirm `IS_REACT_ACT_ENVIRONMENT` is set after `GlobalRegistrator.register()` in `setup.ts` — Vitest with `environment: "happy-dom"` installs `window`/`document` before `setupFiles` runs, so the guard in `setup.ts` short-circuits safely; no change needed

Import migration (mechanical find-replace):
- [x] Task 8: In all test files under `frontend/src/` and `frontend/test/`, replace `from "bun:test"` with `from "vitest"` — run: `find frontend/src frontend/test -name "*.test.*" | xargs sed -i 's|from "bun:test"|from "vitest"|g'` — verify with `grep -r '"bun:test"' frontend/src frontend/test` returning nothing

Mock API migration (9 files using `mock.*`):
- [x] Task 9: `src/__tests__/App.commandEvents.test.tsx` — replace `mock` import + usages with `vi` (see Implementation Notes §C for the mapping table); `vi.mock()` calls at file top level are hoisted automatically
- [x] Task 10: `src/components/__tests__/DonationButton.test.tsx` — same `mock` → `vi` migration
- [x] Task 11: `src/components/__tests__/NavBar.test.tsx` — same
- [x] Task 12: `src/components/__tests__/NotificationCenter.test.tsx` — same
- [x] Task 13: `src/pages/shares/__tests__/ShareEditDialog.test.tsx` — same
- [x] Task 14: `src/pages/shares/__tests__/Shares.test.tsx` — same **and remove `describe.skip`**; with Vitest worker isolation the contamination is gone
- [x] Task 15: `src/pages/volumes/components/__tests__/VolumeMountDialog.test.tsx` — same
- [x] Task 16: `src/pages/volumes/__tests__/Volumes.test.tsx` — same
- [x] Task 17: `src/pages/__tests__/SmbConf.test.tsx` — same

`bunfig.toml` cleanup:
- [x] Task 18: Remove or comment out the `[test]` section in `frontend/bunfig.toml` — it is read only by `bun test` and is dead config after migration; the Vitest equivalents live in `vitest.config.ts`

Documentation updates:
- [x] Task 19: Update `.github/instructions/fontend_test.instructions.md` — change all references from `bun:test` to `vitest`; update the Code Standard example in §6; update §1 to list Vitest as the runner
- [x] Task 20: Update `docs/test-setup-patterns.md` — replace `bun:test` runner references with Vitest; update any lifecycle hook examples that import from `"bun:test"`

Verification:
- [x] Task 21: Run `bunx vitest run --reporter=verbose 2>&1 | tail -40` — confirm all files pass, `Shares.test.tsx` is no longer skipped
- [x] Task 22: Run `bunx vitest run --coverage` — confirm `frontend/coverage/lcov.info` is generated
- [x] Task 23: Run `mise run //frontend:lint` — confirm zero new errors
- [x] Task 24: Run `tsgo --noEmit` — confirm no TypeScript errors introduced
- [x] Task 25: If prepare-refactor was run, complete the post-refactor verification phase in `docs/refactors/migrate-frontend-test-runner-vitest.md`

## 🧠 Implementation Notes

### A — `vitest.config.ts`

```typescript
import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "happy-dom",
    globals: true,
    setupFiles: ["./test/setup.ts"],
    include: [
      "src/**/*.test.{ts,tsx}",
      "test/__tests__/**/*.test.{ts,tsx}",
    ],
    exclude: ["**/node_modules/**", "**/dist/**", "**/build/**"],
    // "forks" = one subprocess per file → isolated module registry per file
    pool: "forks",
    testTimeout: 60000,
    hookTimeout: 60000,
    bail: 0,
    reporter: ["dot"],
    coverage: {
      provider: "v8",
      reporter: ["text", "lcov"],
      exclude: [
        "**/node_modules/**",
        "**/dist/**",
        "**/build/**",
        "**/coverage/**",
        "**/src/mocks/**",
        "**/macro/**",
        "**/.devcontainer/**",
        "src/store/sratApi.ts",
        "test/**",
      ],
    },
  },
});
```

Key choices:
- `pool: "forks"` — each file in a subprocess. Module mocks are subprocess-scoped; no shared registry.
- `environment: "happy-dom"` — Vitest installs happy-dom per worker. The `GlobalRegistrator.register()` guard in `setup.ts` short-circuits (`if (!window || !document)`) because Vitest pre-installs the DOM.
- `globals: true` — required so `@testing-library/jest-dom` can extend the global `expect` during setup-file execution.
- `setupFiles: ["./test/setup.ts"]` — replaces `--preload ./test/setup.ts`. Runs once per worker before the first test file in that worker.
- `bail: 0` — show all failures; CI adds `--bail=1` via the script.

### B — Script and task updates

**`package.json` scripts** (replace the `_test*` entries):
```json
"_test":        "tsgo --noEmit && AGENT=1 bunx vitest run --reporter=dot",
"_test:ci":     "bun run test:prepare && bunx vitest run --coverage --bail=1",
"_test:new":    "tsgo --noEmit && AGENT=1 bunx vitest run --reporter=dot --changed",
"_test:watch":  "bunx vitest"
```

Remove: `BUN_TEST_NO_COVERAGE_THRESHOLD=1`, `--preload ./test/setup.ts`, `--max-concurrency=1`, `--timeout 10000` (now in `vitest.config.ts`).

**`.mise.toml` task `test`** (replace the `run` array):
```toml
[tasks.test]
run = [
  "bun ./scripts/add-test-setup.js",
  "tsgo --noEmit",
  "AGENT=1 bunx vitest run --reporter=dot",
]
quiet = true

[tasks."test:new"]
run = [
  "bun ./scripts/add-test-setup.js",
  "tsgo --noEmit",
  "AGENT=1 bunx vitest run --reporter=dot --changed",
]
quiet = true

[tasks."test:ci"]
run = [
  "bun ./scripts/add-test-setup.js",
  "bunx vitest run --coverage --bail=1",
]
quiet = true
```

### C — `mock.*` → `vi.*` API mapping

| `bun:test` | `vitest` |
|---|---|
| `import { mock } from "bun:test"` | `import { vi } from "vitest"` |
| `mock(fn)` | `vi.fn(fn)` |
| `mock.fn()` | `vi.fn()` |
| `mock.module(id, factory)` | `vi.mock(id, factory)` |
| `mock.restore()` | `vi.restoreAllMocks()` |
| `mockFn.mockClear()` | `mockFn.mockClear()` — identical |
| `mockFn.mockImplementation(fn)` | `mockFn.mockImplementation(fn)` — identical |
| `mockFn.mock.calls` | `mockFn.mock.calls` — identical |

**Important — `vi.mock()` hoisting:** Vitest statically hoists `vi.mock()` calls to the top of the file (like Jest). This means:
- `vi.mock()` calls do not need to be inside `beforeEach`. Move any top-level `mock.module()` calls that were inside `beforeEach`/`setupMocks()` to file top level if they are unconditional.
- If the mock factory needs variables from the test scope (like `wsState`), use `vi.fn()` inside the factory and update the implementation in `beforeEach` — the factory closure captures the function reference, not the variable value.

Example migration for `App.commandEvents.test.tsx`:
```typescript
// BEFORE (bun:test)
import { mock } from "bun:test";
const registerModuleMocks = () => {
  mock.module("../store/wsApi", () => ({ ... }));
};
beforeEach(() => { mock.restore(); registerModuleMocks(); });
afterEach(() => { mock.restore(); });

// AFTER (vitest)
import { vi } from "vitest";
vi.mock("../store/wsApi", () => ({ ... }));
beforeEach(() => { vi.clearAllMocks(); });
afterEach(() => { vi.restoreAllMocks(); });
```

### D — `Shares.test.tsx` unskip

Remove the `describe.skip(...)` wrapper and the comment above it. The test body is otherwise correct. With Vitest workers, `vi.mock()` in this file has no effect on any other file's module registry.

The `path.resolve(__dirname, ...)` absolute-path mock calls at the bottom of `setupMocks()` are a workaround for Bun's module resolution — verify whether Vitest still needs them or if the relative-path mocks are sufficient. Remove the absolute-path duplicates if they are redundant.

### E — `scripts/add-test-setup.js` preamble check

The script enforces that every test file starts with `import '../../../../test/setup'`. With Vitest's `setupFiles`, this import is redundant (setup runs globally per worker). Options:

1. **Keep as-is (safest):** The import is a no-op since the module is cached after the first load. No files need to change immediately.
2. **Update the script to warn only:** Change `process.exit(2)` to a console warning so it no longer blocks the test run while the redundant imports are cleaned up over time as a separate PR.

Recommendation: update to warn-only in this PR, plan a follow-up to strip the per-file imports.

### F — Potential issues to watch for

- **`@testing-library/jest-dom` matchers:** These are imported in `test/setup.ts` via `import '@testing-library/jest-dom'`. Vitest supports this natively when using `globals: false`. Verify matchers work in the first smoke run; if not, add `import '@testing-library/jest-dom/vitest'` instead.
- **`happy-dom` version:** `@happy-dom/global-registrator` (`20.9.0`) is installed. Vitest's `environment: "happy-dom"` uses its own bundled copy of happy-dom. The two may diverge. If the `GlobalRegistrator.register()` guard in `setup.ts` short-circuits correctly (window already exists), both copies are irrelevant — only Vitest's environment copy is active. Run the smoke test and watch for DOM-related failures before removing the explicit registration.
- **`IS_REACT_ACT_ENVIRONMENT`:** Must still be set after `GlobalRegistrator.register()` (or after Vitest's environment setup). The existing code in `setup.ts` already handles this correctly.
- **`bun:test` snapshot format:** If any test uses `.toMatchSnapshot()`, the snapshot files use Bun's format. Vitest uses a different format. Delete `.snap` files and regenerate with `bunx vitest run -u` after migration. Check for any `.snap` files with `find frontend -name "*.snap"`.

## 🔗 Code References & TODOs

- [ ] `TODO: frontend/test/bun-setup.ts:13` — change `from "bun:test"` to `from "vitest"`
- [ ] `TODO: frontend/src/pages/shares/__tests__/Shares.test.tsx:8` — remove `describe.skip`
- [ ] `TODO: frontend/bunfig.toml:[test]` — remove/comment the `[test]` section after migration
- [ ] `TODO: .github/instructions/fontend_test.instructions.md` — update runner references from bun:test to vitest throughout
- [ ] `TODO: docs/test-setup-patterns.md` — update runner references
- [ ] Files with `mock.*` needing `vi.*` migration (9 total): `App.commandEvents.test.tsx`, `DonationButton.test.tsx`, `NavBar.test.tsx`, `NotificationCenter.test.tsx`, `ShareEditDialog.test.tsx`, `Shares.test.tsx`, `VolumeMountDialog.test.tsx`, `Volumes.test.tsx`, `SmbConf.test.tsx`
