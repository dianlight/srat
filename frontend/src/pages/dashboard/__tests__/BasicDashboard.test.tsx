import { render, screen } from "@testing-library/react";
import React from "react";
import { Provider } from "react-redux";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it } from "vitest";
import { createTestStore } from "/test/testing";
import { Dashboard } from "../Dashboard";

describe("Dashboard Basic Functionality", () => {
    beforeEach(() => {
        // Shared setup handles cleanup.
    });

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
});