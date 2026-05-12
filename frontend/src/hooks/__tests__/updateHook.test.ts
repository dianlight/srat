import { beforeEach, describe, expect, it } from "vitest";

describe("useUpdate hook", () => {
    beforeEach(() => {
        // noop
    });

    it("initializes with default update state", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("/test/testing");
        const { useUpdate } = await import("../updateHook");

        const store = await createTestStore();
        const wrapper = ({ children }: React.PropsWithChildren) =>
            React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useUpdate(), { wrapper });

        expect(result.current.update).toBeTruthy();
        expect(result.current.update.Available).toBe(false);
        expect(typeof result.current.update.Progress.progress).toBe("number");
        expect(typeof result.current.isLoading).toBe("boolean");
        expect("error" in result.current).toBe(true);
    });

    it("exposes isLoading and error properties", async () => {
        const React = await import("react");
        const { renderHook } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("/test/testing");
        const { useUpdate } = await import("../updateHook");

        const store = await createTestStore();
        const wrapper = ({ children }: React.PropsWithChildren) =>
            React.createElement(Provider, { store, children });

        const { result } = renderHook(() => useUpdate(), { wrapper });

        expect(typeof result.current.isLoading).toBe("boolean");
        expect("error" in result.current).toBe(true);
    });
});
