// Or from '@reduxjs/toolkit/query' if not using the auto-generated hooks
import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { getApiUrl } from "../macro/Environment" with { type: 'macro' };
import normalizeUrl from "normalize-url";

export const apiUrl = normalizeUrl((getApiUrl() === "dynamic" ? window.location.href.substring(
	0,
	window.location.href.lastIndexOf("/") + 1
) : getApiUrl()) + "/");

// initialize an empty api service that we'll inject endpoints into later as needed
export const emptySplitApi = createApi({
	baseQuery: fetchBaseQuery({
		baseUrl: apiUrl,
		// Enrich outgoing requests with MDC headers when available in state
		prepareHeaders: (headers, { getState }) => {
			try {
				const state: any = getState();
				const mdc = state?.mdc;
				const setIfMissing = (key: string, val?: string | null) => {
					if (!val) return;
					// Headers implements has()/get()/set()
					if (typeof (headers as any).has === "function") {
						if (!(headers as any).has(key)) (headers as any).set(key, val);
					} else {
						// Fallback for non-Header-like implementations
						(headers as any)[key] ??= val;
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


console.debug("API URL is", apiUrl);
