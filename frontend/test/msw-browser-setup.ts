/**
 * Browser MSW setup for Vitest browser runs.
 */

import { afterAll, afterEach, beforeAll } from "vitest";
import { setupWorker } from "msw/browser";
import { loadMswRuntime } from "./msw-setup";

type SetupWorkerHandler = Parameters<typeof setupWorker>[number];

let worker: ReturnType<typeof setupWorker> | null = null;
let defaultHandlers: SetupWorkerHandler[] = [];
let clearFilesystemSupportOverrides: () => void = () => {};
let resetApiCounters: () => void = () => {};

beforeAll(async () => {
	const runtime = await loadMswRuntime();
	defaultHandlers = runtime.handlers;
	clearFilesystemSupportOverrides = runtime.clearFilesystemSupportOverrides;
	resetApiCounters = runtime.resetApiCounters;
	worker = setupWorker(...defaultHandlers);
	await worker.start({ onUnhandledRequest: "warn" });

	(globalThis as any).__SRAT_MSW_ADAPTER__ = {
		getMswServer: () => ({
			use: (...handlers: SetupWorkerHandler[]) => getMswWorker().use(...handlers),
			resetHandlers: (...handlers: SetupWorkerHandler[]) =>
				getMswWorker().resetHandlers(...handlers),
			listHandlers: () => [...defaultHandlers],
		}),
		withTestHandlers: <T>(
			handlers: SetupWorkerHandler | SetupWorkerHandler[],
			run: () => Promise<T>,
		) => withTestHandlers(handlers, run),
	};
});

afterEach(() => {
	clearFilesystemSupportOverrides();
	resetApiCounters();
	if (worker) {
		worker.resetHandlers(...defaultHandlers);
	}
});

afterAll(() => {
	if (worker) {
		worker.stop();
	}
});

export function getMswWorker() {
	if (!worker) {
		throw new Error("MSW worker not initialized — call from within a test");
	}
	return worker;
}

export async function withTestHandlers<T>(
	handlers: SetupWorkerHandler | SetupWorkerHandler[],
	run: () => Promise<T>,
): Promise<T> {
	const activeWorker = getMswWorker();
	const scopedHandlers = Array.isArray(handlers) ? handlers : [handlers];
	activeWorker.resetHandlers(...scopedHandlers, ...defaultHandlers);
	try {
		return await run();
	} finally {
		activeWorker.resetHandlers(...defaultHandlers);
	}
}

export const mswWorker = worker;
