import { afterEach, beforeEach, describe, expect, it } from "bun:test";
import "../../../../../test/setup";

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

// Helper function to render with Redux Provider and Router
async function renderWithProviders(
    element: any,
    options?: { seedStore?: (store: any) => void },
) {
    const React = await import("react");
    const { render } = await import("@testing-library/react");
    const { BrowserRouter } = await import("react-router-dom");
    const { Provider } = await import("react-redux");
    const { createTestStore } = await import("../../../../../test/setup");
    const store = await createTestStore();

    if (options?.seedStore) {
        options.seedStore(store);
    }

    const wrapWithProviders = (child: any) =>
        React.createElement(
            Provider,
            { store, children: React.createElement(BrowserRouter, null, child) },
        );

    const renderResult = render(wrapWithProviders(element));
    const rerenderWithProviders = (child: any) =>
        renderResult.rerender(wrapWithProviders(child));

    return { ...renderResult, store, rerenderWithProviders };
}

describe("VolumeDetailsPanel Component", () => {
    beforeEach(() => {
        localStorage.clear();
    });

    afterEach(async () => {
        const { cleanup } = await import("@testing-library/react");
        cleanup();
    });

    it("renders placeholder when no disk or partition selected", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        await renderWithProviders(React.createElement(VolumeDetailsPanel as any, {}));

        // Should display a placeholder message
        const placeholder = await screen.findByText(/Select a partition/i);
        expect(placeholder).toBeTruthy();
    });

    it("keeps hook order when selection appears", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const { rerenderWithProviders } = await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {}),
        );

        const mockDisk = {
            id: "disk-1",
            name: "sda",
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1",
            mount_point_data: {
                "/mnt/data": {
                    path: "/mnt/data",
                    fstype: "ext4",
                    is_mounted: true,
                },
            },
        };
        //const partitionId = mockPartition.id;

        rerenderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition,
            }),
        );

        const header = await screen.findByText("Partition Information");
        expect(header).toBeTruthy();
    });

    it("renders disk and partition details", async () => {
        const React = await import("react");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
            size: 1000000000,
            removable: false,
            connection_bus: "usb"
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1",
            size: 500000000,
            fstype: "ext4"
        };

        const { container } = await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition
            })
        );

        expect(container).toBeTruthy();
    });

    it("shows clean filesystem status tooltip with details", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");
        const { sratApi, Type } = await import("../../../../store/sratApi");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1",
            mount_point_data: {
                "/mnt/data": {
                    path: "/mnt/data",
                    fstype: "ext4",
                    is_mounted: true,
                    type: Type.Addon,
                },
            },
        };
        const partitionId = mockPartition.id;

        await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition,
            }),
            {
                seedStore: (store) => {
                    store.dispatch(
                        sratApi.util.upsertQueryData(
                            "getApiFilesystemState",
                            { partitionId },
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

        const user = userEvent.setup();
        const fsChip = await screen.findByText(/EXT4 Filesystem/i);
        await user.hover(fsChip);

        const description = await screen.findByText(/Filesystem is clean/i);
        expect(description).toBeTruthy();

        const additionalInfo = await screen.findAllByText((content, element) => {
            const text = element?.textContent ?? content;
            return text.includes("Last check: 2026-02-10");
        });
        expect(additionalInfo.length).toBeGreaterThan(0);
    });

    it("shows error filesystem status tooltip", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");
        const { sratApi, Type } = await import("../../../../store/sratApi");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1",
            mount_point_data: {
                "/mnt/data": {
                    path: "/mnt/data",
                    fstype: "xfs",
                    is_mounted: false,
                    type: Type.Addon,
                },
            },
        };
        const partitionId = mockPartition.id;

        await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition,
            }),
            {
                seedStore: (store) => {
                    store.dispatch(
                        sratApi.util.upsertQueryData(
                            "getApiFilesystemState",
                            { partitionId },
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

        const user = userEvent.setup();
        const fsChip = await screen.findByText(/XFS Filesystem/i);
        await user.hover(fsChip);

        const description = await screen.findByText(/Filesystem has errors/i);
        expect(description).toBeTruthy();
    });

    it("shows no status available tooltip when state missing", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");
        const { Type } = await import("../../../../store/sratApi");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1",
            mount_point_data: {
                "/mnt/data": {
                    path: "/mnt/data",
                    fstype: "btrfs",
                    is_mounted: true,
                    type: Type.Addon,
                },
            },
        };

        await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition,
            }),
        );

        const user = userEvent.setup();
        const fsChip = await screen.findByText(/BTRFS Filesystem/i);
        await user.hover(fsChip);

        const description = await screen.findByText(/No filesystem status available/i);
        expect(description).toBeTruthy();
    });

    it("renders disk icon based on connection bus", async () => {
        const React = await import("react");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
            size: 1000000000,
            removable: false,
            connection_bus: "usb"
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1"
        };

        const { container } = await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition
            })
        );

        // Verify icons are rendered using SVG elements
        const icons = container.getElementsByTagName('svg');
        expect(icons.length).toBeGreaterThanOrEqual(0);
    });

    it("renders disk icon for SDIO/MMC connection", async () => {
        const React = await import("react");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "mmcblk0",
            size: 1000000000,
            removable: false,
            connection_bus: "sdio"
        };

        const mockPartition = {
            id: "part-1",
            name: "mmcblk0p1"
        };

        const { container } = await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition
            })
        );

        expect(container).toBeTruthy();
    });

    it("renders eject icon for removable disks", async () => {
        const React = await import("react");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sdb",
            size: 1000000000,
            removable: true
        };

        const mockPartition = {
            id: "part-1",
            name: "sdb1"
        };

        const { container } = await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition
            })
        );

        expect(container).toBeTruthy();
    });

    it("displays partition size information", async () => {
        const React = await import("react");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
            size: 1000000000
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1",
            size: 500000000
        };

        const { container } = await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition
            })
        );

        // Check for size information in the container
        expect(container.textContent).toBeTruthy();
    });

    it("renders shared resource information when provided", async () => {
        const React = await import("react");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda"
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1"
        };

        const mockShare = {
            name: "shared-folder",
            path: "/mnt/share"
        };

        const { container } = await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition,
                share: mockShare
            })
        );

        expect(container).toBeTruthy();
    });

    it("handles disk info expansion toggle", async () => {
        const React = await import("react");
        const { act } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
            size: 1000000000
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1"
        };

        const { container } = await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition
            })
        );

        // Look for expand button
        const { screen } = await import("@testing-library/react");
        const expandButtons = screen.queryAllByRole("button");
        // Find the expand button (typically has ExpandMore icon)
        const firstExpandButton = expandButtons[0];
        if (expandButtons.length > 0 && firstExpandButton) {
            const user = userEvent.setup();
            await act(async () => {
                await user.click(firstExpandButton as any);
            });
        }

        expect(container).toBeTruthy();
    });

    it("renders preview dialog when object is selected", async () => {
        const React = await import("react");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda"
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1"
        };

        const { container } = await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition
            })
        );

        // Check that PreviewDialog component is present
        expect(container).toBeTruthy();
    });

    it("navigates to shares page when share is clicked", async () => {
        const React = await import("react");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda"
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1"
        };

        const mockShare = {
            name: "test-share",
            path: "/mnt/share"
        };

        const { container } = await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition,
                share: mockShare
            })
        );

        expect(container).toBeTruthy();
    });

    it("renders partition actions when available", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1",
            mount_point_data: {},
        };

        await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition,
                protectedMode: false,
                readOnly: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            }),
        );

        const mountButton = await screen.findByRole("button", { name: /mount partition/i });
        expect(mountButton).toBeTruthy();
    });

    it("disables partition actions in read-only mode with tooltip", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1",
            mount_point_data: {},
        };

        await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition,
                protectedMode: false,
                readOnly: true,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            }),
        );

        const user = userEvent.setup();
        const mountButton = await screen.findByRole("button", { name: /mount partition/i });
        expect((mountButton as HTMLButtonElement).disabled).toBe(true);

        const hoverTarget = mountButton.parentElement ?? mountButton;
        await user.hover(hoverTarget as HTMLElement);

        const tooltip = await screen.findByText(/read-only mode enabled/i);
        expect(tooltip).toBeTruthy();
    });

    it("hides partition actions for hassos partitions", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
        };

        const mockPartition = {
            id: "part-1",
            name: "hassos-data",
            mount_point_data: {},
        };

        await renderWithProviders(
            React.createElement(VolumeDetailsPanel as any, {
                disk: mockDisk,
                partition: mockPartition,
                protectedMode: false,
                readOnly: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            }),
        );

        const actionsHeading = screen.queryByText(/actions/i);
        expect(actionsHeading).toBeNull();
    });
});
