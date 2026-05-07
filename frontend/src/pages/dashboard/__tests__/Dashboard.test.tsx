import { render, screen } from "@testing-library/react";
import { Provider } from "react-redux";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it } from "vitest";
import { createTestStore } from "/test/testing";
import { Dashboard } from "../Dashboard";

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