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

const registerModuleMocks = () => {
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
    Slide: undefined,
    ToastContainer: () => null,
    toast: {
      error: (...args: unknown[]) => toastErrorMock(...args),
      info: (..._args: unknown[]) => undefined,
      success: (..._args: unknown[]) => undefined,
      warn: (..._args: unknown[]) => undefined,
      warning: (..._args: unknown[]) => undefined,
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

  mock.module("../components/GlobalEventTracker", () => ({
    __esModule: true,
    default: () => <div data-testid="mock-event-monitor">EventMonitor</div>,
    useSystemLogs: () => ({ logs: [], clearLogs: () => undefined }),
  }));

  mock.module("../components/BaseConfigModal", () => ({
    default: () => null,
  }));

  mock.module("../components/TelemetryModal", () => ({
    default: () => null,
  }));
};

describe("App command events", () => {
  afterEach(() => {
    mock.restore();
    cleanup();
    document.body.innerHTML = "";
  });

  beforeEach(() => {
    mock.restore();
    registerModuleMocks();
    wsState = { heartbeat: { alive: true } };
    toastErrorMock.mockClear();
    document.body.innerHTML = "";
    localStorage.clear();
    sessionStorage.clear();
  });

  it("does not show stderr toast while exit code is unavailable", async () => {
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
      expect(toastErrorMock.mock.calls.length).toBe(0);
    });
  });

  it("shows failure toast after command termination with a non-zero exit code", async () => {
    const { App } = await import("../App");
    const store = await createTestStore();
    const server = await getMswServer();
    server.use(
      http.get("/api/settings/app-config", () => {
        return HttpResponse.json({ requires_restart: false });
      }),
    );

    const { rerender } = render(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    wsState = {
      heartbeat: { alive: true },
      command_output: {
        execution_id: "exec-1c",
        command_id: "cmd-1c",
        channel: "stderr",
        line: "permission denied",
        timestamp: 3,
      },
    };

    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    await waitFor(() => {
      expect(toastErrorMock.mock.calls.length).toBe(0);
    });

    wsState = {
      heartbeat: { alive: true },
      command_terminated: {
        execution_id: "exec-1c",
        command_id: "cmd-1c",
        exit_code: 9,
        finished_at: 4,
        success: false,
        error: "exit 9",
      },
    };

    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    await waitFor(() => {
      expect(toastErrorMock.mock.calls.length).toBe(1);
    });
  });

  it("shows stderr toast when stderr output arrives with a non-zero exit code", async () => {
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
        execution_id: "exec-1b",
        command_id: "cmd-1b",
        channel: "stderr",
        line: "permission denied",
        timestamp: 2,
        exit_code: 1,
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
        exit_code: 2,
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

  it("does not leak react-toastify helper props into DOM when rendering stderr toast content", async () => {
    const { App } = await import("../App");
    const store = await createTestStore();
    const server = await getMswServer();
    server.use(
      http.get("/api/settings/app-config", () => {
        return HttpResponse.json({ requires_restart: false });
      }),
    );

    const originalConsoleError = console.error;
    const consoleErrorMock = mock((..._args: unknown[]) => undefined);
    console.error = consoleErrorMock as typeof console.error;

    try {
      wsState = {
        heartbeat: { alive: true },
        command_output: {
          execution_id: "exec-3",
          command_id: "cmd-3",
          channel: "stderr",
          line: "permission denied",
          timestamp: 103,
          exit_code: 1,
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

      const calls = toastErrorMock.mock.calls as unknown as unknown[][];
      const lastCall = calls[calls.length - 1] ?? [];
      const toastContent = lastCall[0] as React.ReactElement<Record<string, unknown>>;
      const toastRender = render(
        React.cloneElement<Record<string, unknown>>(toastContent, {
          closeToast: () => undefined,
          toastProps: {},
          isPaused: false,
          data: undefined,
        }),
      );

      const loggedWarnings = consoleErrorMock.mock.calls
        .flat()
        .map((entry) => String(entry))
        .join("\n");

      expect(loggedWarnings).not.toContain("closeToast");
      expect(loggedWarnings).not.toContain("toastProps");
      expect(loggedWarnings).not.toContain("isPaused");

      toastRender.unmount();
    } finally {
      console.error = originalConsoleError;
    }
  });
});
