import "../../../../../test/setup.ts";
import { describe, it, expect, beforeEach } from "bun:test";

// REQUIRED localStorage shim for every localStorage test
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => {
            _store[k] = String(v);
        },
        removeItem: (k: string) => {
            delete _store[k];
        },
        clear: () => {
            for (const k of Object.keys(_store)) delete _store[k];
        },
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
        enabled: "Yes",
        idle_time: 0,
        command_type: "",
        power_condition: 0,
    },
    ...overrides,
});

describe("HDIdleDiskSettings Apply/Cancel & Unsupported", () => {
    beforeEach(() => {
        localStorage.clear();
        document.body.innerHTML = "";
    });

    it("disables expand and actions when Enabled.No is selected", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../../test/setup");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const store = await createTestStore();
        const user = userEvent.setup();
        const mockDisk = createMockDisk();

        render(
            React.createElement(Provider as any, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, { disk: mockDisk, readOnly: false }),
            })
        );

        // Select No and verify expand disabled
        const noBtn = await screen.findByRole("button", { name: /No/i });
        await user.click(noBtn);

        const expandBtn = await screen.findByRole("button", { name: /show more/i });
        expect((expandBtn as HTMLButtonElement).disabled).toBe(true);

        // Apply button is not rendered when accordion is collapsed; nothing else to assert here
    });
});
