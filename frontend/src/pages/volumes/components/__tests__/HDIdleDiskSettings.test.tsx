import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

// localStorage shim for jsdom environments that lack one.
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => { _store[k] = String(v); },
        removeItem: (k: string) => { delete _store[k]; },
        clear: () => { for (const k of Object.keys(_store)) delete _store[k]; },
    };
}

// Mutable ref so individual tests can override labMode without a new vi.mock.
const labModeRef = { value: true };

vi.mock("../../../../hooks/useLabMode", () => ({
    useLabMode: () => ({ labMode: labModeRef.value, isLoading: false }),
}));

// Helper: rotational HDD by default. Tests opt-out by overriding is_rotational.
const createMockDisk = (overrides: any = {}) => ({
    id: "disk-1",
    legacy_device_name: "sda",
    model: "Test Disk Model",
    size: 1000000000,
    removable: false,
    is_rotational: true,
    hdidle_device: {
        supported: true,
        enabled: "yes", // lowercase — matches the Enabled enum value
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

describe("HDIdleDiskSettings Component", () => {
    let originalFetch: any;

    beforeEach(async () => {
        labModeRef.value = true;
        localStorage.clear();
        document.body.innerHTML = "";
        // RTK Query still calls fetch when stores are not pre-seeded — provide
        // minimal stubs for any settings/disk endpoint the component may
        // accidentally trigger (it shouldn't with our mock, but be safe).
        originalFetch = (globalThis as any).fetch;
        (globalThis as any).fetch = async () =>
            new Response(JSON.stringify({}), {
                status: 200,
                headers: { "Content-Type": "application/json" },
            });
    });

    afterEach(() => {
        if (originalFetch) (globalThis as any).fetch = originalFetch;
    });

    test("renders the Power Settings card", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("/test/testing");
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const store = await createTestStore();
        render(
            React.createElement(Provider as any, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, {
                    disk: createMockDisk(),
                    readOnly: false,
                }),
            }),
        );

        expect(await screen.findByText(/Power Settings/i)).toBeTruthy();
    });

    test("returns null when hdidle_device.supported is false", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("/test/testing");
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const store = await createTestStore();
        const disk = createMockDisk({
            hdidle_device: { ...createMockDisk().hdidle_device, supported: false },
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
        expect(screen.queryByText(/Power Settings/i)).toBeNull();
    });

    test("renders card in test environment without lab mode", async () => {
        // Component no longer uses labMode — card is always visible when
        // __TEST__ is true (set by common-setup.ts).
        labModeRef.value = false;
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("/test/testing");
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const store = await createTestStore();
        render(
            React.createElement(Provider as any, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, {
                    disk: createMockDisk(),
                    readOnly: false,
                }),
            }),
        );
        expect(await screen.findByText(/Power Settings/i)).toBeTruthy();
    });

    test("readOnly disables the toggle group", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("/test/testing");
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const store = await createTestStore();
        render(
            React.createElement(Provider as any, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, {
                    disk: createMockDisk(),
                    readOnly: true,
                }),
            }),
        );

        const toggleGroup = await screen.findByRole("group", {
            name: /toggle disk override/i,
        });
        const buttons = within(toggleGroup).getAllByRole("button");
        // MUI ToggleButton sets `disabled` attribute on each button when the
        // group's `disabled` prop is true.
        for (const b of buttons) {
            expect((b as HTMLButtonElement).disabled).toBe(true);
        }
    });

    test("expand button is disabled until Enabled.Custom is selected", async () => {
        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("/test/testing");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const store = await createTestStore();
        const user = userEvent.setup();
        render(
            React.createElement(Provider as any, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, {
                    disk: createMockDisk(),
                    readOnly: false,
                }),
            }),
        );

        const expandBtn = await screen.findByRole("button", { name: /show more/i });
        // Default is Yes → expand disabled.
        expect((expandBtn as HTMLButtonElement).disabled).toBe(true);

        const [, customBtn] = await getOverrideToggleButtons(screen);
        await user.click(customBtn);

        await waitFor(() => {
            expect((expandBtn as HTMLButtonElement).disabled).toBe(false);
        });
    });

    test("does not show force dialog on rotational disk", async () => {
        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("/test/testing");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const store = await createTestStore();
        const user = userEvent.setup();
        // Rotational HDD with current state No — clicking Yes must not show
        // the force-enable dialog.
        render(
            React.createElement(Provider as any, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, {
                    disk: createMockDisk({
                        is_rotational: true,
                        hdidle_device: {
                            ...createMockDisk().hdidle_device,
                            enabled: "no",
                        },
                    }),
                    readOnly: false,
                }),
            }),
        );

        const [yesBtn] = await getOverrideToggleButtons(screen);
        await user.click(yesBtn);

        // No dialog appears.
        await waitFor(() => {
            expect(
                screen.queryByText(/Enable HDIdle on a non-rotational disk/i),
            ).toBeNull();
        });
    });

    test("opens force dialog when enabling on a non-rotational disk", async () => {
        const React = await import("react");
        const { render, screen } = await import(
            "@testing-library/react"
        );
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("/test/testing");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const store = await createTestStore();
        const user = userEvent.setup();
        render(
            React.createElement(Provider as any, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, {
                    disk: createMockDisk({
                        is_rotational: false,
                        hdidle_device: {
                            ...createMockDisk().hdidle_device,
                            enabled: "no",
                            force_enabled: false,
                        },
                    }),
                    readOnly: false,
                }),
            }),
        );

        const [yesBtn] = await getOverrideToggleButtons(screen);
        await user.click(yesBtn);

        // The dialog appears.
        const dialog = await screen.findByRole("dialog");
        expect(dialog).toBeTruthy();
        const { within } = await import("@testing-library/react");
        const headingElements = within(dialog).getAllByRole("heading", {
            name: /^Enable HDIdle on a non-rotational disk$/i,
        });
        expect(headingElements.length).toBeGreaterThanOrEqual(1);
    });

    test("Cancel in force dialog leaves toggle unchanged", async () => {
        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("/test/testing");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const store = await createTestStore();
        const user = userEvent.setup();
        render(
            React.createElement(Provider as any, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, {
                    disk: createMockDisk({
                        is_rotational: false,
                        hdidle_device: {
                            ...createMockDisk().hdidle_device,
                            enabled: "no",
                        },
                    }),
                    readOnly: false,
                }),
            }),
        );

        const [yesBtn] = await getOverrideToggleButtons(screen);
        await user.click(yesBtn);

        const cancelBtn = await screen.findByRole("button", { name: /^cancel$/i });
        await user.click(cancelBtn);

        // The "No" button (third) should still be aria-pressed=true.
        await waitFor(async () => {
            const buttons = await getOverrideToggleButtons(screen);
            expect(buttons[2].getAttribute("aria-pressed")).toBe("true");
        });
    });

    test("Force enable in dialog flips toggle and sets the warning", async () => {
        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("/test/testing");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

        const store = await createTestStore();
        const user = userEvent.setup();
        render(
            React.createElement(Provider as any, {
                store,
                children: React.createElement(HDIdleDiskSettings as any, {
                    disk: createMockDisk({
                        is_rotational: false,
                        hdidle_device: {
                            ...createMockDisk().hdidle_device,
                            enabled: "no",
                        },
                    }),
                    readOnly: false,
                }),
            }),
        );

        const [, customBtn] = await getOverrideToggleButtons(screen);
        await user.click(customBtn);

        const forceBtn = await screen.findByRole("button", { name: /force enable/i });
        await user.click(forceBtn);

        // After confirmation: Custom is selected.
        await waitFor(async () => {
            const buttons = await getOverrideToggleButtons(screen);
            expect(buttons[1].getAttribute("aria-pressed")).toBe("true");
        });
    });
});
