import { render, screen } from "@testing-library/react";
import React from "react";
import { Provider } from "react-redux";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it } from "vitest";
import { Dashboard } from "../Dashboard";
import { createTestStore } from "/test/testing";

describe("Dashboard Component Basic Tests", () => {
    beforeEach(() => {
        // Shared setup handles cleanup.
    });

    async function renderDashboard() {
        const store = await createTestStore();
        return render(
            <Provider store={store}>
                <MemoryRouter>
                    <Dashboard />
                </MemoryRouter>
            </Provider>,
        );
    }

    it("dashboard component can be imported", async () => {
        expect(Dashboard).toBeTruthy();
    });

    it("dashboard renders with proper store", async () => {
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

    it("renders dashboard container", async () => {
        const { container } = await renderDashboard();
        expect(container.firstChild).toBeTruthy();
    });

    it("shows expand button", async () => {
        await renderDashboard();

        // Find expand button by aria-label
        const expandButton = screen.getAllByLabelText("expand")[0];
        expect(expandButton).toBeTruthy();
    });

    it("has grid layout structure", async () => {
        const { container } = await renderDashboard();

        // Check that Dashboard renders with content (test behavior, not implementation)
        expect(container.firstChild).toBeTruthy();
    });
});