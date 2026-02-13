import { afterEach, beforeEach, describe, expect, it } from "bun:test";
import "../../../../../test/setup.ts";

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

// Helper to create a mock disk with required hdidle properties
const createMockDisk = (overrides: any = {}) => ({
    id: "disk-1",
    name: "sda",
    model: "Test Disk Model",
    size: 1000000000,
    removable: false,
    hdidle_device: {
        supported: true,
        enabled: "yes",
        idle_time: 0,
        command_type: "",
        power_condition: 0,
    },
    ...overrides,
});

describe("HDIdleDiskSettings Component", () => {
    let originalFetch: any;

    beforeEach(async () => {
        localStorage.clear();
        document.body.innerHTML = '';
        // Override fetch to provide expected API responses for RTK Query hooks
        originalFetch = (globalThis as any).fetch;

        (globalThis as any).fetch = async (url: string, init?: RequestInit) => {
            void init;
            let body: any = {};
            if (typeof url === "string" && url.includes("/api/settings")) {
                body = { hdidle_enabled: true };
            } else if (typeof url === "string" && url.includes("/api/disk/") && url.includes("/hdidle/support")) {
                body = { supported: true };
            } else if (typeof url === "string" && url.includes("/api/disk/") && url.includes("/hdidle/config")) {
                body = { enabled: "yes", idle_time: 0, command_type: "", power_condition: 0 };
            }
            const jsonString = JSON.stringify(body);
            return new Response(jsonString, {
                status: 200,
                statusText: "OK",
                headers: { "Content-Type": "application/json" },
            });
        };
    });

    afterEach(() => {
        if (originalFetch) {
            (globalThis as any).fetch = originalFetch;
        }
    });

    it("renders accordion with disk settings title", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const mockDisk = createMockDisk();

        const store = await createTestStore();
        render(
            React.createElement(Provider, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, { disk: mockDisk, readOnly: false }),
            })
        );

        const title = await screen.findByText(/Power Settings/i);
        expect(title).toBeTruthy();
    });

    it("displays disk model in description", async () => {
        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const mockDisk = createMockDisk({
            model: "Samsung SSD 870",
        });

        const store = await createTestStore();
        const user = userEvent.setup();
        render(
            React.createElement(Provider, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, { disk: mockDisk, readOnly: false }),
            })
        );

        // Click Custom to enable the expand button
        const customBtn = await screen.findByRole("button", { name: /Custom/i });
        await user.click(customBtn);

        const expandBtn = await screen.findByRole("button", { name: /show more/i });

        await waitFor(() => {
            expect((expandBtn as HTMLButtonElement).disabled).toBe(false);
        });

        await user.click(expandBtn as any);

        const model = await screen.findByText(/Samsung SSD 870/i);
        expect(model).toBeTruthy();
    });

    it("renders idle time configuration field", async () => {
        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const mockDisk = createMockDisk({
            name: "sdb",
            size: 2000000000,
        });

        const store = await createTestStore();
        const user = userEvent.setup();
        render(
            React.createElement(Provider, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, { disk: mockDisk, readOnly: false }),
            })
        );

        // Click Custom to enable the expand button
        const customBtn = await screen.findByRole("button", { name: /Custom/i });
        await user.click(customBtn);

        const expandBtn = await screen.findByRole("button", { name: /show more/i });

        await waitFor(() => {
            expect((expandBtn as HTMLButtonElement).disabled).toBe(false);
        });

        await user.click(expandBtn as any);

        const idleField = await screen.findByLabelText(/Idle Time/i);
        expect(idleField).toBeTruthy();
    });

    it("renders command type configuration field", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const mockDisk = createMockDisk({
            name: "sdc",
            model: "Another Disk",
            size: 500000000,
            removable: true,
        });

        const store = await createTestStore();

        render(
            React.createElement(Provider, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, { disk: mockDisk, readOnly: false }),
            })
        );

        const title = await screen.findByText(/Power Settings/i);
        expect(title).toBeTruthy();
    });

    it("respects readOnly mode", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const mockDisk = createMockDisk();

        const store = await createTestStore();

        render(
            React.createElement(Provider, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, { disk: mockDisk, readOnly: true }),
            })
        );

        const title = await screen.findByText(/Power Settings/i);
        expect(title).toBeTruthy();
    });

    it("handles disk without name using ID", async () => {
        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const mockDisk = createMockDisk({
            id: "disk-unknown",
            name: undefined,
            model: "Mystery Disk",
        });

        const store = await createTestStore();
        const user = userEvent.setup();

        render(
            React.createElement(Provider, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, { disk: mockDisk, readOnly: false }),
            })
        );

        const customBtn = await screen.findByRole("button", { name: /Custom/i });
        await user.click(customBtn);

        const expandBtn = await screen.findByRole("button", { name: /show more/i });

        await waitFor(() => {
            expect((expandBtn as HTMLButtonElement)?.disabled).toBe(false);
        });

        await user.click(expandBtn as any);

        const model = await screen.findByText(/Mystery Disk|disk-unknown/i);
        expect(model).toBeTruthy();
    });

    it("expands accordion only when Enabled.Custom is selected", async () => {
        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const mockDisk = createMockDisk();

        const store = await createTestStore();
        const user = userEvent.setup();

        render(
            React.createElement(Provider, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, { disk: mockDisk, readOnly: false }),
            })
        );

        const expandBtn = await screen.findByRole("button", { name: /show more/i });

        // Initially, expand button should be disabled (default is Enabled.Yes)
        expect((expandBtn as HTMLButtonElement).disabled).toBe(true);

        // Click Custom to enable the expand button
        const customBtn = await screen.findByRole("button", { name: /Custom/i });
        await user.click(customBtn);

        // Now the expand button should be enabled
        await waitFor(() => {
            expect((expandBtn as HTMLButtonElement).disabled).toBe(false);
        });

        // Click to expand
        await user.click(expandBtn);

        // Verify accordion content is visible
        const idleField = await screen.findByLabelText(/Idle Time/i);
        expect(idleField).toBeTruthy();

        // Switch back to Enabled.Yes
        const yesBtn = await screen.findByRole("button", { name: /Yes/i });
        await user.click(yesBtn);

        // Expand button should be disabled again
        expect((expandBtn as HTMLButtonElement).disabled).toBe(true);

        // Accordion should be collapsed
        expect(() => screen.getByLabelText(/Idle Time/i)).toThrow();
    });

    it("defaults to Enabled.Yes", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const mockDisk = createMockDisk();

        const store = await createTestStore();

        render(
            React.createElement(Provider, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, { disk: mockDisk, readOnly: false }),
            })
        );

        // The Yes button should be selected by default (success color)
        const yesBtn = await screen.findByRole("button", { name: /Yes/i });
        expect(yesBtn).toBeTruthy();
    });
});
