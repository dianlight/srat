import { createTheme, ThemeProvider } from "@mui/material/styles";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { Provider } from "react-redux";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createTestStore, getMswServer } from "/test/testing";

const { wsStateRef } = vi.hoisted(() => {
  const wsStateRef = {
    current: {
      hello: {
        read_only: false,
        build_version: "2026.1.0",
        message: "mock",
      },
    } as Record<string, unknown>,
  };
  return { wsStateRef };
});

vi.mock("../../../store/wsApi", () => ({
  wsApi: {
    reducerPath: "wsApi",
    reducer: () => ({}),
    util: {
      resetApiState: () => ({ type: "wsApi/resetApiState" }),
    },
    middleware: () => (next: (action: unknown) => unknown) => (action: unknown) => next(action),
  },
  useGetServerEventsQuery: () => ({
    data: wsStateRef.current,
    isLoading: false,
    error: undefined,
  }),
}));

import { Settings } from "../Settings";

async function renderSettings() {
  const store = await createTestStore();
  const theme = createTheme();
  const user = userEvent.setup();
  render(
    <Provider store={store}>
      <ThemeProvider theme={theme}>
        <Settings />
      </ThemeProvider>
    </Provider>,
  );
  await screen.findByPlaceholderText("Search settings...");
  return { user, store };
}

describe("Settings read_only gating", () => {
  beforeEach(() => {
    localStorage.clear();
    wsStateRef.current = {
      hello: { read_only: false, build_version: "2026.1.0", message: "mock" },
    } as unknown as Record<string, unknown>;
  });

  afterEach(() => {
    cleanup();
    wsStateRef.current = {
      hello: { read_only: false, build_version: "2026.1.0", message: "mock" },
    } as unknown as Record<string, unknown>;
  });

  it("enables controls when read_only is false", async () => {
    const server = getMswServer();
    server.use(
      http.get(/.*\/api\/settings$/, () =>
        HttpResponse.json({
          hostname: "homeassistant",
          workgroup: "WORKGROUP",
          telemetry_mode: "Disabled",
          bind_all_interfaces: true,
          interfaces: [],
        }),
      ),
    );
    await renderSettings();

    const hostInputs = screen.getAllByLabelText(/hostname/i);
    expect(hostInputs[0]).toBeEnabled();
    const wgInputs = screen.getAllByLabelText(/workgroup/i);
    expect(wgInputs[0]).toBeEnabled();
    // switches should be enabled (check one)
    expect(screen.getByRole("switch", { name: /local master/i })).toBeEnabled();
    expect(screen.getByRole("button", { name: /^reset$/i })).toBeDisabled(); // dirty still disabled
    expect(screen.getByRole("button", { name: /^apply$/i })).toBeDisabled();
  });

  it("disables all controls when read_only is true and hides dirty", async () => {
    wsStateRef.current = {
      hello: { read_only: true, build_version: "2026.1.0", message: "mock" },
    } as unknown as Record<string, unknown>;

    const server = getMswServer();
    server.use(
      http.get(/.*\/api\/settings$/, () =>
        HttpResponse.json({
          hostname: "homeassistant",
          workgroup: "WORKGROUP",
          telemetry_mode: "Disabled",
          bind_all_interfaces: true,
          interfaces: [],
        }),
      ),
    );

    await renderSettings();

    const hostInputs = screen.getAllByLabelText(/hostname/i);
    expect(hostInputs[0]).toBeDisabled();
    const wgInputs = screen.getAllByLabelText(/workgroup/i);
    expect(wgInputs[0]).toBeDisabled();
    expect(screen.getByRole("switch", { name: /local master/i })).toBeDisabled();
    expect(screen.getByRole("switch", { name: /experimental lab mode/i })).toBeDisabled();

    // bottom bar stays disabled even if we try to mutate (form disabled prevents dirty)
    expect(screen.getByRole("button", { name: /^reset$/i })).toBeDisabled();
    expect(screen.getByRole("button", { name: /^apply$/i })).toBeDisabled();

    // Verify Devices panel also disabled (match Settings.test.tsx helper to avoid treeitem name mismatch)
    const user = userEvent.setup();
    const treeItems = await screen.findAllByRole("treeitem");
    const devicesLabel = treeItems
      .map((item) => within(item).queryByText("Devices"))
      .find((el) => el != null);
    expect(devicesLabel).toBeTruthy();
    await user.click(devicesLabel as HTMLElement);
    expect(await screen.findByRole("heading", { name: /^devices$/i })).toBeInTheDocument();
    // bind all interfaces checkbox may be rendered as checkbox, not switch - check both
    const bindAll = screen.queryByLabelText(/bind all interfaces/i) ?? screen.queryByRole("checkbox", { name: /bind all interfaces/i });
    expect(bindAll).toBeDisabled();
  });

  it("re-enables after read_only toggled back to false (regression for suggestion)", async () => {
    wsStateRef.current = {
      hello: { read_only: true, build_version: "2026.1.0", message: "mock" },
    } as unknown as Record<string, unknown>;
    const server = getMswServer();
    server.use(
      http.get(/.*\/api\/settings$/, () =>
        HttpResponse.json({
          hostname: "homeassistant",
          workgroup: "WORKGROUP",
          telemetry_mode: "Disabled",
          bind_all_interfaces: true,
          interfaces: [],
        }),
      ),
    );

    const { store } = await renderSettings();
    expect(screen.getAllByLabelText(/hostname/i)[0]).toBeDisabled();

    // toggle back
    wsStateRef.current = {
      hello: { read_only: false, build_version: "2026.1.0", message: "mock" },
    } as unknown as Record<string, unknown>;
    // force re-render by re-mounting Settings (simplest: cleanup and re-render)
    cleanup();
    const theme = createTheme();
    render(
      <Provider store={store}>
        <ThemeProvider theme={theme}>
          <Settings />
        </ThemeProvider>
      </Provider>,
    );
    await screen.findByPlaceholderText("Search settings...");
    expect(screen.getAllByLabelText(/hostname/i)[0]).toBeEnabled();
  });
});
