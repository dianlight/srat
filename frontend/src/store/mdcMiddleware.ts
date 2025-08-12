import type { Middleware } from "@reduxjs/toolkit";
import { mdcSlice } from "./mdcSlice";

// Middleware to capture X-* IDs from outgoing RTK Query requests and incoming responses
export const mdcMiddleware: Middleware = (storeApi) => (next) => (action) => {
	// Before the request is processed: look at initiate actions' originalArgs to read headers
	try {
		const args = (action as any)?.meta?.arg?.originalArgs;
		if (
			args &&
			(args["X-Request-Id"] || args["X-Span-Id"] || args["X-Trace-Id"])
		) {
			storeApi.dispatch(
				mdcSlice.actions.setAllData({
					spanId: args["X-Span-Id"] ?? null,
					traceId: args["X-Trace-Id"] ?? null,
				}),
			);
		}
	} catch {
		// ignore
	}

	const result = next(action);

	// After the request resolves: fulfilled/rejected actions include baseQueryMeta with response headers
	try {
		const baseQueryMeta = (action as any)?.meta?.baseQueryMeta;
		if (baseQueryMeta) {
			const headers: Headers | Record<string, string> | undefined =
				baseQueryMeta?.response?.headers ?? baseQueryMeta?.headers;

			let spanId: string | null = null;
			let traceId: string | null = null;

			if (headers) {
				// Headers may be a Fetch Headers object or a plain object
				const getHeader = (key: string): string | null => {
					if (typeof (headers as any).get === "function") {
						return ((headers as any).get(key) as string | null) ?? null;
					}
					const h = headers as Record<string, string>;
					const v =
						h[key] ??
						(h as any)[key.toLowerCase()] ??
						(h as any)[key.toUpperCase()];
					return (v as string | undefined) ?? null;
				};

				spanId = getHeader("X-Span-Id");
				traceId = getHeader("X-Trace-Id");
			}

			if (spanId || traceId) {
				storeApi.dispatch(
					mdcSlice.actions.setAllData({
						spanId: spanId ?? null,
						traceId: traceId ?? null,
					}),
				);
			}
		}
	} catch {
		// ignore
	}

	return result;
};

export default mdcMiddleware;
