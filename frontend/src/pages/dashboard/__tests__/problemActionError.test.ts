import { beforeEach, describe, expect, it, vi } from "vitest";

import { type Problem, Severity, Status } from "../../../store/sratApi";
import { normalizeSentryExtras } from "../../../utils/sentrySerialize";
import {
  clearLocalProblemIgnore,
  describeProblemActionFailure,
  dismissProblemToast,
  getLocalProblemIgnore,
  getProblemIgnoreOverride,
  isInactiveProblem,
  isProblemIgnoreOverrideActive,
  isRetryableProblemStatus,
  mergeLiveProblems,
  problemToastId,
  reconcileProblemIgnoreFromServer,
  reconcileProblemIgnoreMutationResult,
  resolveReenableTarget,
  setLocalProblemIgnore,
  subscribeProblemIgnoreOverrides,
} from "../problemActionError";

const { toastDismissMock } = vi.hoisted(() => ({
  toastDismissMock: vi.fn(),
}));

vi.mock("react-toastify", () => ({
  toast: {
    dismiss: (...args: unknown[]) => toastDismissMock(...args),
    error: vi.fn(),
    warn: vi.fn(),
    info: vi.fn(),
    update: vi.fn(),
    isActive: vi.fn(() => false),
  },
}));

function makeProblem(overrides: Partial<Problem> = {}): Problem {
  return {
    id: 1,
    problem_key: "protected_mode",
    title: "Addon in Protected Mode",
    description: "Protection mode is on",
    severity: Severity.Warning,
    status: Status.Created,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ignored: false,
    repeating: 1,
    ...overrides,
  };
}

describe("describeProblemActionFailure (#1360)", () => {
  beforeEach(() => {
    clearLocalProblemIgnore("protected_mode");
    toastDismissMock.mockClear();
  });

  it("prefers backend detail over message and status", () => {
    const failure = describeProblemActionFailure(
      "dismiss",
      "addon_config_changed",
      {
        status: 500,
        data: { detail: "device busy", message: "other", status: 500 },
      },
    );
    expect(failure.problemKey).toBe("addon_config_changed");
    expect(failure.status).toBe(500);
    expect(failure.detail).toContain("device busy");
    expect(failure.message).toContain("addon_config_changed");
    expect(failure.message).toContain("device busy");
    expect(failure.retryable).toBe(true);
    expect(failure.toastMessage).toContain("Retrying may help");
  });

  it("falls back to message then status", () => {
    const fromMessage = describeProblemActionFailure(
      "ignore",
      "protected_mode",
      {
        status: 400,
        data: { message: "validation failed" },
      },
    );
    expect(fromMessage.detail).toContain("validation failed");
    expect(fromMessage.retryable).toBe(false);
    expect(fromMessage.toastMessage).not.toContain("Retrying may help");

    const fromStatus = describeProblemActionFailure(
      "re-enable",
      "protected_mode",
      {
        status: 404,
        data: {},
      },
    );
    expect(fromStatus.status).toBe(404);
    expect(fromStatus.detail).toContain("404");
  });

  it("marks fetch errors as retryable", () => {
    expect(isRetryableProblemStatus("FETCH_ERROR")).toBe(true);
    expect(isRetryableProblemStatus("TIMEOUT_ERROR")).toBe(true);
    expect(isRetryableProblemStatus("TIMEOUT")).toBe(true);
    expect(isRetryableProblemStatus(503)).toBe(true);
    expect(isRetryableProblemStatus("503")).toBe(true);
    expect(isRetryableProblemStatus(400)).toBe(false);
    expect(isRetryableProblemStatus("oops")).toBe(false);
    expect(isRetryableProblemStatus(undefined)).toBe(false);
  });

  it("handles Error-instance and string rejections", () => {
    const fromError = describeProblemActionFailure(
      "dismiss",
      "addon_config_changed",
      new Error("kaput"),
    );
    expect(fromError.detail).toContain("kaput");
    expect(fromError.message).toContain("addon_config_changed");
    expect(fromError.retryable).toBe(false);

    const fromString = describeProblemActionFailure(
      "dismiss",
      "addon_config_changed",
      "plain failure",
    );
    expect(fromString.detail).toContain("plain failure");
    expect(fromString.retryable).toBe(false);
  });

  it("marks numeric-string statuses retryable with a toast hint", () => {
    const failure = describeProblemActionFailure(
      "dismiss",
      "addon_config_changed",
      {
        status: "503",
        data: { detail: "slow backend" },
      },
    );
    expect(failure.retryable).toBe(true);
    expect(failure.toastMessage).toContain("Retrying may help");
  });

  it("produces Sentry-safe extras without [Object] placeholders", () => {
    const failure = describeProblemActionFailure(
      "dismiss",
      "addon_config_changed",
      {
        status: 500,
        data: { detail: "dismiss boom" },
      },
    );
    const normalized = normalizeSentryExtras({
      problemKey: failure.problemKey,
      status: failure.status,
      detail: failure.detail,
      response: failure.response,
    });
    const asString = JSON.stringify(normalized);
    expect(asString).toContain("addon_config_changed");
    expect(asString).toContain("dismiss boom");
    expect(asString).not.toContain("[Object]");
  });

  it("resolves re-enable targets for hit, miss, and blank keys", () => {
    const issue = { problem_key: "addon_config_changed" } as never;
    expect(resolveReenableTarget([issue], "addon_config_changed")).toEqual({
      type: "upsert",
      issue,
    });
    expect(resolveReenableTarget([issue], "gone")).toEqual({
      type: "dismiss",
    });
    expect(resolveReenableTarget([], "gone")).toEqual({ type: "dismiss" });
    expect(resolveReenableTarget([{ problem_key: "" } as never], "")).toEqual({
      type: "dismiss",
    });
  });

  it("uses the stable problem toast id when dismissing", () => {
    const toastId = problemToastId("protected_mode");
    expect(toastId).toBe("problem-protected_mode");

    dismissProblemToast(toastId);
    expect(toastDismissMock).toHaveBeenCalledWith("problem-protected_mode");
  });

  it("tracks local ignore overrides and notifies subscribers", () => {
    const listener = vi.fn();
    const unsubscribe = subscribeProblemIgnoreOverrides(listener);

    setLocalProblemIgnore("protected_mode", true);
    expect(getLocalProblemIgnore("protected_mode")).toBe(true);
    expect(listener).toHaveBeenCalledTimes(1);

    setLocalProblemIgnore("protected_mode", false);
    expect(getLocalProblemIgnore("protected_mode")).toBe(false);
    expect(listener).toHaveBeenCalledTimes(2);

    clearLocalProblemIgnore("protected_mode");
    expect(getLocalProblemIgnore("protected_mode")).toBeUndefined();
    unsubscribe();
  });

  it("keeps a local ignore over a stale active WS snapshot", () => {
    setLocalProblemIgnore("protected_mode", true, "2026-01-02T00:00:00Z");
    const staleActive = makeProblem({
      updated_at: "2026-01-01T00:00:00Z",
      ignored: false,
      status: Status.Created,
    });

    expect(
      isProblemIgnoreOverrideActive(
        getProblemIgnoreOverride("protected_mode"),
        staleActive,
      ),
    ).toBe(true);

    reconcileProblemIgnoreFromServer([], staleActive);
    expect(getLocalProblemIgnore("protected_mode")).toBe(true);
  });

  it("clears an override when a server snapshot is newer or matches the click", () => {
    setLocalProblemIgnore("protected_mode", true, "2026-01-01T00:00:00Z");
    reconcileProblemIgnoreFromServer(
      [],
      makeProblem({
        updated_at: "2026-01-02T00:00:00Z",
        ignored: false,
        status: Status.Created,
      }),
    );
    expect(getLocalProblemIgnore("protected_mode")).toBeUndefined();

    setLocalProblemIgnore("protected_mode", false, "2026-01-02T00:00:00Z");
    reconcileProblemIgnoreFromServer(
      [
        makeProblem({
          updated_at: "2026-01-02T00:00:00Z",
          ignored: false,
          status: Status.Created,
        }),
      ],
      undefined,
    );
    expect(getLocalProblemIgnore("protected_mode")).toBeUndefined();
  });

  it("clears an override from a successful mutation response", () => {
    setLocalProblemIgnore("protected_mode", true, "2026-01-01T00:00:00Z");
    const cleared = reconcileProblemIgnoreMutationResult(
      makeProblem({
        updated_at: "2026-01-02T00:00:00Z",
        ignored: true,
        status: Status.Ignored,
      }),
    );

    expect(cleared).toBe(true);
    expect(getLocalProblemIgnore("protected_mode")).toBeUndefined();
  });

  it("ignores older WS payloads when merging with REST state", () => {
    const ignoredServerState = makeProblem({
      updated_at: "2026-01-03T00:00:00Z",
      ignored: true,
      status: Status.Ignored,
    });
    const staleActiveEvent = makeProblem({
      updated_at: "2026-01-01T00:00:00Z",
      ignored: false,
      status: Status.Created,
    });

    expect(mergeLiveProblems([ignoredServerState], staleActiveEvent)).toEqual([
      ignoredServerState,
    ]);
  });

  it("recognizes inactive problem states", () => {
    expect(isInactiveProblem({ ignored: true, status: Status.Created })).toBe(
      true,
    );
    expect(isInactiveProblem({ ignored: false, status: Status.Ignored })).toBe(
      true,
    );
    expect(isInactiveProblem({ ignored: false, status: Status.Fixed })).toBe(
      true,
    );
    expect(isInactiveProblem({ ignored: false, status: Status.Updated })).toBe(
      false,
    );
    expect(isInactiveProblem(null)).toBe(false);
  });
});
