/* eslint-disable */
import { act, render } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";
import { Provider } from "react-redux";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createTestStore, getMswServer } from "/test/testing";

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
    data: undefined,
    isLoading: false,
    error: { status: 500 },
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

describe("App error auto-reset timer", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  beforeEach(async () => {
    vi.restoreAllMocks();
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

  it("logs debug when the 5s error auto-reset timer fires", async () => {
    const { App } = await import("../App");
    const store = await createTestStore();
    const debugSpy = vi.spyOn(console, "debug").mockImplementation(() => {});

    vi.useFakeTimers();
    try {
      render(
        <Provider store={store}>
          <App />
        </Provider>,
      );

      // Server-events query reports an error, arming the 5000ms timer.
      expect(debugSpy).not.toHaveBeenCalledWith(
        "Error auto-reset timer triggered",
      );

      await act(async () => {
        await vi.advanceTimersByTimeAsync(5000);
      });

      expect(debugSpy).toHaveBeenCalledWith(
        "Error auto-reset timer triggered",
      );
    } finally {
      vi.useRealTimers();
      debugSpy.mockRestore();
    }
  });
});
