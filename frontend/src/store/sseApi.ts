import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { getServerEventBackend } from "../macro/Environment" with {
	type: "macro",
};
import { apiUrl } from "./emptyApi";
import type {
	Disk,
	HealthPing,
	SharedResource,
	UpdateProgress,
	Welcome,
} from "./sratApi";
import { Supported_events } from "./sratApi";

export type EventData = {
	[Supported_events.Heartbeat]: HealthPing;
	[Supported_events.Volumes]: Disk[];
	[Supported_events.Share]: SharedResource[];
	[Supported_events.Hello]: Welcome;
	[Supported_events.Updating]: UpdateProgress;
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
				const eventSource = new EventSource(`${apiUrl}/api/sse`, {
					withCredentials: true,
				});
				try {
					// wait for the initial query to resolve before proceeding
					await cacheDataLoaded;
					let faultCount = 0;

					eventSource.addEventListener("error", (event) => {
						console.warn(`* SSE connection error ${faultCount}`, event);
						faultCount++;
					});
					eventSource.addEventListener("open", (_event) => {
						//console.debug("* SSE connection open", event);
						faultCount = 0;

						Object.values(Supported_events).forEach((event) => {
							eventSource.addEventListener(event, (data) => {
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
					});
				} catch (error) {
					// no-op in case `cacheEntryRemoved` resolves before `cacheDataLoaded`,
					// in which case `cacheDataLoaded` will throw
					console.error("* Error in SSE connection:", error);
				}
				// cacheEntryRemoved will resolve when the cache subscription is no longer active
				await cacheEntryRemoved;
				// perform cleanup steps once the `cacheEntryRemoved` promise resolves
				eventSource.close();
			},
		}),
	}),
});

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
				let ws: WebSocket = undefined as unknown as WebSocket;
				try {
					// wait for the initial query to resolve before proceeding
					await cacheDataLoaded;

					//console.log("* Starting WebSocket connection");
					// create a websocket connection when the cache subscription starts
					ws = new WebSocket(`${apiUrl.replace(/^http/, "ws")}/ws`);
					// when data is received from the socket connection to the server,
					// if it is a message and for the appropriate channel,
					// update our query result with the received message
					const listener = (event: MessageEvent) => {
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
				} catch (error) {
					// no-op in case `cacheEntryRemoved` resolves before `cacheDataLoaded`,
					// in which case `cacheDataLoaded` will throw
					console.error("* Error in WebSocket connection:", error);
				} finally {
					// cacheEntryRemoved will resolve when the cache subscription is no longer active
					await cacheEntryRemoved;
					// perform cleanup steps once the `cacheEntryRemoved` promise resolves
					if (ws) ws.close();
				}
			},
		}),
	}),
});

// Export the hook
export const { useGetServerEventsQuery } =
	getServerEventBackend() === "SSE" ? sseApi : wsApi;
