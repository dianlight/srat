/* eslint-disable */
import { getMswServer, renderWithTestStore } from "/test/testing";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { delay, http, HttpResponse } from "msw";
import React from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DashboardActions } from "../DashboardActions";
import {
  clearLocalProblemIgnore,
  getLocalProblemIgnore,
  setLocalProblemIgnore,
} from "../problemActionError";

const { toastDismissMock, toastErrorMock, toastWarnMock, toastInfoMock } =
  vi.hoisted(() => ({
    toastDismissMock: vi.fn(),
    toastErrorMock: vi.fn(),
    toastWarnMock: vi.fn(),
    toastInfoMock: vi.fn(),
  }));

vi.mock("react-toastify", () => ({
  toast: {
    dismiss: (...args: unknown[]) => toastDismissMock(...args),
    error: (...args: unknown[]) => toastErrorMock(...args),
    warn: (...args: unknown[]) => toastWarnMock(...args),
    info: (...args: unknown[]) => toastInfoMock(...args),
  },
}));

function makeProtectedModeProblem(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    problem_key: "protected_mode",
    title: "Addon in Protected Mode",
    description: "Protection mode is on",
    severity: "warning",
    status: "created",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ignored: false,
    repeating: 1,
    ...overrides,
  };
}

describe("DashboardActions ignore flow (issue 1302)", () => {
  beforeEach(() => {
    localStorage.clear();
    clearLocalProblemIgnore("protected_mode");
    toastDismissMock.mockClear();
    toastErrorMock.mockClear();
    toastWarnMock.mockClear();
    toastInfoMock.mockClear();
    const server = getMswServer();
    server.use(
      http.get(/\/api\/problems$/, () =>
        HttpResponse.json([makeProtectedModeProblem()]),
      ),
    );
  });

  it("sends PUT /api/problems/protected_mode with ignored:true on Ignore click", async () => {
    const putBodies: unknown[] = [];
    const putUrls: string[] = [];
    const server = getMswServer();
    server.use(
      http.put(/\/api\/problems\/.+$/, async ({ request }) => {
        putUrls.push(request.url);
        const body = await request.json().catch(() => ({}));
        putBodies.push(body);
        return HttpResponse.json(body, { status: 200 });
      }),
    );

    const user = userEvent.setup();
    await renderWithTestStore(
      React.createElement(
        MemoryRouter,
        null,
        React.createElement(DashboardActions),
      ),
    );

    const ignoreButton = await screen.findByRole("button", {
      name: /^ignore$/i,
    });
    await user.click(ignoreButton);

    await waitFor(() => {
      expect(putBodies.length).toBeGreaterThan(0);
    });
    expect(
      putUrls.some((url) => url.includes("/api/problems/protected_mode")),
    ).toBe(true);
    expect(putBodies[0]).toMatchObject({
      problem_key: "protected_mode",
      ignored: true,
      status: "ignored",
    });
  });

  it("hides the card and dismisses its toast immediately on Ignore click", async () => {
    const server = getMswServer();
    let putCalled = false;
    server.use(
      http.put(/\/api\/problems\/protected_mode$/, async ({ request }) => {
        putCalled = true;
        const body = await request.json().catch(() => ({}));
        await delay(10);
        return HttpResponse.json(body, { status: 200 });
      }),
    );

    const user = userEvent.setup();
    await renderWithTestStore(
      React.createElement(
        MemoryRouter,
        null,
        React.createElement(DashboardActions),
      ),
    );

    expect(await screen.findByText("Addon in Protected Mode")).toBeTruthy();

    await user.click(await screen.findByRole("button", { name: /^ignore$/i }));

    // Optimistic state must win before the PUT resolves or a WS event arrives.
    expect(screen.queryByText("Addon in Protected Mode")).toBeNull();
    expect(toastDismissMock).toHaveBeenCalledWith("problem-protected_mode");

    await waitFor(() => {
      expect(putCalled).toBe(true);
    });
  });

  it("logs an error instead of silently dropping when problem_key is missing", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    try {
      const server = getMswServer();
      server.use(
        http.get(/\/api\/problems$/, () =>
          HttpResponse.json([makeProtectedModeProblem({ problem_key: "" })]),
        ),
      );

      const user = userEvent.setup();
      await renderWithTestStore(
        React.createElement(
          MemoryRouter,
          null,
          React.createElement(DashboardActions),
        ),
      );

      const ignoreButton = await screen.findByRole("button", {
        name: /^ignore$/i,
      });
      await user.click(ignoreButton);

      await waitFor(() => {
        expect(consoleSpy).toHaveBeenCalledWith(
          expect.stringContaining("Cannot ignore problem"),
          expect.anything(),
        );
      });
    } finally {
      consoleSpy.mockRestore();
    }
  });

  it("restores the active card, override and toast when IGNORE fails", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    try {
      const server = getMswServer();
      server.use(
        http.put(/\/api\/problems\/protected_mode$/, () =>
          HttpResponse.json(
            { detail: "ignore failed", status: 500 },
            { status: 500 },
          ),
        ),
      );

      const user = userEvent.setup();
      await renderWithTestStore(
        React.createElement(
          MemoryRouter,
          null,
          React.createElement(DashboardActions),
        ),
      );

      await user.click(
        await screen.findByRole("button", { name: /^ignore$/i }),
      );
      await waitFor(() => {
        expect(toastErrorMock).toHaveBeenCalledTimes(1);
      });

      expect(await screen.findByText("Addon in Protected Mode")).toBeTruthy();
      expect(getLocalProblemIgnore("protected_mode")).toBeUndefined();
      expect(toastWarnMock).toHaveBeenCalledWith(
        "Addon in Protected Mode",
        expect.objectContaining({ toastId: "problem-protected_mode" }),
      );
      expect(String(toastErrorMock.mock.calls[0]?.[0])).toContain(
        "ignore failed",
      );
    } finally {
      consoleSpy.mockRestore();
    }
  });

  it("restores ignored state and dismisses the toast when re-enable fails", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    try {
      const server = getMswServer();
      server.use(
        http.get(/\/api\/problems$/, () =>
          HttpResponse.json([
            makeProtectedModeProblem({
              ignored: true,
              status: "ignored",
              updated_at: "2026-01-02T00:00:00Z",
            }),
          ]),
        ),
        http.put(/\/api\/problems\/protected_mode$/, () =>
          HttpResponse.json(
            { detail: "re-enable failed", status: 500 },
            { status: 500 },
          ),
        ),
      );

      const user = userEvent.setup();
      await renderWithTestStore(
        React.createElement(
          MemoryRouter,
          null,
          React.createElement(DashboardActions),
        ),
      );
      await user.click(screen.getByRole("switch", { name: /show ignored/i }));
      await user.click(
        await screen.findByRole("button", { name: /^re-enable$/i }),
      );

      await waitFor(() => {
        expect(toastErrorMock).toHaveBeenCalledTimes(1);
      });
      expect(getLocalProblemIgnore("protected_mode")).toBeUndefined();
      expect(toastDismissMock).toHaveBeenCalledWith("problem-protected_mode");
      expect(toastWarnMock).not.toHaveBeenCalled();
      expect(
        await screen.findByRole("button", { name: /^re-enable$/i }),
      ).toBeTruthy();
      expect(String(toastErrorMock.mock.calls[0]?.[0])).toContain(
        "re-enable failed",
      );
    } finally {
      consoleSpy.mockRestore();
    }
  });

  it("drops a local ignore when REST reports the problem re-enabled", async () => {
    const server = getMswServer();
    server.use(
      http.get(/\/api\/problems$/, () =>
        HttpResponse.json([
          makeProtectedModeProblem({ updated_at: "2026-01-02T00:00:00Z" }),
        ]),
      ),
    );
    setLocalProblemIgnore("protected_mode", true, "2026-01-01T00:00:00Z");

    await renderWithTestStore(
      React.createElement(
        MemoryRouter,
        null,
        React.createElement(DashboardActions),
      ),
    );

    await waitFor(() => {
      expect(getLocalProblemIgnore("protected_mode")).toBeUndefined();
    });
    expect(await screen.findByText("Addon in Protected Mode")).toBeTruthy();
  });

  it("drops a local re-enable when REST reports the problem ignored", async () => {
    const server = getMswServer();
    server.use(
      http.get(/\/api\/problems$/, () =>
        HttpResponse.json([
          makeProtectedModeProblem({
            ignored: true,
            status: "ignored",
            updated_at: "2026-01-02T00:00:00Z",
          }),
        ]),
      ),
    );
    setLocalProblemIgnore("protected_mode", false, "2026-01-01T00:00:00Z");

    await renderWithTestStore(
      React.createElement(
        MemoryRouter,
        null,
        React.createElement(DashboardActions),
      ),
    );

    await waitFor(() => {
      expect(getLocalProblemIgnore("protected_mode")).toBeUndefined();
    });
    expect(screen.queryByText("Addon in Protected Mode")).toBeNull();
  });
});
