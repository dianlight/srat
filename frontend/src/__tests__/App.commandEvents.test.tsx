import { GlobalRegistrator } from "@happy-dom/global-registrator";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, mock } from "bun:test";
import { http, HttpResponse } from "msw";
import React from "react";
import { Provider } from "react-redux";
import { getMswServer } from "../../test/bun-setup";
import { createTestStore } from "../../test/setup";

if (!(globalThis as any).window || !(globalThis as any).document) {
  GlobalRegistrator.register({
    settings: {
      enableJavaScriptEvaluation: true,
      suppressCodeGenerationFromStringsWarning: true,
    },
    url: "http://localhost:3000/",
  });
}

if (!(globalThis as any).localStorage) {
  const store: Record<string, string> = {};
  (globalThis as any).localStorage = {
    getItem: (key: string) => (key in store ? store[key] : null),
    setItem: (key: string, value: string) => {
      store[key] = String(value);
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      for (const key of Object.keys(store)) delete store[key];
    },
  };
}

const toastErrorMock = mock((..._args: unknown[]) => undefined);

let wsState: Record<string, unknown> = {
  heartbeat: { alive: true },
};

mock.module("../store/wsApi", () => ({
  wsApi: {
    reducerPath: "wsApi",
    reducer: () => ({}),
    middleware: () => (next: (action: unknown) => unknown) => (action: unknown) =>
      next(action),
  },
  useGetServerEventsQuery: () => ({
    data: wsState,
    isLoading: false,
    error: undefined,
  }),
}));

mock.module("react-toastify", () => ({
  toast: {
    error: (...args: unknown[]) => toastErrorMock(...args),
  },
}));

mock.module("../hooks/useTelemetryModal", () => ({
  useTelemetryModal: () => ({ shouldShow: false, dismiss: () => undefined }),
}));

mock.module("../hooks/useBaseConfigModal", () => ({
  useBaseConfigModal: () => ({ shouldShow: false, dismiss: () => undefined }),
}));

mock.module("../components/NavBar", () => ({
  NavBar: () => <div data-testid="mock-navbar">NavBar</div>,
}));

mock.module("../components/Footer", () => ({
  Footer: () => <div data-testid="mock-footer">Footer</div>,
}));

mock.module("../components/GlobalEventTracker", () => ({
  default: () => <div data-testid="mock-event-monitor">EventMonitor</div>,
}));

mock.module("../components/BaseConfigModal", () => ({
  default: () => null,
}));

mock.module("../components/TelemetryModal", () => ({
  default: () => null,
}));

describe("App command events", () => {
  afterEach(() => {
    cleanup();
    document.body.innerHTML = "";
  });

  beforeEach(() => {
    wsState = { heartbeat: { alive: true } };
    toastErrorMock.mockClear();
    document.body.innerHTML = "";
    localStorage.clear();
    sessionStorage.clear();
  });

  it("shows stderr toast when stderr output event arrives and dialog is closed", async () => {
    const { App } = await import("../App");
    const store = await createTestStore();
    const server = await getMswServer();
    server.use(
      http.get("/api/settings/app-config", () => {
        return HttpResponse.json({ requires_restart: false });
      }),
    );

    wsState = {
      heartbeat: { alive: true },
      command_output: {
        execution_id: "exec-1",
        command_id: "cmd-1",
        channel: "stderr",
        line: "permission denied",
        timestamp: 1,
      },
    };

    render(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    await waitFor(() => {
      expect(toastErrorMock.mock.calls.length).toBe(1);
    });
  });

  it("opens popup from stderr toast action and displays command output", async () => {
    const { App } = await import("../App");
    const store = await createTestStore();
    const server = await getMswServer();
    server.use(
      http.get("/api/settings/app-config", () => {
        return HttpResponse.json({ requires_restart: false });
      }),
    );
    const user = userEvent.setup();

    const { rerender } = render(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    wsState = {
      heartbeat: { alive: true },
      command_started: {
        execution_id: "exec-2",
        command_id: "cmd-2",
        label: "Run Check",
        command: "sh",
        args: ["-c", "echo test"],
        started_at: 100,
      },
    };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    wsState = {
      heartbeat: { alive: true },
      command_output: {
        execution_id: "exec-2",
        command_id: "cmd-2",
        channel: "stderr",
        line: "boom",
        timestamp: 101,
      },
    };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    await waitFor(() => {
      expect(toastErrorMock.mock.calls.length).toBeGreaterThan(0);
    });

    const calls = toastErrorMock.mock.calls as unknown as unknown[][];
    const lastCall = calls[calls.length - 1] ?? [];
    const toastContent = lastCall[0] as React.ReactElement;
    const toastRender = render(toastContent);

    await user.click(screen.getByRole("button", { name: "Open Output" }));

    wsState = {
      heartbeat: { alive: true },
      command_terminated: {
        execution_id: "exec-2",
        command_id: "cmd-2",
        exit_code: 2,
        finished_at: 102,
        success: false,
        error: "exit 2",
      },
    };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    await waitFor(() => {
      expect(screen.getByText(/Command Output:/)).toBeTruthy();
      expect(screen.getByText(/Execution: exec-2/)).toBeTruthy();
      expect(screen.getByText(/\[stderr\]/)).toBeTruthy();
      expect(screen.getByText("boom")).toBeTruthy();
    });

    toastRender.unmount();
  });
});
