// Or from '@reduxjs/toolkit/query' if not using the auto-generated hooks
import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import normalizeUrl from "normalize-url";
import { getApiUrl } from "../macro/Environment" with { type: "macro" };

export const apiUrl = normalizeUrl(
	`${
		getApiUrl() === "dynamic"
			? window.location.href.substring(
					0,
					window.location.href.lastIndexOf("/") + 1,
				)
			: getApiUrl()
	}/`,
);

// initialize an empty api service that we'll inject endpoints into later as needed
export const emptySplitApi = createApi({
	baseQuery: fetchBaseQuery({
		baseUrl: apiUrl,
		// Enrich outgoing requests with MDC headers when available in state
		prepareHeaders: (headers, { getState }) => {
			try {
				const state = getState() as { mdc?: { traceId?: string | null } };
				const mdc = state?.mdc;
				const setIfMissing = (key: string, val?: string | null) => {
					if (!val) return;
					// Headers implements has()/get()/set()
					const isHeaders = (obj: unknown): obj is Headers =>
						typeof obj === "object" &&
						obj !== null &&
						"has" in obj &&
						typeof (obj as Headers).has === "function";
					if (isHeaders(headers)) {
						if (!headers.has(key)) headers.set(key, val);
					} else {
						// Fallback for non-Header-like implementations
						const headerObj = headers as Record<string, string>;
						if (!(key in headerObj)) {
							headerObj[key] = val;
						}
					}
				};
				const spanId =
					typeof globalThis.crypto?.randomUUID === "function"
						? globalThis.crypto.randomUUID()
						: "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
								const r = (Math.random() * 16) | 0;
								const v = c === "x" ? r : (r & 0x3) | 0x8;
								return v.toString(16);
							});
				setIfMissing("X-Span-Id", spanId);
				setIfMissing("X-Trace-Id", mdc?.traceId); // Don't touch need to identify a transaction
			} catch (err) {
				console.warn("Failed to set MDC headers, continuing without them", err);
			}
			return headers;
		},
	}),
	endpoints: () => ({}),
});

//console.debug("API URL is", apiUrl);
