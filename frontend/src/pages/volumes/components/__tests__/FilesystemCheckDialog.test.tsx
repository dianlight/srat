import { describe, expect, it } from "bun:test";
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
});
