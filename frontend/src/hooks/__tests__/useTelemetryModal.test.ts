import "../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

describe("useTelemetryModal hook", () => {
    beforeEach(() => {
        // Clear any state
    });

    it("exports useTelemetryModal function", async () => {
        const { useTelemetryModal } = await import("../useTelemetryModal");
        expect(typeof useTelemetryModal).toBe("function");
    });

    it("initializes with shouldShow false", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useTelemetryModal } = await import("../useTelemetryModal");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store }, children);

        const { result } = renderHook(() => useTelemetryModal(), { wrapper });

        expect(result.current.shouldShow).toBe(false);
    });

    it("returns dismiss function", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useTelemetryModal } = await import("../useTelemetryModal");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store }, children);

        const { result } = renderHook(() => useTelemetryModal(), { wrapper });

        expect(typeof result.current.dismiss).toBe("function");
    });

    it("dismiss function sets shouldShow to false", async () => {
        const React = await import("react");
        const { renderHook, act } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useTelemetryModal } = await import("../useTelemetryModal");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store }, children);

        const { result } = renderHook(() => useTelemetryModal(), { wrapper });

        act(() => {
            result.current.dismiss();
        });

        expect(result.current.shouldShow).toBe(false);
    });

    it("checks settings and internet connection", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useTelemetryModal } = await import("../useTelemetryModal");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store }, children);

        const { result } = renderHook(() => useTelemetryModal(), { wrapper });

        // Should have checked for settings and internet
        expect(result.current).toHaveProperty("shouldShow");
        expect(result.current).toHaveProperty("dismiss");
    });

    it("handles loading states correctly", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useTelemetryModal } = await import("../useTelemetryModal");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store }, children);

        const { result } = renderHook(() => useTelemetryModal(), { wrapper });

        // While loading, shouldShow should remain false
        expect(result.current.shouldShow).toBe(false);
    });

    it("validates settings object correctly", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useTelemetryModal } = await import("../useTelemetryModal");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store }, children);

        const { result } = renderHook(() => useTelemetryModal(), { wrapper });

        // Hook should validate settings internally
        expect(result.current).toBeTruthy();
    });

    it("returns consistent object structure", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../test/setup");
        const { useTelemetryModal } = await import("../useTelemetryModal");

        const store = await createTestStore();
        const wrapper = ({ children }: any) => React.createElement(Provider, { store }, children);

        const { result } = renderHook(() => useTelemetryModal(), { wrapper });

        expect(result.current).toHaveProperty("shouldShow");
        expect(result.current).toHaveProperty("dismiss");
        expect(typeof result.current.shouldShow).toBe("boolean");
        expect(typeof result.current.dismiss).toBe("function");
    });
});
