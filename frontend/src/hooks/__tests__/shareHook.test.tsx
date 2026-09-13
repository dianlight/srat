/* eslint-disable */
import { http, HttpResponse } from "msw";
import { beforeEach, describe, expect, it, vi } from "vitest";

// Controllable WS mock state: the real wsApi never delivers a `shares`
// payload in the test environment (streaming is disabled), so tests must
// inject the WS side to exercise the "WS wins" path of useShare.
const wsMock = vi.hoisted(() => ({
  data: undefined as unknown,
  isLoading: false,
  error: undefined as unknown,
}));

const fakeReducer = (state: any = {}, _action: any) => state;
const makeMiddleware = () => () => (next: any) => (action: any) => next(action);

vi.mock("../../store/wsApi", () => ({
  wsApi: {
    reducerPath: "wsApi",
    reducer: fakeReducer,
    middleware: makeMiddleware(),
    util: {
      resetApiState: () => ({ type: "wsApi/resetApiState" }),
    },
  },
  useGetServerEventsQuery: () => ({
    data: wsMock.data,
    isLoading: wsMock.isLoading,
    error: wsMock.error,
  }),
}));

const restShare = {
  name: "media",
  usage: "media",
  mount_point_data: {
    path: "/mnt/media",
    is_mounted: true,
  },
};
const wsShare = {
  name: "backup",
  usage: "backup",
  mount_point_data: {
    path: "/mnt/backup",
    is_mounted: true,
  },
};

describe("useShare hook", () => {
  beforeEach(() => {
    wsMock.data = undefined;
    wsMock.isLoading = false;
    wsMock.error = undefined;
  });

  it("returns shares as a record with loading and error fields", async () => {
    const React = await import("react");
    const { renderHook } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { useShare } = await import("../shareHook");
    const { createTestStore, getMswServer } = await import("/test/testing");

    getMswServer().use(
      http.get(/.*\/api\/shares(?:\?.*)?$/, () => HttpResponse.json([])),
    );

    const store = await createTestStore();
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(Provider, { store, children });

    const { result } = renderHook(() => useShare(), { wrapper });

    expect(result.current).toHaveProperty("shares");
    expect(result.current).toHaveProperty("isLoading");
    expect(result.current).toHaveProperty("error");
    expect(Object.keys(result.current).length).toBe(3);
  });

  it("falls back to REST data when no WS shares payload has arrived", async () => {
    const React = await import("react");
    const { renderHook, waitFor } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { useShare } = await import("../shareHook");
    const { createTestStore, getMswServer } = await import("/test/testing");

    getMswServer().use(
      http.get(/.*\/api\/shares(?:\?.*)?$/, () =>
        HttpResponse.json([restShare]),
      ),
    );

    const store = await createTestStore();
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(Provider, { store, children });

    const { result } = renderHook(() => useShare(), { wrapper });

    await waitFor(() => {
      expect(result.current.shares).toEqual({ media: restShare });
    });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeUndefined();
  });

  it("prefers the WS shares payload over REST data", async () => {
    const React = await import("react");
    const { renderHook, waitFor } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { useShare } = await import("../shareHook");
    const { createTestStore, getMswServer } = await import("/test/testing");

    wsMock.data = { shares: [wsShare] };
    // REST returns a *different* share; WS must win.
    getMswServer().use(
      http.get(/.*\/api\/shares(?:\?.*)?$/, () =>
        HttpResponse.json([restShare]),
      ),
    );

    const store = await createTestStore();
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(Provider, { store, children });

    const { result } = renderHook(() => useShare(), { wrapper });

    await waitFor(() => {
      expect(result.current.shares).toEqual({ backup: wsShare });
    });
    // With a WS payload present, REST loading must not gate isLoading.
    expect(result.current.isLoading).toBe(false);
  });

  it("switches from REST to WS data when a shares payload arrives late", async () => {
    const React = await import("react");
    const { renderHook, waitFor } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { useShare } = await import("../shareHook");
    const { createTestStore, getMswServer } = await import("/test/testing");

    getMswServer().use(
      http.get(/.*\/api\/shares(?:\?.*)?$/, () =>
        HttpResponse.json([restShare]),
      ),
    );

    const store = await createTestStore();
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(Provider, { store, children });

    const { result, rerender } = renderHook(() => useShare(), { wrapper });

    // First paint comes from REST without waiting for the socket.
    await waitFor(() => {
      expect(result.current.shares).toEqual({ media: restShare });
    });

    // A late WS event replaces the REST snapshot.
    wsMock.data = { shares: [wsShare] };
    rerender();

    await waitFor(() => {
      expect(result.current.shares).toEqual({ backup: wsShare });
    });
  });

  it("keeps isLoading true while REST loads and no WS payload is present", async () => {
    const React = await import("react");
    const { renderHook, waitFor } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { useShare } = await import("../shareHook");
    const { createTestStore, getMswServer } = await import("/test/testing");

    let resolveShares: (value: Response) => void = () => {};
    const pending = new Promise<Response>((resolve) => {
      resolveShares = resolve;
    });
    getMswServer().use(
      http.get(/.*\/api\/shares(?:\?.*)?$/, () => pending),
    );

    const store = await createTestStore();
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(Provider, { store, children });

    const { result } = renderHook(() => useShare(), { wrapper });

    // REST is still pending and no WS payload: loading must stay true and
    // the tree must see an empty record (loading skeleton), not stale data.
    expect(result.current.isLoading).toBe(true);
    expect(result.current.shares).toEqual({});

    resolveShares(HttpResponse.json([restShare]));
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });
    expect(result.current.shares).toEqual({ media: restShare });
  });

  it("returns an empty record and surfaces the error when REST fails", async () => {
    const React = await import("react");
    const { renderHook, waitFor } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { useShare } = await import("../shareHook");
    const { createTestStore, getMswServer } = await import("/test/testing");

    getMswServer().use(
      http.get(/.*\/api\/shares(?:\?.*)?$/, () =>
        HttpResponse.json({ detail: "boom" }, { status: 500 }),
      ),
    );

    const store = await createTestStore();
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(Provider, { store, children });

    const { result } = renderHook(() => useShare(), { wrapper });

    await waitFor(() => {
      expect(result.current.error).toBeTruthy();
    });
    expect(result.current.shares).toEqual({});
  });

  it("surfaces the WS error when REST succeeds but WS has no payload", async () => {
    const React = await import("react");
    const { renderHook, waitFor } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { useShare } = await import("../shareHook");
    const { createTestStore, getMswServer } = await import("/test/testing");

    wsMock.error = { status: 500, data: { detail: "ws down" } };
    getMswServer().use(
      http.get(/.*\/api\/shares(?:\?.*)?$/, () =>
        HttpResponse.json([restShare]),
      ),
    );

    const store = await createTestStore();
    const wrapper = ({ children }: { children: React.ReactNode }) =>
      React.createElement(Provider, { store, children });

    const { result } = renderHook(() => useShare(), { wrapper });

    await waitFor(() => {
      expect(result.current.error).toEqual({
        status: 500,
        data: { detail: "ws down" },
      });
    });
    await waitFor(() => {
      expect(result.current.shares).toEqual({ media: restShare });
    });
  });
});
