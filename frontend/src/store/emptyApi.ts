// Or from '@reduxjs/toolkit/query' if not using the auto-generated hooks
import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import normalizeUrl from "normalize-url";

let APIURL = process.env.APIURL;
console.debug("Configuration APIURL", APIURL);
async function testURL(url: string): Promise<boolean> {
	try {
		const _parsedUrl = new URL(url);
		return await fetch(url, { method: "GET" })
			.then((response) => {
				if (response.ok) {
					console.log(`API URL is reachable: ${url}`);
					return true;
				} else {
					console.error(`API URL is not reachable: ${url}`);
					return false;
				}
			})
			.catch((error) => {
				console.error(`Error fetching API URL: ${error}`);
				return false;
			});
	} catch (_e) {
		return false;
	}
}

if (APIURL === undefined || APIURL === "" || APIURL === "dynamic") {
	APIURL = window.location.href.substring(
		0,
		window.location.href.lastIndexOf("/"),
	);
	console.info(
		`Dynamic APIURL provided, using generated: ${APIURL}/ from ${window.location.href}`,
	);
	if (!(await testURL(`${APIURL}/api/health`))) {
		APIURL = "http://localhost:8080";
		console.error(
			"APIURL is not reachable, using default: http://localhost:8080",
		);
	}
}

/*
// test if APIURL is set and if is reaceable
if (APIURL === undefined || APIURL === "") {
	console.error("APIURL is not set, using default: http://localhost:8080");
	APIURL = "http://localhost:8080";
} else if (process.env.APIURL === "dynamic" || !(await testURL(APIURL))) {
	APIURL = window.location.href.substring(0, window.location.href.lastIndexOf('/'));
	console.info(`Dynamic APIURL provided, using generated: ${APIURL}/ from ${window.location.href}`)
}
	*/
console.log("* API URL", `${APIURL}`, "Reachable: ", await testURL(APIURL));

// initialize an empty api service that we'll inject endpoints into later as needed
export const emptySplitApi = createApi({
	baseQuery: fetchBaseQuery({
		baseUrl: APIURL,
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

export const apiUrl = normalizeUrl(`${APIURL}/`);
