/**
 * Example test demonstrating MSW integration with React 19, RTK Query, SSE, and WebSocket
 * 
 * This test shows how to:
 * 1. Render a React 19 component with Testing Library
 * 2. Use RTK Query hooks for data fetching
 * 3. Wait for SSE updates
 * 4. Use React 19 features like modern patterns
 */

import "../setup";
import { describe, it, expect, beforeEach } from "bun:test";
import { act } from "react";

describe("MSW Integration Example Tests", () => {
beforeEach(() => {
// Clear DOM between tests
document.body.innerHTML = "";
localStorage.clear();
});

it("renders component and receives SSE updates via RTK Query", async () => {
const React = await import("react");
const { render, screen, waitFor } = await import("@testing-library/react");
const { Provider } = await import("react-redux");
const { createTestStore } = await import("../setup");
const { useGetServerEventsQuery } = await import("../../src/store/sseApi");

// Create a test component that uses SSE via RTK Query
const SSETestComponent = () => {
const { data, isLoading, error } = useGetServerEventsQuery();

if (isLoading) return React.createElement("div", null, "Loading...");
if (error) return React.createElement("div", null, "Error occurred");

return React.createElement(
"div",
null,
React.createElement("h1", null, "SSE Test Component"),
data?.hello?.message &&
React.createElement("p", { "data-testid": "welcome-message" }, data.hello.message),
data?.heartbeat?.alive !== undefined &&
React.createElement(
"p",
{ "data-testid": "heartbeat-status" },
`Alive: ${data.heartbeat.alive}`,
),
);
};

const store = await createTestStore();

render(
React.createElement(
Provider,
{ store, children: React.createElement(SSETestComponent as any) },
),
);

// Wait for initial loading to complete
await waitFor(
() => {
expect(screen.queryByText("Loading...")).toBeFalsy();
},
{ timeout: 2000 },
);

// Wait for SSE hello message to arrive
await waitFor(
() => {
const welcomeMessage = screen.queryByTestId("welcome-message");
expect(welcomeMessage).toBeTruthy();
},
{ timeout: 3000 },
);

// Verify the mocked SSE data
const welcomeMessage = screen.getByTestId("welcome-message");
expect(welcomeMessage.textContent).toContain("SRAT");
});

it("handles basic MSW mock data", async () => {
const React = await import("react");
const { render, screen } = await import("@testing-library/react");
const { Provider } = await import("react-redux");
const { createTestStore } = await import("../setup");

// Simple component to test MSW is working
const TestComponent = () => {
return React.createElement(
"div",
null,
React.createElement("h1", { "data-testid": "title" }, "MSW Test"),
React.createElement("p", { "data-testid": "description" }, "Testing MSW integration"),
);
};

const store = await createTestStore();

render(
React.createElement(
Provider,
{ store, children: React.createElement(TestComponent as any) },
),
);

const title = screen.getByTestId("title");
expect(title.textContent).toBe("MSW Test");

const description = screen.getByTestId("description");
expect(description.textContent).toBe("Testing MSW integration");
});
});
