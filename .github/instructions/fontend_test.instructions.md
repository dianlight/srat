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

## **2\. Core Philosophy**

- **Test Behavior, Not Implementation:** Focus on user outcomes, not internal component logic, state, or DOM structure.
- **Resilience:** Tests must pass as long as the functional requirements are met, regardless of HTML tag changes or CSS refactoring.

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
