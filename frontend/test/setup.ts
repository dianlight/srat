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
        enableJavaScriptEvaluation: true,
        suppressCodeGenerationFromStringsWarning: true
    },
    url: "http://localhost:3000/"
});
(globalThis as any).window = win as any;
(globalThis as any).document = win.document as any;
(globalThis as any).HTMLElement = win.HTMLElement as any;

// Ensure document.body exists for @testing-library/react screen
if (!document.body) {
    document.body = document.createElement('body');
}

// Mock fetch globally to prevent network calls during tests
(globalThis as any).fetch = async (url: string, options?: any) => {
    // Create a proper Response-like object that RTK Query can work with
    const mockResponse = {
        ok: true,
        status: 200,
        statusText: 'OK',
        headers: new Map([['content-type', 'application/json']]),
        json: async () => ({}),
        text: async () => '',
        clone: () => ({ ...mockResponse }), // RTK Query needs this
        arrayBuffer: async () => new ArrayBuffer(0),
        blob: async () => new Blob(),
        formData: async () => new FormData(),
    };
    return mockResponse;
};

// Ensure APIURL is set so modules that compute API url at import time behave
if (typeof process !== "undefined") {
    process.env = process.env || {};
    process.env.API_URL = process.env.API_URL || "http://localhost:8080";
    // Disable @testing-library/react automatic act environment setup
    process.env.RTL_SKIP_AUTO_CLEANUP = "true";
} else {
    (globalThis as any).process = { env: { API_URL: "http://localhost:8080", RTL_SKIP_AUTO_CLEANUP: "true" } };
}

// Configure Bun test environment for React Testing Library
// Override beforeAll/afterAll to no-ops to prevent @testing-library/react from setting up React act environment
// that conflicts with Bun's test runner
(globalThis as any).beforeAll = () => { };
(globalThis as any).afterAll = () => { };

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
