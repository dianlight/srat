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
      if (extraData) {
        Sentry.withScope((scope) => {
          scope.setContext("extra", extraData);
          if (typeof error === "string") {
            Sentry.captureMessage(error, "error");
          } else {
            Sentry.captureException(error);
          }
        });
      } else if (typeof error === "string") {
        Sentry.captureMessage(error, "error");
      } else {
        Sentry.captureException(error);
      }
    }
  };

  const reportEvent = (event: string, data?: Record<string, unknown>) => {
    if (telemetryMode === Telemetry_mode.All) {
      const eventData = {
        ...data,
        event_type: event,
        timestamp: new Date().toISOString(),
      };

      Sentry.captureMessage(`Event: ${event}`, {
        level: "info",
        contexts: {
          event: eventData,
        },
      });
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
