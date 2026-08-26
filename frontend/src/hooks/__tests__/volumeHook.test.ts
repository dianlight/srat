import { http, HttpResponse } from "msw";
import { beforeEach, describe, expect, it, vi } from "vitest";

// Controllable SSE mock state: the real wsApi never delivers a `volumes`
// payload in the test environment (streaming is disabled), so tests must
// inject the SSE side to exercise the "SSE wins" path of useVolume.
const sseMock = vi.hoisted(() => ({
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
    data: sseMock.data,
    isLoading: sseMock.isLoading,
    error: sseMock.error,
  }),
}));

const restDisk = {
  id: "ata-REST-DISK",
  device_path: "/dev/disk/by-id/ata-REST-DISK",
  partitions: [],
};
const sseDisk = {
  id: "ata-SSE-DISK",
  device_path: "/dev/disk/by-id/ata-SSE-DISK",
  partitions: [],
};

describe("useVolume hook", () => {
  beforeEach(() => {
    sseMock.data = undefined;
    sseMock.isLoading = false;
    sseMock.error = undefined;
  });

  it("falls back to REST data when no SSE volumes payload has arrived", async () => {
    const React = await import("react");
    const { renderHook, waitFor } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { createTestStore, getMswServer } = await import("/test/testing");
    const { useVolume } = await import("../../hooks/volumeHook");

    getMswServer().use(
      http.get(/.*\/api\/volumes(?:\?.*)?$/, () => HttpResponse.json([restDisk])),
    );

    const store = await createTestStore();
    const wrapper = ({ children }: React.PropsWithChildren) =>
      React.createElement(Provider, { store, children });

    const { result } = renderHook(() => useVolume(), { wrapper });

    await waitFor(() => {
      expect(result.current.disks).toEqual([restDisk]);
    });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeUndefined();
  });

  it("prefers the SSE volumes payload over REST data", async () => {
    const React = await import("react");
    const { renderHook, waitFor } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { createTestStore, getMswServer } = await import("/test/testing");
    const { useVolume } = await import("../../hooks/volumeHook");

    sseMock.data = { volumes: [sseDisk] };
    // REST returns a *different* disk; SSE must win.
    getMswServer().use(
      http.get(/.*\/api\/volumes(?:\?.*)?$/, () => HttpResponse.json([restDisk])),
    );

    const store = await createTestStore();
    const wrapper = ({ children }: React.PropsWithChildren) =>
      React.createElement(Provider, { store, children });

    const { result } = renderHook(() => useVolume(), { wrapper });

    await waitFor(() => {
      expect(result.current.disks).toEqual([sseDisk]);
    });
    // With an SSE payload present, REST loading must not gate isLoading.
    expect(result.current.isLoading).toBe(false);
  });

  it("returns an empty array (not undefined) and surfaces the error when REST fails", async () => {
    const React = await import("react");
    const { renderHook, waitFor } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { createTestStore, getMswServer } = await import("/test/testing");
    const { useVolume } = await import("../../hooks/volumeHook");

    getMswServer().use(
      http.get(/.*\/api\/volumes(?:\?.*)?$/, () =>
        HttpResponse.json({ detail: "boom" }, { status: 500 }),
      ),
    );

    const store = await createTestStore();
    const wrapper = ({ children }: React.PropsWithChildren) =>
      React.createElement(Provider, { store, children });

    const { result } = renderHook(() => useVolume(), { wrapper });

    await waitFor(() => {
      expect(result.current.error).toBeTruthy();
    });
    expect(result.current.disks).toEqual([]);
    expect(result.current.disks).not.toBeUndefined();
  });

  it("surfaces the SSE error when REST succeeds but SSE has no payload", async () => {
    const React = await import("react");
    const { renderHook, waitFor } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { createTestStore, getMswServer } = await import("/test/testing");
    const { useVolume } = await import("../../hooks/volumeHook");

    sseMock.error = { status: 500, data: { detail: "ws down" } };
    getMswServer().use(
      http.get(/.*\/api\/volumes(?:\?.*)?$/, () => HttpResponse.json([restDisk])),
    );

    const store = await createTestStore();
    const wrapper = ({ children }: React.PropsWithChildren) =>
      React.createElement(Provider, { store, children });

    const { result } = renderHook(() => useVolume(), { wrapper });

    await waitFor(() => {
      expect(result.current.error).toEqual({ status: 500, data: { detail: "ws down" } });
    });
    await waitFor(() => {
      expect(result.current.disks).toEqual([restDisk]);
    });
  });

  it("keeps isLoading true while REST loads and no SSE payload is present", async () => {
    const React = await import("react");
    const { renderHook, waitFor } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { createTestStore, getMswServer } = await import("/test/testing");
    const { useVolume } = await import("../../hooks/volumeHook");

    let resolveVolumes: (value: Response) => void = () => {};
    const pending = new Promise<Response>((resolve) => {
      resolveVolumes = resolve;
    });
    getMswServer().use(
      http.get(/.*\/api\/volumes(?:\?.*)?$/, () => pending),
    );

    const store = await createTestStore();
    const wrapper = ({ children }: React.PropsWithChildren) =>
      React.createElement(Provider, { store, children });

    const { result } = renderHook(() => useVolume(), { wrapper });

    // REST is still pending and no SSE payload: loading must stay true.
    expect(result.current.isLoading).toBe(true);

    resolveVolumes(HttpResponse.json([restDisk]));
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });
    expect(result.current.disks).toEqual([restDisk]);
  });

  it("returns a stable disks reference when SSE delivers identical content", async () => {
    const React = await import("react");
    const { renderHook, waitFor, act } = await import("@testing-library/react");
    const { Provider } = await import("react-redux");
    const { createTestStore, getMswServer } = await import("/test/testing");
    const { useVolume } = await import("../../hooks/volumeHook");

    // Seed the SSE mock with initial volume data.
    sseMock.data = { volumes: [{ ...sseDisk, partitions: { sda1: { id: "part-1", device_path: "/dev/sda1" } } }] };
    getMswServer().use(
      http.get(/.*\/api\/volumes(?:\?.*)?$/, () => HttpResponse.json([restDisk])),
    );

    const store = await createTestStore();
    const wrapper = ({ children }: React.PropsWithChildren) =>
      React.createElement(Provider, { store, children });

    const { result } = renderHook(() => useVolume(), { wrapper });

    await waitFor(() => {
      expect(result.current.disks).toHaveLength(1);
    });
    const disksRefBefore = result.current.disks;

    // Simulate an SSE heartbeat that delivers the same content via a new
    // JSON.parse() — different object references, identical data.
    await act(async () => {
      sseMock.data = {
        volumes: JSON.parse(JSON.stringify(result.current.disks)),
      };
    });

    // The disks reference must be identical — no new array was created.
    expect(result.current.disks).toBe(disksRefBefore);
  });
});
