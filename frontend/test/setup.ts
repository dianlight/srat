// Shared test setup for bun:test
// - installs a happy-dom Window as global window/document/localStorage
// - ensures process.env.APIURL is set
// - provides a helper to create a minimal Redux store with RTK Query APIs

import { Window } from "happy-dom";
import { configureStore } from "@reduxjs/toolkit";
import { GlobalRegistrator } from "@happy-dom/global-registrator";
import '@testing-library/jest-dom'

GlobalRegistrator.register();

// Install DOM globals immediately when this module is imported
const win = new Window({
    settings: {
        enableJavaScriptEvaluation: true
    }
});
(globalThis as any).window = win as any;
(globalThis as any).document = win.document as any;
(globalThis as any).HTMLElement = win.HTMLElement as any;
//(globalThis as any).localStorage = win.localStorage as any;

// Mock fetch globally to prevent network calls during tests
(globalThis as any).fetch = async () => {
    return {
        ok: true,
        status: 200,
        json: async () => ({}),
        text: async () => '',
        headers: new Map(),
    };
};

// Ensure APIURL is set so modules that compute API url at import time behave
if (typeof process !== "undefined") {
    process.env = process.env || {};
    process.env.APIURL = process.env.APIURL || "http://localhost:8080";
} else {
    (globalThis as any).process = { env: { APIURL: "http://localhost:8080" } };
}

// Ensure React Testing Library is evaluated before any test runs so that
// it can register its automatic cleanup hooks with Bun's afterEach.
await import("@testing-library/react");

// Create the store after the above globals are set. Do dynamic imports to
// avoid loading modules (that inspect window/process.env at module import)
// before we've set up the test environment.
export async function createTestStore() {
    const { sratApi } = await import("../src/store/sratApi");
    const { sseApi, wsApi } = await import("../src/store/sseApi");

    const reducers: any = { [sratApi.reducerPath]: sratApi.reducer };
    const middlewares: any[] = [sratApi.middleware];
    if (sseApi && sseApi.reducer) {
        reducers[sseApi.reducerPath] = sseApi.reducer;
        middlewares.push(sseApi.middleware);
    }
    if (wsApi && wsApi.reducer) {
        reducers[wsApi.reducerPath] = wsApi.reducer;
        middlewares.push(wsApi.middleware);
    }

    const store = configureStore({
        reducer: reducers,
        middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(...middlewares),
    });
    return store;
}
