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

    const extras: Record<string, unknown> = {};
    if (rest.length > 0) {
      extras.console_args = rest.map((v) => {
        try {
          if (v instanceof Error)
            return { name: v.name, message: v.message, stack: v.stack };
          if (
            typeof v === "string" ||
            typeof v === "number" ||
            typeof v === "boolean"
          )
            return v;
          return JSON.parse(JSON.stringify(v));
        } catch {
          return String(v);
        }
      });
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
      reportError(first, extras);
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
