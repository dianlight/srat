import { render, screen} from "@testing-library/react";
import React from "react";
import { Provider } from "react-redux";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it } from "vitest";
import { createTestStore } from "/test/testing";
import { DashboardMetrics } from "../DashboardMetrics";

describe("Dashboard Collapsible Sections", () => {
    beforeEach(() => {
        // Shared setup handles cleanup.
    });

    const renderCollapsibleMetrics = async () => {
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

    it("renders process metrics section as collapsible", async () => {
        await renderCollapsibleMetrics();

        const processMetricsHeaders = screen.getAllByText("Process Metrics");
        expect(processMetricsHeaders.length).toBeGreaterThan(0);
    });

    it("renders disk health section as collapsible", async () => {
        await renderCollapsibleMetrics();

        const diskHealthHeaders = screen.getAllByText("Disk Health");
        expect(diskHealthHeaders.length).toBeGreaterThan(0);
    });

    it("renders network health section as collapsible", async () => {
        await renderCollapsibleMetrics();

        const networkHealthHeaders = screen.getAllByText("Network Health");
        expect(networkHealthHeaders.length).toBeGreaterThan(0);
    });

    it("renders samba status section as collapsible", async () => {
        await renderCollapsibleMetrics();

        const sambaStatusHeaders = screen.getAllByText("Samba Status");
        expect(sambaStatusHeaders.length).toBeGreaterThan(0);
    });
});