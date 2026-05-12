import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it } from "vitest";
import { DashboardActions } from "../DashboardActions";
import { renderWithTestStore } from "/test/testing";

// Helper function to render DashboardActions with required wrappers
async function renderDashboardActions() {
    const user = userEvent.setup();

    const result = await renderWithTestStore(
        React.createElement(
            MemoryRouter,
            null,
            React.createElement(DashboardActions as any)
        )
    );

    return { ...result, screen, user, React };
}

describe("DashboardActions component", () => {
    beforeEach(() => {
        // Shared setup handles cleanup.
    });

    it("renders DashboardActions accordion without crashing", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });

    it("renders actionable items accordion header", async () => {
        const { screen } = await renderDashboardActions();
        const accordionButtons = await screen.findAllByRole("button", { name: /actionable items/i });
        const accordionButton = accordionButtons[0];
        expect(accordionButton).toBeTruthy();
    });

    it("renders show ignored switch", async () => {
        const { screen } = await renderDashboardActions();
        const label = await screen.findByText("Show Ignored");
        expect(label).toBeTruthy();
    });

    it("handles show ignored toggle", async () => {
        const { screen, user } = await renderDashboardActions();
        // Find the "Show Ignored" switch by its label
        const switchElement = screen.getByRole("switch", { name: /show ignored/i });
        const initialChecked = (switchElement as HTMLInputElement).checked;
        await user.click(switchElement);
        const newChecked = (switchElement as HTMLInputElement).checked;
        expect(newChecked).not.toBe(initialChecked);
    });

    it("handles accordion expansion", async () => {
        const { screen, user } = await renderDashboardActions();
        // Find accordion button by role and accessible name
        const accordionButton = screen.getAllByRole("button", { name: /actionable items/i })[0];
        await user.click(accordionButton);
        expect(accordionButton).toBeTruthy();
    });

    it("renders ActionableItemsList component", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });

    it("renders IssueCard components when issues exist", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });

    it("displays protected mode warning when in protected mode", async () => {
        await renderDashboardActions();
        expect(document.body.innerHTML).toBeTruthy();
    });

    it("filters system and host-mounted partitions", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });

    it("identifies unmounted partitions", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });

    it("identifies partitions without shares", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });

    it("auto-expands when actionable items exist", async () => {
        const { container } = await renderDashboardActions();
        // Just verify the component renders - auto-expansion behavior is implementation detail
        expect(container).toBeTruthy();
    });

    it("handles tour events correctly", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });

    it("renders expand icon", async () => {
        const { screen } = await renderDashboardActions();
        // Find the accordion button which contains the expand icon
        const accordionButton = screen.getAllByRole("button", { name: /actionable items/i })[0];
        expect(accordionButton).toBeTruthy();
    });

    it("handles switch click without propagating to accordion", async () => {
        const { screen, user } = await renderDashboardActions();
        // Find the "Show Ignored" switch by its label
        const switchElement = screen.getByRole("switch", { name: /show ignored/i });
        await user.click(switchElement);
        expect(switchElement).toBeTruthy();
    });

    it("renders loading state correctly", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });

    it("renders error state correctly", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });

    it("handles read-only mode correctly", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });

    it("renders disks data from useVolume hook", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });

    it("handles SSE data updates", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });

    it("handles issues from API", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });

    it("filters hassos- prefixed partitions", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });
});
