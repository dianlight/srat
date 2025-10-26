import type { Middleware } from "@reduxjs/toolkit";
import { mdcSlice } from "./mdcSlice";

// Types for RTK Query action metadata
interface RTKQueryMeta {
	meta?: {
		arg?: {
			originalArgs?: Record<string, unknown>;
		};
		baseQueryMeta?: {
			response?: {
				headers?: Headers | Record<string, string>;
			};
			headers?: Record<string, string>;
		};
	};
}

// Middleware to capture X-* IDs from outgoing RTK Query requests and incoming responses
export const mdcMiddleware: Middleware = (storeApi) => (next) => (action) => {
	// Before the request is processed: look at initiate actions' originalArgs to read headers
	try {
		const meta = (action as RTKQueryMeta)?.meta;
		const args = meta?.arg?.originalArgs;
		if (
			args &&
			(args["X-Request-Id"] || args["X-Span-Id"] || args["X-Trace-Id"])
		) {
			storeApi.dispatch(
				mdcSlice.actions.setAllData({
					spanId: (args["X-Span-Id"] as string | undefined) ?? null,
					traceId: (args["X-Trace-Id"] as string | undefined) ?? null,
				}),
			);
		}
	} catch {
		// ignore
	}

	const result = next(action);

	// After the request resolves: fulfilled/rejected actions include baseQueryMeta with response headers
	try {
		const baseQueryMeta = (action as RTKQueryMeta)?.meta?.baseQueryMeta;
		if (baseQueryMeta) {
			const headers: Headers | Record<string, string> | undefined =
				baseQueryMeta?.response?.headers ?? baseQueryMeta?.headers;

			let spanId: string | null = null;
			let traceId: string | null = null;

			if (headers) {
				// Headers may be a Fetch Headers object or a plain object
				const getHeader = (key: string): string | null => {
					if (typeof (headers as Headers).get === "function") {
						return (headers as Headers).get(key) ?? null;
					}
					const h = headers as Record<string, string>;
					const v =
						h[key] ??
						(h as Record<string, string>)[key.toLowerCase()] ??
						(h as Record<string, string>)[key.toUpperCase()];
					return v ?? null;
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
