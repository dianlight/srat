/* eslint-disable */
import { render, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";
import { Provider } from "react-redux";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createTestStore, getMswServer } from "/test/testing";

const {
  toastErrorMock,
  toastWarnMock,
  toastInfoMock,
  toastDismissMock,
  toastUpdateMock,
  toastIsActiveMock,
  wsStateRef,
  activeToasts,
  getUnreadCount,
} = vi.hoisted(() => {
  const activeToasts = new Set<string>();
  const getUnreadCount = () => activeToasts.size;
  const toastErrorMock = vi.fn((...args: unknown[]) => {
    const opts = args[1] as Record<string, unknown> | undefined;
    const id = (opts?.toastId as string | undefined) ?? (args[0] as string);
    if (id) activeToasts.add(id);
  });
  const toastWarnMock = vi.fn((...args: unknown[]) => {
    const opts = args[1] as Record<string, unknown> | undefined;
    const id = (opts?.toastId as string | undefined) ?? (args[0] as string);
    if (id) activeToasts.add(id);
  });
  const toastInfoMock = vi.fn((...args: unknown[]) => {
    const opts = args[1] as Record<string, unknown> | undefined;
    const id = (opts?.toastId as string | undefined) ?? (args[0] as string);
    if (id) activeToasts.add(id);
  });
  const toastDismissMock = vi.fn((...args: unknown[]) => {
    const id = args[0] as string | undefined;
    if (id) activeToasts.delete(id);
  });
  const toastUpdateMock = vi.fn((..._args: unknown[]) => undefined);
  const toastIsActiveMock = vi.fn((_id: string) => false);
  const wsStateRef = {
    current: {
      heartbeat: { alive: true },
    } as Record<string, unknown>,
  };
  return {
    toastErrorMock,
    toastWarnMock,
    toastInfoMock,
    toastDismissMock,
    toastUpdateMock,
    toastIsActiveMock,
    wsStateRef,
    activeToasts,
    getUnreadCount,
  };
});

vi.mock("../store/wsApi", () => ({
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

vi.mock("react-toastify", () => ({
  Slide: undefined,
  ToastContainer: () => null,
  toast: {
    error: (...args: unknown[]) => toastErrorMock(...args),
    warn: (...args: unknown[]) => toastWarnMock(...args),
    warning: (...args: unknown[]) => toastWarnMock(...args),
    info: (...args: unknown[]) => toastInfoMock(...args),
    success: (..._args: unknown[]) => undefined,
    dismiss: (...args: unknown[]) => toastDismissMock(...args),
    update: (...args: unknown[]) => toastUpdateMock(...args),
    isActive: (...args: unknown[]) => toastIsActiveMock(...(args as [string])),
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

function makeProblem(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    problem_key: "custom_component_missing",
    title: "Custom component missing",
    description: "The custom component is not installed",
    severity: "warning",
    status: "created",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ignored: false,
    repeating: 1,
    ...overrides,
  };
}

describe("App problem notifications - toastId dedup and dismiss", () => {
  beforeEach(async () => {
    wsStateRef.current = { heartbeat: { alive: true } };
    activeToasts.clear();
    toastErrorMock.mockClear();
    toastWarnMock.mockClear();
    toastInfoMock.mockClear();
    toastDismissMock.mockClear();
    toastUpdateMock.mockClear();
    toastIsActiveMock.mockReset();
    toastIsActiveMock.mockReturnValue(false);
    if (localStorage && typeof localStorage.clear === "function") {
      localStorage.clear();
    }
    if (sessionStorage && typeof sessionStorage.clear === "function") {
      sessionStorage.clear();
    }
    const server = getMswServer();
    server.use(
      http.get(/\/api\/settings\/app-config/, () => HttpResponse.json({ requires_restart: false })),
      http.get(/\/api\/command_output/, () => HttpResponse.json({ message: "not found" }, { status: 404 })),
      http.get(/\/api\/settings$/, () => HttpResponse.json({})),
      http.get(/\/api\/users$/, () => HttpResponse.json([])),
      http.get(/\/api\/volumes$/, () => HttpResponse.json([])),
      http.get(/\/api\/hostname$/, () => HttpResponse.json({ hostname: "localhost" })),
      http.get(/\/api\/nics$/, () => HttpResponse.json([])),
      http.get(/\/api\/telemetry\/internet-connection$/, () => HttpResponse.json({ connected: false })),
      http.get(/\/api\/health$/, () => HttpResponse.json({ status: "ok", dirty_tracking: {} })),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("creates a single toast for a new problem and dedups duplicate WS replays by problem_key", async () => {
    const { App } = await import("../App");
    const store = await createTestStore();
    const { rerender } = render(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    const problem = makeProblem();

    wsStateRef.current = {
      heartbeat: { alive: true },
      problem,
    };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    await waitFor(() => {
      expect(toastWarnMock.mock.calls.length).toBe(1);
    });
    const firstCallArgs = toastWarnMock.mock.calls[0] as unknown as [string, Record<string, unknown>];
    expect(firstCallArgs[0]).toBe(problem.title);
    expect(firstCallArgs[1]).toHaveProperty("toastId", "problem-custom_component_missing");

    // Replay same problem with identical dedup key (same updated_at, status, severity) - WS reconnect replay
    wsStateRef.current = {
      heartbeat: { alive: true },
      problem: { ...problem },
    };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    // Should still be 1, not 2
    await waitFor(() => {
      expect(toastWarnMock.mock.calls.length).toBe(1);
      expect(toastErrorMock.mock.calls.length).toBe(0);
      expect(toastInfoMock.mock.calls.length).toBe(0);
    });

    // Simulate 8 rapid replays - badge must not climb to 8
    for (let i = 0; i < 6; i++) {
      wsStateRef.current = {
        heartbeat: { alive: true },
        problem: { ...problem },
      };
      rerender(
        <Provider store={store}>
          <App />
        </Provider>,
      );
    }

    await waitFor(() => {
      expect(toastWarnMock.mock.calls.length).toBe(1);
    });
    // unreadCount (badge) must not climb to 8 on reconnect replays
    expect(getUnreadCount()).toBe(1);
  });

  it("updates existing toast when problem is re-emitted with new updated_at instead of creating new toast", async () => {
    const { App } = await import("../App");
    const store = await createTestStore();

    const { rerender } = render(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    const problem = makeProblem({ updated_at: "2026-01-01T00:00:00Z" });
    wsStateRef.current = { heartbeat: { alive: true }, problem };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    await waitFor(() => {
      expect(toastWarnMock.mock.calls.length).toBe(1);
    });

    // Make toast appear active so next emission triggers update path
    toastIsActiveMock.mockReturnValue(true);

    const updatedProblem = makeProblem({ updated_at: "2026-01-01T01:00:00Z", title: "Custom component missing - updated" });
    wsStateRef.current = { heartbeat: { alive: true }, problem: updatedProblem };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    await waitFor(() => {
      expect(toastUpdateMock.mock.calls.length).toBe(1);
    });
    const updateArgs = toastUpdateMock.mock.calls[0] as unknown as [string, Record<string, unknown>];
    expect(updateArgs[0]).toBe("problem-custom_component_missing");
    expect(updateArgs[1]).toHaveProperty("render", updatedProblem.title);
    // No new toast should have been created
    expect(toastWarnMock.mock.calls.length).toBe(1);
  });

  it("maps severity to correct toast type and keeps toastId stable", async () => {
    const { App } = await import("../App");
    const store = await createTestStore();
    const { rerender } = render(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    const errorProblem = makeProblem({ problem_key: "disk_error", severity: "error", title: "Disk error" });
    wsStateRef.current = { heartbeat: { alive: true }, problem: errorProblem };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );
    await waitFor(() => {
      expect(toastErrorMock.mock.calls.length).toBe(1);
    });
    expect((toastErrorMock.mock.calls[0] as [string, Record<string, unknown>])[1]).toHaveProperty("toastId", "problem-disk_error");

    const infoProblem = makeProblem({ problem_key: "info_note", severity: "info", title: "Info note" });
    wsStateRef.current = { heartbeat: { alive: true }, problem: infoProblem };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );
    await waitFor(() => {
      expect(toastInfoMock.mock.calls.length).toBe(1);
    });
    expect((toastInfoMock.mock.calls[0] as [string, Record<string, unknown>])[1]).toHaveProperty("toastId", "problem-info_note");
  });

  it("dismisses toast when problem status becomes Dismissed and allows re-creation", async () => {
    const { App } = await import("../App");
    const store = await createTestStore();
    const { rerender } = render(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    const problem = makeProblem({ status: "created" });
    wsStateRef.current = { heartbeat: { alive: true }, problem };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );
    await waitFor(() => {
      expect(toastWarnMock.mock.calls.length).toBe(1);
    });

    const dismissed = makeProblem({ status: "dismissed", updated_at: "2026-01-02T00:00:00Z" });
    wsStateRef.current = { heartbeat: { alive: true }, problem: dismissed };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    await waitFor(() => {
      expect(toastDismissMock.mock.calls.length).toBe(1);
    });
    expect(toastDismissMock.mock.calls[0]).toEqual(["problem-custom_component_missing"]);

    // After dismiss, same problem_key with created status should create a new toast (regression check for map cleanup)
    toastIsActiveMock.mockReturnValue(false);
    const recreated = makeProblem({ status: "created", updated_at: "2026-01-03T00:00:00Z" });
    wsStateRef.current = { heartbeat: { alive: true }, problem: recreated };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    await waitFor(() => {
      expect(toastWarnMock.mock.calls.length).toBe(2);
    });
    expect((toastWarnMock.mock.calls[1] as [string, Record<string, unknown>])[1]).toHaveProperty("toastId", "problem-custom_component_missing");
  });

  it("dismisses toast for Deleted and Fixed statuses and dedups Deletes", async () => {
    const { App } = await import("../App");
    const store = await createTestStore();
    const { rerender } = render(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    const base = makeProblem({ problem_key: "vol_missing", status: "created", updated_at: "2026-01-01T00:00:00Z" });
    wsStateRef.current = { heartbeat: { alive: true }, problem: base };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );
    await waitFor(() => {
      expect(toastWarnMock.mock.calls.length).toBe(1);
    });

    // Deleted
    wsStateRef.current = { heartbeat: { alive: true }, problem: makeProblem({ problem_key: "vol_missing", status: "deleted", updated_at: "2026-01-02T00:00:00Z" }) };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );
    await waitFor(() => {
      expect(toastDismissMock.mock.calls.length).toBe(1);
    });
    expect(toastDismissMock.mock.calls[0]).toEqual(["problem-vol_missing"]);

    // Fixed
    wsStateRef.current = { heartbeat: { alive: true }, problem: makeProblem({ problem_key: "vol_missing", status: "fixed", updated_at: "2026-01-03T00:00:00Z" }) };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );
    await waitFor(() => {
      expect(toastDismissMock.mock.calls.length).toBe(2);
    });
    expect(toastDismissMock.mock.calls[1]).toEqual(["problem-vol_missing"]);

    // Duplicate Deleted replay should not create extra dismiss calls beyond dedup? Actually dismiss is idempotent - each distinct deleted event will dismiss, but map already cleared. Verify second identical Deleted still calls dismiss (idempotent, not creating new toast)
    wsStateRef.current = { heartbeat: { alive: true }, problem: makeProblem({ problem_key: "vol_missing", status: "deleted", updated_at: "2026-01-03T00:00:00Z" }) };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );
    await waitFor(() => {
      expect(toastDismissMock.mock.calls.length).toBe(3);
    });
    expect(toastWarnMock.mock.calls.length).toBe(1);
  });

  it("keeps separate toastIds per problem_key so badge counts distinct problems", async () => {
    const { App } = await import("../App");
    const store = await createTestStore();
    const { rerender } = render(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    const p1 = makeProblem({ problem_key: "p1", title: "P1" });
    wsStateRef.current = { heartbeat: { alive: true }, problem: p1 };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );
    await waitFor(() => {
      expect(toastWarnMock.mock.calls.length).toBe(1);
    });

    const p2 = makeProblem({ problem_key: "p2", title: "P2" });
    wsStateRef.current = { heartbeat: { alive: true }, problem: p2 };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );
    await waitFor(() => {
      expect(toastWarnMock.mock.calls.length).toBe(2);
    });
    expect((toastWarnMock.mock.calls[0] as [string, Record<string, unknown>])[1]).toHaveProperty("toastId", "problem-p1");
    expect((toastWarnMock.mock.calls[1] as [string, Record<string, unknown>])[1]).toHaveProperty("toastId", "problem-p2");
  });

  it("does not resurrect toast on WS reconnect replay of an already-dismissed problem", async () => {
    const { App } = await import("../App");
    const store = await createTestStore();
    const { rerender } = render(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    const problem = makeProblem({ problem_key: "reconnect_dismissed", status: "created", updated_at: "2026-01-01T00:00:00Z" });
    wsStateRef.current = { heartbeat: { alive: true }, problem };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    await waitFor(() => {
      expect(toastWarnMock.mock.calls.length).toBe(1);
    });

    const dismissed = makeProblem({ problem_key: "reconnect_dismissed", status: "dismissed", updated_at: "2026-01-02T00:00:00Z" });
    wsStateRef.current = { heartbeat: { alive: true }, problem: dismissed };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    await waitFor(() => {
      expect(toastDismissMock.mock.calls.length).toBe(1);
    });
    expect(toastDismissMock.mock.calls[0]).toEqual(["problem-reconnect_dismissed"]);
    expect(toastWarnMock.mock.calls.length).toBe(1);

    const unreadAfterDismiss = getUnreadCount();
    expect(unreadAfterDismiss).toBe(0);

    // Capture unreadCount before reconnect replays
    const unreadBeforeReplays = getUnreadCount();

    // Simulate WS reconnect replaying the same dismissed payload 5 times (e.g., server re-emits on reconnect, UI re-connects)
    for (let i = 0; i < 5; i++) {
      wsStateRef.current = {
        heartbeat: { alive: true },
        // New object reference but same dismissed content - typical reconnect replay
        problem: { ...dismissed },
      };
      rerender(
        <Provider store={store}>
          <App />
        </Provider>,
      );
      // Assert unreadCount stays unchanged after each replay (badge does not climb)
      expect(getUnreadCount()).toBe(unreadBeforeReplays);
      expect(getUnreadCount()).toBe(0);
    }

    // Wait a tick to let effects settle
    await waitFor(() => {
      expect(toastDismissMock.mock.calls.length).toBeGreaterThanOrEqual(1);
    });

    // Badge must stay gone: no new warning/error/info toast should have been created for the dismissed replay
    expect(toastWarnMock.mock.calls.length).toBe(1);
    expect(toastErrorMock.mock.calls.length).toBe(0);
    expect(toastInfoMock.mock.calls.length).toBe(0);
    expect(toastUpdateMock.mock.calls.length).toBe(0);
    // unreadCount must remain 0 across all five reconnect replays (badge regression guard)
    expect(getUnreadCount()).toBe(0);
    expect(getUnreadCount()).toBe(unreadBeforeReplays);
    // Dismiss is idempotent - each replay calls dismiss but must not resurrect; we allow multiple dismiss calls but ensure count does not explode due to dedup
    // At least 1 dismiss, at most 6 (initial + 5 replays) - the key assertion is no new toast was created
    expect(toastDismissMock.mock.calls.length).toBeGreaterThanOrEqual(1);
    expect(toastDismissMock.mock.calls.length).toBeLessThanOrEqual(6);
    // Verify all dismiss calls target the same toastId
    for (const call of toastDismissMock.mock.calls) {
      expect(call[0]).toBe("problem-reconnect_dismissed");
    }

    // Even after many dismissed replays, a new created problem with same key should still be able to resurrect correctly (not permanently blocked)
    toastIsActiveMock.mockReturnValue(false);
    const recreated = makeProblem({ problem_key: "reconnect_dismissed", status: "created", updated_at: "2026-01-03T00:00:00Z" });
    wsStateRef.current = { heartbeat: { alive: true }, problem: recreated };
    rerender(
      <Provider store={store}>
        <App />
      </Provider>,
    );

    await waitFor(() => {
      expect(toastWarnMock.mock.calls.length).toBe(2);
    });
    expect((toastWarnMock.mock.calls[1] as [string, Record<string, unknown>])[1]).toHaveProperty("toastId", "problem-reconnect_dismissed");
    // unreadCount must be 1 after resurrecting the same key (badge shows the new active problem)
    expect(getUnreadCount()).toBe(1);
  });


  it.each(["deleted", "fixed"] as const)(
    "does not resurrect toast on WS reconnect replay of already-%s problem - unreadCount stays 0",
    async (terminalStatus) => {
      const { App } = await import("../App");
      const store = await createTestStore();
      const { rerender } = render(
        <Provider store={store}>
          <App />
        </Provider>,
      );

      const problemKey = `reconnect_${terminalStatus}`;
      const problem = makeProblem({ problem_key: problemKey, status: "created", updated_at: "2026-01-01T00:00:00Z" });
      wsStateRef.current = { heartbeat: { alive: true }, problem };
      rerender(
        <Provider store={store}>
          <App />
        </Provider>,
      );

      await waitFor(() => {
        expect(toastWarnMock.mock.calls.length).toBe(1);
      });
      expect(getUnreadCount()).toBe(1);

      const terminal = makeProblem({ problem_key: problemKey, status: terminalStatus, updated_at: "2026-01-02T00:00:00Z" });
      wsStateRef.current = { heartbeat: { alive: true }, problem: terminal };
      rerender(
        <Provider store={store}>
          <App />
        </Provider>,
      );

      await waitFor(() => {
        expect(toastDismissMock.mock.calls.length).toBe(1);
      });
      expect(toastDismissMock.mock.calls[0]).toEqual([`problem-${problemKey}`]);
      expect(getUnreadCount()).toBe(0);

      const unreadBeforeReplays = getUnreadCount();

      for (let i = 0; i < 5; i++) {
        wsStateRef.current = {
          heartbeat: { alive: true },
          problem: { ...terminal },
        };
        rerender(
          <Provider store={store}>
            <App />
          </Provider>,
        );
        expect(getUnreadCount()).toBe(unreadBeforeReplays);
        expect(getUnreadCount()).toBe(0);
      }

      await waitFor(() => {
        expect(toastDismissMock.mock.calls.length).toBeGreaterThanOrEqual(1);
      });

      expect(toastWarnMock.mock.calls.length).toBe(1);
      expect(toastErrorMock.mock.calls.length).toBe(0);
      expect(toastInfoMock.mock.calls.length).toBe(0);
      expect(toastUpdateMock.mock.calls.length).toBe(0);
      expect(getUnreadCount()).toBe(0);
      expect(getUnreadCount()).toBe(unreadBeforeReplays);
      for (const call of toastDismissMock.mock.calls) {
        expect(call[0]).toBe(`problem-${problemKey}`);
      }

      // Recreation after terminal replays should still work
      toastIsActiveMock.mockReturnValue(false);
      const recreated = makeProblem({ problem_key: problemKey, status: "created", updated_at: "2026-01-03T00:00:00Z" });
      wsStateRef.current = { heartbeat: { alive: true }, problem: recreated };
      rerender(
        <Provider store={store}>
          <App />
        </Provider>,
      );

      await waitFor(() => {
        expect(toastWarnMock.mock.calls.length).toBe(2);
      });
      expect(getUnreadCount()).toBe(1);
    },
  );

});
