import { describe, expect, it } from "bun:test";
import { http, HttpResponse } from "msw";
import { getMswServer } from "../../../../../test/bun-setup";
import "../../../../../test/setup";

async function renderWithProviders(element: any, options?: { seedStore?: (store: any) => void }) {
    const React = await import("react");
    const { render } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { createTestStore } = await import("../../../../../test/setup");

    const store = await createTestStore();
    if (options?.seedStore) {
        options.seedStore(store);
    }

    const result = render(React.createElement(Provider, { store, children: element }));
    return { ...result, store };
}

describe("FilesystemCheckDialog", () => {
    it("shows switches and logs area when verbose is enabled", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const { FilesystemCheckDialog } = await import("../FilesystemCheckDialog");

        const partition = {
            id: "part-1",
            name: "data",
            device_path: "/dev/sdb1",
        };

        await renderWithProviders(
            React.createElement(FilesystemCheckDialog as any, {
                open: true,
                partition,
                initialVerbose: true,
                onClose: () => { },
            }),
        );

        const logsField = await screen.findByRole("textbox");
        expect(logsField).toBeTruthy();
    });

    it("renders progress and logs from filesystem_task events", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const { FilesystemCheckDialog } = await import("../FilesystemCheckDialog");

        const partition = {
            id: "part-1",
            name: "data",
            device_path: "/dev/sdb1",
        };

        await renderWithProviders(
            React.createElement(FilesystemCheckDialog as any, {
                open: true,
                partition,
                initialVerbose: true,
                taskOverride: {
                    device: "/dev/sdb1",
                    operation: "check",
                    status: "running",
                    progress: 42,
                    notes: ["Checking filesystem"],
                },
                onClose: () => { },
            }),
        );

        const logsField = await screen.findByRole("textbox");
        expect((logsField as HTMLInputElement).value).toContain("Checking filesystem");
        const progressBar = await screen.findByRole("progressbar");
        expect(progressBar).toBeTruthy();
    });

    it("shows running command output in logs when notes are not provided", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const { FilesystemCheckDialog } = await import("../FilesystemCheckDialog");

        const partition = {
            id: "part-2",
            name: "data2",
            device_path: "/dev/sde1",
        };

        await renderWithProviders(
            React.createElement(FilesystemCheckDialog as any, {
                open: true,
                partition,
                initialVerbose: true,
                taskOverride: {
                    device: "/dev/sde1",
                    operation: "check",
                    status: "running",
                    progress: 999,
                    message: "fsck.ext4: checking inode tables",
                },
                onClose: () => { },
            }),
        );

        const logsField = await screen.findByRole("textbox");
        expect((logsField as HTMLInputElement).value).toContain(
            "fsck.ext4: checking inode tables",
        );
    });

    it("shows completion details in logs when a check succeeds without notes", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const { FilesystemCheckDialog } = await import("../FilesystemCheckDialog");

        const partition = {
            id: "part-success",
            name: "success-data",
            device_path: "/dev/sdf1",
        };

        await renderWithProviders(
            React.createElement(FilesystemCheckDialog as any, {
                open: true,
                partition,
                initialVerbose: true,
                taskOverride: {
                    device: "/dev/sdf1",
                    operation: "check",
                    status: "success",
                    progress: 100,
                    message: "Filesystem check completed successfully for /dev/sdf1",
                    result: {
                        Message: "fsck.ext4: clean, 12/1024 files, 128/8192 blocks",
                    },
                },
                onClose: () => { },
            }),
        );

        const logsField = await screen.findByRole("textbox");
        expect((logsField as HTMLInputElement).value).toContain(
            "Filesystem check completed successfully for /dev/sdf1",
        );
        expect((logsField as HTMLInputElement).value).toContain(
            "fsck.ext4: clean, 12/1024 files, 128/8192 blocks",
        );
    });

    it("shows stderr details in logs when a check fails", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const { FilesystemCheckDialog } = await import("../FilesystemCheckDialog");

        const partition = {
            id: "part-failure",
            name: "failure-data",
            device_path: "/dev/sdz9",
        };

        await renderWithProviders(
            React.createElement(FilesystemCheckDialog as any, {
                open: true,
                partition,
                initialVerbose: true,
                taskOverride: {
                    device: "/dev/sdz9",
                    operation: "check",
                    status: "failure",
                    progress: 100,
                    message: "Check operation failed for /dev/sdz9",
                    error: "fsck.ext4: Superblock invalid on /dev/sdz9",
                },
                onClose: () => { },
            }),
        );

        const logsField = await screen.findByRole("textbox");
        expect((logsField as HTMLInputElement).value).toContain(
            "fsck.ext4: Superblock invalid on /dev/sdz9",
        );
    });

    it("keeps the reported success state when the selected partition object refreshes", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { FilesystemCheckDialog } = await import("../FilesystemCheckDialog");

        const taskOverride = {
            device: "/dev/sdg1",
            operation: "check",
            status: "success",
            progress: 100,
            message: "Filesystem check completed successfully for /dev/sdg1",
        };

        const partition = {
            id: "part-refresh",
            name: "refresh-data",
            device_path: "/dev/sdg1",
        };

        const { rerender, store } = await renderWithProviders(
            React.createElement(FilesystemCheckDialog as any, {
                open: true,
                partition,
                initialVerbose: true,
                taskOverride,
                onClose: () => { },
            }),
        );

        const initialLogsField = await screen.findByRole("textbox");
        expect((initialLogsField as HTMLInputElement).value).toContain(
            "Filesystem check completed successfully for /dev/sdg1",
        );

        rerender(
            React.createElement(Provider, {
                store,
                children: React.createElement(FilesystemCheckDialog as any, {
                    open: true,
                    partition: { ...partition },
                    initialVerbose: true,
                    taskOverride,
                    onClose: () => { },
                }),
            }),
        );

        const refreshedLogsField = await screen.findByRole("textbox");
        expect((refreshedLogsField as HTMLInputElement).value).toContain(
            "Filesystem check completed successfully for /dev/sdg1",
        );
        expect(await screen.findByText("SUCCESS")).toBeTruthy();
    });

    it("disables start and shows unsupported hint when check is not available", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const { FilesystemCheckDialog } = await import("../FilesystemCheckDialog");

        const partition = {
            id: "part-unsupported",
            name: "archive",
            device_path: "/dev/sdc1",
            filesystem_info: {
                support: {
                    canCheck: false,
                },
            },
        };

        await renderWithProviders(
            React.createElement(FilesystemCheckDialog as any, {
                open: true,
                partition,
                onClose: () => { },
            }),
        );

        const startButton = await screen.findByRole("button", { name: /start/i });
        expect((startButton as HTMLButtonElement).disabled).toBe(true);

        const hint = await screen.findByText(/Check tools are not available/i);
        expect(hint).toBeTruthy();
    });

    it("shows indeterminate progress note when progress is unsupported", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const { FilesystemCheckDialog } = await import("../FilesystemCheckDialog");

        const partition = {
            id: "part-1",
            name: "data",
            device_path: "/dev/sdb1",
        };

        await renderWithProviders(
            React.createElement(FilesystemCheckDialog as any, {
                open: true,
                partition,
                taskOverride: {
                    device: "/dev/sdb1",
                    operation: "check",
                    status: "running",
                    progress: 999,
                },
                onClose: () => { },
            }),
        );

        const explanation = await screen.findByText(/does not report incremental progress/i);
        expect(explanation).toBeTruthy();
    });

    it("does not animate progress when check is idle", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const { FilesystemCheckDialog } = await import("../FilesystemCheckDialog");

        const partition = {
            id: "part-idle",
            name: "idle-data",
            device_path: "/dev/sdd1",
        };

        await renderWithProviders(
            React.createElement(FilesystemCheckDialog as any, {
                open: true,
                partition,
                taskOverride: {
                    device: "/dev/sdd1",
                    operation: "check",
                    status: "idle",
                    progress: 0,
                },
                onClose: () => { },
            }),
        );

        expect(screen.queryByText(/Working.../i)).toBeNull();
        const zeroPercent = await screen.findByText("0%");
        expect(zeroPercent).toBeTruthy();
    });

    it("shows unsupported check hint for zfs support profile from preflight", async () => {
        const React = await import("react");
        const { screen } = await import("@testing-library/react");
        const { FilesystemCheckDialog } = await import("../FilesystemCheckDialog");

        const server = await getMswServer();
        server.use(
            http.get("/api/filesystem/support", () =>
                HttpResponse.json({
                    canMount: true,
                    canFormat: false,
                    canCheck: false,
                    canSetLabel: false,
                    canGetState: true,
                    alpinePackage: "zfs",
                    missingTools: ["zpool"],
                }),
            ),
        );

        const partition = {
            id: "part-zfs",
            name: "tank",
            device_path: "/dev/sdz1",
            fs_type: "zfs",
            filesystem_info: {
                support: {
                    canCheck: true,
                },
            },
        };

        await renderWithProviders(
            React.createElement(FilesystemCheckDialog as any, {
                open: true,
                partition,
                onClose: () => { },
            }),
        );

        const startButton = await screen.findByRole("button", { name: /start/i });
        expect((startButton as HTMLButtonElement).disabled).toBe(true);

        const hints = await screen.findAllByText(/Check tools are not available/i);
        expect(hints.length).toBeGreaterThan(0);
        const missingTools = await screen.findAllByText(/Missing tools: zpool/i);
        expect(missingTools.length).toBeGreaterThan(0);
        const installHints = await screen.findAllByText(/apk add zfs/i);
        expect(installHints.length).toBeGreaterThan(0);
    });
});
