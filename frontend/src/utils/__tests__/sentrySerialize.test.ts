import { describe, expect, it } from "vitest";
import {
  buildWizardFailureExtras,
  buildWizardFailureMessage,
  buildWizardFailureToast,
  describeMutationError,
  normalizeSentryExtras,
  serializeSentryValue,
} from "../sentrySerialize";

describe("sentrySerialize", () => {
  it("serializes Error instances without losing message", () => {
    const value = serializeSentryValue(new Error("boom")) as Record<
      string,
      unknown
    >;
    expect(value["message"]).toBe("boom");
    expect(value["name"]).toBe("Error");
  });

  it("normalizes extras without [Object] placeholders", () => {
    const extras = normalizeSentryExtras({
      console_args: [{ status: 400, data: { detail: "bad mount" } }],
    });
    const asString = JSON.stringify(extras);
    expect(asString).toContain("bad mount");
    expect(asString).not.toContain("[Object]");
  });

  it("extracts backend detail from RTK FetchBaseQueryError", () => {
    const described = describeMutationError({
      status: 400,
      data: { detail: "mount path invalid", title: "Bad Request", status: 400 },
    });
    expect(described.status).toBe(400);
    expect(described.detail).toContain("mount path invalid");
  });

  it("serializes Error cause chains without [Object] placeholders", () => {
    const rtkError = { status: 406, data: { detail: "device busy" } };
    const mountError = new Error("Mount failed: device busy") as Error & {
      cause?: unknown;
    };
    mountError.cause = rtkError;
    const normalized = normalizeSentryExtras({
      console_args: [serializeSentryValue(mountError)],
    });
    const asString = JSON.stringify(normalized);
    expect(asString).toContain("device busy");
    expect(asString).toContain("Mount failed");
    expect(asString).not.toContain("[Object]");
  });
  it("handles circular structures", () => {
    const circular: Record<string, unknown> = {};
    circular["self"] = circular;
    const normalized = normalizeSentryExtras({ circular });
    expect(JSON.stringify(normalized)).toContain("[Circular]");
  });

  it("builds a wizard failure message naming the failed op", () => {
    const message = buildWizardFailureMessage([
      { op: "settings", detail: "HTTP 400: mount path invalid" },
    ]);
    expect(message).toContain("[settings]");
    expect(message).toContain("mount path invalid");
  });

  it("builds a wizard toast surfacing the backend detail", () => {
    const toast = buildWizardFailureToast([
      { op: "settings", detail: "HTTP 400: mount path invalid" },
    ]);
    expect(toast).toContain("settings");
    expect(toast).toContain("mount path invalid");
  });

  it("builds wizard extras with serialized backend response", () => {
    const extras = buildWizardFailureExtras({
      shareName: "Media",
      partitionId: "p1",
      failures: [
        {
          op: "settings",
          status: 400,
          detail: "HTTP 400: mount path invalid",
          context: { mountPath: "/mnt/Media" },
          response: { detail: "mount path invalid" },
        },
      ],
    });
    const asString = JSON.stringify(extras);
    expect(asString).toContain("mount path invalid");
    expect(asString).toContain("settings");
    expect(asString).not.toContain("[Object]");
  });
});
