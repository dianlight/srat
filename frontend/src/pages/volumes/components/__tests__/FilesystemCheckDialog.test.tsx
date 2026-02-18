import { afterEach, beforeEach, describe, expect, it } from "bun:test";
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
    const originalEventSource = (globalThis as any).EventSource;

    beforeEach(() => {
        (globalThis as any).EventSource = class {
            addEventListener() { }
            removeEventListener() { }
            close() { }
        } as any;
    });

    afterEach(() => {
        (globalThis as any).EventSource = originalEventSource;
    });

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
});
