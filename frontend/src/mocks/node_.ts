/**
 * MSW Server setup for Bun test environment (Node.js runtime)
 * 
 * This file configures MSW to intercept HTTP and WebSocket requests
 * in the Bun test runner environment.
 */

import { setupServer } from "msw/node";
import { customHandlers } from "./customHandlers";
import { streamingHandlers } from "./streamingHandlers";

/**
 * Combine all handlers:
 * - Generated REST API handlers from OpenAPI spec
 * - Manual streaming handlers for SSE and WebSocket
 */
export const handlers = [...customHandlers, ...streamingHandlers];

/**
 * Create MSW server instance
 * 
 * This server will intercept network requests in the Node.js (Bun) environment
 * and respond with mocked data.
 */
export const server = setupServer(...handlers);

/**
 * Start the MSW server
 * Call this in your test setup (e.g., beforeAll hook)
 */
export function startMockServer() {
	server.listen({
		onUnhandledRequest: "warn", // Warn about unhandled requests
	});
}

/**
 * Reset handlers between tests
 * Call this in your afterEach hook to ensure test isolation
 */
export function resetMockServer() {
	server.resetHandlers();
}

/**
 * Stop the MSW server
 * Call this in your test teardown (e.g., afterAll hook)
 */
export function stopMockServer() {
	server.close();
}

/**
 * Export the server instance for advanced usage
 */
export default server;
