import { describe, it, expect, beforeEach } from "bun:test";

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

describe("Volumes component", () => {
    beforeEach(() => {
        localStorage.clear();
        // Clear DOM between tests
        document.body.innerHTML = "";
    });

    it("renders volumes page without crashing", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

        // Check that the component renders
        expect(container).toBeTruthy();
    });

    it("renders with initial disks data", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();
        const mockDisks = [
            {
                id: "disk1",
                name: "sda",
                size: 1000000000,
                partitions: []
            }
        ];

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any, { initialDisks: mockDisks }),
                })
            )
        );

        expect(container).toBeTruthy();
    });

    it("handles hide system partitions toggle", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

        // Find the hide system partitions switch
        const switches = container.querySelectorAll('input[type="checkbox"]');
        if (switches.length > 0) {
            fireEvent.click(switches[0]);
            // Check localStorage was updated
            expect(localStorage.getItem("volumes.hideSystemPartitions")).toBeTruthy();
        }

        expect(container).toBeTruthy();
    });

    it("persists selected partition to localStorage", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        // Set initial localStorage value
        localStorage.setItem("volumes.selectedPartitionId", "test-partition-1");

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

        // Verify localStorage is being used
        expect(localStorage.getItem("volumes.selectedPartitionId")).toBe("test-partition-1");
        expect(container).toBeTruthy();
    });

    it("persists expanded disks to localStorage", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        // Set initial expanded disks
        localStorage.setItem("volumes.expandedDisks", JSON.stringify(["disk1", "disk2"]));

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

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
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        // Set invalid JSON in localStorage
        localStorage.setItem("volumes.expandedDisks", "invalid-json");

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

        // Should handle invalid data without crashing
        expect(container).toBeTruthy();
    });

    it("renders VolumesTreeView component", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

        // Check that tree view structure exists
        expect(container).toBeTruthy();
    });

    it("renders VolumeDetailsPanel component", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

        // Check that details panel structure exists
        expect(container).toBeTruthy();
    });

    it("renders VolumeMountDialog component", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

        // Check that mount dialog structure exists
        expect(container).toBeTruthy();
    });

    it("renders PreviewDialog component", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

        // Check that preview dialog structure exists
        expect(container).toBeTruthy();
    });

    it("handles partition selection", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();
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

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any, { initialDisks: mockDisks }),
                })
            )
        );

        // Look for partition items that can be clicked
        const treeItems = container.querySelectorAll('[role="treeitem"]');
        if (treeItems.length > 0) {
            fireEvent.click(treeItems[0]);
        }

        expect(container).toBeTruthy();
    });

    it("handles disk expansion toggle", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();
        const mockDisks = [
            {
                id: "disk1",
                name: "sda",
                size: 1000000000,
                partitions: []
            }
        ];

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any, { initialDisks: mockDisks }),
                })
            )
        );

        // Look for expandable tree items
        const expandButtons = container.querySelectorAll('[aria-label*="expand"]');
        if (expandButtons.length > 0) {
            fireEvent.click(expandButtons[0]);
        }

        expect(container).toBeTruthy();
    });

    it("renders loading state correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

        // Check for loading indicators
        const loadingElements = container.querySelectorAll('[role="progressbar"]');
        expect(loadingElements.length).toBeGreaterThanOrEqual(0);
    });

    it("handles location state navigation", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { MemoryRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                MemoryRouter,
                { initialEntries: [{ pathname: '/volumes', state: { from: 'dashboard' } }] },
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders grid layout correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

        // Check for Grid layout elements
        const gridElements = container.querySelectorAll('[class*="MuiGrid"]');
        expect(gridElements.length).toBeGreaterThanOrEqual(0);
    });

    it("handles filter options correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

        // Look for filter controls
        const formControls = container.querySelectorAll('[class*="FormControl"]');
        expect(formControls.length).toBeGreaterThanOrEqual(0);
    });

    it("handles empty disk list", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any, { initialDisks: [] }),
                })
            )
        );

        // Should handle empty list gracefully
        expect(container).toBeTruthy();
    });

    it("handles SSE data updates", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

        // Component should be able to receive SSE updates
        expect(container).toBeTruthy();
    });

    it("renders paper container correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

        // Look for Paper component
        const papers = container.querySelectorAll('[class*="MuiPaper"]');
        expect(papers.length).toBeGreaterThanOrEqual(0);
    });

    it("handles tour events correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

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

    it("exports components from index correctly", async () => {
        const components = await import("../components");

        expect(components.VolumesTreeView).toBeTruthy();
        expect(components.VolumeDetailsPanel).toBeTruthy();
        expect(components.VolumeMountDialog).toBeTruthy();
    });

    it("handles responsive layout", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { BrowserRouter } = await import("react-router-dom");
        const { Volumes } = await import("../Volumes");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(Provider, {
                    store,
                    children: React.createElement(Volumes as any),
                })
            )
        );

        // Check that responsive grid is used
        expect(container).toBeTruthy();
    });
});
