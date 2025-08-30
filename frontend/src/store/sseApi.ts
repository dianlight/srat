import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
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

// Define SSE event types based on backend
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
			/*
            const response = await fetch(url, options);
            if (!response.ok) {
                throw new Error("Network response was not ok");
            }
            return response;
            */
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
				console.log("* Starting SSE connection");
				const eventSource = new EventSource(`${apiUrl}/api/sse`, {
					withCredentials: true,
				});
				try {
					// wait for the initial query to resolve before proceeding
					await cacheDataLoaded;
					let faultCount = 0;

					eventSource.addEventListener("error", (event) => {
						console.warn(`* SSE connection error ${faultCount}`, event);
						/*
                        this.heartbeatListener.forEach((func) => {
                            try {
                                func({ data: '{ "alive": false, "read_only": true }' });
                            } catch (error) {
                                console.error("Error in heartbeat listener", error);
                            }
                        });
                        */
						faultCount++;
					});
					eventSource.addEventListener("open", (event) => {
						console.debug("* SSE connection open", event);
						faultCount = 0;

						Object.values(Supported_events).forEach((event) => {
							eventSource.addEventListener(event, (data) => {
								updateCachedData((draft) => {
									console.log(
										`* SSE event ${event} received:`,
										event,
										data,
										draft[event],
									);
									if (draft !== undefined && draft !== null) {
										console.log(`* Updating draft for event ${event}`);
										draft[event] = JSON.parse(data.data);
									}
								});
							});
						});
					});

					/*
                    const listener = (event: MessageEvent) => {
                        const data = JSON.parse(event.data)
                        if (!isMessage(data) || data.channel !== arg) return

                        updateCachedData((draft) => {
                            draft.push(data)
                        })
                    }

                    ws.addEventListener('message', listener)
                    */
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

// Export the hook
export const { useGetServerEventsQuery } = sseApi;
