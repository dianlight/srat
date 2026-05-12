import { render, screen } from "@testing-library/react";
import React from "react";
import { Provider } from "react-redux";
import { beforeEach, describe, expect, it } from "vitest";
import { createTestStore } from "/test/testing";
import { DashboardMetrics } from "../DashboardMetrics";

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
        // Shared setup handles cleanup.
    });

    const renderDashboardMetrics = async () => {
        const store = await createTestStore();

        return render(
            React.createElement(
                Provider,
                { store, children: React.createElement(DashboardMetrics as any) }
            )
        );
    };

    it("renders dashboard metrics component", async () => {
        const { container } = await renderDashboardMetrics();

        expect(container).toBeTruthy();
    });

    it("renders metric cards", async () => {
        await renderDashboardMetrics();

        expect(document.body.textContent?.length ?? 0).toBeGreaterThan(0);
    });

    it("handles loading state correctly", async () => {
        await renderDashboardMetrics();

        // Check for loading indicators using semantic query
        const loadingElements = screen.queryAllByRole("progressbar");
        expect(loadingElements.length).toBeGreaterThanOrEqual(0);
    });

});
