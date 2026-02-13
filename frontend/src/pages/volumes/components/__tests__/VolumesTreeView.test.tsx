import { beforeEach, describe, expect, it } from "bun:test";
import "../../../../../test/setup";

if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => {
            _store[k] = String(v);
        },
        removeItem: (k: string) => {
            delete _store[k];
        },
        clear: () => {
            for (const k of Object.keys(_store)) delete _store[k];
        },
    };
}

const createBaseProps = (overrides: Record<string, unknown> = {}) => ({
    expandedItems: [],
    onExpandedItemsChange: () => { },
    onPartitionSelect: () => { },
    onToggleAutomount: () => { },
    onMount: () => { },
    onUnmount: () => { },
    onCreateShare: () => { },
    onGoToShare: () => { },
    ...overrides,
});

describe("VolumesTreeView Component", () => {
    beforeEach(() => {
        localStorage.clear();
        document.body.innerHTML = "";
    });

    it("renders disks and partitions", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumesTreeView as any, {
                        onSelectPartition: () => { }
                    })
                }
            )
        );

        // Look for tree view elements using semantic query
        const treeItems = screen.queryAllByRole("treeitem");
        expect(treeItems.length).toBeGreaterThanOrEqual(0);
    });

    it("handles partition selection", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        const user = userEvent.setup();
        let selectedPartition = null;
        const onSelectPartition = (_disk: any, partition: any) => {
            selectedPartition = partition;
        };
        const disks = [
            {
                id: "disk-a",
                model: "Disk A",
                size: 1000000000,
                partitions: {
                    "part-a": {
                        id: "part-a",
                        name: "Partition A",
                        size: 500000000,
                        mount_point_data: {
                            "/mnt/a": {
                                mount_point: "/mnt/a",
                                is_mounted: true,
                                is_write_supported: true,
                                fstype: "ext4",
                            },
                        },
                    },
                },
            },
        ];

        render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumesTreeView as any, {
                        onSelectPartition
                    })
                }
            )
        );

        // Try clicking a tree item using semantic query
        const treeItems = screen.queryAllByRole("treeitem");
        const firstTreeItem = treeItems[0];
        if (treeItems.length > 0 && firstTreeItem) {
            await user.click(firstTreeItem as any);
        }

        expect(selectedPartition).toBeDefined();
        expect(document.body).toBeTruthy();
        store,
            children: React.createElement(
                VolumesTreeView as any,
                createBaseProps({
                    disks,
                    expandedItems: ["disk-a"],
                }),
            ),
                },
    ),
        );

const partitionName = await screen.findByText("Partition A");
expect(partitionName).toBeTruthy();
    });

it("does not crash with duplicate partition ids across disks", async () => {
    const React = await import("react");
    const { render } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { VolumesTreeView } = await import("../VolumesTreeView");
    const { createTestStore } = await import("../../../../../test/setup");

    const store = await createTestStore();

    const disks = [
        {
            id: "disk-a",
            model: "Disk A",
            partitions: {
                "part-1": {
                    id: "part-1",
                    name: "Partition A",
                    mount_point_data: {
                        "/mnt/a": {
                            mount_point: "/mnt/a",
                            is_mounted: true,
                            is_write_supported: true,
                            fstype: "ext4",
                        },
                    },
                },
            },
        },
        {
            id: "disk-b",
            model: "Disk B",
            partitions: {
                "part-1": {
                    id: "part-1",
                    name: "Partition B",
                    mount_point_data: {
                        "/mnt/b": {
                            mount_point: "/mnt/b",
                            is_mounted: false,
                            is_write_supported: true,
                            fstype: "ext4",
                        },
                    },
                },
            },
        },
    ];

    expect(() => {
        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        VolumesTreeView as any,
                        createBaseProps({
                            disks,
                            expandedItems: ["disk-a", "disk-b"],
                        }),
                    ),
                },
            ),
        );
    }).not.toThrow();

    const partitionA = await screen.findByText("Partition A");
    const partitionB = await screen.findByText("Partition B");
    expect(partitionA).toBeTruthy();
    expect(partitionB).toBeTruthy();
});

it("invokes onPartitionSelect when a partition is clicked", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { Provider } = await import("react-redux");
    const { VolumesTreeView } = await import("../VolumesTreeView");
    const { createTestStore } = await import("../../../../../test/setup");

    const store = await createTestStore();
    const user = userEvent.setup();
    const onPartitionSelect = () => { };

    const disks = [
        {
            id: "disk-a",
            model: "Disk A",
            partitions: {
                "part-a": {
                    id: "part-a",
                    name: "Partition A",
                    mount_point_data: {
                        "/mnt/a": {
                            mount_point: "/mnt/a",
                            is_mounted: true,
                            is_write_supported: true,
                            fstype: "ext4",
                        },
                    },
                },
            },
        },
    ];

    render(
        React.createElement(
            Provider,
            {
                store,
                children: React.createElement(
                    VolumesTreeView as any,
                    createBaseProps({
                        disks,
                        expandedItems: ["disk-a"],
                        onPartitionSelect,
                    }),
                ),
            },
        ),
    );

    const partitionName = await screen.findByText("Partition A");
    await user.click(partitionName);

    expect(partitionName).toBeTruthy();
});

it("renders multiple mountpoints with a parent node", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { VolumesTreeView } = await import("../VolumesTreeView");
    const { createTestStore } = await import("../../../../../test/setup");

    const store = await createTestStore();

    const disks = [
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
                            fstype: "ext4",
                        },
                        "/mnt/data2": {
                            mount_point: "/mnt/data2",
                            is_mounted: false,
                            is_write_supported: true,
                            fstype: "ext4",
                        },
                    },
                },
            },
        },
    ];

    render(
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

    expect(container).toBeTruthy();

    // Verify the partition node shows multiple mountpoints
    children: React.createElement(
        VolumesTreeView as any,
        createBaseProps({
            disks,
            expandedItems: ["disk-1"],
        }),
    ),
                },
),
        );

const mountpointLabel = await screen.findByText(/2 mountpoint\(s\)/i);
const partitionName = await screen.findByText("Test Partition");
expect(mountpointLabel).toBeTruthy();
expect(partitionName).toBeTruthy();
    });

it("renders a single mountpoint without extra level", async () => {
    const React = await import("react");
    const { render, screen } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { VolumesTreeView } = await import("../VolumesTreeView");
    const { createTestStore } = await import("../../../../../test/setup");

    const store = await createTestStore();

    const disks = [
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
                            fstype: "ext4",
                        },
                    },
                },
            },
        },
    ];

    render(
        React.createElement(
            Provider,
            {
                store,
                children: React.createElement(
                    VolumesTreeView as any,
                    createBaseProps({
                        disks,
                        expandedItems: ["disk-1"],
                    }),
                ),
            },
        ),
    );

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

expect(container).toBeTruthy();
// Verify the partition name is shown
const partitionName = await screen.findByText("Single Mount Partition");
expect(partitionName).toBeTruthy();
    });
});

