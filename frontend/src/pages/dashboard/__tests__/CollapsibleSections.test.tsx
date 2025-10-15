import { describe, it, expect, beforeEach } from "bun:test";

describe("Dashboard Collapsible Sections", () => {
    beforeEach(() => {
        // Clear any state before each test
    });

    it("renders process metrics section as collapsible", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { MemoryRouter } = await import("react-router");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        MemoryRouter,
                        null,
                        React.createElement(DashboardMetrics as any)
                    ),
                },
            )
        );

        // Check that Process Metrics section header is present - use getAllByText since there may be multiple instances
        const processMetricsHeaders = screen.getAllByText("Process Metrics");
        expect(processMetricsHeaders.length).toBeGreaterThan(0);

        // Should be expandable (look for expand icon or button) - button name includes status metrics
        const processMetricsButtons = screen.getAllByRole("button", { name: /Process Metrics/ });
        expect(processMetricsButtons.length).toBeGreaterThan(0);
    });

    it("renders disk health section as collapsible", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { MemoryRouter } = await import("react-router");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        MemoryRouter,
                        null,
                        React.createElement(DashboardMetrics as any)
                    ),
                },
            )
        );

        // Check that Disk Health section header is present - use getAllByText since there may be multiple instances
        const diskHealthHeaders = screen.getAllByText("Disk Health");
        expect(diskHealthHeaders.length).toBeGreaterThan(0);

        // Should be expandable - use getAllByRole since there may be multiple instances
        const diskHealthButtons = screen.getAllByRole("button", { name: "Disk Health" });
        expect(diskHealthButtons.length).toBeGreaterThan(0);
    });

    it("renders network health section as collapsible", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { MemoryRouter } = await import("react-router");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        MemoryRouter,
                        null,
                        React.createElement(DashboardMetrics as any)
                    ),
                },
            )
        );

        // Check that Network Health section header is present - use getAllByText since there may be multiple instances
        const networkHealthHeaders = screen.getAllByText("Network Health");
        expect(networkHealthHeaders.length).toBeGreaterThan(0);

        // Should be expandable - use getAllByRole since there may be multiple instances
        const networkHealthButtons = screen.getAllByRole("button", { name: "Network Health" });
        expect(networkHealthButtons.length).toBeGreaterThan(0);
    });

    it("renders samba status section as collapsible", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { MemoryRouter } = await import("react-router");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        MemoryRouter,
                        null,
                        React.createElement(DashboardMetrics as any)
                    ),
                },
            )
        );

        // Check that Samba Status section header is present - use getAllByText since there may be multiple instances
        const sambaStatusHeaders = screen.getAllByText("Samba Status");
        expect(sambaStatusHeaders.length).toBeGreaterThan(0);

        // Should be expandable - button name includes session/tcon counts
        const sambaStatusButtons = screen.getAllByRole("button", { name: /Samba Status/ });
        expect(sambaStatusButtons.length).toBeGreaterThan(0);
    });

    it("expands process metrics section when clicked", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { MemoryRouter } = await import("react-router");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        MemoryRouter,
                        null,
                        React.createElement(DashboardMetrics as any)
                    ),
                },
            )
        );

        // Find and click the first Process Metrics button - button name includes status metrics
        const processMetricsButtons = screen.getAllByRole("button", { name: /Process Metrics/ });
        expect(processMetricsButtons.length).toBeGreaterThan(0);
        const firstProcessButton = processMetricsButtons[0];
        if (firstProcessButton) {
            fireEvent.click(firstProcessButton);
        }

        // After expanding, should show process table content
        // The table should contain process names like smbd, nmbd, wsdd2, srat
        const tableElements = screen.getAllByRole("table");
        expect(tableElements.length).toBeGreaterThan(0);
    });

    it("expands disk health section when clicked", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { MemoryRouter } = await import("react-router");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        MemoryRouter,
                        null,
                        React.createElement(DashboardMetrics as any)
                    ),
                },
            )
        );

        // Find and click the first Disk Health button - use getAllByRole since there may be multiple instances
        const diskHealthButtons = screen.getAllByRole("button", { name: "Disk Health" });
        expect(diskHealthButtons.length).toBeGreaterThan(0);
        const firstDiskButton = diskHealthButtons[0];
        if (firstDiskButton) {
            fireEvent.click(firstDiskButton);
        }

        // After expanding, should show disk health table content
        const tableElements = screen.getAllByRole("table");
        expect(tableElements.length).toBeGreaterThan(0);
    });

    it("expands samba status section when clicked", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { MemoryRouter } = await import("react-router");
        const { DashboardMetrics } = await import("../DashboardMetrics");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        MemoryRouter,
                        null,
                        React.createElement(DashboardMetrics as any)
                    ),
                },
            )
        );

        // Find and click the first Samba Status button - button name includes session/tcon counts
        const sambaStatusButtons = screen.getAllByRole("button", { name: /Samba Status/ });
        expect(sambaStatusButtons.length).toBeGreaterThan(0);
        const firstSambaButton = sambaStatusButtons[0];
        if (firstSambaButton) {
            fireEvent.click(firstSambaButton);
        }

        // After expanding, should show samba sessions and tcons tables
        // Look for "Samba Sessions" header - use getAllByText since there may be multiple instances
        const sambaSessionsHeaders = screen.getAllByText("Samba Sessions");
        expect(sambaSessionsHeaders.length).toBeGreaterThan(0);
    });
});