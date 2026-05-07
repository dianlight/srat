/**
 * Shared MSW runtime loader for frontend tests.
 */

import type { RequestHandler, WebSocketHandler } from "msw";

type MswHandler = RequestHandler | WebSocketHandler;

export type MswRuntime = {
	handlers: MswHandler[];
	clearFilesystemSupportOverrides: () => void;
	resetApiCounters: () => void;
};

export async function loadMswRuntime(): Promise<MswRuntime> {
	const [
		{ handlers: generatedHandlers, resetApiCounters },
		{ customHandlers, clearFilesystemSupportOverrides },
		{ streamingHandlers },
	] = await Promise.all([
		import("../src/mocks/handlers"),
		import("../src/mocks/customHandlers"),
		import("../src/mocks/streamingHandlers"),
	]);

	return {
		handlers: [...customHandlers, ...streamingHandlers, ...generatedHandlers],
		clearFilesystemSupportOverrides,
		resetApiCounters,
	};
}
