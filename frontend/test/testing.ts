/**
 * Shared test helpers for both happy-dom and browser Vitest runs.
 * This module intentionally avoids the "setup" name so tests don't import setup entrypoints.
 */

import type { ReactElement } from "react";
import { createTestStore } from "./common-setup";

export { createTestStore };

type MswAdapter = {
	getMswServer: () => any;
	withTestHandlers: <T>(handlers: any | any[], run: () => Promise<T>) => Promise<T>;
};

function getAdapter(): MswAdapter {
	const adapter = (globalThis as any).__SRAT_MSW_ADAPTER__ as MswAdapter | undefined;
	if (!adapter) {
		throw new Error("MSW adapter not initialized. Ensure Vitest setupFiles loaded.");
	}
	return adapter;
}

export function getMswServer() {
	return getAdapter().getMswServer();
}

export async function withTestHandlers<T>(
	handlers: any | any[],
	run: () => Promise<T>,
): Promise<T> {
	return getAdapter().withTestHandlers(handlers, run);
}

type TestStore = Awaited<ReturnType<typeof createTestStore>>;

type RenderWithTestStoreOptions = {
	seedStore?: (store: TestStore) => void | Promise<void>;
};

export async function renderWithTestStore(
	element: ReactElement,
	options?: RenderWithTestStoreOptions,
) {
	const React = await import("react");
	const { render } = await import("@testing-library/react");
	const { Provider } = await import("react-redux");

	const store = await createTestStore();
	await options?.seedStore?.(store);

	const result = render(React.createElement(Provider, { store, children: element }));
	return { ...result, store };
}
