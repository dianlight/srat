<!-- DOCTOC SKIP -->

---

description: This file describes the frontend testing guidelines for the project.
applyTo: **/frontend/**/\*.test.{js,jsx,ts,tsx}

---

# **Copilot Rule: Robust React Testing with Bun & TypeScript**

**📖 See also**: `docs/test-setup-patterns.md` for unified test lifecycle patterns across languages and critical anti-patterns.

## **1\. Environment & Tools**

- **Test Runner:** Use bun:test (import { test, expect, describe, it, beforeEach, afterEach } from "bun:test").
- **Library:** Use @testing-library/react.
- **Language:** TypeScript (ensure strict typing for props and mocks).
- **Matchings:** Use @testing-library/jest-dom matchers (manually imported or configured via setup file).
- **Timeout:** Use a default test timeout of 5000ms on every test (configurable in bun:test options `--timeout 5000`).
- **Speedup:** To speed up test runs, use `--test-name-pattern <pattern>` to run specific tests.
- **Test Git Changes:** When modifying a component, run only the tests related to that component using `--test-name-pattern` to quickly verify changes without running the entire suite. Also use `--changed` to run tests related to changed files in the current branch.
- **Pre-handoff Verification (Required):** Before finalizing frontend changes, always run `tsgo --noEmit` (or a task that includes it) and `mise run //frontend:test:new` to catch TypeScript and changed-file regressions early.
- **Test Stability:** For flaky tests, use `--rerun-each 10` to automatically rerun failed tests up to 10 times before marking them as failed.
- **Test Isolation:** Use beforeEach and afterEach hooks to set up and clean up test environments, ensuring no shared state between tests.
- **Mocking:** Use `msw` (Mock Service Worker) and `msw-auto-mock`  for API mocking when testing components that make network requests, ensuring tests are fast and reliable without hitting real endpoints.
- **`IS_REACT_ACT_ENVIRONMENT`:** Set `(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true` **after** `GlobalRegistrator.register()` in `test/setup.ts`. Placing it before registration has no effect — GlobalRegistrator overwrites the global context, leaving `@testing-library/react` unaware of the act() environment and printing "not configured to support act(...)" for every render. 

- **Test Noise Policy:** Never hide test noise by muting `console` output or swallowing errors. Always fix the underlying cause. Use this **triage decision tree**:

  1. **See "not configured to support act(...)" or `dispatchEvent` errors?**
     → **Fix:** Event constructor mismatch in `test/setup.ts`. Check for Bun-native constructors overriding happy-dom's `Event`/`EventTarget`/`MessageEvent` in the `nativeGlobals` capture block. Remove them and restore happy-dom versions.

  2. **See "not wrapped in act(...)" warnings during `fireEvent` or user interactions?**
     → **Fix:** Stop using `fireEvent` immediately—it doesn't use the act() environment. Replace with `@testing-library/user-event` via `await user.event(...)` (no manual act wrapper needed). Also gate hooks with `skip: !<requiredProp>` so background subscriptions (WebSocket, RTK Query fetches) don't fire when component props aren't available.

  3. **See "Failed to fetch" or RTK Query cache mismatches after seeding with `upsertQueryData`?**
     → **Fix:** Use MSW endpoint overrides (`getMswServer().use(http.get(...))`) instead of cache seeding. Live queries can overwrite seeded values before assertions run, causing CI flakes.

  4. **Intermittent `act(...)` warnings or MUI Dialog content still visible after close?**
     → **Fix:** Remove manual `cleanup()` or `document.body.innerHTML = ""` in test files—`frontend/test/bun-setup.ts` already does this in `afterEach`. Duplicate teardown races with MUI Transition unmount timing. For Dialog assertion, wrap in `waitFor` to let async unmount complete.

- **Avoid duplicate teardown:** When shared test setup already performs cleanup (for example `frontend/test/bun-setup.ts` calling `cleanup()` in `afterEach`), do not add extra per-test-file `cleanup()` calls or manual `document.body.innerHTML = ""` resets. Duplicate teardown can race with MUI `Transition` unmount timing and cause intermittent `act(...)` warnings.

## **2\. Core Philosophy**

- **Test Behavior, Not Implementation:** Focus on user outcomes, not internal component logic, state, or DOM structure.
- **Resilience:** Tests must pass as long as the functional requirements are met, regardless of HTML tag changes or CSS refactoring.
- **Semantic Queries:** Always prefer queries that reflect how users interact with the UI (for example, getByRole, getByLabelText) over implementation-specific selectors (for example, getByTestId).
- **Avoid Test Fragility:** Do not use queries that rely on CSS classes, IDs, or deep DOM structures, as these are prone to break with UI changes.
- **Isolate Tests:** Each test should be independent and not rely on the state or side effects of other tests. Use beforeEach and afterEach to set up and tear down any necessary state or mocks.
- **Mock External Dependencies:** When testing components that interact with external APIs or services, use mocking libraries like `msw` to simulate responses and ensure tests are fast and reliable without hitting real endpoints.

## **3\. Query Priority (The "No-Break" Rule)**

Always use screen from @testing-library/react. Use queries in this order:

1. getByRole: Primary choice. Always prefer the name option (for example, screen.getByRole('button', { name: /submit/i })).
2. getByLabelText: For form inputs.
3. getByPlaceholderText: Only if no label exists.
4. getByText: For non-interactive elements.
5. getByTestId: Use ONLY as a last resort for dynamic content without semantic meaning.

**❌ STRICTLY PROHIBITED:**

- No container.querySelector().
- No CSS class selectors (.my-class), IDs (\#id), or deep HTML tag selectors (div \> span).

## **4\. User Interactions**

- Use @testing-library/user-event.
- All interactions must be await-ed.
- Initialize const user \= userEvent.setup(); before render().

## **5\. TypeScript Requirements**

- Define interfaces for mocked props.
- Use as casting only when absolutely necessary (for example, for mocked functions: const mockFn \= myFunc as Mock;).
- Ensure all screen queries are properly typed (RTL does this automatically, but avoid any).

## **6\. Code Standard (Bun \+ TS)**

```typescript
import { test, expect, describe } from "bun:test";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MyComponent } from "./MyComponent";

describe("MyComponent", () \=\> {
  test("should update state when user interacts", async () \=\> {
    const user \= userEvent.setup();
    render(\<MyComponent /\>);

    // ✅ GOOD: Semantic and resilient
    const input \= screen.getByLabelText(/user name/i);
    const button \= screen.getByRole('button', { name: /save/i });

    await user.type(input, "John Doe");
    await user.click(button);

    expect(screen.getByText(/data saved/i)).toBeInTheDocument();
  });
});
```

## **7\. Additional Guidelines**
- **Test Coverage:** Aim for comprehensive test coverage of critical components and user flows, but prioritize quality and maintainability over quantity.
- **Connection to Backend:** When testing components that rely on backend data, use `msw` to mock API responses, ensuring tests are fast and do not depend on the availability of the backend. If you see errors like "Failed to fetch" or "Network error" in your tests, check your `msw` setup and ensure that all expected API calls are properly mocked. Also an message like `[MSW] Warning: intercepted a request without a matching request handler: GET http://localhost:3000/api/data` indicates that your component is making a request that `msw` does not have a handler for. To fix this, add a request handler in your `msw` setup file that matches the endpoint being called by your component.

### MSW-Specific Patterns

- **Request body consumption:** When MSW handlers inspect request bodies (e.g., POST body parsing), use `await request.clone().json()` instead of `await request.json()` to avoid `InvalidStateError: Body has already been used` during test reruns with `--rerun-each 10`.
- **Prefer MSW handlers over cache seeding:** For components with multiple data sources, use explicit MSW endpoint overrides (`getMswServer().use(http.get(...))`) instead of seeding RTK Query cache with `upsertQueryData`. Cache seeding can be overwritten by live queries before assertions run, causing flakes.
- **Shared external mocks:** For recurring external requests in this repo, prefer adding a shared handler in `frontend/src/mocks/customHandlers.ts` instead of repeating one-off test-local handlers. This is especially important for `/api/filesystem/support` and the GitHub announcements discussions endpoint used by dashboard widgets.
- **Restore global fetch mocks:** If a test directly overrides `globalThis.fetch`, always restore the original value in `afterEach` to prevent cross-test leaks that surface later as unexpected `Network error` failures.
- **Mock the real source of truth:** Before adding MSW handlers, verify where the component state actually comes from. If behavior is driven by props/local state (for example `disk.hdidle_device` in `HDIdleDiskSettings`), set deterministic fixture props in the test and assert transitions with `waitFor`; do not rely on mocking unrelated endpoints like `/api/disk/:diskId/hdidle/config`.

### MUI & Component-Specific Patterns

- **MUI v9 Dialog in tests:** To disable dialog animation in a test theme, use `createTheme({ components: { MuiDialog: { defaultProps: { transitionDuration: 0 } } } })` — the MUI v5/v6 pattern `slotProps: { transition: { timeout: 0 } }` has no effect in v9. Even with `transitionDuration: 0`, MUI's default `closeAfterTransition={true}` unmounts dialog content asynchronously after `onExited` fires; always wrap assertions that check for the *absence* of dialog content in `waitFor`.
- **Avoid test output noise:** Never hide test noise by muting `console` output; always fix the underlying cause. Never add manual `cleanup()` or `document.body.innerHTML = ""` teardown inside test files when shared teardown in `frontend/test/bun-setup.ts` already runs cleanup after each test — duplicate cleanup can trigger MUI Transition updates outside React act() timing.
- **Test Naming:** Use descriptive test names that clearly indicate the expected behavior being tested (for example, "should display error message when API call fails").
- **Test Organization:** Group related tests together using describe blocks to improve readability and maintainability of the test suite.
- **Test Data:** Use realistic test data that closely mimics actual user input and API responses to ensure tests are meaningful and effective at catching potential issues.