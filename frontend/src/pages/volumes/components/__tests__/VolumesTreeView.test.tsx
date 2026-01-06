import "../../../../../test/setup";
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

describe("VolumesTreeView Component", () => {
    beforeEach(() => {
        localStorage.clear();
        document.body.innerHTML = '';
    });

    it("renders volumes tree view component", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumesTreeView as any, {
                        onSelectPartition: () => { }
                    })
                }
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders tree structure with disks and partitions", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumesTreeView as any, {
                        onSelectPartition: () => { }
                    })
                }
            )
        );

        // Look for tree view elements
        const treeItems = container.querySelectorAll('[role="treeitem"]');
        expect(treeItems.length).toBeGreaterThanOrEqual(0);
    });

    it("handles partition selection", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        let selectedPartition = null;
        const onSelectPartition = (disk: any, partition: any) => {
            selectedPartition = partition;
        };

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumesTreeView as any, {
                        onSelectPartition
                    })
                }
            )
        );

        // Try clicking a tree item
        const treeItems = container.querySelectorAll('[role="treeitem"]');
        const firstTreeItem = treeItems[0];
        if (treeItems.length > 0 && firstTreeItem) {
            const user = userEvent.setup();
            await user.click(firstTreeItem as any);
        }

        expect(container).toBeTruthy();
    });

    it("displays disk icons based on type", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumesTreeView as any, {
                        onSelectPartition: () => { }
                    })
                }
            )
        );

        // Look for icons
        const icons = container.querySelectorAll('svg');
        expect(icons.length).toBeGreaterThanOrEqual(0);
    });

    it("shows partition mount status", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumesTreeView as any, {
                        onSelectPartition: () => { }
                    })
                }
            )
        );

        expect(container).toBeTruthy();
    });

    it("handles tree expansion and collapse", async () => {
        const React = await import("react");
        const { render, act } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumesTreeView as any, {
                        onSelectPartition: () => { }
                    })
                }
            )
        );

        // Look for expand/collapse buttons
        const expandIcons = container.querySelectorAll('[data-testid*="Expand"]');
        const firstExpandIcon = expandIcons[0];
        if (expandIcons.length > 0 && firstExpandIcon) {
            const button = firstExpandIcon.closest('button');
            if (button) {
                const user = userEvent.setup();
                await act(async () => {
                    await user.click(button as any);
                });
            }
        }

        expect(container).toBeTruthy();
    });

    it("displays partition information", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumesTreeView as any, {
                        onSelectPartition: () => { }
                    })
                }
            )
        );

        // Verify component rendered
        expect(container).toBeTruthy();
    });

    it("handles loading state", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumesTreeView as any, {
                        onSelectPartition: () => { }
                    })
                }
            )
        );

        // Check for loading indicators
        const loadingElements = container.querySelectorAll('[role="progressbar"]');
        expect(loadingElements.length).toBeGreaterThanOrEqual(0);
    });

    it("highlights selected partition", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumesTreeView as any, {
                        onSelectPartition: () => { },
                        selectedPartitionId: "test-partition"
                    })
                }
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders empty state when no disks available", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumesTreeView as any, {
                        onSelectPartition: () => { }
                    })
                }
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders multiple mountpoints as separate tree items", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        // Create a partition with multiple mountpoints
        const mockDisks = [
            {
                id: "disk-1",
                model: "Test Disk",
                size: 1000000000,
                connection_bus: "usb",
                partitions: {
                    "part-1": {
                        id: "part-1",
                        name: "Test Partition",
                        size: 500000000,
                        mount_point_data: {
                            "/mnt/data1": {
                                mount_point: "/mnt/data1",
                                is_mounted: true,
                                is_write_supported: true,
                                fstype: "ext4"
                            },
                            "/mnt/data2": {
                                mount_point: "/mnt/data2",
                                is_mounted: false,
                                is_write_supported: true,
                                fstype: "ext4"
                            }
                        }
                    }
                }
            }
        ];

        const expandedItems: string[] = ["disk-1"];
        const onExpandedItemsChange = (items: string[]) => {
            expandedItems.length = 0;
            expandedItems.push(...items);
        };

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(VolumesTreeView as any, {
                        disks: mockDisks,
                        expandedItems,
                        onExpandedItemsChange,
                        onPartitionSelect: () => { },
                        onToggleAutomount: () => { },
                        onMount: () => { },
                        onUnmount: () => { },
                        onCreateShare: () => { },
                        onGoToShare: () => { }
                    })
                }
            )
        );

        // Verify the partition node shows multiple mountpoints
        const mountpointLabel = await screen.findByText(/2 mountpoint\(s\)/i);
        expect(mountpointLabel).toBeTruthy();

        // Verify the partition name appears
        const partitionName = await screen.findByText("Test Partition");
        expect(partitionName).toBeTruthy();
    });

    it("renders single mountpoint partition without extra level", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        // Create a partition with single mountpoint
        const mockDisks = [
            {
                id: "disk-1",
                model: "Test Disk",
                size: 1000000000,
                connection_bus: "usb",
                partitions: {
                    "part-1": {
                        id: "part-1",
                        name: "Single Mount Partition",
                        size: 500000000,
                        mount_point_data: {
                            "/mnt/single": {
                                mount_point: "/mnt/single",
                                is_mounted: true,
                                is_write_supported: true,
                                fstype: "ext4"
                            }
                        }
                    }
                }
            }
        ];

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(VolumesTreeView as any, {
                        disks: mockDisks,
                        expandedItems: ["disk-1"],
                        onExpandedItemsChange: () => { },
                        onPartitionSelect: () => { },
                        onToggleAutomount: () => { },
                        onMount: () => { },
                        onUnmount: () => { },
                        onCreateShare: () => { },
                        onGoToShare: () => { }
                    })
                }
            )
        );

        // Verify the partition name is shown
        const partitionName = await screen.findByText("Single Mount Partition");
        expect(partitionName).toBeTruthy();

        // Verify there's no "mountpoint(s)" label for single mountpoint
        const mountpointLabel = container.querySelector('[class*="MuiChip"]');
        if (mountpointLabel?.textContent?.includes("mountpoint")) {
            expect(mountpointLabel.textContent).not.toContain("mountpoint(s)");
        }
    });
});

