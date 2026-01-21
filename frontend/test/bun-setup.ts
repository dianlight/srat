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

import { beforeAll, afterEach, afterAll } from "bun:test";
import { setupServer } from "msw/node";

// We'll initialize the server later to avoid circular imports
let server: ReturnType<typeof setupServer> | null = null;

/**
 * Start MSW server before all tests
 * 
 * This sets up request interception for all network calls made during tests.
 * The server will automatically respond with mocked data from the handlers.
 */
beforeAll(async () => {
	// Dynamic import to avoid loading store modules before globals are set up
	const { handlers: generatedHandlers } = await import("../src/mocks/handlers");
	const { customHandlers } = await import("../src/mocks/customHandlers");
	const { streamingHandlers } = await import("../src/mocks/streamingHandlers");
	
	const handlers = [...generatedHandlers, ...streamingHandlers, ...customHandlers];
	server = setupServer(...handlers);
	
	server.listen({
		onUnhandledRequest: "warn", // Warn about unhandled requests
	});
	
	console.log("[MSW] Mock server started for Bun tests");
});

/**
 * Reset handlers after each test
 * 
 * This ensures that any runtime handler modifications made during a test
 * don't leak into other tests, maintaining test isolation.
 */
afterEach(() => {
	if (server) {
		server.resetHandlers();
	}
});

/**
 * Stop MSW server after all tests
 * 
 * This cleans up the server instance and removes request interception.
 */
afterAll(() => {
	if (server) {
		server.close();
		console.log("[MSW] Mock server stopped");
	}
});

/**
 * Export MSW server for advanced test usage
 * 
 * You can use this to add custom handlers for specific tests:
 * 
 * import { getMswServer } from './test/bun-setup';
 * import { http, HttpResponse } from 'msw';
 * 
 * test('specific test with custom handler', async () => {
 *   const server = await getMswServer();
 *   server.use(
 *     http.get('/api/custom', () => {
 *       return HttpResponse.json({ custom: 'data' });
 *     })
 *   );
 *   // ... rest of test
 * });
 */
export async function getMswServer() {
	// Wait for beforeAll to complete if needed
	let retries = 0;
	while (!server && retries < 10) {
		await new Promise(resolve => setTimeout(resolve, 100));
		retries++;
	}
	if (!server) {
		throw new Error("MSW server not initialized");
	}
	return server;
}

export const mswServer = server;
