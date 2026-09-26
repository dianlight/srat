/* eslint-disable */
import { render, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";
import { Provider } from "react-redux";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createTestStore, getMswServer } from "/test/testing";

const { wsStateRef } = vi.hoisted(() => {
  const wsStateRef = {
    current: {
      heartbeat: { alive: true },
    } as Record<string, unknown>,
  };
  return { wsStateRef };
});

vi.mock("../store/wsApi", () => ({
  wsApi: {
    reducerPath: "wsApi",
    reducer: () => ({}),
    util: {
      resetApiState: () => ({ type: "wsApi/resetApiState" }),
    },
    middleware: () => (next: (action: unknown) => unknown) => (action: unknown) =>
      next(action),
  },
  useGetServerEventsQuery: () => ({
    data: wsStateRef.current,
    isLoading: false,
    error: undefined,
  }),
}));

vi.mock("react-toastify", () => ({
  Slide: undefined,
  ToastContainer: () => null,
  toast: {
    error: (..._args: unknown[]) => undefined,
    info: (..._args: unknown[]) => undefined,
    success: (..._args: unknown[]) => undefined,
    warn: (..._args: unknown[]) => undefined,
    warning: (..._args: unknown[]) => undefined,
  },
}));

vi.mock("../hooks/useTelemetryModal", () => ({
  useTelemetryModal: () => ({ shouldShow: false, dismiss: () => undefined }),
}));

vi.mock("../hooks/useBaseConfigModal", () => ({
  useBaseConfigModal: () => ({ shouldShow: false, dismiss: () => undefined }),
}));

vi.mock("../hooks/useSetupWizard", () => ({
  useSetupWizard: () => ({ shouldShow: false, dismiss: () => undefined }),
}));

vi.mock("../components/NavBar", () => ({
  NavBar: () => <div data-testid="mock-navbar">NavBar</div>,
}));

vi.mock("../components/GlobalEventTracker", () => ({
  __esModule: true,
  default: () => <div data-testid="mock-event-monitor">EventMonitor</div>,
  useSystemLogs: () => ({ logs: [], clearLogs: () => undefined }),
}));

vi.mock("../components/BaseConfigModal", () => ({
  default: () => null,
}));

vi.mock("../components/Footer", () => ({
  Footer: () => <div data-testid="mock-footer">Footer</div>,
}));

vi.mock("../components/wizard/SetupWizard", () => ({
  SetupWizard: () => null,
  WizardOpenContext: React.createContext<() => void>(() => undefined),
}));

describe("App backdrop soft reconnect", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  beforeEach(async () => {
    vi.restoreAllMocks();
    wsStateRef.current = { heartbeat: { alive: true } };
    if (localStorage && typeof localStorage.clear === "function") {
      localStorage.clear();
    }
    if (sessionStorage && typeof sessionStorage.clear === "function") {
      sessionStorage.clear();
    }

    const server = getMswServer();
    server.use(
      http.get(/\/api\/settings\/app-config/, () =>
        HttpResponse.json({ requires_restart: false }),
      ),
      http.get(/\/api\/command_output/, () =>
        HttpResponse.json({ message: "not found" }, { status: 404 }),
      ),
      http.get(/\/api\/settings$/, () => HttpResponse.json({})),
      http.get(/\/api\/users$/, () => HttpResponse.json([])),
      http.get(/\/api\/volumes$/, () => HttpResponse.json([])),
      http.get(/\/api\/hostname$/, () => HttpResponse.json({ hostname: "localhost" })),
      http.get(/\/api\/nics$/, () => HttpResponse.json([])),
      http.get(/\/api\/telemetry\/internet-connection$/, () =>
        HttpResponse.json({ connected: false }),
      ),
      http.get(/\/api\/health$/, () =>
        HttpResponse.json({ status: "ok", dirty_tracking: {} }),
      ),
    );
  });

  it("does not reload on transient heartbeat outage and recovery", async () => {
    const { App } = await import("../App");
    const store = await createTestStore();
    const reloadMock = vi
      .spyOn(window.location, "reload")
      .mockImplementation(() => undefined);

    const { rerender } = render(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    // Stable connected state.
    wsStateRef.current = { heartbeat: { alive: true } };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    // Transient outage: server not reachable.
    wsStateRef.current = { heartbeat: { alive: false } };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    // Recovery: heartbeat alive again. Must NOT trigger a full page reload.
    wsStateRef.current = { heartbeat: { alive: true } };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    await waitFor(() => {
      expect(reloadMock).toHaveBeenCalledTimes(0);
    });
  });
});
