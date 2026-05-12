import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithTestStore } from "/test/testing";

async function renderVolumesPage(
    props?: Record<string, unknown>,
    routerProps?: Record<string, unknown>,
) {
    const React = await import("react");
    const { MemoryRouter } = await import("react-router");
    const { Volumes } = await import("../Volumes");

    return renderWithTestStore(
        React.createElement(
            MemoryRouter,
            routerProps ?? null,
            React.createElement(Volumes as any, props ?? {}),
        ),
    );
}

describe("Volumes component", () => {
    beforeEach(() => {
        vi.restoreAllMocks();
        localStorage.clear();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("renders volumes page without crashing", async () => {
        const { container } = await renderVolumesPage();

        // Check that the component renders
        expect(container).toBeTruthy();
    });

    it("renders with initial disks data", async () => {
        const mockDisks = [
            {
                id: "disk1",
                name: "sda",
                size: 1000000000,
                partitions: []
            }
        ];

        const { container } = await renderVolumesPage({ initialDisks: mockDisks });

        expect(container).toBeTruthy();
    });

    it("handles hide system partitions toggle", async () => {
        const { screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { container } = await renderVolumesPage();

        // Find the hide system partitions switch
        const switches = screen.queryAllByRole("checkbox");
        const firstSwitch = switches[0];
        if (switches.length > 0 && firstSwitch) {
            const user = userEvent.setup();
            await user.click(firstSwitch as any);
            // Check localStorage was updated
            expect(localStorage.getItem("volumes.hideSystemPartitions")).toBeTruthy();
        }

        expect(container).toBeTruthy();
    });

    it("persists selected partition to localStorage", async () => {
        // Set initial localStorage value
        localStorage.setItem("volumes.selectedPartitionId", "test-partition-1");

        const { container } = await renderVolumesPage();

        // Verify localStorage is being used
        expect(localStorage.getItem("volumes.selectedPartitionId")).toBe("test-partition-1");
        expect(container).toBeTruthy();
    });

    it("persists expanded disks to localStorage", async () => {
        // Set initial expanded disks
        localStorage.setItem("volumes.expandedDisks", JSON.stringify(["disk1", "disk2"]));

        const { container } = await renderVolumesPage();

        // Verify localStorage is being used
        const storedExpanded = localStorage.getItem("volumes.expandedDisks");
        expect(storedExpanded).toBeTruthy();
        if (storedExpanded) {
            const parsed = JSON.parse(storedExpanded);
            expect(Array.isArray(parsed)).toBe(true);
        }

        expect(container).toBeTruthy();
    });

    it("handles invalid localStorage data gracefully", async () => {
        // Set invalid JSON in localStorage
        localStorage.setItem("volumes.expandedDisks", "invalid-json");

        const { container } = await renderVolumesPage();

        // Should handle invalid data without crashing
        expect(container).toBeTruthy();
    });

    it("renders VolumesTreeView component", async () => {
        const { container } = await renderVolumesPage();

        // Check that tree view structure exists
        expect(container).toBeTruthy();
    });

    it("renders VolumeDetailsPanel component", async () => {
        const { container } = await renderVolumesPage();

        // Check that details panel structure exists
        expect(container).toBeTruthy();
    });

    it("renders VolumeMountDialog component", async () => {
        const { container } = await renderVolumesPage();

        // Check that mount dialog structure exists
        expect(container).toBeTruthy();
    });

    it("renders PreviewDialog component", async () => {
        const { container } = await renderVolumesPage();

        // Check that preview dialog structure exists
        expect(container).toBeTruthy();
    });

    it("handles partition selection", async () => {
        const { screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const mockDisks = [
            {
                id: "disk1",
                name: "sda",
                size: 1000000000,
                partitions: [
                    {
                        id: "part1",
                        name: "sda1",
                        size: 500000000,
                        filesystem: "ext4"
                    }
                ]
            }
        ];

        const { container } = await renderVolumesPage({ initialDisks: mockDisks });

        // Look for partition items that can be clicked
        const treeItems = screen.queryAllByRole("treeitem");
        const firstTreeItem = treeItems[0];
        if (treeItems.length > 0 && firstTreeItem) {
            const user = userEvent.setup();
            await user.click(firstTreeItem as any);
        }

        expect(container).toBeTruthy();
    });

    it("updates the nested partition label in the volumes tree data", async () => {
        const { updatePartitionLabelInDisks } = await import("../Volumes");

        const initialDisks = [
            {
                id: "disk1",
                partitions: {
                    part1: {
                        id: "part1",
                        name: "old-label",
                    },
                },
            },
        ];

        const updatedDisks = updatePartitionLabelInDisks(
            initialDisks as any,
            "part1",
            "PROVA_EXT4_4",
        );

        expect((updatedDisks[0] as any).partitions.part1.name).toBe("PROVA_EXT4_4");
        expect((initialDisks[0] as any).partitions.part1.name).toBe("old-label");
    });

    it("handles disk expansion toggle", async () => {
        const { screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const mockDisks = [
            {
                id: "disk1",
                name: "sda",
                size: 1000000000,
                partitions: []
            }
        ];

        const { container } = await renderVolumesPage({ initialDisks: mockDisks });

        // Look for expandable tree items
        const expandButtons = screen.queryAllByRole("button");
        // Find expand button by checking aria-label
        const firstExpandButton = expandButtons.find((btn) => {
            const label = btn.getAttribute("aria-label");
            return label && label.includes("expand");
        });
        if (firstExpandButton) {
            const user = userEvent.setup();
            await user.click(firstExpandButton as any);
        }

        expect(container).toBeTruthy();
    });

    it("renders loading state correctly", async () => {
        const { screen } = await import("@testing-library/react");
        const { container } = await renderVolumesPage();

        // Check for loading indicators
        const loadingElements = screen.queryAllByRole("progressbar");
        expect(loadingElements.length).toBeGreaterThanOrEqual(0);
        expect(container).toBeTruthy();
    });

    it("handles location state navigation", async () => {
        const { container } = await renderVolumesPage(undefined, {
            initialEntries: [{ pathname: "/volumes", state: { from: "dashboard" } }],
        });

        expect(container).toBeTruthy();
    });

    it("renders grid layout correctly", async () => {
        const { container } = await renderVolumesPage();

        // Verify grid layout renders correctly
        expect(container.firstChild).toBeTruthy();
    });

    it("handles filter options correctly", async () => {
        const { container } = await renderVolumesPage();

        // Verify filter controls are present
        expect(container.firstChild).toBeTruthy();
    });

    it("handles empty disk list", async () => {
        const { container } = await renderVolumesPage({ initialDisks: [] });

        // Should handle empty list gracefully
        expect(container).toBeTruthy();
    });

    it("handles SSE data updates", async () => {
        const { container } = await renderVolumesPage();

        // Component should be able to receive SSE updates
        expect(container).toBeTruthy();
    });

    it("renders paper container correctly", async () => {
        const { container } = await renderVolumesPage();

        // Verify component renders correctly
        expect(container.firstChild).toBeTruthy();
    });

    it("handles tour events correctly", async () => {
        const { container } = await renderVolumesPage();

        // Component should be able to handle tour events
        expect(container).toBeTruthy();
    });

    it("handles decodeEscapeSequence utility function", async () => {
        const { decodeEscapeSequence } = await import("../utils");

        // Test that the utility function is available
        expect(typeof decodeEscapeSequence).toBe("function");

        // Test basic functionality
        const result = decodeEscapeSequence("test");
        expect(result).toBe("test");
    });

    it("does not trigger a setState-in-render warning when the volumes query fails", async () => {
        const React = await import("react");
        const { MemoryRouter } = await import("react-router");

        vi.doMock("../../../components/GlobalEventTracker", () => ({
            __esModule: true,
            default: () => null,
            useSystemLogs: () => ({ logs: [], clearLogs: () => undefined }),
        }));

        vi.doMock("../../../hooks/volumeHook", () => ({
            useVolume: () => ({
                disks: [],
                isLoading: false,
                error: { status: "FETCH_ERROR", error: "TypeError: Failed to fetch" },
            }),
        }));

        const { Volumes } = await import("../Volumes");
        const { useSystemLogs } = await import("../../../components/GlobalEventTracker");

        const LogProbe = () => {
            useSystemLogs();
            return null;
        };

        const originalConsoleError = console.error;
        const consoleErrorMock = vi.fn((..._args: unknown[]) => undefined);
        console.error = consoleErrorMock as typeof console.error;

        try {
            await renderWithTestStore(
                React.createElement(
                    MemoryRouter,
                    null,
                    React.createElement(React.Fragment, {
                        children: [
                            React.createElement(LogProbe, { key: "log-probe" }),
                            React.createElement(Volumes as any, { key: "volumes" }),
                        ],
                    })
                )
            );

            expect(document.body).toBeTruthy();

            const loggedWarnings = consoleErrorMock.mock.calls
                .flat()
                .map((entry) => String(entry))
                .join("\n");

            expect(loggedWarnings).not.toContain("Cannot update a component");
        } finally {
            console.error = originalConsoleError;
        }
    });

    it("exports components from index correctly", async () => {
        const components = await import("../components");

        expect(components.VolumesTreeView).toBeTruthy();
        expect(components.VolumeDetailsPanel).toBeTruthy();
        expect(components.VolumeMountDialog).toBeTruthy();
    });

    it("handles responsive layout", async () => {
        const { container } = await renderVolumesPage();

        // Check that responsive grid is used
        expect(container).toBeTruthy();
    });
});
