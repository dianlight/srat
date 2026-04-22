import { beforeEach, describe, expect, it } from "bun:test";
import "../../../../../test/setup.ts";

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

async function getOverrideToggleButtons(screen: any) {
    const { within } = await import("@testing-library/react");
    const toggleGroup = await screen.findByRole("group", {
        name: /toggle disk override/i,
    });

    return within(toggleGroup).getAllByRole("button");
}

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
        const toggleButtons = await getOverrideToggleButtons(screen);
        const noBtn = toggleButtons[2];
        await user.click(noBtn);

        const expandBtn = await screen.findByRole("button", { name: /show more/i });
        expect((expandBtn as HTMLButtonElement).disabled).toBe(true);

        // Apply button is not rendered when accordion is collapsed; nothing else to assert here
    });
});
