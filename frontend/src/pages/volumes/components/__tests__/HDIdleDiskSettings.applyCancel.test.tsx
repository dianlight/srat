/* eslint-disable */
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

const { saveConfigMock } = vi.hoisted(() => ({
  saveConfigMock: vi.fn(),
}));

vi.mock("../../../../store/sratApi", async () => {
  const actual = await vi.importActual<
    typeof import("../../../../store/sratApi")
  >("../../../../store/sratApi");
  return {
    ...actual,
    usePutApiDiskByDiskIdHdidleConfigMutation: () => [
      saveConfigMock,
      { isLoading: false },
    ],
  };
});

const createMockDisk = (overrides: any = {}) => ({
  id: "disk-1",
  legacy_device_name: "sda",
  model: "Test Disk Model",
  size: 1000000000,
  removable: false,
  is_rotational: true,
  hdidle_device: {
    supported: true,
    // lowercase — matches the Enabled enum value. Use the "custom" sentinel
    // so the mock stays a valid Enabled value; tests override as needed.
    enabled: "custom",
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
    saveConfigMock.mockReset();
    saveConfigMock.mockReturnValue({ unwrap: () => Promise.resolve({}) });
  });

  afterEach(() => {
    if (originalFetch !== undefined) (globalThis as any).fetch = originalFetch;
  });

  test("expand button stays enabled when Enabled.No is selected", async () => {
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
    const noBtn = toggleButtons[1];
    await user.click(noBtn);

    await waitFor(() => {
      expect(noBtn.getAttribute("aria-pressed")).toBe("true");
    });
    // Expand button stays enabled even when No is selected,
    // so the user can still inspect the settings.
    await waitFor(() => {
      expect((expandBtn as HTMLButtonElement).disabled).toBe(false);
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

    // Custom auto-opens the Collapse section, so Apply is already visible.
    const applyBtn = await screen.findByRole("button", { name: /apply/i });

    // Switch to No — form becomes dirty.
    const toggleButtons = await getOverrideToggleButtons(screen);
    const noBtn = toggleButtons[1];
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

  test("auto-saves and invalidates the volumes cache when toggling hd idle off", async () => {
    const React = await import("react");
    const { render, screen, waitFor } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { createTestStore } = await import("/test/testing");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

    const store = await createTestStore();
    const dispatchSpy = vi.spyOn(store, "dispatch");
    const user = userEvent.setup();
    const disk = createMockDisk();

    render(
      React.createElement(Provider as any, {
        store,
        children: React.createElement(HDIdleDiskSettings as any, {
          disk,
          readOnly: false,
        }),
      }),
    );

    const toggleButtons = await getOverrideToggleButtons(screen);
    const noBtn = toggleButtons[1];
    await user.click(noBtn);

    await waitFor(() => {
      expect(saveConfigMock).toHaveBeenCalledTimes(1);
    });
    expect(saveConfigMock).toHaveBeenCalledWith(
      expect.objectContaining({ diskId: "disk-1" }),
    );
    // Saving must refresh cached volume data so the dashboard reflects
    // the new HDIdle on/off state without a full page reload.
    await waitFor(() => {
      expect(dispatchSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          type: expect.stringContaining("invalidateTags"),
        }),
      );
    });
  });

  test("auto-saves when enabling hd idle via the toggle", async () => {
    const React = await import("react");
    const { render, screen, waitFor } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { createTestStore } = await import("/test/testing");
    const userEvent = (await import("@testing-library/user-event")).default;
    const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");

    const store = await createTestStore();
    const user = userEvent.setup();
    const disk = createMockDisk({
      hdidle_device: { ...createMockDisk().hdidle_device, enabled: "no" },
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

    const toggleButtons = await getOverrideToggleButtons(screen);
    const customBtn = toggleButtons[0];
    await user.click(customBtn);

    await waitFor(() => {
      expect(saveConfigMock).toHaveBeenCalledTimes(1);
    });
    expect(saveConfigMock).toHaveBeenCalledWith(
      expect.objectContaining({
        diskId: "disk-1",
        hdIdleDevice: expect.objectContaining({ enabled: "custom" }),
      }),
    );
  });
});
