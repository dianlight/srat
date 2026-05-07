/**
 * Happy-dom MSW setup for Bun/Vitest runs.
 */

import { afterEach, beforeAll } from "vitest";
import { setupServer } from "msw/node";
import { loadMswRuntime } from "./msw-setup";

type SetupServerHandler = Parameters<typeof setupServer>[number];

let server: ReturnType<typeof setupServer> | null = null;
let defaultHandlers: SetupServerHandler[] = [];
let clearFilesystemSupportOverrides: () => void = () => {};
let resetApiCounters: () => void = () => {};

beforeAll(async () => {
	const runtime = await loadMswRuntime();
	defaultHandlers = runtime.handlers;
	clearFilesystemSupportOverrides = runtime.clearFilesystemSupportOverrides;
	resetApiCounters = runtime.resetApiCounters;
	server = setupServer(...defaultHandlers);
	server.listen({ onUnhandledRequest: "warn" });

	(globalThis as any).__SRAT_MSW_ADAPTER__ = {
		getMswServer: () => getMswServer(),
		withTestHandlers: <T>(
			handlers: SetupServerHandler | SetupServerHandler[],
			run: () => Promise<T>,
		) => withTestHandlers(handlers, run),
	};
});

afterEach(() => {
	clearFilesystemSupportOverrides();
	resetApiCounters();
	if (server) {
		server.resetHandlers(...defaultHandlers);
	}
});

export function getMswServer() {
	if (!server) {
		throw new Error("MSW server not initialized — call from within a test");
	}
	return server;
}

export async function withTestHandlers<T>(
	handlers: SetupServerHandler | SetupServerHandler[],
	run: () => Promise<T>,
): Promise<T> {
	const activeServer = getMswServer();
	const scopedHandlers = Array.isArray(handlers) ? handlers : [handlers];
	activeServer.resetHandlers(...scopedHandlers, ...defaultHandlers);
	try {
		return await run();
	} finally {
		activeServer.resetHandlers(...defaultHandlers);
	}
}

export const mswServer = server;
