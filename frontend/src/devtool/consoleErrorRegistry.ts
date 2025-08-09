/**
 * Console Error Callback Registry
 *
 * Allows registering callbacks that run asynchronously whenever
 * console.error is called. Callbacks receive the same arguments
 * passed to console.error (compatible with console.log style).
 *
 * Behavior:
 * - The original console.error is always invoked first.
 * - Callbacks are dispatched in a microtask to avoid affecting
 *   React render/error flows.
 * - Patching is idempotent and applied on first registration.
 */

export type ConsoleErrorCallback = (...args: unknown[]) => void;

let patched = false;
let originalConsoleError: ((...args: unknown[]) => void) | null = null;
const listeners = new Set<ConsoleErrorCallback>();

function ensurePatched() {
	if (patched) return;
	if (typeof console === "undefined" || typeof console.error !== "function") {
		patched = true; // nothing to patch
		return;
	}

	originalConsoleError = console.error.bind(console);
	// Accessing dynamic property on console; cast to unknown then to Record
	(console as unknown as Record<string, (...args: unknown[]) => void>).error = (
		...args: unknown[]
	) => {
		try {
			originalConsoleError?.(...args);
		} finally {
			if (listeners.size > 0) {
				const snapshot = Array.from(listeners);
				queueMicrotask(() => {
					for (const cb of snapshot) {
						try {
							cb(...args);
						} catch {
							// prevent callback errors from cascading
						}
					}
				});
			}
		}
	};

	patched = true;
}

/** Register a callback; returns an unsubscribe function. */
export function registerConsoleErrorCallback(
	cb: ConsoleErrorCallback,
): () => void {
	ensurePatched();
	listeners.add(cb);
	let active = true;
	return () => {
		if (!active) return;
		active = false;
		listeners.delete(cb);
	};
}

/** Count of active callbacks (diagnostics/testing). */
export function getConsoleErrorCallbackCount(): number {
	return listeners.size;
}

/** Force-enable the console.error patch. Usually not needed. */
export function enableConsoleErrorPatch(): void {
	ensurePatched();
}

/** Restore original console.error. Use for tests only. */
export function disableConsoleErrorPatch(): void {
	if (!patched) return;
	if (originalConsoleError) {
		(console as unknown as Record<string, (...args: unknown[]) => void>).error =
			originalConsoleError;
	}
	patched = false;
}
