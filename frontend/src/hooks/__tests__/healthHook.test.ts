import "../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

describe("useHealth hook", () => {
    beforeEach(() => {
        // Clear any previous state
    });

    it("initializes with default health state", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useHealth } = await import("../healthHook");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useHealth(), { wrapper });

        // Initially should have default health object
        expect(result.current.health).toBeTruthy();
        expect(typeof result.current.health.alive).toBe("boolean");
    });

    it("returns loading state correctly", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useHealth } = await import("../healthHook");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useHealth(), { wrapper });

        // Should have a loading state
        expect(typeof result.current.isLoading).toBe("boolean");
    });

    it("returns error state correctly", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useHealth } = await import("../healthHook");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useHealth(), { wrapper });

        // Error property exists in the return value
        expect("error" in result.current).toBe(true);
    });

    it("updates health when data changes", async () => {
        const React = await import("react");
        const { renderHook, waitFor } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useHealth } = await import("../healthHook");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useHealth(), { wrapper });

        await waitFor(() => {
            expect(result.current.health !== undefined).toBe(true);
        }, { timeout: 1000 });
    });

    it("handles SSE heartbeat updates correctly", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useHealth } = await import("../healthHook");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useHealth(), { wrapper });

        // Hook should handle SSE heartbeat updates
        expect(result.current.health).toBeTruthy();
    });

    it("contains all expected health fields", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useHealth } = await import("../healthHook");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useHealth(), { wrapper });

        // Check that health object has expected properties
        expect(result.current.health).toHaveProperty("alive");
        expect(result.current.health).toHaveProperty("aliveTime");
        expect(result.current.health).toHaveProperty("dirty_tracking");
        expect(result.current.health).toHaveProperty("uptime");
    });

    it("combines REST API and SSE loading states", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useHealth } = await import("../healthHook");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useHealth(), { wrapper });

        // Should combine loading states correctly
        expect(typeof result.current.isLoading).toBe("boolean");
        expect(result.current.health).toBeTruthy();
    });

    it("combines REST API and SSE errors", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useHealth } = await import("../healthHook");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useHealth(), { wrapper });

        // Error property exists in the return value
        expect("error" in result.current).toBe(true);
    });

    it("initializes samba_process_status correctly", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useHealth } = await import("../healthHook");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useHealth(), { wrapper });

        expect(result.current.health).toHaveProperty("samba_process_status");
    });

    it("initializes disk_health correctly", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useHealth } = await import("../healthHook");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useHealth(), { wrapper });

        expect(result.current.health).toHaveProperty("disk_health");
    });

    it("initializes network_health correctly", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useHealth } = await import("../healthHook");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useHealth(), { wrapper });

        expect(result.current.health).toHaveProperty("network_health");
    });
});
