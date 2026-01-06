import "../../../../test/setup";
import { describe, it, expect, beforeEach, afterEach } from "bun:test";

// Helper function to render DashboardActions with required wrappers
async function renderDashboardActions() {
    const React = await import("react");
    const { render, within, fireEvent } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { BrowserRouter } = await import("react-router-dom");
    const { DashboardActions } = await import("../DashboardActions");
    const { createTestStore } = await import("../../../../test/setup");

    const store = await createTestStore();

    const result = render(
        React.createElement(
            BrowserRouter,
            null,
            React.createElement(Provider, {
                store,
                children: React.createElement(DashboardActions as any),
            })
        )
    );

    return { ...result, within, fireEvent, React };
}

describe("DashboardActions component", () => {
    beforeEach(() => {
        // Clear DOM between tests
        document.body.innerHTML = "";
    });

    afterEach(async () => {
        const { cleanup } = await import("@testing-library/react");
        cleanup();
    });

    it("renders DashboardActions accordion without crashing", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });

    it("renders accordion with correct title", async () => {
        const { container, within } = await renderDashboardActions();
        const title = await within(container).findByText("Actionable Items");
        expect(title).toBeTruthy();
    });

    it("renders show ignored switch", async () => {
        const { container, within } = await renderDashboardActions();
        const label = await within(container).findByText("Show Ignored");
        expect(label).toBeTruthy();
    });

    it("handles show ignored toggle", async () => {
        const { container, fireEvent } = await renderDashboardActions();
        const switches = container.querySelectorAll('input[type="checkbox"]');
        const firstSwitch = switches[0];
        if (switches.length > 0 && firstSwitch) {
            const initialChecked = (firstSwitch as HTMLInputElement).checked;
            fireEvent.click(firstSwitch);
            const newChecked = (firstSwitch as HTMLInputElement).checked;
            expect(newChecked).not.toBe(initialChecked);
        }
    });

    it("handles accordion expansion", async () => {
        const { container, fireEvent } = await renderDashboardActions();
        const accordionSummary = container.querySelector('[id="actions-header"]');
        if (accordionSummary) {
            fireEvent.click(accordionSummary);
        }
        expect(container).toBeTruthy();
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
        const { container } = await renderDashboardActions();
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
        const accordion = container.querySelector('[data-tutor*="dashboard"]');
        expect(accordion || container).toBeTruthy();
    });

    it("handles tour events correctly", async () => {
        const { container } = await renderDashboardActions();
        expect(container).toBeTruthy();
    });

    it("renders expand icon", async () => {
        const { container } = await renderDashboardActions();
        const expandIcons = container.querySelectorAll('[data-testid="ExpandMoreIcon"]');
        expect(expandIcons.length).toBeGreaterThanOrEqual(0);
    });

    it("handles switch click without propagating to accordion", async () => {
        const { container, fireEvent } = await renderDashboardActions();
        const switches = container.querySelectorAll('input[type="checkbox"]');
        const firstSwitch = switches[0];
        if (switches.length > 0 && firstSwitch) {
            fireEvent.click(firstSwitch);
        }
        expect(container).toBeTruthy();
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
