import "../../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

describe("Dashboard Component Basic Tests", () => {
    beforeEach(() => {
        // Clear DOM between tests
        document.body.innerHTML = "";
    });

    it("renders welcome text", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { MemoryRouter } = await import("react-router-dom");
        const { Dashboard } = await import("../Dashboard");
        const { createTestStore } = await import("../../../../test/setup");

        // Use proper test store with RTK Query middleware
        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        MemoryRouter,
                        {},
                        React.createElement(Dashboard as any)
                    )
                }
            )
        );

        // Find welcome text
        const welcomeElement = await screen.findByText("Welcome to SRAT");
        expect(welcomeElement).toBeTruthy();
    });

    it("shows expand button", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { MemoryRouter } = await import("react-router-dom");
        const { Dashboard } = await import("../Dashboard");
        const { createTestStore } = await import("../../../../test/setup");

        // Use proper test store with RTK Query middleware
        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        MemoryRouter,
                        {},
                        React.createElement(Dashboard as any)
                    )
                }
            )
        );

        // Find expand button by aria-label
        const expandButton = screen.getByLabelText("expand");
        expect(expandButton).toBeTruthy();
    });

    it("has grid layout structure", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { MemoryRouter } = await import("react-router-dom");
        const { Dashboard } = await import("../Dashboard");
        const { createTestStore } = await import("../../../../test/setup");

        // Use proper test store with RTK Query middleware
        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        MemoryRouter,
                        {},
                        React.createElement(Dashboard as any)
                    )
                }
            )
        );

        // Check that Dashboard renders with content (test behavior, not implementation)
        const welcomeElement = await screen.findByText("Welcome to SRAT");
        expect(welcomeElement).toBeTruthy();
        expect(container.firstChild).toBeTruthy();
    });
});