<!-- DOCTOC SKIP -->

# [FEATURE]: Improve Overlay Helper System & Tour Events

**Target Repo:** `srat` **Status:** ✅ Done **Issue Link:** [#515](https://github.com/dianlight/srat/issues/515)

## 🎯 Objective

Refactor and improve the `TourEvents` overlay helper system to accurately reflect the current state of UI pages after recent changes, implement automated testing, and establish clear frontend guidelines for maintaining the system across all pages (Dashboard, Shares, Volumes, Settings, Users).

## 🛠️ Technical Specifications

- **Inputs:**
  - Current `frontend/src/utils/TourEvents.ts` (event emitter system)
  - Tour step components across pages: `UsersSteps.tsx`, `SharesTourStep.tsx`, dashboard, volumes, settings tour steps
  - Frontend instruction files (`.github/instructions/reactjs.instructions.md`)

- **Outputs:**
  - Improved `TourEvents.ts` with type-safe event handlers and proper cleanup semantics
  - Updated tour step components reflecting actual current page state/behavior
  - Automated test suite for TourEvents system (mock, emit, listen, cleanup)
  - New frontend maintenance guidelines in `reactjs.instructions.md` (event listener patterns, cleanup requirements, testing)

- **Dependencies:**
  - React hooks, event emitters (Emittery library)
  - All page components using TourEvents: `Users.tsx`, `SharesTourStep.tsx`, dashboard, volumes, settings
  - Frontend test infrastructure (bun:test, React Testing Library)

## 📝 Task List

- [x] Task 1: Audit all tour step components and page state to identify outdated tour steps
- [x] Task 2: Refactor `TourEvents.ts` to improve type safety, error handling, and listener lifecycle
- [x] Task 3: Create automated test suite for TourEvents (emit, listen, cleanup patterns)
- [x] Task 4: Update tour step definitions in all page components to match current UI/behavior
- [x] Task 5: Add frontend maintenance guidelines to `reactjs.instructions.md`
- [x] Task 6: Add JSDoc and inline documentation to TourEvents and related components
- [x] Task 7: Run full frontend test suite and verify tour functionality on test pages
- [x] Task 8: Integration testing and documentation

## 🧠 Implementation Notes (Copilot Context)

### Current State
- `TourEvents.ts` uses Emittery for event emission with these methods: `on()`, `off()`, `emit()`
- `TourEventTypes` enum defines step events for Dashboard (8 steps), Shares (2 steps), Volumes (3 steps), Settings (3 steps), Users (1 step)
- Tour steps are triggered manually in component render/effects by calling `TourEvents.emit()`
- Tour steps are consumed by page containers using `TourEvents.on()` to position overlay tooltips

### Problem Areas
1. **State Mismatch:** Tour steps were designed before recent UI changes; many steps no longer reflect current page layout/behavior
2. **Type Unsafety:** Events use `unknown[]` for parameters, no type safety for event payloads
3. **Listener Cleanup:** No systematic cleanup of event listeners (potential memory leaks in long-lived pages)
4. **Error Handling:** No error callbacks or error event propagation
5. **Testing:** Manual testing only; no automated way to verify tour steps work correctly
6. **No Guidelines:** Frontend developers lack clear rules for maintaining TourEvents across changes

### Solutions to Implement
1. **Audit & Fix:** Scan all page components, verify tour step elements exist and match current DOM
2. **Type Safety:** Create typed event payloads (e.g., `TourStepPayload<T>` with element ref, metadata)
3. **Listener Lifecycle:** Implement proper cleanup hooks (useEffect return cleanup, off-on-unmount)
4. **Testing:** Create test utilities to mock, spy, and verify event emission/listening
5. **Guidelines:** Add to `reactjs.instructions.md`:
   - Always use `useEffect` with cleanup for `TourEvents.on()`
   - Define strict TypeScript types for event payloads
   - Add JSDoc comments with examples to torn step emitters
   - Verify DOM elements exist before emitting tour events
   - Add unit tests for tour step emission

## 🔗 Code References & TODOs

- [x] `frontend/src/utils/TourEvents.ts` — Main system file updated with typed events, lifecycle helpers, and JSDoc
- [x] `frontend/src/pages/users/Users.tsx` — Tour listener moved into `useEffect` with cleanup
- [x] `frontend/src/pages/users/UsersSteps.tsx` — Tour step content updated to match current behavior
- [ ] `frontend/src/pages/shares/SharesTourStep.tsx` (line 60) — Another emitter example
- [ ] `frontend/src/pages/shares/__tests__/SharesTourStep.test.tsx` (line 7) — Manual mocking in tests
- [x] `.github/instructions/reactjs.instructions.md` — Added TourEvents maintenance guidelines section
- [x] `frontend/src/pages/settings/Settings.tsx` — Added concrete `data-tutor` anchors for steps 0/2/3/4/5/6/7/8/9 and aligned step-8 action with `network_devices`
- [x] `frontend/src/pages/volumes/Volumes.tsx` — Added missing anchors for steps 0/4/5
- [x] `frontend/src/pages/users/Users.tsx` — Moved tour listener into `useEffect` with cleanup
- [x] `frontend/src/pages/dashboard/Dashboard.tsx` — Moved tour listener into `useEffect` with cleanup
- [x] `frontend/src/pages/dashboard/DashboardActions.tsx` — Moved tour listener into `useEffect` with cleanup
- [x] `frontend/src/pages/dashboard/metrics/MetricDetails.tsx` — Moved all tour listeners into `useEffect` with cleanup
- [x] `frontend/src/utils/__tests__/TourEvents.test.ts` — Added dedicated event-system tests

### Known Issues
- SHARES_STEP_2 is commented out (line 45 in SharesTourStep.tsx) — determine if should be removed or reactivated
- Manual test mocking pattern in SharesTourStep.test.tsx should be formalized into a test utility

### Phase Updates
- 2026-03-20: Completed Task 2 by refactoring `frontend/src/utils/TourEvents.ts` with a typed event payload map, explicit `on/off/once` lifecycle helpers (including unsubscribe return), protected `emit` error handling, and listener cleanup helper (`clearListeners`).
- 2026-03-20: Targeted validation passed with `cd frontend && bun tsgo --noEmit` (exit code 0).
- 2026-03-20: Completed Task 3 by adding `frontend/src/utils/__tests__/TourEvents.test.ts` covering emit/listen/off/once/clear/error behavior; targeted test run passed (`bun test --preload ./test/setup.ts src/utils/__tests__/TourEvents.test.ts`).
- 2026-03-20: Completed Task 4 by removing outdated users tour steps, fixing missing settings/volumes tour anchors, and aligning guided-tour actions with current page structure.
- 2026-03-20: Completed Task 5 by adding SRAT-specific guided-tour maintenance rules to `.github/instructions/reactjs.instructions.md`.
- 2026-03-20: Completed Task 6 by adding JSDoc to public `TourEvents` contracts and API methods.
- 2026-03-20: Validation after Task 4/5/6 passed (`bun test --preload ./test/setup.ts` for focused tour tests + `bun tsgo --noEmit`, all green).
- 2026-03-20: Completed Task 7 by running the full frontend suite (`cd frontend && bun test --preload ./test/setup.ts --bail --timeout 10000 --max-concurrency=1 --reporter=dot`) with 607 pass / 1 skip / 0 fail.
- 2026-03-20: Completed Task 8 by finalizing integration-level validation notes and documentation updates.
- 2026-03-20: Documentation validation also passed from repo root with `make docs-validate`.
