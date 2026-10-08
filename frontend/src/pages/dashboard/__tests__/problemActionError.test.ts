import { describe, expect, it } from "vitest";
import { normalizeSentryExtras } from "../../../utils/sentrySerialize";
import {
  describeProblemActionFailure,
  isRetryableProblemStatus,
  resolveReenableTarget,
} from "../problemActionError";

describe("describeProblemActionFailure (#1360)", () => {
  it("prefers backend detail over message and status", () => {
    const failure = describeProblemActionFailure("dismiss", "addon_config_changed", {
      status: 500,
      data: { detail: "device busy", message: "other", status: 500 },
    });
    expect(failure.problemKey).toBe("addon_config_changed");
    expect(failure.status).toBe(500);
    expect(failure.detail).toContain("device busy");
    expect(failure.message).toContain("addon_config_changed");
    expect(failure.message).toContain("device busy");
    expect(failure.retryable).toBe(true);
    expect(failure.toastMessage).toContain("Retrying may help");
  });

  it("falls back to message then status", () => {
    const fromMessage = describeProblemActionFailure("ignore", "protected_mode", {
      status: 400,
      data: { message: "validation failed" },
    });
    expect(fromMessage.detail).toContain("validation failed");
    expect(fromMessage.retryable).toBe(false);
    expect(fromMessage.toastMessage).not.toContain("Retrying may help");

    const fromStatus = describeProblemActionFailure("re-enable", "protected_mode", {
      status: 404,
      data: {},
    });
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
    const failure = describeProblemActionFailure("dismiss", "addon_config_changed", {
      status: "503",
      data: { detail: "slow backend" },
    });
    expect(failure.retryable).toBe(true);
    expect(failure.toastMessage).toContain("Retrying may help");
  });

  it("produces Sentry-safe extras without [Object] placeholders", () => {
    const failure = describeProblemActionFailure("dismiss", "addon_config_changed", {
      status: 500,
      data: { detail: "dismiss boom" },
    });
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
    expect(
      resolveReenableTarget([{ problem_key: "" } as never], ""),
    ).toEqual({ type: "dismiss" });
  });
});
