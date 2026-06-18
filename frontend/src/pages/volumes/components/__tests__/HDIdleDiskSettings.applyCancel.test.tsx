import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

// localStorage shim
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

vi.mock("../../../../hooks/useLabMode", () => ({
    useLabMode: () => ({ labMode: true, isLoading: false }),
}));

const createMockDisk = (overrides: any = {}) => ({
    id: "disk-1",
    legacy_device_name: "sda",
    model: "Test Disk Model",
    size: 1000000000,
    removable: false,
    is_rotational: true,
    hdidle_device: {
        supported: true,
        // lowercase — matches the Enabled enum value (was "Yes" before; the
        // capitalised string did not match Enabled.Yes="yes" and broke
        // togglegroup selection).
        enabled: "yes",
        idle_time: 0,
        command_type: "",
        power_condition: 0,
        force_enabled: false,
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

describe("HDIdleDiskSettings Apply/Cancel", () => {
    let originalFetch: any;

    beforeEach(() => {
        localStorage.clear();
        document.body.innerHTML = "";
        originalFetch = (globalThis as any).fetch;
        // Minimal fetch stub so any background RTK Query call returns sane JSON.
        (globalThis as any).fetch = async () =>
            new Response(JSON.stringify({}), { status: 200 });
    });

    afterEach(() => {
        if (originalFetch !== undefined) (globalThis as any).fetch = originalFetch;
    });

    test("disables the expand button when Enabled.No is selected", async () => {
        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("/test/testing");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const store = await createTestStore();
        const user = userEvent.setup();
        const disk = createMockDisk({
            hdidle_device: {
                supported: true,
                enabled: "custom",
                idle_time: 0,
                command_type: "",
                power_condition: 0,
                force_enabled: false,
            },
        });

        render(
            React.createElement(Provider as any, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, {
                    disk,
                    readOnly: false,
                }),
            }),
        );

        const expandBtn = await screen.findByRole("button", { name: /show more/i });

        // Initial state is Custom → expand button enabled.
        await waitFor(() => {
            expect((expandBtn as HTMLButtonElement).disabled).toBe(false);
        });

        // Switch to No.
        const toggleButtons = await getOverrideToggleButtons(screen);
        const noBtn = toggleButtons[2];
        await user.click(noBtn);

        await waitFor(() => {
            expect(noBtn.getAttribute("aria-pressed")).toBe("true");
        });
        await waitFor(() => {
            expect((expandBtn as HTMLButtonElement).disabled).toBe(true);
        });
    });

    test("Apply button is enabled when switching to Enabled.No (form is dirty)", async () => {
        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("/test/testing");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const store = await createTestStore();
        const user = userEvent.setup();
        // Start with Custom so the expand button is enabled and the Collapse section
        // (which contains Apply) can be opened before we switch to No.
        const disk = createMockDisk({
            hdidle_device: {
                supported: true,
                enabled: "custom",
                idle_time: 0,
                command_type: "",
                power_condition: 0,
                force_enabled: false,
            },
        });

        render(
            React.createElement(Provider as any, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, {
                    disk,
                    readOnly: false,
                }),
            }),
        );

        // Expand the collapse section to make Apply visible.
        const expandBtn = await screen.findByRole("button", { name: /show more/i });
        await user.click(expandBtn);

        // Wait for Apply to appear inside the expanded collapse, then capture it
        // before switching to No (which auto-collapses the section).
        const applyBtn = await screen.findByRole("button", { name: /apply/i });

        // Switch to No — form becomes dirty.
        const toggleButtons = await getOverrideToggleButtons(screen);
        const noBtn = toggleButtons[2];
        await user.click(noBtn);

        await waitFor(() => {
            expect(noBtn.getAttribute("aria-pressed")).toBe("true");
        });

        // Apply must be enabled so the user can persist the "disable" change.
        // (Prior to the fix, fieldsDisabled included Enabled.No and blocked Apply.)
        await waitFor(() => {
            expect((applyBtn as HTMLButtonElement).disabled).toBe(false);
        });
    });
});
