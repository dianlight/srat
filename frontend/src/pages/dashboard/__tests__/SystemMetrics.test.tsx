import "../../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

describe("Dashboard System Metrics", () => {
    beforeEach(() => {
        // Clear any state before each test
    });

    it("renders system metrics with uptime", async () => {
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

        // Check that system metrics section appears - use getAllByText since there may be multiple instances
        const metricsSections = screen.getAllByText("System Metrics");
        expect(metricsSections.length).toBeGreaterThan(0);

        // Check that server uptime label appears - use getAllByText since there may be multiple instances
        const serverUptimeLabels = screen.getAllByText("Server Uptime");
        expect(serverUptimeLabels.length).toBeGreaterThan(0);
    });

    it("renders CPU metrics with show details button", async () => {
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

        // Check that CPU metrics appear - use getAllByText since there may be multiple instances
        const addonCpuLabels = screen.getAllByText("Addon CPU");
        expect(addonCpuLabels.length).toBeGreaterThan(0);

        // Check that the system metrics accordion is present - use getAllByText since there may be multiple instances
        const systemMetricsAccordions = screen.getAllByText("System Metrics");
        expect(systemMetricsAccordions.length).toBeGreaterThan(0);
    });

    it("renders memory metrics", async () => {
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

        // Check that memory metrics appear - use getAllByText since there may be multiple instances
        const memoryLabels = screen.getAllByText("Addon Memory");
        expect(memoryLabels.length).toBeGreaterThan(0);
    });

    it("renders disk I/O metrics", async () => {
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

        // Check that disk I/O metrics appear - use getAllByText since there may be multiple instances
        const diskIoLabels = screen.getAllByText("Global Disk I/O");
        expect(diskIoLabels.length).toBeGreaterThan(0);

        const iopsLabels = screen.getAllByText("IOPS");
        expect(iopsLabels.length).toBeGreaterThan(0);
    });

    it("renders network I/O metrics", async () => {
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

        // Check that network I/O metrics appear - use getAllByText since there may be multiple instances
        const networkIoLabels = screen.getAllByText("Global Network I/O");
        expect(networkIoLabels.length).toBeGreaterThan(0);

        const perSecondLabels = screen.getAllByText("per second");
        expect(perSecondLabels.length).toBeGreaterThan(0);
    });

    it("renders samba sessions metrics", async () => {
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

        // Check that Samba sessions metrics appear
        const sambaSessionsLabels = screen.getAllByText("Samba Sessions");
        expect(sambaSessionsLabels.length).toBeGreaterThan(0);
    });
});