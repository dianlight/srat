import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, mock } from "bun:test";
import type { ComponentProps } from "react";
import { Provider } from "react-redux";
import { BrowserRouter } from "react-router-dom";
import "../../../../../test/setup";
import { createTestStore } from "../../../../../test/setup";
import { type Partition, sratApi } from "../../../../store/sratApi";
import { VolumeDetailsPanel } from "../VolumeDetailsPanel";

interface RenderOptions {
    seedStore?: (store: any) => void;
}

const baseDisk = {
    id: "disk-1",
    model: "Samsung SSD",
    vendor: "Samsung",
    size: 1_000_000_000,
    removable: false,
    connection_bus: "usb",
    partitions: {},
};

const createPartition = (overrides: Record<string, unknown> = {}): Partition => ({
    id: "part-1",
    name: "data-1",
    fs_type: "ext4",
    size: 500_000_000,
    mount_point_data: {},
    ...overrides,
}) as Partition;

const renderPanel = async (
    props: ComponentProps<typeof VolumeDetailsPanel>,
    options?: RenderOptions,
) => {
    const store = await createTestStore();
    if (options?.seedStore) {
        options.seedStore(store);
    }

    return {
        store,
        ...render(
            <Provider store={store}>
                <BrowserRouter>
                    <VolumeDetailsPanel {...props} />
                </BrowserRouter>
            </Provider>,
        ),
    };
};

describe("VolumeDetailsPanel", () => {
    beforeEach(() => {
        localStorage.clear();
    });

    afterEach(() => {
        cleanup();
    });

    it("shows placeholder when nothing is selected", async () => {
        await renderPanel({});

        expect(
            await screen.findByText(/select a partition from the tree/i),
        ).toBeTruthy();
    });

    it("renders disk and partition sections", async () => {
        const partition = createPartition({
            mount_point_data: {
                "/mnt/data": {
                    path: "/mnt/data",
                    fstype: "ext4",
                    is_mounted: false,
                },
            },
        });

        await renderPanel({ disk: baseDisk as any, partition });

        expect(await screen.findByText("Disk Information")).toBeTruthy();
        expect(screen.getByText("Partition Information")).toBeTruthy();
        expect(screen.getByText(/partition id/i)).toBeTruthy();
    });

    it("toggles disk details expansion", async () => {
        const user = userEvent.setup();
        await renderPanel({ disk: baseDisk as any, partition: createPartition() });

        expect(screen.queryByText(/^Vendor$/)).toBeNull();

        await user.click(screen.getByRole("button", { name: /show more/i }));

        expect(screen.getByText(/^Vendor$/)).toBeTruthy();
        expect(screen.getByText("Samsung")).toBeTruthy();
    });

    it("opens preview dialog from disk preview button", async () => {
        const user = userEvent.setup();
        await renderPanel({ disk: baseDisk as any, partition: createPartition() });

        await user.click(screen.getByRole("button", { name: /disk preview/i }));

        expect(await screen.findByRole("dialog")).toBeTruthy();
        expect(screen.getByText(/Disk: Samsung SSD/i)).toBeTruthy();
    });

    it("shows clean filesystem tooltip details", async () => {
        const user = userEvent.setup();
        const partition = createPartition({
            fs_type: "ext4",
            mount_point_data: {
                "/mnt/data": {
                    path: "/mnt/data",
                    fstype: "ext4",
                    is_mounted: true,
                },
            },
            filesystem_info: {
                Description: "EXT4 Filesystem",
            } as any,
        });

        await renderPanel(
            { disk: baseDisk as any, partition },
            {
                seedStore: (store) => {
                    store.dispatch(
                        sratApi.util.upsertQueryData(
                            "getApiFilesystemState",
                            { partitionId: partition.id },
                            {
                                isClean: true,
                                hasErrors: false,
                                isMounted: true,
                                stateDescription: "Filesystem is clean",
                                additionalInfo: {
                                    "Last check": "2026-02-10",
                                },
                            },
                        ),
                    );
                },
            },
        );

        await user.hover(screen.getByText(/EXT4 Filesystem/i));

        const tooltip = await screen.findByRole("tooltip");
        //expect(within(tooltip).getByText(/filesystem is clean/i)).toBeTruthy();
        expect(within(tooltip).getByText(/last check/i)).toBeTruthy();
        expect(within(tooltip).getByText(/2026-02-10/i)).toBeTruthy();
    });

    it("shows error filesystem tooltip", async () => {
        const user = userEvent.setup();
        const partition = createPartition({
            fs_type: "xfs",
            mount_point_data: {
                "/mnt/data": {
                    path: "/mnt/data",
                    fstype: "xfs",
                    is_mounted: false,
                },
            },
            filesystem_info: {
                Description: "XFS Filesystem",
            } as any,
        });

        await renderPanel(
            { disk: baseDisk as any, partition },
            {
                seedStore: (store) => {
                    store.dispatch(
                        sratApi.util.upsertQueryData(
                            "getApiFilesystemState",
                            { partitionId: partition.id },
                            {
                                isClean: false,
                                hasErrors: true,
                                isMounted: false,
                                stateDescription: "Filesystem has errors",
                                additionalInfo: {},
                            },
                        ),
                    );
                },
            },
        );

        await user.hover(screen.getByText(/XFS Filesystem/i));
        const tooltip = await screen.findByRole("tooltip");
        expect(within(tooltip).getByText(/filesystem has errors/i)).toBeTruthy();
    });

    it("shows fallback filesystem tooltip when state is missing", async () => {
        const user = userEvent.setup();
        const partition = {
            name: "data-1",
            fs_type: "btrfs",
            mount_point_data: {
                "/mnt/data": {
                    path: "/mnt/data",
                    fstype: "btrfs",
                    is_mounted: true,
                },
            },
            filesystem_info: {
                Description: "BTRFS Filesystem",
            },
        } as unknown as Partition;

        await renderPanel(
            { disk: baseDisk as any, partition },
            {
                seedStore: (store) => {
                    store.dispatch(
                        sratApi.util.upsertQueryData(
                            "getApiFilesystemState",
                            { partitionId: partition.id },
                            null as any,
                        ),
                    );
                },
            },
        );

        await user.hover(screen.getByText(/BTRFS Filesystem/i));
        const tooltip = await screen.findByRole("tooltip");
        expect(within(tooltip).getByText(/no filesystem status available/i)).toBeTruthy();
    });

    it("renders mount settings when exactly one mount point is mounted", async () => {
        const partition = createPartition({
            mount_point_data: {
                "/mnt/data": {
                    path: "/mnt/data",
                    fstype: "ext4",
                    is_mounted: true,
                    is_to_mount_at_startup: true,
                    is_write_supported: false,
                    flags: [{ name: "uid", value: "1000" }],
                    custom_flags: [{ name: "compress", value: "zstd" }],
                },
            },
        });

        await renderPanel({ disk: baseDisk as any, partition });

        expect(await screen.findByText("Mount Settings")).toBeTruthy();
        expect(screen.getByText(/automatic mount/i)).toBeTruthy();
        expect(screen.getByText(/filesystem-specific mount flags/i)).toBeTruthy();
        expect(screen.getByText(/write support/i)).toBeTruthy();
    });

    it("renders mount action and triggers callback", async () => {
        const user = userEvent.setup();
        const onMount = mock(() => undefined);

        await renderPanel({
            disk: baseDisk as any,
            partition: createPartition({ mount_point_data: {} }),
            onToggleAutomount: mock(() => undefined),
            onMount,
            onUnmount: mock(() => undefined),
            onCreateShare: mock(() => undefined),
            onGoToShare: mock(() => undefined),
        });

        const mountButton = await screen.findByRole("button", {
            name: /mount partition/i,
        });
        await user.click(mountButton);

        expect(onMount).toHaveBeenCalledTimes(1);
    });

    it("disables partition actions in read-only mode and shows tooltip", async () => {
        const user = userEvent.setup();

        await renderPanel({
            disk: baseDisk as any,
            partition: createPartition({ mount_point_data: {} }),
            readOnly: true,
            onToggleAutomount: mock(() => undefined),
            onMount: mock(() => undefined),
            onUnmount: mock(() => undefined),
            onCreateShare: mock(() => undefined),
            onGoToShare: mock(() => undefined),
        });

        const mountButton = await screen.findByRole("button", {
            name: /mount partition/i,
        });
        expect((mountButton as HTMLButtonElement).disabled).toBe(true);

        const hoverTarget = mountButton.parentElement ?? mountButton;
        await user.hover(hoverTarget as HTMLElement);

        expect(await screen.findByText(/read-only mode enabled/i)).toBeTruthy();
    });

    it("hides actions for hassos partitions", async () => {
        await renderPanel({
            disk: baseDisk as any,
            partition: createPartition({ name: "hassos-data" }),
            onToggleAutomount: mock(() => undefined),
            onMount: mock(() => undefined),
            onUnmount: mock(() => undefined),
            onCreateShare: mock(() => undefined),
            onGoToShare: mock(() => undefined),
        });

        expect(screen.queryByText(/^Actions$/)).toBeNull();
    });
});
