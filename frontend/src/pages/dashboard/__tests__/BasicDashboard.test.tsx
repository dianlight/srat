import { describe, it, expect, beforeEach } from "bun:test";

describe("Dashboard Basic Functionality", () => {
    beforeEach(() => {
        // Clear DOM between tests
        document.body.innerHTML = "";
    });

    it("dashboard component can be imported", async () => {
        const { Dashboard } = await import("../Dashboard");
        expect(Dashboard).toBeTruthy();
    });

    it("dashboard renders with proper store", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { MemoryRouter } = await import("react-router-dom");
        const { createTestStore } = await import("../../../../test/setup");
        const { Dashboard } = await import("../Dashboard");

        const store = await createTestStore();

        // Should render without throwing errors
        const container = render(
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

        expect(container).toBeTruthy();
    });

    it("welcome text appears in dashboard", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { MemoryRouter } = await import("react-router-dom");
        const { createTestStore } = await import("../../../../test/setup");
        const { Dashboard } = await import("../Dashboard");

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
});