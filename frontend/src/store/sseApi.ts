import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { apiUrl } from "./emptyApi";
import type {
	DataDirtyTracker,
	Disk,
	FilesystemTask,
	HealthPing,
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
	[Supported_events.SmartTestStatus]: SmartTestStatus;
	[Supported_events.FilesystemTask]: FilesystemTask;
} & {
	__wsConnected?: boolean;
};

const DEFAULT_INACTIVITY_TIMEOUT_MS = 30_000;
const DEFAULT_RECONNECT_DELAY_MS = 1_000;

const getGlobalNumber = (key: string, fallback: number) => {
	const value = (globalThis as Record<string, unknown>)[key];
	return typeof value === "number" && value >= 0 ? value : fallback;
};

/**
 * SSE API for handling Server-Sent Events using RTK Query and EventSource
 *
 * This API provides:
 * - useGetServerEventsQuery: RTK Query hook for SSE endpoint (for compatibility)
 * - useGetServerEvents: Custom hook that establishes EventSource connection for real-time SSE streaming
 *
 * The custom hook automatically dispatches actions to update the SSE slice state
 * when events are received from the server.
 */

// Create a separate API for SSE operations
/*
export const sseApi = createApi({
	reducerPath: "sseApi",
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
						message: "SSE connection established",
					}),
				}),
			); // Dummy response as we won't use fetch for SSE
		},
	}),
	tagTypes: ["system"],
	endpoints: (build) => ({
		// This endpoint is for RTK Query compatibility but won't be used for actual SSE
		getServerEvents: build.query<EventData, void>({
			query: () => "/api/sse",
			providesTags: ["system"],
			async onCacheEntryAdded(
				_arg,
				{ updateCachedData, cacheDataLoaded, cacheEntryRemoved },
			) {
				//console.log("* Starting SSE connection");
				const inactivityTimeoutMs = getGlobalNumber(
					"__SRAT_SSE_INACTIVITY_MS",
					DEFAULT_INACTIVITY_TIMEOUT_MS,
				);
				const reconnectDelayMs = getGlobalNumber(
					"__SRAT_SSE_RECONNECT_MS",
					DEFAULT_RECONNECT_DELAY_MS,
				);

				let eventSource: EventSource | null = null;
				let inactivityTimer: ReturnType<typeof setTimeout> | null = null;
				let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
				let isStopped = false;
				try {
					// wait for the initial query to resolve before proceeding
					await cacheDataLoaded;

					let faultCount = 0;

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

					const scheduleReconnect = (reason: string) => {
						if (isStopped || reconnectTimer) return;
						reconnectTimer = setTimeout(() => {
							reconnectTimer = null;
							if (isStopped) return;
							if (eventSource) eventSource.close();
							connect();
						}, reconnectDelayMs);
						console.warn("* SSE reconnect scheduled:", reason);
					};

					const connect = () => {
						if (isStopped) return;
						clearReconnectTimer();
						clearInactivityTimer();
						eventSource = new EventSource(`${apiUrl}/api/sse`, {
							withCredentials: true,
						});

						eventSource.addEventListener("error", (event) => {
							console.warn(`* SSE connection error ${faultCount}`, event);
							faultCount++;
							scheduleReconnect("error");
						});
						eventSource.addEventListener("open", (_event) => {
							//console.debug("* SSE connection open", event);
							faultCount = 0;
							scheduleInactivityTimer();
						});

						Object.values(Supported_events).forEach((event) => {
							eventSource?.addEventListener(event, (data) => {
								scheduleInactivityTimer();
								updateCachedData((draft) => {
									//console.log(
									//	`* SSE event ${event} received:`,
									//	event,
									//	data,
									//	draft[event],
									//);
									if (draft !== undefined && draft !== null) {
										// console.log(`* Updating draft for event ${event}`);
										draft[event] = JSON.parse(data.data);
									}
								});
							});
						});
					};

					connect();
				} catch (error) {
					// no-op in case `cacheEntryRemoved` resolves before `cacheDataLoaded`,
					// in which case `cacheDataLoaded` will throw
					console.error("* Error in SSE connection:", error);
				}
				// cacheEntryRemoved will resolve when the cache subscription is no longer active
				await cacheEntryRemoved;
				// perform cleanup steps once the `cacheEntryRemoved` promise resolves
				isStopped = true;
				if (reconnectTimer) clearTimeout(reconnectTimer);
				if (inactivityTimer) clearTimeout(inactivityTimer);
				if (eventSource) (eventSource as EventSource).close();
			},
		}),
	}),
});
*/

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
			); // Dummy response as we won't use fetch for WebSocket
		},
	}),
	tagTypes: ["system"],
	endpoints: (build) => ({
		// This endpoint is for RTK Query compatibility but won't be used for actual WebSocket
		getServerEvents: build.query<EventData, void>({
			query: () => "/ws", // "/api/ws",
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

				let ws: WebSocket | null = null;
				let inactivityTimer: ReturnType<typeof setTimeout> | null = null;
				let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
				let isStopped = false;
				try {
					// wait for the initial query to resolve before proceeding
					await cacheDataLoaded;

					const setWsConnected = (connected: boolean) => {
						updateCachedData((draft) => {
							if (draft !== undefined && draft !== null) {
								draft.__wsConnected = connected;
							}
						});
					};

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

					const scheduleReconnect = (reason: string) => {
						if (isStopped || reconnectTimer) return;
						reconnectTimer = setTimeout(() => {
							reconnectTimer = null;
							if (isStopped) return;
							if (ws) ws.close();
							connect();
						}, reconnectDelayMs);
						//console.warn("* WebSocket reconnect scheduled:", reason);
					};

					const connect = () => {
						if (isStopped) return;
						clearReconnectTimer();
						clearInactivityTimer();
						setWsConnected(false);

						//console.log("* Starting WebSocket connection");
						// create a websocket connection when the cache subscription starts
						ws = new WebSocket(`${apiUrl.replace(/^http/, "ws")}/ws`);

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

						// when data is received from the socket connection to the server,
						// if it is a message and for the appropriate channel,
						// update our query result with the received message
						const listener = (event: MessageEvent) => {
							scheduleInactivityTimer();
							//console.log("* WebSocket event received:", event.data)
							let [id, eventType, data] = event.data.split("\n") as [
								string,
								string,
								string,
							];
							id = id.substring(4);
							eventType = eventType.substring(7);
							data = data.substring(6);
							//console.log("* Parsed:", { id, eventType, data })

							const eventTypeEnum = Object.entries(Supported_events).find(
								([_key, value]) => value === eventType,
							)?.[1];

							//console.log("* WebSocket event:", id, eventType, eventTypeEnum, data);

							if (eventTypeEnum) {
								updateCachedData((draft) => {
									if (draft !== undefined && draft !== null) {
										// console.log(`* Updating draft for event ${event}`);
										draft[eventTypeEnum] = JSON.parse(data);
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
							//const data = JSON.parse(event.data)
						};

						ws.addEventListener("message", listener);
					};

					connect();
				} catch (error) {
					// no-op in case `cacheEntryRemoved` resolves before `cacheDataLoaded`,
					// in which case `cacheDataLoaded` will throw
					console.error("* Error in WebSocket connection:", error);
				} finally {
					// cacheEntryRemoved will resolve when the cache subscription is no longer active
					await cacheEntryRemoved;
					// perform cleanup steps once the `cacheEntryRemoved` promise resolves
					isStopped = true;
					if (reconnectTimer) clearTimeout(reconnectTimer);
					if (inactivityTimer) clearTimeout(inactivityTimer);
					if (ws) (ws as WebSocket).close();
				}
			},
		}),
	}),
});

//const useSseServerEventsQuery = sseApi.endpoints.getServerEvents.useQuery;

const useWsServerEventsQuery = () => {
	const result = wsApi.endpoints.getServerEvents.useQuery();
	const isConnected = Boolean(result.data?.__wsConnected);
	return {
		...result,
		isLoading: result.isLoading || !isConnected,
	};
};

// Export the hook
/*
export const useGetServerEventsQuery =
	getServerEventBackend() === "SSE"
		? useSseServerEventsQuery
		: useWsServerEventsQuery;
*/

export const useGetServerEventsQuery = useWsServerEventsQuery;