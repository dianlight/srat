import { describe, expect, it } from "vitest";
import { normalizeSentryExtras } from "../../../utils/sentrySerialize";
import {
  describeProblemActionFailure,
  isRetryableProblemStatus,
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
    expect(isRetryableProblemStatus(503)).toBe(true);
    expect(isRetryableProblemStatus(400)).toBe(false);
    expect(isRetryableProblemStatus(undefined)).toBe(false);
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
});
