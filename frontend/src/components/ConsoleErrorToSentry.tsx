import { useConsoleErrorCallback } from "../hooks/useConsoleErrorCallback";
import { useSentryTelemetry } from "../hooks/useSentryTelemetry";
import {
  describeMutationError,
  serializeSentryValue,
} from "../utils/sentrySerialize";

/**
 * Extracts the backend `detail` (or `message` fallback) from an RTK Query
 * rejection shaped like `{ data: { detail, message }, status }`.
 */
function extractRtkDetail(value: unknown): string | undefined {
  if (!value || typeof value !== "object") return undefined;
  const candidate = value as { data?: unknown };
  if (!candidate.data || typeof candidate.data !== "object") return undefined;
  const data = candidate.data as Record<string, unknown>;
  if (typeof data.detail === "string" && data.detail.trim()) {
    return data.detail.trim();
  }
  if (typeof data.message === "string" && data.message.trim()) {
    return data.message.trim();
  }
  return undefined;
}

/**
 * Mount this component once to forward console.error calls to Sentry.
 * It respects telemetry mode via useSentryTelemetry.
 */
export const ConsoleErrorToSentry: React.FC = () => {
  const { reportError } = useSentryTelemetry();

  useConsoleErrorCallback((...args: unknown[]) => {
    const [first, ...rest] = args;

    const safeSerialize = (v: unknown): unknown => {
      try {
        // #1354: reuse the Sentry serializer so Error.cause, backend
        // detail/message, and circular refs survive (no "[Object]").
        return serializeSentryValue(v);
      } catch {
        return String(v);
      }
    };

    // Finds backend detail inside console rest-args: Error messages first,
    // then RTK `{ data: { detail/message } }`, then described mutations.
    // Skips generic fallbacks so labels are never suffixed with emptiness.
    const findDetailInArgs = (args: unknown[]): string | undefined => {
      for (const arg of args) {
        if (arg instanceof Error && arg.message.trim()) {
          return arg.message.trim();
        }
        const rtkDetail = extractRtkDetail(arg);
        if (rtkDetail) return rtkDetail;
      }
      for (const arg of args) {
        if (arg instanceof Error) continue;
        const described = describeMutationError(arg);
        if (
          described.detail &&
          described.detail !== "Unknown error" &&
          described.detail !== "Request failed" &&
          !described.detail.startsWith("Request failed with status") &&
          !described.detail.startsWith("console.error")
        ) {
          return described.detail;
        }
      }
      return undefined;
    };

    const extras: Record<string, unknown> = {};
    if (rest.length > 0) {
      extras.console_args = rest.map(safeSerialize);
    }

    try {
      const g = globalThis as unknown as Record<string, unknown>;
      const store = g.__SRAT_STORE__ ?? g.store ?? g.reduxStore;
      type StoreLike = {
        getState: () => Record<string, unknown>;
      };
      const typedStore: StoreLike | undefined =
        typeof store === "object" &&
        store !== null &&
        "getState" in store &&
        typeof (store as { getState?: unknown }).getState === "function"
          ? (store as StoreLike)
          : undefined;

      const state = typedStore?.getState();
      const mds = state?.mdsSlice ?? state?.mds;

      if (mds != null) {
        extras.mds = mds;
      }
    } catch {
      console.warn("Failed to attach mdsSlice to Sentry extras"); // eslint-disable-line no-console
    }

    if (first instanceof Error) {
      // console.error(new Error("Mount failed"), rtkError): surface the
      // backend detail at the top level (#1344) so it survives Sentry's
      // normalization instead of hiding inside console_args.
      const rtkDetail = extractRtkDetail(rest[0]);
      if (rtkDetail && !first.message.includes(rtkDetail)) {
        const enriched = new Error(`${first.message}: ${rtkDetail}`);
        (enriched as { cause?: unknown }).cause = first;
        extras.rtk_detail = rtkDetail;
        reportError(enriched, extras);
      } else {
        reportError(first, extras);
      }
    } else if (typeof first === "string") {
      // console.error("Label:", rtkError) previously became a bare
      // captureMessage("Label:") with console_args [Object] and no stack
      // (#1344). Promote it to an Error carrying the backend detail so
      // Sentry groups on message + stack instead of an empty string.
      // #1354: also look for Error objects in rest-args (mount/unmount
      // log `console.error(label, errorObj)`), otherwise the title stays
      // a bare "Mount Error:" / "Unmount Error:" with empty suffix.
      const candidateDetail =
        extractRtkDetail(rest[0]) ?? findDetailInArgs(rest);
      if (candidateDetail && !first.includes(candidateDetail)) {
        const err = new Error(`${first} ${candidateDetail}`);
        (err as { cause?: unknown }).cause = rest.length === 1 ? rest[0] : rest;
        reportError(err, extras);
      } else if (candidateDetail) {
        reportError(first, extras);
      } else {
        reportError(first, extras);
      }
    } else if (first) {
      let message = "console.error called";
      try {
        message =
          typeof first === "object" ? JSON.stringify(first) : String(first);
      } catch {
        message = String(first);
      }
      reportError(message, extras);
    } else {
      reportError("console.error called with no arguments", extras);
    }
  });

  return null;
};
