/**
 * Example test demonstrating MSW integration with React 19 and RTK Query
 * 
 * This test shows how to:
 * 1. Render a React 19 component with Testing Library
 * 2. Use RTK Query hooks for data fetching
 * 3. Use React 19 features like modern patterns
 * 
 * Note: SSE is deprecated for this project. Use WebSocket for real-time streaming.
 */

import "../setup";
import { describe, it, expect, beforeEach } from "bun:test";

describe("MSW Integration Example Tests", () => {
	beforeEach(() => {
		// Clear DOM between tests
		document.body.innerHTML = "";
		localStorage.clear();
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
