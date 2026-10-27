/* eslint-disable */
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import React from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getMswServer, renderWithTestStore } from "/test/testing";
import { DashboardActions } from "../DashboardActions";

const { toastErrorMock, reportErrorMock } = vi.hoisted(() => {
  return {
    toastErrorMock: vi.fn((..._args: unknown[]) => undefined),
    reportErrorMock: vi.fn((..._args: unknown[]) => undefined),
  };
});

vi.mock("react-toastify", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-toastify")>();
  return {
    ...actual,
    toast: {
      ...actual.toast,
      error: (...args: unknown[]) => toastErrorMock(...args),
    },
  };
});

vi.mock("../../../hooks/useSentryTelemetry", () => ({
  useSentryTelemetry: () => ({
    reportError: (...args: unknown[]) => reportErrorMock(...args),
    reportEvent: vi.fn(),
    telemetryMode: "errors",
    isLoading: false,
    error: undefined,
  }),
}));

function makeProblem(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    problem_key: "addon_config_changed",
    title: "Addon config changed",
    description: "Config drift detected",
    severity: "warning",
    status: "created",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ignored: false,
    repeating: 1,
    ...overrides,
  };
}

describe("DashboardActions problem-action failures (#1360)", () => {
  beforeEach(() => {
    localStorage.clear();
    toastErrorMock.mockClear();
    reportErrorMock.mockClear();
    const server = getMswServer();
    server.use(
      http.get(/\/api\/problems$/, () =>
        HttpResponse.json([makeProblem()]),
      ),
    );
  });

  it("reports dismiss failure with problem key + backend status/detail", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    try {
      const server = getMswServer();
      server.use(
        http.delete(/\/api\/problems\/.+$/, () =>
          HttpResponse.json(
            { detail: "dismiss boom", status: 500 },
            { status: 500 },
          ),
        ),
      );

      const user = userEvent.setup();
      await renderWithTestStore(
        React.createElement(
          MemoryRouter,
          null,
          React.createElement(DashboardActions as any),
        ),
      );

      const resolveButton = await screen.findByRole("button", {
        name: /^resolve$/i,
      });
      await user.click(resolveButton);

      await waitFor(() => {
        expect(reportErrorMock).toHaveBeenCalledTimes(1);
      });
      const [sentryError, extras] = reportErrorMock.mock.calls[0] as [
        Error,
        Record<string, unknown>,
      ];
      expect(sentryError).toBeInstanceOf(Error);
      expect(sentryError.message).toContain("addon_config_changed");
      expect(sentryError.message).toContain("dismiss boom");
      expect(extras["problemKey"]).toBe("addon_config_changed");
      expect(extras["status"]).toBe(500);
      expect(String(extras["detail"])).toContain("dismiss boom");
      expect(JSON.stringify(extras)).not.toContain("[Object]");

      // console.error must log the Error first with serialized context
      expect(consoleSpy).toHaveBeenCalled();
      const firstArg = consoleSpy.mock.calls[0]?.[0];
      expect(firstArg).toBeInstanceOf(Error);
      expect((firstArg as Error).message).toContain("addon_config_changed");
      const contextArg = consoleSpy.mock.calls[0]?.[1] as
        | Record<string, unknown>
        | undefined;
      expect(contextArg).toMatchObject({
        problemKey: "addon_config_changed",
        status: 500,
      });
      expect(String((contextArg as Record<string, unknown>)["detail"])).toContain(
        "dismiss boom",
      );

      // toast carries the same detail plus a retry hint for 5xx
      await waitFor(() => {
        expect(toastErrorMock).toHaveBeenCalledTimes(1);
      });
      expect(String(toastErrorMock.mock.calls[0]?.[0])).toContain(
        "addon_config_changed",
      );
      expect(String(toastErrorMock.mock.calls[0]?.[0])).toContain("dismiss boom");
      expect(String(toastErrorMock.mock.calls[0]?.[0])).toContain(
        "Retrying may help",
      );
    } finally {
      consoleSpy.mockRestore();
    }
  });

  it("reports ignore failure with the same context", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    try {
      const server = getMswServer();
      server.use(
        http.put(/\/api\/problems\/.+$/, () =>
          HttpResponse.json(
            { detail: "ignore boom", status: 500 },
            { status: 500 },
          ),
        ),
      );

      const user = userEvent.setup();
      await renderWithTestStore(
        React.createElement(
          MemoryRouter,
          null,
          React.createElement(DashboardActions as any),
        ),
      );

      const ignoreButton = await screen.findByRole("button", {
        name: /^ignore$/i,
      });
      await user.click(ignoreButton);

      await waitFor(() => {
        expect(reportErrorMock).toHaveBeenCalledTimes(1);
      });
      const [sentryError, extras] = reportErrorMock.mock.calls[0] as [
        Error,
        Record<string, unknown>,
      ];
      expect((sentryError as Error).message).toContain("ignore");
      expect((sentryError as Error).message).toContain("ignore boom");
      expect(extras["problemKey"]).toBe("addon_config_changed");
      expect(JSON.stringify(extras)).not.toContain("[Object]");

      await waitFor(() => {
        expect(toastErrorMock).toHaveBeenCalledTimes(1);
      });
      expect(String(toastErrorMock.mock.calls[0]?.[0])).toContain("ignore boom");
    } finally {
      consoleSpy.mockRestore();
    }
  });

  it("reports re-enable failure with the same context", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    try {
      const server = getMswServer();
      server.use(
        http.get(/\/api\/problems$/, () =>
          HttpResponse.json([
            makeProblem({ ignored: true, status: "ignored" }),
          ]),
        ),
        http.put(/\/api\/problems\/.+$/, () =>
          HttpResponse.json(
            { detail: "re-enable boom", status: 500 },
            { status: 500 },
          ),
        ),
      );

      const user = userEvent.setup();
      await renderWithTestStore(
        React.createElement(
          MemoryRouter,
          null,
          React.createElement(DashboardActions as any),
        ),
      );

      const showIgnored = screen.getByRole("switch", {
        name: /show ignored/i,
      });
      await user.click(showIgnored);

      const reenableButton = await screen.findByRole("button", {
        name: /^re-enable$/i,
      });
      await user.click(reenableButton);

      await waitFor(() => {
        expect(reportErrorMock).toHaveBeenCalledTimes(1);
      });
      const [sentryError, extras] = reportErrorMock.mock.calls[0] as [
        Error,
        Record<string, unknown>,
      ];
      expect((sentryError as Error).message).toContain("re-enable");
      expect((sentryError as Error).message).toContain(
        "addon_config_changed",
      );
      expect((sentryError as Error).message).toContain("re-enable boom");
      expect(extras["problemKey"]).toBe("addon_config_changed");
      expect(extras["status"]).toBe(500);
      expect(JSON.stringify(extras)).not.toContain("[Object]");

      expect(consoleSpy).toHaveBeenCalled();
      expect(consoleSpy.mock.calls[0]?.[0]).toBeInstanceOf(Error);

      await waitFor(() => {
        expect(toastErrorMock).toHaveBeenCalledTimes(1);
      });
      expect(String(toastErrorMock.mock.calls[0]?.[0])).toContain(
        "re-enable boom",
      );
      expect(String(toastErrorMock.mock.calls[0]?.[0])).toContain(
        "Retrying may help",
      );
    } finally {
      consoleSpy.mockRestore();
    }
  });
});
