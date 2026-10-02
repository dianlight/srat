import { describe, expect, it, vi } from "vitest";

const wsState = vi.hoisted(() => ({
  data: undefined as
    | { smart_test_status?: { disk_id?: string } }
    | undefined,
}));

vi.mock("../../store/wsApi", () => ({
  wsApi: {
    reducerPath: "wsApi",
    reducer: () => ({}),
    util: {
      resetApiState: () => ({ type: "wsApi/resetApiState" }),
    },
    middleware:
      () => (next: (action: unknown) => unknown) => (action: unknown) =>
        next(action),
  },
  useGetServerEventsQuery: () => ({
    data: wsState.data,
    isLoading: false,
    error: undefined,
  }),
}));

describe("useSmartTestStatus hook - SSE events", () => {
  it("logs debug when smart test status arrives for a different disk", async () => {
    const sseStatus = {
      disk_id: "other-disk",
      running: false,
      status: "idle",
      percent_complete: 0,
      test_type: "none",
    };
    wsState.data = { smart_test_status: sseStatus };

    const debugSpy = vi.spyOn(console, "debug").mockImplementation(() => {});
    try {
      const React = await import("react");
      const { renderHook } = await import("@testing-library/react");
      const { Provider } = await import("react-redux");
      const { createTestStore } = await import("/test/testing");
      const { useSmartTestStatus } = await import("../smartTestStatusHook");

      const store = await createTestStore();
      const wrapper = ({ children }: React.PropsWithChildren) =>
        React.createElement(Provider, { store, children });

      renderHook(() => useSmartTestStatus("disk-1"), { wrapper });

      expect(debugSpy).toHaveBeenCalledWith(
        "Received smart test status for different disk:",
        sseStatus,
      );
    } finally {
      debugSpy.mockRestore();
    }
  });
});
