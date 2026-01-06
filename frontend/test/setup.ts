// Shared test setup for bun:test
// - installs a happy-dom Window as global window/document/localStorage
// - ensures process.env.APIURL is set
// - provides a helper to create a minimal Redux store with RTK Query APIs

import { Window } from "happy-dom";
import { configureStore } from "@reduxjs/toolkit";
import '@testing-library/jest-dom'

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
(globalThis as any).navigator = win.navigator as any;
(globalThis as any).location = win.location as any;

// Expose common DOM classes/globals used by libraries (MUI, Popper, Prism, etc.)
(globalThis as any).Node = win.Node as any;
(globalThis as any).Element = win.Element as any;
(globalThis as any).Document = (win as any).Document || win.document.constructor;
(globalThis as any).DocumentFragment = win.DocumentFragment as any;
(globalThis as any).HTMLElement = win.HTMLElement as any;
(globalThis as any).SVGElement = win.SVGElement as any;
(globalThis as any).Text = win.Text as any;
(globalThis as any).Comment = win.Comment as any;
(globalThis as any).ShadowRoot = (win as any).ShadowRoot as any;
(globalThis as any).customElements = (win as any).customElements as any;
(globalThis as any).getComputedStyle = win.getComputedStyle.bind(win);
// Mark environment as test for components that conditionally load heavy browser-only modules
; (globalThis as any).__TEST__ = true;

// Polyfill CSSStyleSheet + adoptedStyleSheets used by lit/openapi-explorer
if (!(globalThis as any).CSSStyleSheet) {
    class CSSStyleSheetPolyfill {
        private _text = "";
        replaceSync(text: string) { this._text = String(text ?? ""); return this; }
        async replace(text: string) { this.replaceSync(text); return this; }
        toString() { return this._text; }
    }
    (globalThis as any).CSSStyleSheet = CSSStyleSheetPolyfill as any;
}
// Define adoptedStyleSheets on Document/ShadowRoot if missing
const defineAdoptedStyleSheets = (proto: any) => {
    if (!proto) return;
    if (!('adoptedStyleSheets' in proto)) {
        let _sheets: any[] = [];
        Object.defineProperty(proto, 'adoptedStyleSheets', {
            get() { return _sheets; },
            set(v: any[]) { _sheets = Array.isArray(v) ? v : []; },
            configurable: true,
        });
    }
};
defineAdoptedStyleSheets((globalThis as any).Document?.prototype);
defineAdoptedStyleSheets((globalThis as any).ShadowRoot?.prototype);

// Polyfills/stubs for browser-only APIs referenced in tests/components
if (!(globalThis as any).matchMedia) {
    (globalThis as any).matchMedia = (query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: () => { /* deprecated */ },
        removeListener: () => { /* deprecated */ },
        addEventListener: () => { },
        removeEventListener: () => { },
        dispatchEvent: () => true,
    });
}

if (!(globalThis as any).ResizeObserver) {
    (globalThis as any).ResizeObserver = class {
        observe() { }
        unobserve() { }
        disconnect() { }
    } as any;
}

if (!(globalThis as any).IntersectionObserver) {
    (globalThis as any).IntersectionObserver = class {
        constructor(_cb: any, _options?: any) { }
        observe() { }
        unobserve() { }
        disconnect() { }
        takeRecords() { return []; }
    } as any;
}

if (!(globalThis as any).requestAnimationFrame) {
    (globalThis as any).requestAnimationFrame = (cb: (ts: number) => void) => setTimeout(() => cb(Date.now()), 0);
}
if (!(globalThis as any).cancelAnimationFrame) {
    (globalThis as any).cancelAnimationFrame = (id: any) => clearTimeout(id);
}

// Happy-DOM can throw on duplicate/unmatched removeChild during React strict/effects.
// Make it tolerant to avoid cross-test unmount noise.
try {
    const originalRemoveChild = (globalThis as any).Node?.prototype?.removeChild;
    if (originalRemoveChild && !('__patched_removeChild' in (globalThis as any).Node.prototype)) {
        Object.defineProperty((globalThis as any).Node.prototype, '__patched_removeChild', { value: true });
        (globalThis as any).Node.prototype.removeChild = function(child: any) {
            try { return originalRemoveChild.call(this, child); }
            catch { return child; }
        };
    }
} catch { /* ignore */ }

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

// Do not override test runner lifecycle hooks; tests rely on them.

// Create the store after the above globals are set. Do dynamic imports to
// avoid loading modules (that inspect window/process.env at module import)
// before we've set up the test environment.
export async function createTestStore() {
    // const { emptySplitApi: api } = await import("../src/store/emptyApi");
    const { sseApi, wsApi } = await import("../src/store/sseApi");
    const { sratApi } = await import("../src/store/sratApi");
    const { errorSlice } = await import("../src/store/errorSlice");
    const { mdcSlice } = await import("../src/store/mdcSlice");
    const mdcMiddleware = (await import("../src/store/mdcMiddleware")).default;
    const { setupListeners } = await import("@reduxjs/toolkit/query");

    // CRITICAL: Create a fresh store with RTK Query middleware
    // Each test must have its own completely isolated store to prevent
    // subscription state from leaking between tests
    const store = configureStore({
        reducer: {
            errors: errorSlice.reducer,
            mdc: mdcSlice.reducer,
            //  [api.reducerPath]: api.reducer,
            [sratApi.reducerPath]: sratApi.reducer,
            [sseApi.reducerPath]: sseApi.reducer,
            [wsApi.reducerPath]: wsApi.reducer,
        },
        middleware: (getDefaultMiddleware) =>
            getDefaultMiddleware()
                .concat(mdcMiddleware)
                .concat(sratApi.middleware)
                .concat(sseApi.middleware)
                .concat(wsApi.middleware),
    });

    // Setup listeners - CRITICAL for RTK Query to work properly!
    // This initializes connection tracking for RTK Query subscription handling
    setupListeners(store.dispatch);

    // Clear RTK Query caches and subscription state after store is created
    // This needs to happen after the store is created and listeners are set up
    try {
        // Reset all API state to ensure clean slate for cache
        store.dispatch(sratApi.util.resetApiState());
        store.dispatch(sseApi.util.resetApiState());
        store.dispatch(wsApi.util.resetApiState());
    } catch {
        // Silently ignore errors in case dispatch is not fully ready
    }

    return store;
}
