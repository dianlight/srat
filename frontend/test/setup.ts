// Shared test setup for bun:test
// - installs a happy-dom Window as global window/document/localStorage
// - ensures process.env.APIURL is set
// - provides a helper to create a minimal Redux store with RTK Query APIs

import { configureStore } from "@reduxjs/toolkit";
import '@testing-library/jest-dom';
import { Window } from "happy-dom";
// Initialize MSW for API mocking
// This must be imported after all globals are set up
import "./bun-setup";

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

// happy-dom often returns zero-sized rectangles, which causes MUI Popover/Menu
// to warn that anchorEl is not part of the document layout. Provide a stable,
// non-zero default rectangle for test environments.
if ((globalThis as any).HTMLElement?.prototype) {
    (globalThis as any).HTMLElement.prototype.getBoundingClientRect = () => ({
        x: 0,
        y: 0,
        top: 0,
        left: 0,
        right: 120,
        bottom: 40,
        width: 120,
        height: 40,
        toJSON: () => ({}),
    });
}

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
// Create a matchMedia mock that always reports large screen (matches: false for small-screen queries)
const createMatchMediaMock = (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => { /* deprecated */ },
    removeListener: () => { /* deprecated */ },
    addEventListener: () => { },
    removeEventListener: () => { },
    dispatchEvent: () => true,
});
// Apply matchMedia to both globalThis and window to ensure MUI's useMediaQuery finds it
if (!(globalThis as any).matchMedia) {
    (globalThis as any).matchMedia = createMatchMediaMock;
}
// Also set on window object since MUI's useMediaQuery may look there directly
if (!(win as any).matchMedia) {
    (win as any).matchMedia = createMatchMediaMock;
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
        (globalThis as any).Node.prototype.removeChild = function (child: any) {
            try { return originalRemoveChild.call(this, child); }
            catch { return child; }
        };
    }
} catch { /* ignore */ }

// Ensure document.body exists for @testing-library/react screen
if (!document.body) {
    document.body = document.createElement('body');
}

// Mock fetch is now handled by MSW (Mock Service Worker)
// See test/bun-setup.ts for MSW configuration
// The fetch mock below is commented out to allow MSW to intercept requests
// Uncomment only if you need to run tests without MSW

// (globalThis as any).fetch = async (url: string, options?: any) => {
//     const mockResponse = {
//         ok: true,
//         status: 200,
//         statusText: 'OK',
//         headers: new Map([['content-type', 'application/json']]),
//         json: async () => ({}),
//         text: async () => '',
//         clone: () => ({ ...mockResponse }),
//         arrayBuffer: async () => new ArrayBuffer(0),
//         blob: async () => new Blob(),
//         formData: async () => new FormData(),
//     };
//     return mockResponse;
// };

// LocalStorage mock for the tests
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) =>
            _store.hasOwnProperty(k) ? _store[k] : null,
        setItem: (k: string, v: string) => {
            _store[k] = String(v);
        },
        removeItem: (k: string) => {
            delete _store[k];
        },
        clear: () => {
            for (const k of Object.keys(_store)) delete _store[k];
        },
    };
}

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

// Cache API module instances to ensure consistency across all test imports
// This prevents the module loader from creating multiple instances of the same API
let cachedApiModules: {
    sratApi: any;
    wsApi: any;
    errorSlice: any;
    mdcSlice: any;
    mdcMiddleware: any;
} | null = null;

// Create the store after the above globals are set. Do dynamic imports to
// avoid loading modules (that inspect window/process.env at module import)
// before we've set up the test environment.
export async function createTestStore() {
    // CRITICAL: Cache API modules on first import to ensure all components
    // use the same instances as the store middleware.
    // Without this, in CI environments with different module loading behavior,
    // components may import different instances of the APIs than what's in the store,
    // causing "middleware not added" errors from RTK Query.
    if (!cachedApiModules) {
        const [
            { wsApi },
            { sratApi },
            { errorSlice },
            { mdcSlice },
            mdcMiddlewareModule,
        ] = await Promise.all([
            import("../src/store/sseApi"),
            import("../src/store/sratApi"),
            import("../src/store/errorSlice"),
            import("../src/store/mdcSlice"),
            import("../src/store/mdcMiddleware"),
        ]);

        cachedApiModules = {
            sratApi,
            wsApi,
            errorSlice,
            mdcSlice,
            mdcMiddleware: mdcMiddlewareModule.default,
        };
    }

    const { sratApi, wsApi, errorSlice, mdcSlice, mdcMiddleware } = cachedApiModules;
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
            [wsApi.reducerPath]: wsApi.reducer,
        },
        middleware: (getDefaultMiddleware) =>
            getDefaultMiddleware()
                .concat(mdcMiddleware)
                .concat(sratApi.middleware)
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
        store.dispatch(wsApi.util.resetApiState());
    } catch {
        // Silently ignore errors in case dispatch is not fully ready
    }

    return store;
}


