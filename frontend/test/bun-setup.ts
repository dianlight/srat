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
import { server, startMockServer, resetMockServer, stopMockServer } from "../src/mocks/node";

/**
 * Start MSW server before all tests
 * 
 * This sets up request interception for all network calls made during tests.
 * The server will automatically respond with mocked data from the handlers.
 */
beforeAll(() => {
	startMockServer();
	console.log("[MSW] Mock server started for Bun tests");
});

/**
 * Reset handlers after each test
 * 
 * This ensures that any runtime handler modifications made during a test
 * don't leak into other tests, maintaining test isolation.
 */
afterEach(() => {
	resetMockServer();
});

/**
 * Stop MSW server after all tests
 * 
 * This cleans up the server instance and removes request interception.
 */
afterAll(() => {
	stopMockServer();
	console.log("[MSW] Mock server stopped");
});

/**
 * Export MSW server for advanced test usage
 * 
 * You can use this to add custom handlers for specific tests:
 * 
 * import { mswServer } from './test/bun-setup';
 * import { http, HttpResponse } from 'msw';
 * 
 * test('specific test with custom handler', () => {
 *   mswServer.use(
 *     http.get('/api/custom', () => {
 *       return HttpResponse.json({ custom: 'data' });
 *     })
 *   );
 *   // ... rest of test
 * });
 */
export const mswServer = server;

/**
 * Helper function to enable/disable specific handlers during tests
 */
export function useMockHandlers(...handlers: Parameters<typeof server.use>) {
	server.use(...handlers);
}

/**
 * Helper function to restore default handlers
 */
export function restoreHandlers() {
	server.restoreHandlers();
}

export { startMockServer, resetMockServer, stopMockServer };
