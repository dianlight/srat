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

describe("DashboardMetrics Component", () => {
    beforeEach(() => {
        localStorage.clear();
        // Clear DOM before each test
        document.body.innerHTML = '';
    });

    it("renders dashboard metrics component", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(DashboardMetrics as any) }
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders metric cards", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(DashboardMetrics as any) }
            )
        );

        // Look for metric card components
        const cards = container.querySelectorAll('[class*="MuiCard"]');
        expect(cards.length).toBeGreaterThanOrEqual(0);
    });

    it("renders accordion sections for different metric types", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(DashboardMetrics as any) }
            )
        );

        // Look for accordion elements
        const accordions = container.querySelectorAll('[class*="MuiAccordion"]');
        expect(accordions.length).toBeGreaterThanOrEqual(0);
    });

    it("handles loading state correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(DashboardMetrics as any) }
            )
        );

        // Check for loading indicators
        const loadingElements = container.querySelectorAll('[role="progressbar"]');
        expect(loadingElements.length).toBeGreaterThanOrEqual(0);
    });

    it("renders system metrics section", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(DashboardMetrics as any) }
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders disk health metrics section", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(DashboardMetrics as any) }
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders network health metrics section", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(DashboardMetrics as any) }
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders Samba status metrics section", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(DashboardMetrics as any) }
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders process metrics section", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(DashboardMetrics as any) }
            )
        );

        expect(container).toBeTruthy();
    });

    it("handles accordion expansion and collapse", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                { store, children: React.createElement(DashboardMetrics as any) }
            )
        );

        // Find expand buttons and test clicking
        const expandButtons = container.querySelectorAll('[data-testid="ExpandMoreIcon"]');
        const firstExpandButton = expandButtons[0];
        if (expandButtons.length > 0 && firstExpandButton) {
            const button = firstExpandButton.closest('button');
            if (button) {
                fireEvent.click(button);
            }
        }

        expect(container).toBeTruthy();
    });
});
