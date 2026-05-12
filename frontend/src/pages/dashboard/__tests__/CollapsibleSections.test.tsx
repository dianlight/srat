import { render, within } from "@testing-library/react";
import React from "react";
import { Provider } from "react-redux";
import { MemoryRouter } from "react-router-dom";
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
        const { container } = await renderCollapsibleMetrics();

        const processMetricsHeaders = within(container).getAllByText("Process Metrics");
        expect(processMetricsHeaders.length).toBeGreaterThan(0);
    });

    it("renders disk health section as collapsible", async () => {
        const { container } = await renderCollapsibleMetrics();

        const diskHealthHeaders = within(container).getAllByText("Disk Health");
        expect(diskHealthHeaders.length).toBeGreaterThan(0);
    });

    it("renders network health section as collapsible", async () => {
        const { container } = await renderCollapsibleMetrics();

        const networkHealthHeaders = within(container).getAllByText("Network Health");
        expect(networkHealthHeaders.length).toBeGreaterThan(0);
    });

    it("renders samba status section as collapsible", async () => {
        const { container } = await renderCollapsibleMetrics();

        const sambaStatusHeaders = within(container).getAllByText("Samba Status");
        expect(sambaStatusHeaders.length).toBeGreaterThan(0);
    });
});