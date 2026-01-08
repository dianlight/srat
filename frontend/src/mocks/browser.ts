/**
 * MSW Worker setup for browser environment
 * 
 * This file configures MSW to intercept HTTP and WebSocket requests
 * in the browser environment for manual testing and development.
 * 
 * To use this in your app:
 * 1. Run: bunx msw init public/ --save
 * 2. Import this file in your main app entry point (conditionally for dev only)
 * 3. Call startMockWorker() to start intercepting requests
 */

import { setupWorker } from "msw/browser";
import { generatedHandlers } from "./generatedHandlers";
import { streamingHandlers } from "./streamingHandlers";

/**
 * Combine all handlers:
 * - Generated REST API handlers from OpenAPI spec
 * - Manual streaming handlers for SSE and WebSocket
 */
export const handlers = [...generatedHandlers, ...streamingHandlers];

/**
 * Create MSW worker instance for browser
 * 
 * This worker will intercept network requests in the browser
 * and respond with mocked data.
 */
export const worker = setupWorker(...handlers);

/**
 * Start the MSW worker in the browser
 * 
 * Usage in your app entry point (e.g., index.tsx):
 * 
 * if (process.env.NODE_ENV === 'development' && process.env.ENABLE_MSW === 'true') {
 *   import('./mocks/browser').then(({ startMockWorker }) => {
 *     startMockWorker().then(() => {
 *       // Start your app after MSW is ready
 *     });
 *   });
 * }
 */
export async function startMockWorker() {
	if (typeof window === "undefined") {
		throw new Error(
			"MSW browser worker can only be started in a browser environment",
		);
	}

	await worker.start({
		onUnhandledRequest: "warn", // Warn about unhandled requests
		serviceWorker: {
			// Customize the Service Worker URL if needed
			url: "/mockServiceWorker.js",
		},
	});

	console.log("[MSW] Mock Service Worker started in browser");
}

/**
 * Stop the MSW worker in the browser
 */
export function stopMockWorker() {
	worker.stop();
	console.log("[MSW] Mock Service Worker stopped");
}

/**
 * Export the worker instance for advanced usage
 */
export default worker;
