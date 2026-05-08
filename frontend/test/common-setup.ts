/**
 * Shared frontend test setup for all Vitest environments.
 */

import { configureStore } from "@reduxjs/toolkit";
import * as matchers from "@testing-library/jest-dom/matchers";
import { afterEach } from "vitest";
import { expect } from "vitest";

expect.extend(matchers);

if (!(globalThis as any).__TEST__) {
	(globalThis as any).__TEST__ = true;
}

if (!(globalThis as any).Bun) {
	(globalThis as any).Bun = {
		spawnSync: () => ({ stdout: Buffer.from("test-commit-hash") }),
	};
}

const nativeGlobals = {
	AbortController: (globalThis as any).AbortController,
	AbortSignal: (globalThis as any).AbortSignal,
	BroadcastChannel: (globalThis as any).BroadcastChannel,
	WebSocket: (globalThis as any).WebSocket,
};

const win = (globalThis as any).window as Window & typeof globalThis;

if (!(win as any).SyntaxError) {
	(win as any).SyntaxError = globalThis.SyntaxError;
}
(globalThis as any).SyntaxError = (win as any).SyntaxError;

(globalThis as any).Node = win.Node as any;
(globalThis as any).Element = win.Element as any;
(globalThis as any).Document = (win as any).Document || win.document.constructor;
(globalThis as any).DocumentFragment = win.DocumentFragment as any;
(globalThis as any).HTMLElement = win.HTMLElement as any;
(globalThis as any).SVGElement = win.SVGElement as any;
(globalThis as any).Text = win.Text as any;
(globalThis as any).Comment = win.Comment as any;
(globalThis as any).ShadowRoot = (win as any).ShadowRoot as any;
if (!(globalThis as any).customElements && (win as any).customElements) {
	(globalThis as any).customElements = (win as any).customElements as any;
}
(globalThis as any).getComputedStyle = win.getComputedStyle.bind(win);

const currentFetch = (globalThis as any).fetch ?? (win as any).fetch;
if (currentFetch) {
	Object.defineProperty(globalThis, "fetch", {
		value: currentFetch,
		writable: true,
		configurable: true,
	});
	Object.defineProperty(win, "fetch", {
		value: currentFetch,
		writable: true,
		configurable: true,
	});
}

const makeWritableGlobal = (target: any, key: string, value: unknown) => {
	if (!value) return;
	Object.defineProperty(target, key, {
		value,
		writable: true,
		configurable: true,
	});
};

(globalThis as any).AbortController = nativeGlobals.AbortController;
(globalThis as any).AbortSignal = nativeGlobals.AbortSignal;
(globalThis as any).BroadcastChannel = nativeGlobals.BroadcastChannel;
(globalThis as any).WebSocket = nativeGlobals.WebSocket;
makeWritableGlobal(globalThis, "BroadcastChannel", nativeGlobals.BroadcastChannel);
makeWritableGlobal(globalThis, "WebSocket", nativeGlobals.WebSocket);
makeWritableGlobal(win, "BroadcastChannel", nativeGlobals.BroadcastChannel);
makeWritableGlobal(win, "WebSocket", nativeGlobals.WebSocket);

(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;
(win as any).IS_REACT_ACT_ENVIRONMENT = true;

if (!(globalThis as any).matchMedia) {
	const createMatchMediaMock = (query: string) => ({
		matches: false,
		media: query,
		onchange: null,
		addListener: () => {},
		removeListener: () => {},
		addEventListener: () => {},
		removeEventListener: () => {},
		dispatchEvent: () => true,
	});

	(globalThis as any).matchMedia = createMatchMediaMock;
}

if (!(win as any).matchMedia) {
	const createMatchMediaMock = (query: string) => ({
		matches: false,
		media: query,
		onchange: null,
		addListener: () => {},
		removeListener: () => {},
		addEventListener: () => {},
		removeEventListener: () => {},
		dispatchEvent: () => true,
	});

	(win as any).matchMedia = createMatchMediaMock;
}

if (!(globalThis as any).ResizeObserver) {
	(globalThis as any).ResizeObserver = class {
		observe() {}
		unobserve() {}
		disconnect() {}
	} as any;
}

if (!(globalThis as any).IntersectionObserver) {
	(globalThis as any).IntersectionObserver = class {
		constructor(_cb: any, _options?: any) {}
		observe() {}
		unobserve() {}
		disconnect() {}
		takeRecords() {
			return [];
		}
	} as any;
}

if (!(globalThis as any).requestAnimationFrame) {
	(globalThis as any).requestAnimationFrame = (cb: (ts: number) => void) =>
		setTimeout(() => cb(Date.now()), 0);
}
if (!(globalThis as any).cancelAnimationFrame) {
	(globalThis as any).cancelAnimationFrame = (id: any) => clearTimeout(id);
}

if (!document.body) {
	document.body = document.createElement("body");
}

// Ensure localStorage is properly set up with all required methods
// happy-dom creates localStorage, but we need to ensure all methods exist
const localStorage = (globalThis as any).localStorage || {};
if (!localStorage.getItem || typeof localStorage.getItem !== 'function') {
	const store: Record<string, string> = {};
	const localStorageImpl = {
		getItem: (key: string) => (key in store ? store[key] : null),
		setItem: (key: string, value: string) => {
			store[key] = String(value);
		},
		removeItem: (key: string) => {
			delete store[key];
		},
		clear: () => {
			for (const key of Object.keys(store)) delete store[key];
		},
	};
	(globalThis as any).localStorage = localStorageImpl;
	if (typeof window !== "undefined") {
		(window as any).localStorage = localStorageImpl;
	}
}

if (typeof process !== "undefined") {
	process.env = process.env || {};
	process.env.API_URL = "http://localhost:3000";
	process.env.RTL_SKIP_AUTO_CLEANUP = "true";
} else {
	(globalThis as any).process = {
		env: {
			API_URL: "http://localhost:3000",
			RTL_SKIP_AUTO_CLEANUP: "true",
		},
	};
}

const testListenerCleanupRegistry = ((globalThis as any).__SRAT_TEST_LISTENER_CLEANUPS__ ??=
	new Set<() => void>()) as Set<() => void>;

afterEach(async () => {
	const { cleanup } = await import("@testing-library/react");
	cleanup();

	const listenerCleanups = (globalThis as any).__SRAT_TEST_LISTENER_CLEANUPS__ as
		| Set<() => void>
		| undefined;
	if (listenerCleanups) {
		for (const listenerCleanup of listenerCleanups) {
			try {
				listenerCleanup();
			} catch {
				// Best-effort teardown for test-only listeners.
			}
		}
		listenerCleanups.clear();
	}

	document.querySelectorAll(".MuiModal-root").forEach((el) =>
		el.parentNode?.removeChild(el),
	);
});

let cachedApiModules: {
	sratApi: any;
	wsApi: any;
	githubRestApi: any;
	errorSlice: any;
	mdcSlice: any;
	mdcMiddleware: any;
} | null = null;

export async function createTestStore() {
	if (!cachedApiModules) {
		const [
			{ wsApi },
			{ sratApi },
			{ githubRestApi },
			{ errorSlice },
			{ mdcSlice },
			mdcMiddlewareModule,
		] = await Promise.all([
			import("../src/store/wsApi"),
			import("../src/store/sratApi"),
			import("../src/store/githubRestApi"),
			import("../src/store/errorSlice"),
			import("../src/store/mdcSlice"),
			import("../src/store/mdcMiddleware"),
		]);

		cachedApiModules = {
			sratApi,
			wsApi,
			githubRestApi,
			errorSlice,
			mdcSlice,
			mdcMiddleware: mdcMiddlewareModule.default,
		};
	}

	const { sratApi, wsApi, githubRestApi, errorSlice, mdcSlice, mdcMiddleware } =
		cachedApiModules;
	const { setupListeners } = await import("@reduxjs/toolkit/query");

	const store = configureStore({
		reducer: {
			errors: errorSlice.reducer,
			mdc: mdcSlice.reducer,
			[sratApi.reducerPath]: sratApi.reducer,
			[wsApi.reducerPath]: wsApi.reducer,
			[githubRestApi.reducerPath]: githubRestApi.reducer,
		},
		middleware: (getDefaultMiddleware) =>
			getDefaultMiddleware()
				.concat(mdcMiddleware)
				.concat(sratApi.middleware)
				.concat(wsApi.middleware)
				.concat(githubRestApi.middleware),
	});

	const unsubscribeListeners = setupListeners(store.dispatch);
	if (typeof unsubscribeListeners === "function") {
		testListenerCleanupRegistry.add(unsubscribeListeners);
	}

	try {
		store.dispatch(sratApi.util.resetApiState());
		store.dispatch(wsApi.util.resetApiState());
		store.dispatch(githubRestApi.util.resetApiState());
	} catch (error) {
		console.warn("[createTestStore] RTK Query resetApiState failed:", error);
	}

	return store;
}
