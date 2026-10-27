import * as Sentry from "@sentry/react";
import { useEffect, useState } from "react";
import packageJson from "../../package.json";
import { getCurrentEnv, getSentryDsn } from "../macro/Environment" with {
  type: "macro",
};
import {
  type Settings,
  Telemetry_mode,
  useGetApiSettingsQuery,
} from "../store/sratApi";
import { useGetServerEventsQuery } from "../store/wsApi";
import { normalizeSentryExtras } from "../utils/sentrySerialize";

/**
 * Hook that provides Sentry functionality with telemetry mode checking.
 * Errors/events are only reported based on current telemetry mode.
 */
export const useSentryTelemetry = () => {
  const {
    data: apiSettings,
    isLoading: apiLoading,
    error: apiError,
  } = useGetApiSettingsQuery();
  const [telemetryMode, setTelemetryMode] = useState<Telemetry_mode>(
    Telemetry_mode.Disabled,
  );
  const { data: evdata, isLoading, error: herror } = useGetServerEventsQuery();

  useEffect(() => {
    setTelemetryMode(
      (apiSettings as Settings)?.telemetry_mode || Telemetry_mode.Ask,
    );
  }, [apiSettings]);

  useEffect(() => {
    if (!isLoading && !apiLoading && evdata?.hello && apiSettings) {
      const dsn = getSentryDsn();
      const enabled =
        dsn !== "disabled" &&
        [Telemetry_mode.Errors, Telemetry_mode.All].includes(
          (apiSettings as Settings)?.telemetry_mode || Telemetry_mode.Ask,
        );

      Sentry.init({
        dsn: dsn === "disabled" ? "" : dsn,
        environment: getCurrentEnv(),
        release: packageJson.version,
        enabled,
      });

      Sentry.setTag("version", packageJson.version);
      if (evdata.hello.machine_id) {
        Sentry.setUser({ id: evdata.hello.machine_id });
      }
    }
  }, [isLoading, apiLoading, evdata?.hello, apiSettings]);

  const reportError = (
    error: Error | string,
    extraData?: Record<string, unknown>,
  ) => {
    if ([Telemetry_mode.Errors, Telemetry_mode.All].includes(telemetryMode)) {
      // #1344: bare captureMessage(string) yields minified withScope frames
      // with no stack (SRAT-FRONTEND-2C). Promote strings to Errors so
      // Sentry stores a stack + cause/context.
      const asError = typeof error === "string" ? new Error(error) : error;
      if (typeof error === "string") {
        (asError as { cause?: unknown }).cause = extraData;
      }
      if (extraData) {
        // #1354: capture with contexts directly — withScope adds a
        // minified `withScope` frame that becomes the Sentry title.
        const normalizedExtras = normalizeSentryExtras(extraData);
        Sentry.captureException(asError, {
          contexts: { extra: normalizedExtras },
        });
      } else {
        Sentry.captureException(asError);
      }
    }
  };

  const reportEvent = (event: string, data?: Record<string, unknown>) => {
    if (telemetryMode === Telemetry_mode.All) {
      const eventData = {
        ...normalizeSentryExtras(data),
        event_type: event,
        timestamp: new Date().toISOString(),
      };

      Sentry.captureMessage(`Event: ${event}`, {
        level: "info",
        contexts: {
          event: eventData,
        },
      });
      if (process.env.NODE_ENV !== "production")
        console.debug("Event reported to Sentry:", event, eventData);
    }
  };

  return {
    reportError,
    reportEvent,
    telemetryMode,
    isLoading: apiLoading || isLoading,
    error: apiError || herror,
  };
};
