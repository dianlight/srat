/**
 * Bun test setup for MSW (Mock Service Worker)
 *
 * This file manages the MSW server lifecycle for Bun tests.
 * It integrates with the existing test setup and provides hooks
 * for starting, resetting, and stopping the mock server.
 *
 * Usage:
 * Import this file in your test setup or individual test files:
 * import './test/bun-setup'; // or import from 'path/to/bun-setup.ts'
 */

import { afterEach, beforeAll } from "bun:test";
import { setupServer } from "msw/node";

type SetupServerHandler = Parameters<typeof setupServer>[number];

// Server and handlers are set up lazily in beforeAll because handler modules
// access `window` (set up by GlobalRegistrator in setup.ts) and cannot be
// statically imported at parse time.
let server: ReturnType<typeof setupServer> | null = null;
let defaultHandlers: SetupServerHandler[] = [];
let clearFilesystemSupportOverrides: () => void = () => { };
let resetApiCounters: () => void = () => { };

/**
 * Start MSW server before all tests.
 * Handler imports are parallelized with Promise.all to minimize startup latency.
 */
beforeAll(async () => {
	const [
		{ handlers: generatedHandlers, resetApiCounters: resetFn },
		{ customHandlers, clearFilesystemSupportOverrides: clearFn },
		{ streamingHandlers },
	] = await Promise.all([
		import("../src/mocks/handlers"),
		import("../src/mocks/customHandlers"),
		import("../src/mocks/streamingHandlers"),
	]);

	clearFilesystemSupportOverrides = clearFn;
	resetApiCounters = resetFn;
	defaultHandlers = [...customHandlers, ...streamingHandlers, ...generatedHandlers];
	server = setupServer(...defaultHandlers);
	server.listen({ onUnhandledRequest: "warn" });
});

/**
 * Reset handlers after each test to prevent cross-test leakage.
 * cleanup() is imported dynamically to ensure RTL screen is bound to
 * the happy-dom globals that GlobalRegistrator.register() sets up.
 * (Module is cached after first call, so subsequent imports are free.)
 */
afterEach(async () => {
	const { cleanup } = await import("@testing-library/react");
	cleanup();
	clearFilesystemSupportOverrides();
	resetApiCounters();
	if (server) {
		server.resetHandlers(...defaultHandlers);
	}
	// Remove any MUI Modal portal containers (e.g., Dialog, Drawer) left over by
	// in-progress CSS exit transitions that happy-dom does not automatically
	// complete. The React tree is already unmounted by cleanup() above, so these
	// are orphaned DOM nodes and safe to remove.
	document.querySelectorAll(".MuiModal-root").forEach((el) => el.parentNode?.removeChild(el));
});

/**
 * Return the shared MSW server for advanced per-test handler overrides.
 *
 * Example:
 *   import { getMswServer } from './test/bun-setup';
 *   import { http, HttpResponse } from 'msw';
 *
 *   test('custom handler', async () => {
 *     const srv = getMswServer();
 *     srv.use(http.get('/api/custom', () => HttpResponse.json({ custom: 'data' })));
 *     // ... rest of test
 *   });
 */
export function getMswServer() {
	if (!server) {
		throw new Error("MSW server not initialized — call from within a test");
	}
	return server;
}

/**
 * Run a test block with isolated MSW handlers.
 *
 * Replaces runtime handlers for the duration of the callback, then restores
 * defaults, preventing cross-test interference from shared handler state.
 */
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
