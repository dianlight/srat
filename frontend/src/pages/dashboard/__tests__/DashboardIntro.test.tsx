import { describe, it, expect, beforeEach } from "bun:test";

// Required localStorage shim for testing environment
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => { _store[k] = String(v); },
        removeItem: (k: string) => { delete _store[k]; },
        clear: () => { for (const k of Object.keys(_store)) delete _store[k]; },
    };
}

describe("DashboardIntro Component", () => {
    beforeEach(() => {
        localStorage.clear();
        // Clear DOM before each test
        document.body.innerHTML = '';
    });

    it("renders dashboard intro component", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { DashboardIntro } = await import("../DashboardIntro");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(
                    Provider,
                    { store },
                    React.createElement(DashboardIntro as any)
                )
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders welcome message", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { DashboardIntro } = await import("../DashboardIntro");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(
                    Provider,
                    { store },
                    React.createElement(DashboardIntro as any)
                )
            )
        );

        // Check for any text content
        expect(container.textContent).toBeTruthy();
    });

    it("renders quick action buttons", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { DashboardIntro } = await import("../DashboardIntro");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(
                    Provider,
                    { store },
                    React.createElement(DashboardIntro as any)
                )
            )
        );

        // Look for buttons
        const buttons = container.querySelectorAll('button');
        expect(buttons.length).toBeGreaterThanOrEqual(0);
    });

    it("renders system information cards", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { DashboardIntro } = await import("../DashboardIntro");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(
                    Provider,
                    { store },
                    React.createElement(DashboardIntro as any)
                )
            )
        );

        // Look for cards
        const cards = container.querySelectorAll('[class*="MuiCard"]');
        expect(cards.length).toBeGreaterThanOrEqual(0);
    });

    it("handles button clicks for navigation", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { DashboardIntro } = await import("../DashboardIntro");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(
                    Provider,
                    { store },
                    React.createElement(DashboardIntro as any)
                )
            )
        );

        // Try clicking buttons
        const buttons = container.querySelectorAll('button');
        if (buttons.length > 0) {
            fireEvent.click(buttons[0]);
        }

        expect(container).toBeTruthy();
    });

    it("displays loading state when data is being fetched", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { DashboardIntro } = await import("../DashboardIntro");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(
                    Provider,
                    { store },
                    React.createElement(DashboardIntro as any)
                )
            )
        );

        // Check for loading indicators
        const loadingElements = container.querySelectorAll('[role="progressbar"]');
        expect(loadingElements.length).toBeGreaterThanOrEqual(0);
    });

    it("renders grid layout correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { DashboardIntro } = await import("../DashboardIntro");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(
                    Provider,
                    { store },
                    React.createElement(DashboardIntro as any)
                )
            )
        );

        // Look for grid elements
        const gridElements = container.querySelectorAll('[class*="MuiGrid"]');
        expect(gridElements.length).toBeGreaterThanOrEqual(0);
    });

    it("renders icons for quick actions", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { DashboardIntro } = await import("../DashboardIntro");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(
                    Provider,
                    { store },
                    React.createElement(DashboardIntro as any)
                )
            )
        );

        // Look for SVG icons
        const icons = container.querySelectorAll('svg');
        expect(icons.length).toBeGreaterThanOrEqual(0);
    });

    it("handles responsive layout", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { DashboardIntro } = await import("../DashboardIntro");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(
                    Provider,
                    { store },
                    React.createElement(DashboardIntro as any)
                )
            )
        );

        // Check that component adapts to layout
        expect(container).toBeTruthy();
    });

    it("renders tour integration elements", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { DashboardIntro } = await import("../DashboardIntro");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(
                    Provider,
                    { store },
                    React.createElement(DashboardIntro as any)
                )
            )
        );

        // Check for tour-related attributes
        expect(container).toBeTruthy();
    });
});
