<!-- DOCTOC SKIP -->

# [REFACTOR]: Frontend Code Quality — Anti-patterns & Instruction Violations

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:**

## 🎯 Objective

Fix a set of recurring code-quality violations in the frontend codebase: deprecated testing practices (`fireEvent`, missing `await` on user-event calls, overuse of `getByTestId`), a leftover deprecated TypeScript 5.x HMR cast, raw `console.*` calls in production code paths, and a raw `fetch()` call bypassing the RTK Query layer in a custom hook.

## 🛠️ Technical Specifications

- **Inputs:** Existing frontend source files in `frontend/src/`
- **Outputs:** Cleaner, instruction-compliant code with no regressions in existing tests
- **Dependencies:** `@testing-library/user-event`, `sratApi` RTK Query, TypeScript 6.0+ native `import.meta.hot`

## 📝 Task List

- [ ] Task 1: Fix `fireEvent` usage in `FilesystemLabelFormatDialog.test.tsx` — replace with `userEvent` (via `userEvent.setup()` + `await`)
- [ ] Task 2: Fix non-awaited `userEvent.click/type/hover` calls in `NavBar.test.tsx` — add `await` and ensure `userEvent.setup()` result is used consistently
- [ ] Task 3: Remove `getByTestId` overuse in `DonationButton.test.tsx` — replace with semantic queries (`getByRole`, `getByLabelText`) wherever possible
- [ ] Task 4: Fix deprecated HMR pattern in `DiskHealthMetrics.tsx` — replace `if (import.meta && (import.meta as any).hot)` with native `if (import.meta.hot)` (TS 6.0+)
- [ ] Task 5: Migrate raw `fetch()` in `githubNewsHook.ts` to use the existing `githubApi` RTK Query slice instead of calling `fetch` directly
- [ ] Task 6: Audit and clean up `console.log` / `console.error` / `console.warn` calls in production source files (non-test, non-MSW mock files) — remove debug logs or replace with structured error handling
- [ ] Task 7: Unit testing — verify all modified tests still pass with `mise run //frontend:test --rerun-each 10` for any touched test file
- [ ] Task 8: Integration — run full frontend test suite (`mise run //frontend:test`) and lint (`mise run //frontend:lint`) with zero new errors
- [ ] Task 9: Capture lessons learned and update documentation
- [ ] Task 10: Ask to create a PR with the task implementation and link it here for tracking

## 🧠 Implementation Notes (Copilot Context)

### Task 1 — `fireEvent` → `userEvent`

File: `frontend/src/pages/volumes/components/__tests__/FilesystemLabelFormatDialog.test.tsx`

Lines ~804 and ~928 import `fireEvent` from `@testing-library/react` and call `fireEvent.mouseDown(fsTypeDropdown)`. Per project instructions, **only `@testing-library/user-event` is allowed** for interactions.

Replace with:
```tsx
const user = userEvent.setup();
// ...
await user.click(fsTypeDropdown);
// or pointer-down sequence if needed
```

### Task 2 — Non-awaited `userEvent` calls in NavBar tests

File: `frontend/src/components/__tests__/NavBar.test.tsx`

Multiple lines call `userEvent.click(element)` without `await` (lines 174, 264, 359, 599, 609, 656, 792, 841, 842). Only `userEvent.setup()` is called once at line 96, but the returned `user` object is not consistently used for subsequent calls. All interaction calls **must be awaited**:

```tsx
const user = userEvent.setup();
// ...
await user.click(element); // NOT userEvent.click(element)
```

### Task 3 — `getByTestId` in DonationButton tests

File: `frontend/src/components/__tests__/DonationButton.test.tsx`

`getByTestId("donation-button")` is used on lines 88, 112, 139, 180, 214, 246, 277, 304. Per instructions, `getByTestId` is a last resort. Prefer:
- `getByRole('button', { name: /donate/i })` — if the button has an accessible name
- `getByLabelText(...)` — if it has a label
Check `DonationButton.tsx` component to see if an `aria-label` or visible text label can be used instead of `data-testid`.

### Task 4 — Deprecated HMR cast in DiskHealthMetrics

File: `frontend/src/pages/dashboard/metrics/DiskHealthMetrics.tsx`, lines 431–432

Current (deprecated, TS 5.x):
```ts
if (import.meta && (import.meta as any).hot) {
  (import.meta as any).hot.accept(() => { ... });
}
```

Correct (TS 6.0+ native):
```ts
if (import.meta.hot) {
  import.meta.hot.accept(() => { ... });
}
```

### Task 5 — Raw `fetch()` in `githubNewsHook.ts`

File: `frontend/src/hooks/githubNewsHook.ts`, line 33

The hook calls `fetch(GITHUB_API_URL)` directly. The `githubApi` RTK Query slice (`frontend/src/store/githubApi.ts`) already handles the GitHub Discussions endpoint with proper caching and error handling. Migrate the hook to consume the `githubApi` endpoint, or at minimum wrap the raw fetch in the `githubApi` baseQuery so it benefits from RTK Query lifecycle management.

### Task 6 — Console.* cleanup in production code

Key files with stale/debug console calls:
- `App.tsx:125` — `console.log("Error auto-reset timer triggered")`
- `App.tsx:178` — `console.error("Addon restart failed", error)` (should surface error to UI, not just log)
- `NavBar.tsx:421,435` — `console.log("Doing update")`, `console.error("DoUpdate error:", err)`
- `components/BaseConfigModal.tsx:82,142`
- `hooks/useSmartOperations.ts:90,109`
- `hooks/smartTestStatusHook.ts:52`
- `components/TelemetryModal.tsx:64`
- `components/ReportIssueDialog.tsx:113,124,151`

Remove pure debug `console.log` calls. For error paths, propagate errors to the existing notification / error boundary pattern rather than logging silently.

## 🔗 Code References & TODOs

- [ ] `frontend/src/pages/volumes/components/__tests__/FilesystemLabelFormatDialog.test.tsx:804,928` — `fireEvent` usage
- [ ] `frontend/src/components/__tests__/NavBar.test.tsx:174,264,359,599,609,656,792,841,842` — non-awaited `userEvent`
- [ ] `frontend/src/components/__tests__/DonationButton.test.tsx:88,112,139,180,214,246,277,304` — `getByTestId` overuse
- [ ] `frontend/src/pages/dashboard/metrics/DiskHealthMetrics.tsx:431-432` — deprecated HMR `as any` cast
- [ ] `frontend/src/hooks/githubNewsHook.ts:33` — raw `fetch()` call
- [ ] `frontend/src/App.tsx:125,178` — debug/bare `console.*` in production paths
- [ ] `frontend/src/components/NavBar.tsx:421,435` — debug `console.*`
