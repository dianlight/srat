/* eslint-disable */
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import React from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getMswServer, renderWithTestStore } from "/test/testing";
import { DashboardActions } from "../DashboardActions";

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
        React.createElement(DashboardActions as any),
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

  it("logs an error instead of silently dropping when problem_key is missing", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    try {
      const server = getMswServer();
      server.use(
        http.get(/\/api\/problems$/, () =>
          HttpResponse.json([
            makeProtectedModeProblem({ problem_key: "" }),
          ]),
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
        expect(consoleSpy).toHaveBeenCalledWith(
          expect.stringContaining("Cannot ignore problem"),
          expect.anything(),
        );
      });
    } finally {
      consoleSpy.mockRestore();
    }
  });
});
