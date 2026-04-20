import { type SkipToken, skipToken } from "@reduxjs/toolkit/query";
import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { apiUrl } from "./emptyApi";
import type {
  AppConfigChangedNotification,
  CommandOutputNotification,
  CommandStartedNotification,
  CommandTerminatedNotification,
  DataDirtyTracker,
  Disk,
  FilesystemTask,
  HealthPing,
  Problem,
  RepairCommandMessage,
  SharedResource,
  SmartTestStatus,
  UpdateProgress,
  Welcome,
} from "./sratApi";
import { Supported_events } from "./sratApi";

export type EventData = {
  [Supported_events.Heartbeat]: HealthPing;
  [Supported_events.Volumes]: Disk[];
  [Supported_events.Shares]: SharedResource[];
  [Supported_events.Hello]: Welcome;
  [Supported_events.Updating]: UpdateProgress;
  [Supported_events.DirtyDataTracker]: DataDirtyTracker;
  [Supported_events.AppConfigChanged]: AppConfigChangedNotification;
  [Supported_events.SmartTestStatus]: SmartTestStatus;
  [Supported_events.FilesystemTask]: FilesystemTask;
  [Supported_events.RepairCommand]: RepairCommandMessage;
  [Supported_events.Problem]: Problem;
  command_started?: CommandStartedNotification;
  command_output?: CommandOutputNotification;
  command_terminated?: CommandTerminatedNotification;
} & {
  __wsConnected?: boolean;
};

const DEFAULT_INACTIVITY_TIMEOUT_MS = 30_000;
const DEFAULT_RECONNECT_DELAY_MS = 1_000;

const getGlobalNumber = (key: string, fallback: number) => {
  const value = (globalThis as Record<string, unknown>)[key];
  return typeof value === "number" && value >= 0 ? value : fallback;
};

export const wsApi = createApi({
  reducerPath: "wsApi",
  baseQuery: fetchBaseQuery({
    baseUrl: apiUrl,
    fetchFn: async (_url, _options) => {
      return new Response(
        JSON.stringify({
          status: 200,
          statusText: "OK",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            message: "WebSocket connection established",
          }),
        }),
      );
    },
  }),
  tagTypes: ["system"],
  endpoints: (build) => ({
    getServerEvents: build.query<EventData, void>({
      query: () => "/ws",
      providesTags: ["system"],
      async onCacheEntryAdded(
        _arg,
        { updateCachedData, cacheDataLoaded, cacheEntryRemoved },
      ) {
        const inactivityTimeoutMs = getGlobalNumber(
          "__SRAT_WS_INACTIVITY_MS",
          DEFAULT_INACTIVITY_TIMEOUT_MS,
        );
        const reconnectDelayMs = getGlobalNumber(
          "__SRAT_WS_RECONNECT_MS",
          DEFAULT_RECONNECT_DELAY_MS,
        );

        let ws: WebSocket | undefined;
        let inactivityTimer: ReturnType<typeof setTimeout> | null = null;
        let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
        let isStopped = false;
        try {
          await cacheDataLoaded;

          const setWsConnected = (connected: boolean) => {
            updateCachedData((draft) => {
              if (draft !== undefined && draft !== null) {
                draft.__wsConnected = connected;
              }
            });
          };

          // In test environments we avoid opening a real WebSocket by default
          // to reduce test resource usage. Tests that require real streaming can
          // opt-in by setting MSW_ENABLE_STREAMING=1 in their environment.
          const isTestEnv =
            (globalThis as unknown as { __TEST__?: boolean }).__TEST__ === true;
          const envAllowStreaming =
            typeof process !== "undefined" &&
            (process.env as Record<string, string | undefined>)
              ?.MSW_ENABLE_STREAMING === "1";
          if (isTestEnv && !envAllowStreaming) {
            // Do not connect; leave __wsConnected false and wait until cache removal
            setWsConnected(false);
            await cacheEntryRemoved;
            return;
          }

          setWsConnected(false);

          const clearInactivityTimer = () => {
            if (inactivityTimer) {
              clearTimeout(inactivityTimer);
              inactivityTimer = null;
            }
          };

          const scheduleInactivityTimer = () => {
            clearInactivityTimer();
            if (inactivityTimeoutMs <= 0) return;
            inactivityTimer = setTimeout(() => {
              if (isStopped) return;
              scheduleReconnect("inactivity");
            }, inactivityTimeoutMs);
          };

          const clearReconnectTimer = () => {
            if (reconnectTimer) {
              clearTimeout(reconnectTimer);
              reconnectTimer = null;
            }
          };

          const scheduleReconnect = (_reason: string) => {
            if (isStopped || reconnectTimer) return;
            reconnectTimer = setTimeout(() => {
              reconnectTimer = null;
              if (isStopped) return;
              if (ws) ws.close();
              connect();
            }, reconnectDelayMs);
          };

          const connect = () => {
            if (isStopped) return;
            clearReconnectTimer();
            clearInactivityTimer();
            setWsConnected(false);

            //console.debug("Attempting to connect to WebSocket at", apiUrl);
            const wsUrl = new URL(
              "ws",
              `${apiUrl.replace(/^http/, "ws")}/`,
            ).toString();
            //console.debug("Constructed WebSocket URL is", wsUrl);
            ws = new WebSocket(wsUrl);

            ws.addEventListener("open", () => {
              setWsConnected(true);
              scheduleInactivityTimer();
            });
            ws.addEventListener("close", () => {
              setWsConnected(false);
              scheduleReconnect("close");
            });
            ws.addEventListener("error", () => {
              setWsConnected(false);
              scheduleReconnect("error");
            });

            const listener = (event: MessageEvent) => {
              scheduleInactivityTimer();
              let [id, eventType, data] = event.data.split("\n") as [
                string,
                string,
                string,
              ];
              id = id.substring(4);
              eventType = eventType.substring(7);
              data = data.substring(6);

              const eventTypeEnum = Object.entries(Supported_events).find(
                ([_key, value]) => value === eventType,
              )?.[1];

              if (eventTypeEnum) {
                updateCachedData((draft) => {
                  if (draft !== undefined && draft !== null) {
                    draft[eventTypeEnum] = JSON.parse(data);
                  }
                });
              } else if (
                eventType === "command_started" ||
                eventType === "command_output" ||
                eventType === "command_terminated"
              ) {
                updateCachedData((draft) => {
                  if (draft !== undefined && draft !== null) {
                    draft[eventType] = JSON.parse(data);
                  }
                });
              } else {
                console.error(
                  "* Unsupported WebSocket event type:",
                  id,
                  eventType,
                  data,
                );
              }
            };

            ws.addEventListener("message", listener);
          };

          connect();
        } catch (error) {
          console.error("* Error in WebSocket connection:", error);
        } finally {
          await cacheEntryRemoved;
          isStopped = true;
          if (reconnectTimer) clearTimeout(reconnectTimer);
          if (inactivityTimer) clearTimeout(inactivityTimer);
          ws?.close();
        }
      },
    }),
  }),
});

const useWsServerEventsQuery = () => {
  const isTestEnv =
    (globalThis as unknown as { __TEST__?: boolean }).__TEST__ === true;
  const mockWsInTests =
    typeof process !== "undefined" &&
    (process.env as Record<string, string | undefined>)?.MOCK_WS_IN_TESTS ===
      "1";
  const disableStreaming =
    typeof process !== "undefined" &&
    (process.env as Record<string, string | undefined>)
      ?.MSW_DISABLE_STREAMING === "1";

  // When running tests that use the lightweight wsApi stub or explicitly
  // disable streaming, instruct RTK Query to skip running the query hook.
  // This avoids middleware-checking behavior that can produce warnings in
  // test environments and simplifies test isolation.
  const shouldSkip = isTestEnv && (mockWsInTests || disableStreaming);

  const arg = shouldSkip ? (skipToken as unknown) : undefined;
  const result = wsApi.endpoints.getServerEvents.useQuery(arg as SkipToken);
  const isConnected = Boolean(result.data?.__wsConnected);
  return {
    ...result,
    isLoading: result.isLoading || (!isConnected && !shouldSkip),
  };
};

export const useGetServerEventsQuery = useWsServerEventsQuery;
