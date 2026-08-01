import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ComponentProps } from "react";
import { Provider } from "react-redux";
import { BrowserRouter } from "react-router";
import { createTestStore } from "/test/testing";
import { type Partition } from "../../../../store/sratApi";
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

const createPartition = (overrides: Record<string, unknown> = {}): Partition => (({
    id: "part-1",
    name: "data-1",
    fs_type: "ext4",
    size: 500_000_000,
    mount_point_data: {},
    ...overrides
}) as Partition);

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
    /*
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
    */
    /*
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
 */
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
        const onMount = vi.fn(() => undefined);

        await renderPanel({
            disk: baseDisk as any,
            partition: createPartition({ mount_point_data: {} }),
            onToggleAutomount: vi.fn(() => undefined),
            onMount,
            onUnmount: vi.fn(() => undefined),
            onCreateShare: vi.fn(() => undefined),
            onGoToShare: vi.fn(() => undefined),
        });

        const mountButton = await screen.findByRole("button", {
            name: /mount partition/i,
        });
        await user.click(mountButton);

        expect(onMount).toHaveBeenCalledTimes(1);
    });

    it("lays out partition action buttons in a responsive grid", async () => {
        await renderPanel({
            disk: baseDisk as any,
            partition: createPartition({
                mount_point_data: {},
                filesystem_info: {
                    support: {
                        canCheck: true,
                        canSetLabel: true,
                        canFormat: true,
                    },
                },
            }),
            onToggleAutomount: vi.fn(() => undefined),
            onMount: vi.fn(() => undefined),
            onUnmount: vi.fn(() => undefined),
            onCreateShare: vi.fn(() => undefined),
            onGoToShare: vi.fn(() => undefined),
        });

        const actionsHeading = await screen.findByText(/^Actions$/);
        const actionsContainer = actionsHeading.nextElementSibling as HTMLElement;

        expect(actionsContainer).toBeTruthy();
        expect(getComputedStyle(actionsContainer).display).toBe("grid");
    });

    it("disables partition actions in read-only mode and shows tooltip", async () => {
        const user = userEvent.setup();

        await renderPanel({
            disk: baseDisk as any,
            partition: createPartition({ mount_point_data: {} }),
            readOnly: true,
            onToggleAutomount: vi.fn(() => undefined),
            onMount: vi.fn(() => undefined),
            onUnmount: vi.fn(() => undefined),
            onCreateShare: vi.fn(() => undefined),
            onGoToShare: vi.fn(() => undefined),
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
            onToggleAutomount: vi.fn(() => undefined),
            onMount: vi.fn(() => undefined),
            onUnmount: vi.fn(() => undefined),
            onCreateShare: vi.fn(() => undefined),
            onGoToShare: vi.fn(() => undefined),
        });

        expect(screen.queryByText(/^Actions$/)).toBeNull();
    });

    it("shows raw disk badge when the disk has no partitions", async () => {
        await renderPanel({ disk: baseDisk as any });

        expect(
            await screen.findByText("Raw disk (no partition table)"),
        ).toBeTruthy();
    });

    it("shows a disk-level mount action for a synthesized whole-disk partition", async () => {
        const user = userEvent.setup();
        const onMount = vi.fn(() => undefined);
        const wholeDiskPartition = createPartition({
            id: "part-sda",
            legacy_device_name: "sda",
            mount_point_data: {},
        });
        const disk = {
            ...baseDisk,
            legacy_device_name: "sda",
            partitions: { "part-sda": wholeDiskPartition },
        };

        await renderPanel({
            disk: disk as any,
            onToggleAutomount: vi.fn(() => undefined),
            onMount,
            onUnmount: vi.fn(() => undefined),
            onCreateShare: vi.fn(() => undefined),
            onGoToShare: vi.fn(() => undefined),
        });

        const mountButton = await screen.findByRole("button", {
            name: /mount disk/i,
        });
        await user.click(mountButton);

        expect(onMount).toHaveBeenCalledTimes(1);
        expect(onMount).toHaveBeenCalledWith(wholeDiskPartition);
    });

    it("shows raw disk badge when the only partition is a synthesized whole-disk entry", async () => {
        // Task 044: a disk without a partition table gets a synthesized
        // whole-disk partition with no filesystem type so it stays actionable.
        // It is not a real partition and must not be counted as one.
        const disk = {
            ...baseDisk,
            legacy_device_name: "sdc",
            partitions: {
                "usb-General_USB_Flash_Disk_0111607137301461-0:0": createPartition({
                    id: "usb-General_USB_Flash_Disk_0111607137301461-0:0",
                    legacy_device_name: "sdc",
                    fs_type: null,
                    mount_point_data: {},
                }),
            },
        };

        await renderPanel({ disk: disk as any });

        expect(
            await screen.findByText("Raw disk (no partition table)"),
        ).toBeTruthy();
        expect(screen.queryByText(/1 Partition\(s\)/)).toBeNull();
    });

    it("counts a real whole-disk filesystem partition as a partition", async () => {
        // Task 044: a superfloppy filesystem written directly to the disk
        // (e.g. vfat) is a real partition and should still be counted.
        const disk = {
            ...baseDisk,
            legacy_device_name: "sdd",
            partitions: {
                "usb-General_UDisk-0:0": createPartition({
                    id: "usb-General_UDisk-0:0",
                    legacy_device_name: "sdd",
                    fs_type: "vfat",
                    mount_point_data: {},
                }),
            },
        };

        await renderPanel({ disk: disk as any });

        expect(await screen.findByText("1 Partition(s)")).toBeTruthy();
    });

    it("hides the whole-disk mount action when the partition is named hassos", async () => {
        const disk = {
            ...baseDisk,
            legacy_device_name: "sda",
            partitions: {
                "part-sda": createPartition({
                    id: "part-sda",
                    legacy_device_name: "sda",
                    name: "hassos-data",
                    mount_point_data: {},
                }),
            },
        };

        await renderPanel({
            disk: disk as any,
            onToggleAutomount: vi.fn(() => undefined),
            onMount: vi.fn(() => undefined),
            onUnmount: vi.fn(() => undefined),
            onCreateShare: vi.fn(() => undefined),
            onGoToShare: vi.fn(() => undefined),
        });

        expect(screen.queryByRole("button", { name: /mount disk/i })).toBeNull();
    });

    it("disables the whole-disk mount action in read-only mode with tooltip", async () => {
        const user = userEvent.setup();
        const disk = {
            ...baseDisk,
            legacy_device_name: "sda",
            partitions: {
                "part-sda": createPartition({
                    id: "part-sda",
                    legacy_device_name: "sda",
                    mount_point_data: {},
                }),
            },
        };

        await renderPanel({
            disk: disk as any,
            readOnly: true,
            onToggleAutomount: vi.fn(() => undefined),
            onMount: vi.fn(() => undefined),
            onUnmount: vi.fn(() => undefined),
            onCreateShare: vi.fn(() => undefined),
            onGoToShare: vi.fn(() => undefined),
        });

        const mountButton = await screen.findByRole("button", {
            name: /mount disk/i,
        });
        expect((mountButton as HTMLButtonElement).disabled).toBe(true);

        const hoverTarget = mountButton.parentElement ?? mountButton;
        await user.hover(hoverTarget as HTMLElement);

        expect(await screen.findByText(/read-only mode enabled/i)).toBeTruthy();
    });
});
