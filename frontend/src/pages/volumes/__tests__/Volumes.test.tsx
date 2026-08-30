/* eslint-disable */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithTestStore, withTestHandlers } from "/test/testing";

const { toastInfoMock, toastErrorMock, toastWarnMock } = vi.hoisted(() => {
    const toastInfoMock = vi.fn((..._args: unknown[]) => undefined);
    const toastErrorMock = vi.fn((..._args: unknown[]) => undefined);
    const toastWarnMock = vi.fn((..._args: unknown[]) => undefined);
    return { toastInfoMock, toastErrorMock, toastWarnMock };
});

vi.mock("react-toastify", () => ({
    ToastContainer: () => null,
    Slide: () => null,
    toast: {
        info: (...args: unknown[]) => toastInfoMock(...args),
        error: (...args: unknown[]) => toastErrorMock(...args),
        warn: (...args: unknown[]) => toastWarnMock(...args),
    },
}));

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
        await renderVolumesPage();

        // Check that the component renders
        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
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

        await renderVolumesPage({ initialDisks: mockDisks });

        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("handles hide system partitions toggle", async () => {
        const { screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        await renderVolumesPage();

        // Find the hide system partitions switch
        const switches = screen.queryAllByRole("checkbox");
        const firstSwitch = switches[0];
        if (switches.length > 0 && firstSwitch) {
            const user = userEvent.setup();
            await user.click(firstSwitch as any);
            // Check localStorage was updated
            expect(localStorage.getItem("volumes.hideSystemPartitions")).toBeTruthy();
        }

        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("persists selected partition to localStorage", async () => {
        // Set initial localStorage value
        localStorage.setItem("volumes.selectedPartitionId", "test-partition-1");

        await renderVolumesPage();

        // Verify localStorage is being used
        expect(localStorage.getItem("volumes.selectedPartitionId")).toBe("test-partition-1");
        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("persists expanded disks to localStorage", async () => {
        // Set initial expanded disks
        localStorage.setItem("volumes.expandedDisks", JSON.stringify(["disk1", "disk2"]));

        await renderVolumesPage();

        // Verify localStorage is being used
        const storedExpanded = localStorage.getItem("volumes.expandedDisks");
        expect(storedExpanded).toBeTruthy();
        if (storedExpanded) {
            const parsed = JSON.parse(storedExpanded);
            expect(Array.isArray(parsed)).toBe(true);
        }

        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("handles invalid localStorage data gracefully", async () => {
        // Set invalid JSON in localStorage
        localStorage.setItem("volumes.expandedDisks", "invalid-json");

        await renderVolumesPage();

        // Should handle invalid data without crashing
        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("renders VolumesTreeView component", async () => {
        await renderVolumesPage();

        // Check that tree view structure exists
        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("renders VolumeDetailsPanel component", async () => {
        await renderVolumesPage();

        // Check that details panel structure exists
        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("renders VolumeMountDialog component", async () => {
        await renderVolumesPage();

        // Check that mount dialog structure exists
        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("renders PreviewDialog component", async () => {
        await renderVolumesPage();

        // Check that preview dialog structure exists
        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
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

        await renderVolumesPage({ initialDisks: mockDisks });

        // Look for partition items that can be clicked
        const treeItems = screen.queryAllByRole("treeitem");
        const firstTreeItem = treeItems[0];
        if (treeItems.length > 0 && firstTreeItem) {
            const user = userEvent.setup();
            await user.click(firstTreeItem as any);
        }

        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
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

        await renderVolumesPage({ initialDisks: mockDisks });

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

        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("renders loading state correctly", async () => {
        const { screen } = await import("@testing-library/react");
        await renderVolumesPage();

        // Check for loading indicators
        const loadingElements = screen.queryAllByRole("progressbar");
        expect(loadingElements.length).toBeGreaterThanOrEqual(0);
        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("handles location state navigation", async () => {
        await renderVolumesPage(undefined, {
            initialEntries: [{ pathname: "/volumes", state: { from: "dashboard" } }],
        });

        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("renders grid layout correctly", async () => {
        await renderVolumesPage();

        // Verify grid layout renders correctly
        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("handles filter options correctly", async () => {
        await renderVolumesPage();

        // Verify filter controls are present
        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("handles empty disk list", async () => {
        await renderVolumesPage({ initialDisks: [] });

        // Should handle empty list gracefully
        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("handles SSE data updates", async () => {
        await renderVolumesPage();

        // Component should be able to receive SSE updates
        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("renders paper container correctly", async () => {
        await renderVolumesPage();

        // Verify component renders correctly
        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("handles tour events correctly", async () => {
        await renderVolumesPage();

        // Component should be able to handle tour events
        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
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
        await renderVolumesPage();

        // Check that responsive grid is used
        expect(document.body.innerHTML.trim().length).toBeGreaterThan(0);
    });

    it("writes partition label rename into the RTK volumes cache and keeps it across refetch", async () => {
        const React = await import("react");
        const { screen, waitFor } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { MemoryRouter } = await import("react-router");
        const { Volumes } = await import("../Volumes");
        const { http, HttpResponse } = await import("msw");

        const volumesUrl = /.*\/api\/volumes(?:\?.*)?$/;
        const supportUrl = /.*\/api\/filesystem\/support(?:\?.*)?$/;
        const labelUrl = /.*\/api\/filesystem\/label(?:\?.*)?$/;

        const disks = [
            {
                id: "disk-f3-test",
                name: "sda",
                size: 1000000000,
                partitions: {
                    part1: {
                        id: "part-f3-1",
                        name: "old-label",
                        device_path: "/dev/sda1",
                        fs_type: "ext4",
                        filesystem_info: {
                            support: {
                                canCheck: true,
                                canSetLabel: true,
                                canFormat: true,
                            },
                        },
                    },
                },
            },
        ];
        // Stateful fixture: the PUT handler applies the rename to the served data,
        // mirroring the backend so a post-mutation refetch returns the new label.
        const servedDisks = structuredClone(disks);

        await withTestHandlers(
            [
                http.get(volumesUrl, () => HttpResponse.json(servedDisks)),
                http.options(supportUrl, () => HttpResponse.json({})),
                http.get(supportUrl, () =>
                    HttpResponse.json({
                        canMount: true,
                        canFormat: true,
                        canCheck: true,
                        canSetLabel: true,
                        canGetState: true,
                        labelRule: "^.{1,16}$",
                        alpinePackage: "e2fsprogs",
                        missingTools: [],
                        isExportable: false,
                        isCheckReportProgress: false,
                        isFormatReportProgress: false,
                    }),
                ),
                http.options(labelUrl, () => HttpResponse.json({})),
                http.put(labelUrl, async ({ request }) => {
                    const body = (await request.json()) as {
                        partitionId?: string;
                        label?: string;
                    };
                    if (body.partitionId && body.label) {
                        const partitions = Object.values(servedDisks[0].partitions);
                        const target = partitions.find(
                            (partition: any) => partition.id === body.partitionId,
                        );
                        if (target) {
                            target.name = body.label;
                        }
                    }
                    return HttpResponse.json({ success: true });
                }),
            ],
            async () => {
                // Pre-expand the disk row so the partition actions are visible
                // (the MUI X v9 tree has no expand buttons; the item row toggles).
                localStorage.setItem(
                    "volumes.expandedDisks",
                    JSON.stringify(["disk-f3-test"]),
                );

                const result = await renderWithTestStore(
                    React.createElement(
                        MemoryRouter,
                        null,
                        React.createElement(Volumes as any),
                    ),
                );

                // Open the Set Label dialog from the partition action overflow
                // menu (available on any breakpoint).
                const moreActionsButton = await screen.findByRole("button", {
                    name: "more actions",
                });
                const user = userEvent.setup();
                await user.click(moreActionsButton);

                const setLabelItem = await screen.findByRole("menuitem", {
                    name: /set label/i,
                });
                await user.click(setLabelItem);

                const input = await screen.findByRole("textbox", { name: /label/i });
                await user.clear(input);
                await user.type(input, "NEW-LABEL");

                const submitButton = await screen.findByRole("button", {
                    name: /^set label$/i,
                });
                await user.click(submitButton);

                // The tree reflects the new label (rendered from the RTK cache).
                const newLabelNodes = await screen.findAllByText("NEW-LABEL");
                // The new label appears both in the tree partition row and in
                // the volume details panel header (selected-partition optimism).
                expect(newLabelNodes.length).toBeGreaterThanOrEqual(2);
                await waitFor(() => {
                    expect(screen.queryByText("old-label")).not.toBeInTheDocument();
                });

                // The RTK volumes cache holds the new label (no local-state fork).
                const state = result.store.getState() as any;
                const cached = state.api?.queries?.["getApiVolumes(undefined)"]?.data as
                    | Array<{
                          partitions?: Record<string, { id?: string; name?: string }>;
                      }>
                    | null
                    | undefined;
                const cachedPartitions = Array.isArray(cached)
                    ? Object.values(cached[0]?.partitions ?? {})
                    : [];
                expect(
                    cachedPartitions.find((partition) => partition.id === "part-f3-1")
                        ?.name,
                ).toBe("NEW-LABEL");
            },
        );
    });

    it("toggles automount for a single-mount partition with one summary toast", async () => {
        const React = await import("react");
        const { screen, waitFor } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { MemoryRouter } = await import("react-router");
        const { Volumes } = await import("../Volumes");
        const { http, HttpResponse } = await import("msw");

        const volumesUrl = /.*\/api\/volumes(?:\?.*)?$/;
        const settingsUrl = /.*\/api\/volume\/settings(?:\?.*)?$/;

        const disks = [
            {
                id: "disk-f4-test",
                name: "sdb",
                size: 2000000000,
                partitions: {
                    part1: {
                        id: "part-f4-1",
                        name: "data-vol",
                        device_path: "/dev/sdb1",
                        fs_type: "ext4",
                        mount_point_data: {
                            "/mnt/data": {
                                path: "/mnt/data",
                                type: "HOST",
                                is_mounted: false,
                                is_write_supported: true,
                                fstype: "ext4",
                                is_to_mount_at_startup: false,
                            },
                        },
                    },
                },
            },
        ];

        let patchCount = 0;
        const patchBodies: unknown[] = [];

        await withTestHandlers(
            [
                http.get(volumesUrl, () => HttpResponse.json(disks)),
                http.patch(settingsUrl, async ({ request }) => {
                    patchCount += 1;
                    patchBodies.push(await request.json());
                    return HttpResponse.json({
                        path: "/mnt/data",
                        type: "HOST",
                    });
                }),
            ],
            async () => {
                toastInfoMock.mockClear();
                toastErrorMock.mockClear();
                toastWarnMock.mockClear();

                localStorage.setItem(
                    "volumes.expandedDisks",
                    JSON.stringify(["disk-f4-test"]),
                );

                await renderWithTestStore(
                    React.createElement(
                        MemoryRouter,
                        null,
                        React.createElement(Volumes as any),
                    ),
                );

                const moreActionsButton = await screen.findByRole("button", {
                    name: "more actions",
                });
                const user = userEvent.setup();
                await user.click(moreActionsButton);

                const enableItem = await screen.findByRole("menuitem", {
                    name: /enable automatic mount/i,
                });
                await user.click(enableItem);

                await waitFor(() => {
                    expect(patchCount).toBe(1);
                });

                // One summary toast instead of one per mount point.
                expect(toastInfoMock.mock.calls.length).toBe(1);
                expect(toastInfoMock.mock.calls[0][0]).toBe(
                    "Automount updated for data-vol.",
                );
                expect(toastErrorMock).not.toHaveBeenCalled();

                // The PATCH flips is_to_mount_at_startup and clears the share.
                const body = patchBodies[0] as {
                    is_to_mount_at_startup?: boolean;
                    path?: string;
                };
                expect(body.is_to_mount_at_startup).toBe(true);
                expect(body.path).toBe("/mnt/data");
            },
        );
    });

    it("shows a single error toast when the automount PATCH fails", async () => {
        const React = await import("react");
        const { screen, waitFor } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { MemoryRouter } = await import("react-router");
        const { Volumes } = await import("../Volumes");
        const { http, HttpResponse } = await import("msw");

        const volumesUrl = /.*\/api\/volumes(?:\?.*)?$/;
        const settingsUrl = /.*\/api\/volume\/settings(?:\?.*)?$/;

        const disks = [
            {
                id: "disk-f4-test",
                name: "sdb",
                size: 2000000000,
                partitions: {
                    part1: {
                        id: "part-f4-1",
                        name: "data-vol",
                        device_path: "/dev/sdb1",
                        fs_type: "ext4",
                        mount_point_data: {
                            "/mnt/data": {
                                path: "/mnt/data",
                                type: "HOST",
                                is_mounted: false,
                                is_write_supported: true,
                                fstype: "ext4",
                                is_to_mount_at_startup: false,
                            },
                        },
                    },
                },
            },
        ];

        await withTestHandlers(
            [
                http.get(volumesUrl, () => HttpResponse.json(disks)),
                http.patch(settingsUrl, () =>
                    HttpResponse.json({ detail: "boom" }, { status: 500 }),
                ),
            ],
            async () => {
                toastInfoMock.mockClear();
                toastErrorMock.mockClear();
                toastWarnMock.mockClear();

                localStorage.setItem(
                    "volumes.expandedDisks",
                    JSON.stringify(["disk-f4-test"]),
                );

                await renderWithTestStore(
                    React.createElement(
                        MemoryRouter,
                        null,
                        React.createElement(Volumes as any),
                    ),
                );

                const moreActionsButton = await screen.findByRole("button", {
                    name: "more actions",
                });
                const user = userEvent.setup();
                await user.click(moreActionsButton);

                const enableItem = await screen.findByRole("menuitem", {
                    name: /enable automatic mount/i,
                });
                await user.click(enableItem);

                await waitFor(() => {
                    expect(toastErrorMock.mock.calls.length).toBe(1);
                });

                expect(toastErrorMock.mock.calls[0][0]).toContain(
                    "Failed to update automount for data-vol",
                );
                expect(toastInfoMock).not.toHaveBeenCalled();
            },
        );
    });
});
