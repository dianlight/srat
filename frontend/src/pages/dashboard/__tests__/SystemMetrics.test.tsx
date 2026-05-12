import { render, screen } from "@testing-library/react";
import React from "react";
import { Provider } from "react-redux";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it } from "vitest";
import { createTestStore } from "/test/testing";
import { DashboardMetrics } from "../DashboardMetrics";

describe("Dashboard System Metrics", () => {
    beforeEach(() => {
        // Shared setup handles cleanup.
    });

    const renderMetrics = async () => {
        const store = await createTestStore();

        return render(
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
    };

    it("renders system metrics with uptime", async () => {
        await renderMetrics();

        const metricsSections = screen.getAllByText("System Metrics");
        expect(metricsSections.length).toBeGreaterThan(0);

        const serverUptimeLabels = screen.getAllByText("Server Uptime");
        expect(serverUptimeLabels.length).toBeGreaterThan(0);
    });

    it("renders CPU metrics with show details button", async () => {
        await renderMetrics();

        const addonCpuLabels = screen.getAllByText("Addon CPU");
        expect(addonCpuLabels.length).toBeGreaterThan(0);
    });

    it("renders memory metrics", async () => {
        await renderMetrics();

        const memoryLabels = screen.getAllByText("Addon Memory");
        expect(memoryLabels.length).toBeGreaterThan(0);
    });
});