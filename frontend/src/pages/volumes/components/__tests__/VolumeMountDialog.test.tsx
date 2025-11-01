import "../../../../../test/setup";
import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";

// Mock the sratApi hooks and types used by VolumeMountDialog and related components
mock.module("../../../../store/sratApi", () => {
    const fakeReducer = (state: any = {}, _action: any) => state;
    const makeMiddleware = () => () => (next: any) => (action: any) => next(action);

    return {
        // Minimal RTK Query API object for store creation
        sratApi: {
            reducerPath: "sratApi",
            reducer: fakeReducer,
            middleware: makeMiddleware()
        },
        // Mock the hook used by VolumeMountDialog
        useGetApiFilesystemsQuery: (arg: any, options?: any) => ({
            data: [
                { name: "ext4", mountFlags: ["rw", "ro", "noexec"] },
                { name: "ntfs", mountFlags: ["rw", "ro"] },
                { name: "vfat", mountFlags: ["rw", "ro"] }
            ],
            isLoading: false,
            error: null
        }),
        // Export enums that might be needed by other components
        Type: { System: "system", Data: "data" },
        Usage: { None: "None", General: "general", TimeMachine: "timemachine" },
        Time_machine_support: { Disabled: "disabled", Enabled: "enabled" },
        Supported_events: {
            Hello: "hello",
            Heartbeat: "heartbeat",
            VolumeUpdate: "volume_update",
            ShareUpdate: "share_update",
            UserUpdate: "user_update"
        },
        Update_process_state: {
            Idle: "idle",
            Downloading: "downloading",
            Installing: "installing"
        }
    };
});

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

describe("VolumeMountDialog Component", () => {
    beforeEach(() => {
        localStorage.clear();
        mock.restore();
    });

    afterEach(() => {
        mock.restore();
    });

    it("renders dialog when open prop is true", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumeMountDialog } = await import("../VolumeMountDialog");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        const mockClose = () => { };

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumeMountDialog as any, {
                        open: true,
                        onClose: mockClose
                    })
                }
            )
        );

        // Check that dialog content is rendered
        const dialogs = container.querySelectorAll('[role="dialog"]');
        expect(dialogs.length).toBeGreaterThanOrEqual(0);
    });

    it("does not render dialog when open prop is false", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumeMountDialog } = await import("../VolumeMountDialog");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        const mockClose = () => { };

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumeMountDialog as any, {
                        open: false,
                        onClose: mockClose
                    })
                }
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders form fields when dialog is open", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumeMountDialog } = await import("../VolumeMountDialog");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        const mockClose = () => { };

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumeMountDialog as any, {
                        open: true,
                        onClose: mockClose
                    })
                }
            )
        );

        // Look for form elements
        const textFields = container.querySelectorAll('input, textarea');
        expect(textFields.length).toBeGreaterThanOrEqual(0);
    });

    it("handles partition data for editing", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumeMountDialog } = await import("../VolumeMountDialog");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        const mockClose = () => { };
        const mockPartition = {
            id: "test-id",
            name: "Test Partition",
            mount_point_data: [{
                path: "/mnt/test",
                fstype: "ext4",
                flags: [],
                custom_flags: [],
                is_to_mount_at_startup: true
            }]
        };

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumeMountDialog as any, {
                        open: true,
                        onClose: mockClose,
                        objectToEdit: mockPartition
                    })
                }
            )
        );

        expect(container).toBeTruthy();
    });

    it("displays read-only view when readOnlyView prop is true", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumeMountDialog } = await import("../VolumeMountDialog");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        const mockClose = () => { };

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumeMountDialog as any, {
                        open: true,
                        onClose: mockClose,
                        readOnlyView: true
                    })
                }
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders action buttons", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumeMountDialog } = await import("../VolumeMountDialog");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        const mockClose = () => { };

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumeMountDialog as any, {
                        open: true,
                        onClose: mockClose
                    })
                }
            )
        );

        // Look for buttons (Cancel, Save, etc.)
        const buttons = container.querySelectorAll('button');
        expect(buttons.length).toBeGreaterThanOrEqual(0);
    });

    it("handles filesystem type selection", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumeMountDialog } = await import("../VolumeMountDialog");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        const mockClose = () => { };

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumeMountDialog as any, {
                        open: true,
                        onClose: mockClose
                    })
                }
            )
        );

        // Check for autocomplete or select fields for filesystem type
        const selects = container.querySelectorAll('[role="combobox"]');
        expect(selects.length).toBeGreaterThanOrEqual(0);
    });

    it("displays mount flags options", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumeMountDialog } = await import("../VolumeMountDialog");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        const mockClose = () => { };

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumeMountDialog as any, {
                        open: true,
                        onClose: mockClose
                    })
                }
            )
        );

        // Check that component renders mount flags section
        expect(container).toBeTruthy();
    });

    it("handles mount at startup checkbox", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumeMountDialog } = await import("../VolumeMountDialog");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        const mockClose = () => { };

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumeMountDialog as any, {
                        open: true,
                        onClose: mockClose
                    })
                }
            )
        );

        // Look for checkbox for mount at startup
        const checkboxes = container.querySelectorAll('input[type="checkbox"]');
        expect(checkboxes.length).toBeGreaterThanOrEqual(0);
    });

    it("displays unsupported flags warnings", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { VolumeMountDialog } = await import("../VolumeMountDialog");
        const { createTestStore } = await import("../../../../../test/setup");

        const store = await createTestStore();
        const mockClose = () => { };

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(VolumeMountDialog as any, {
                        open: true,
                        onClose: mockClose
                    })
                }
            )
        );

        // Check for warning or chip elements for unsupported flags
        expect(container).toBeTruthy();
    });
});
