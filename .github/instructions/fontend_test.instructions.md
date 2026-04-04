<!-- DOCTOC SKIP -->

---

description: This file describes the frontend testing guidelines for the project.
applyTo: **/frontend/**/\*.test.{js,jsx,ts,tsx}

---

# **Copilot Rule: Robust React Testing with Bun & TypeScript**

## **1\. Environment & Tools**

- **Test Runner:** Use bun:test (import { test, expect, describe, it, beforeEach, afterEach } from "bun:test").
- **Library:** Use @testing-library/react.
- **Language:** TypeScript (ensure strict typing for props and mocks).
- **Matchings:** Use @testing-library/jest-dom matchers (manually imported or configured via setup file).
- **Timeout:** Use a default test timeout of 5000ms on every test (configurable in bun:test options `--timeout 5000`).
- **Test Stability:** For flaky tests, use `--rerun-each 10` to automatically rerun failed tests up to 10 times before marking them as failed.
- **Test Isolation:** Use beforeEach and afterEach hooks to set up and clean up test environments, ensuring no shared state between tests.
- **Mocking:** Use `msw` (Mock Service Worker) and `msw-auto-mock`  for API mocking when testing components that make network requests, ensuring tests are fast and reliable without hitting real endpoints. 

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
- **Test Naming:** Use descriptive test names that clearly indicate the expected behavior being tested (for example, "should display error message when API call fails").
- **Test Organization:** Group related tests together using describe blocks to improve readability and maintainability of the test suite.
- **Test Data:** Use realistic test data that closely mimics actual user input and API responses to ensure tests are meaningful and effective at catching potential issues.