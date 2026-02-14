import { beforeEach, describe, expect, it } from "bun:test";
import "../../../../../test/setup";

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

    it("renders a single-mountpoint partition", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { getDiskIdentifier } = await import("../../utils");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        const disks = [
            {
                id: "disk-1",
                model: "Test Disk",
                size: 1000000000,
                partitions: {
                    "part-1": {
                        id: "part-1",
                        name: "Single Mount Partition",
                        size: 500000000,
                        mount_point_data: {
                            "/mnt/single": {
                                is_mounted: true,
                                is_write_supported: true,
                                fstype: "ext4",
                                path: "/mnt/single",
                                type: "HOST",
                            },
                        },
                    },
                },
            },
        ];

        const diskIdentifier = getDiskIdentifier(disks[0] as any, 0);

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        VolumesTreeView as any,
                        createBaseProps({
                            disks,
                            expandedItems: [diskIdentifier],
                        }),
                    ),
                },
            ),
        );

        const partitionName = await screen.findByText("Single Mount Partition");
        const mountChip = await screen.findByText("Mounted");
        const fstypeChip = await screen.findByText("ext4");
        expect(partitionName).toBeTruthy();
        expect(mountChip).toBeTruthy();
        expect(fstypeChip).toBeTruthy();
    });

    it("renders multiple mountpoints with a parent node", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { getDiskIdentifier } = await import("../../utils");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        const disks = [
            {
                id: "disk-2",
                model: "Multi Disk",
                size: 2000000000,
                partitions: {
                    "part-2": {
                        id: "part-2",
                        name: "Multi Mount Partition",
                        size: 1000000000,
                        mount_point_data: {
                            "/mnt/data1": {
                                is_mounted: true,
                                is_write_supported: true,
                                fstype: "ext4",
                                path: "/mnt/data1",
                                type: "HOST",
                            },
                            "/mnt/data2": {
                                is_mounted: false,
                                is_write_supported: true,
                                fstype: "ext4",
                                path: "/mnt/data2",
                                type: "HOST",
                            },
                        },
                    },
                },
            },
        ];

        const diskIdentifier = getDiskIdentifier(disks[0] as any, 0);
        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        VolumesTreeView as any,
                        createBaseProps({
                            disks,
                            expandedItems: [diskIdentifier],
                        }),
                    ),
                },
            ),
        );

        const partitionName = await screen.findByText("Multi Mount Partition");
        const mountpointLabel = await screen.findByText(/2 mountpoint\(s\)/i);
        expect(partitionName).toBeTruthy();
        expect(mountpointLabel).toBeTruthy();
    });

    it("invokes onPartitionSelect when a partition is clicked", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { getDiskIdentifier } = await import("../../utils");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        const user = userEvent.setup();
        let selectedPartitionId: string | null = null;
        const onPartitionSelect = (_disk: any, partition: any) => {
            selectedPartitionId = partition?.id ?? null;
        };

        const disks = [
            {
                id: "disk-3",
                model: "Click Disk",
                partitions: {
                    "part-3": {
                        id: "part-3",
                        name: "Clickable Partition",
                        mount_point_data: {
                            "/mnt/click": {
                                is_mounted: true,
                                is_write_supported: true,
                                fstype: "ext4",
                                path: "/mnt/click",
                                type: "HOST",
                            },
                        },
                    },
                },
            },
        ];

        const diskIdentifier = getDiskIdentifier(disks[0] as any, 0);

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        VolumesTreeView as any,
                        createBaseProps({
                            disks,
                            expandedItems: [diskIdentifier],
                            onPartitionSelect,
                        }),
                    ),
                },
            ),
        );

        const partitionName = await screen.findByText("Clickable Partition");
        await user.click(partitionName);

        expect(partitionName).toBeTruthy();
        expect(selectedPartitionId === "part-3").toBe(true);
    });

    it("filters system partitions when hideSystemPartitions is true", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumesTreeView } = await import("../VolumesTreeView");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        const disks = [
            {
                id: "disk-4",
                model: "System Disk",
                partitions: {
                    "part-4": {
                        id: "part-4",
                        name: "hassos-data",
                        system: true,
                        mount_point_data: {},
                        host_mount_point_data: {
                            "/mnt/host": {
                                path: "/mnt/host",
                                type: "HOST",
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
                            expandedItems: [],
                        }),
                    ),
                },
            ),
        );

        expect(screen.queryByText("SYSTEM DISK")).toBeNull();
    });
});

