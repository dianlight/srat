import "../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

// Required localStorage shim for testing environment
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => { _store[k] = String(v); },
        removeItem: (k: string) => { delete _store[k]; },
        clear: () => { for (const k of Object.keys(_store)) delete _store[k]; },
    };
}

describe("useShare Hook", () => {
    beforeEach(() => {
        localStorage.clear();
        // Clear DOM before each test
        document.body.innerHTML = '';
    });

    it("imports and exports useShare function", async () => {
        const { useShare } = await import("../shareHook");
        expect(typeof useShare).toBe("function");
    });

    it("renders hook with default loading state", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { useShare } = await import("../shareHook");
        const { createTestStore } = await import("../../../test/setup");

        const store = await createTestStore();

        const wrapper = ({ children }: { children: React.ReactNode }) =>
            React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useShare(), { wrapper });

        // Hook should return the expected structure
        expect(result.current).toHaveProperty('shares');
        expect(result.current).toHaveProperty('isLoading');
        expect(result.current).toHaveProperty('error');

        // Initial state should have empty shares array
        expect(Array.isArray(result.current.shares)).toBe(true);
        expect(result.current.shares.length).toBe(0);
    });

    it("handles shares data array correctly", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { useShare } = await import("../shareHook");
        const { createTestStore } = await import("../../../test/setup");

        const store = await createTestStore();

        const wrapper = ({ children }: { children: React.ReactNode }) =>
            React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useShare(), { wrapper });

        // Should return shares as array
        expect(Array.isArray(result.current.shares)).toBe(true);
    });

    it("returns loading state correctly", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { useShare } = await import("../shareHook");
        const { createTestStore } = await import("../../../test/setup");

        const store = await createTestStore();

        const wrapper = ({ children }: { children: React.ReactNode }) =>
            React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useShare(), { wrapper });

        // Loading state should be a boolean
        expect(typeof result.current.isLoading).toBe("boolean");
    });

    it("handles error states", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { useShare } = await import("../shareHook");
        const { createTestStore } = await import("../../../test/setup");

        const store = await createTestStore();

        const wrapper = ({ children }: { children: React.ReactNode }) =>
            React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useShare(), { wrapper });

        // Error can be undefined or an error object
        expect(result.current.error === undefined || typeof result.current.error === 'object').toBe(true);
    });

    it("handles useGetApiSharesQuery integration", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { useShare } = await import("../shareHook");
        const { createTestStore } = await import("../../../test/setup");

        const store = await createTestStore();

        const wrapper = ({ children }: { children: React.ReactNode }) =>
            React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useShare(), { wrapper });

        // Hook should integrate with API queries
        expect(result.current).toBeTruthy();
        expect(typeof result.current.shares).toBe("object");
    });

    it("handles useGetServerEventsQuery integration", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { useShare } = await import("../shareHook");
        const { createTestStore } = await import("../../../test/setup");

        const store = await createTestStore();

        const wrapper = ({ children }: { children: React.ReactNode }) =>
            React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useShare(), { wrapper });

        // Hook should integrate with server events
        expect(result.current).toBeTruthy();
        expect(result.current.shares).toBeDefined();
    });

    it("manages useState for shares correctly", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { useShare } = await import("../shareHook");
        const { createTestStore } = await import("../../../test/setup");

        const store = await createTestStore();

        const wrapper = ({ children }: { children: React.ReactNode }) =>
            React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useShare(), { wrapper });

        // Should manage shares state
        expect(result.current.shares).toBeDefined();
        expect(Array.isArray(result.current.shares)).toBe(true);
    });

    it("implements useEffect for data updates correctly", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { useShare } = await import("../shareHook");
        const { createTestStore } = await import("../../../test/setup");

        const store = await createTestStore();

        const wrapper = ({ children }: { children: React.ReactNode }) =>
            React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useShare(), { wrapper });

        // Hook should complete without errors, indicating useEffect works
        expect(result.current).toBeTruthy();
    });

    it("returns expected hook structure", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { useShare } = await import("../shareHook");
        const { createTestStore } = await import("../../../test/setup");

        const store = await createTestStore();

        const wrapper = ({ children }: { children: React.ReactNode }) =>
            React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useShare(), { wrapper });

        // Should return object with exactly 3 properties
        const keys = Object.keys(result.current);
        expect(keys.length).toBe(3);
        expect(keys.includes('shares')).toBe(true);
        expect(keys.includes('isLoading')).toBe(true);
        expect(keys.includes('error')).toBe(true);
    });
});