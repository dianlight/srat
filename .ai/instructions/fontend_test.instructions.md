<!-- DOCTOC SKIP -->

---

description: This file describes the frontend testing guidelines for the project. If test environments or tools change, update this document to reflect the new best practices. If you encounter a test failure or warning that isn't covered here, add a new section with the issue and its resolution to help future contributors.
applyTo: **/frontend/**/*.test.{js,jsx,ts,tsx}

---

# **Copilot Rule: Robust React Testing with Vitest & TypeScript**

> Self‑Evolving Instruction: This document is intended to evolve through practical use. Only apply changes that are triggered by real, reproducible issues or necessary clarifications; avoid speculative edits.

**📖 See also**: `docs/test-setup-patterns.md` for unified test lifecycle patterns across languages and critical anti-patterns.

## **1. Environment & Tools**

- **Test Runner:** Use Vitest (import { test, expect, describe, it, beforeEach, afterEach, vi } from "vitest"). Bun remains the package manager/runtime for scripts, but the frontend test runner is `bunx vitest`.
- **Library:** Use @testing-library/react.
- **Language:** TypeScript (ensure strict typing for props and mocks).
- **Matchings:** Use @testing-library/jest-dom matchers (manually imported or configured via setup file).
- **Timeout:** The shared default timeout lives in `frontend/vitest.config.ts`. Prefer updating config or per-test timeouts there instead of scattering CLI timeout flags.
- **Speedup:** To speed up test runs, run a specific file and optionally filter by test name with `bunx vitest run path/to/file.test.tsx -t "pattern"`.
- **Test Git Changes:** When modifying a component, run only the related test files first, then use `bunx vitest run --changed` for branch-local regression checks.
- **Pre-handoff Verification (Required):** Before finalizing frontend changes, always run `tsgo --noEmit` (or a task that includes it) and `mise run //frontend:test:new` to catch TypeScript and changed-file regressions early.
- **Test Stability:** For flaky tests, use Vitest retries (`--retry 10`) only as a temporary diagnostic aid; fix the root cause before finalizing.
- **Test Isolation:** Use beforeEach and afterEach hooks to set up and clean up test environments, ensuring no shared state between tests.
- **Mocking:** Use `msw` (Mock Service Worker) and `msw-auto-mock` for API mocking when testing components that make network requests, ensuring tests are fast and reliable without hitting real endpoints.
- **`IS_REACT_ACT_ENVIRONMENT`:** Set `(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true` **after** `GlobalRegistrator.register()` in `test/happy-dom-setup.ts`. Placing it before registration has no effect — GlobalRegistrator replaces the global context, leaving `@testing-library/react` unaware of the act() environment and producing "not configured to support act(...)" warnings on render.

- **Test Noise Policy:** Never hide test noise by muting `console` output or swallowing errors. Always fix the underlying cause. Use this **triage decision tree**:

  1. **See "not configured to support act(...)" or `dispatchEvent` errors?**  
     → **Fix:** Event constructor mismatch in `test/happy-dom-setup.ts`. Check for Bun-native constructors overriding happy-dom's `Event`/`EventTarget`/`MessageEvent` in any `nativeGlobals` capture block. Restore happy-dom versions.

  2. **See "not wrapped in act(...)" warnings during interactions?**  
     → **Fix:** Stop using `fireEvent`—it doesn't use the act() environment. Replace with `@testing-library/user-event` (await the user interactions). Also gate hooks with `skip: !<requiredProp>` so background subscriptions (WebSocket, RTK Query fetches) don't fire when component props aren't available.

  3. **See "Failed to fetch" or RTK Query cache mismatches after seeding with `upsertQueryData`?**  
     → **Fix:** Use MSW endpoint overrides (`getMswServer().use(http.get(...))`) instead of cache seeding. Live queries can overwrite seeded values before assertions run, causing CI flakes.

## **2. Core Philosophy**

- **Test Behavior, Not Implementation:** Focus on user outcomes, not internal component logic, state, or DOM structure.
- **Resilience:** Tests must pass as long as the functional requirements are met, regardless of HTML tag changes or CSS refactoring.
- **Semantic Queries:** Always prefer queries that reflect how users interact with the UI (for example, getByRole, getByLabelText) over implementation-specific selectors (for example, getByTestId).
- **Avoid Test Fragility:** Do not use queries that rely on CSS classes, IDs, or deep DOM structures, as these are prone to break with UI changes.
- **Isolate Tests:** Each test should be independent and not rely on the state or side effects of other tests. Use beforeEach and afterEach to set up and tear down any necessary state or mocks.
- **Mock External Dependencies:** When testing components that interact with external APIs or services, use mocking libraries like `msw` to simulate responses and ensure tests are fast and reliable without hitting real endpoints.

## **3. Query Priority (The "No-Break" Rule)**

Always use `screen` from `@testing-library/react`. Use queries in this order:

1. getByRole: Primary choice. Always prefer the name option (for example, screen.getByRole('button', { name: /submit/i })).
2. getByLabelText: For form inputs.
3. getByPlaceholderText: Only if no label exists.
4. getByText: For non-interactive elements.
5. getByTestId: Use ONLY as a last resort for dynamic content without semantic meaning.

❌ STRICTLY PROHIBITED:

- No container.querySelector().
- No CSS class selectors (.my-class), IDs (#id), or deep HTML tag selectors (div > span).

## **4. User Interactions**

- Use `@testing-library/user-event`.
- All interactions must be awaited.
- Initialize `const user = userEvent.setup();` before `render()`.

## **5. TypeScript Requirements**

- Define interfaces for mocked props.
- Use `as` casting only when absolutely necessary (for example, for mocked functions: `const mockFn = myFunc as Mock;`).
- Ensure all screen queries are properly typed (RTL does this automatically; avoid unsafe casts).

## **6. Code Standard (Vitest + TS)**

```typescript
import { test, expect, describe } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MyComponent } from "./MyComponent";

describe("MyComponent", () => {
  test("should update state when user interacts", async () => {
    const user = userEvent.setup();
    render(<MyComponent />);

    // ✅ GOOD: Semantic and resilient
    const input = screen.getByLabelText(/user name/i);
    const button = screen.getByRole("button", { name: /save/i });

    await user.type(input, "John Doe");
    await user.click(button);

    expect(screen.getByText(/data saved/i)).toBeInTheDocument();
  });
});
````
<userPrompt>
Provide the fully rewritten file, incorporating the suggested code change. You must produce the complete file.
</userPrompt>