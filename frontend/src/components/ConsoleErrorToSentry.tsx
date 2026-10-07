import { useConsoleErrorCallback } from "../hooks/useConsoleErrorCallback";
import { useSentryTelemetry } from "../hooks/useSentryTelemetry";

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
        if (v instanceof Error)
          return { name: v.name, message: v.message, stack: v.stack };
        if (
          typeof v === "string" ||
          typeof v === "number" ||
          typeof v === "boolean" ||
          v == null
        )
          return v;
        // RTK Query errors ({status, data: {detail/message}}) lose their
        // message under JSON round-trip when data holds non-serializable
        // values; extract the human message explicitly first.
        if (typeof v === "object") {
          const maybe = v as {
            data?: unknown;
            status?: unknown;
            message?: unknown;
          };
          const data =
            maybe.data && typeof maybe.data === "object"
              ? (maybe.data as Record<string, unknown>)
              : undefined;
          const detail =
            data && typeof data.detail === "string" && data.detail
              ? data.detail
              : undefined;
          const msg =
            detail ??
            (data && typeof data.message === "string" && data.message
              ? data.message
              : undefined) ??
            (typeof maybe.message === "string" ? maybe.message : undefined);
          if (
            msg !== undefined ||
            maybe.status !== undefined ||
            data !== undefined
          ) {
            return {
              status: maybe.status,
              detail,
              message:
                typeof maybe.message === "string" ? maybe.message : undefined,
              data,
            };
          }
        }
        return JSON.parse(JSON.stringify(v));
      } catch {
        return String(v);
      }
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
      reportError(first, extras);
    } else if (typeof first === "string") {
      // console.error("Label:", rtkError) previously became a bare
      // captureMessage("Label:") with console_args [Object] and no stack
      // (#1344). Promote it to an Error carrying the backend detail so
      // Sentry groups on message + stack instead of an empty string.
      const candidate = rest[0] as
        | { data?: unknown; status?: unknown }
        | undefined;
      const candidateData =
        candidate &&
        typeof candidate === "object" &&
        candidate.data &&
        typeof candidate.data === "object"
          ? (candidate.data as Record<string, unknown>)
          : undefined;
      const candidateDetail =
        candidateData &&
        typeof candidateData.detail === "string" &&
        candidateData.detail
          ? candidateData.detail
          : candidateData &&
              typeof candidateData.message === "string" &&
              candidateData.message
            ? candidateData.message
            : undefined;
      if (candidateDetail) {
        const err = new Error(`${first} ${candidateDetail}`);
        (err as { cause?: unknown }).cause = rest[0];
        reportError(err, extras);
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
