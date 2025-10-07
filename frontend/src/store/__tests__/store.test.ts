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

describe("Store slices", () => {
    beforeEach(() => {
        localStorage.clear();
    });

    it("error slice exports correctly", async () => {
        const errorSlice = await import("../errorSlice");
        expect(errorSlice.default).toBeTruthy();
        expect(errorSlice.setError).toBeTruthy();
        expect(errorSlice.clearError).toBeTruthy();
    });

    it("can create setError action", async () => {
        const { setError } = await import("../errorSlice");
        const action = setError("Test error message");
        expect(action.type).toBe("error/setError");
        expect(action.payload).toBe("Test error message");
    });

    it("can create clearError action", async () => {
        const { clearError } = await import("../errorSlice");
        const action = clearError();
        expect(action.type).toBe("error/clearError");
    });

    it("locationState exports TabIDs", async () => {
        const { TabIDs } = await import("../locationState");
        expect(TabIDs.DASHBOARD).toBeTruthy();
        expect(TabIDs.VOLUMES).toBeTruthy();
        expect(TabIDs.SHARES).toBeTruthy();
        expect(TabIDs.USERS).toBeTruthy();
        expect(TabIDs.SETTINGS).toBeTruthy();
        expect(TabIDs.SMB_FILE_CONFIG).toBeTruthy();
        expect(TabIDs.SWAGGER).toBeTruthy();
    });

    it("store configuration exports correctly", async () => {
        const { default: store } = await import("../store");
        expect(store).toBeTruthy();
        expect(typeof store.getState).toBe("function");
        expect(typeof store.dispatch).toBe("function");
        expect(typeof store.subscribe).toBe("function");
    });

    it("store has initial state", async () => {
        const { default: store } = await import("../store");
        const state = store.getState();
        expect(state).toBeTruthy();
        expect(state.error).toBeDefined();
    });

    it("emptyApi exports correctly", async () => {
        const emptyApi = await import("../emptyApi");
        expect(emptyApi.emptyApi).toBeTruthy();
        expect(emptyApi.Configuration).toBeTruthy();
    });

    it("can dispatch actions to store", async () => {
        const { default: store } = await import("../store");
        const { setError } = await import("../errorSlice");
        
        store.dispatch(setError("Test"));
        const state = store.getState();
        expect(state.error.message).toBe("Test");
    });

    it("can clear error in store", async () => {
        const { default: store } = await import("../store");
        const { setError, clearError } = await import("../errorSlice");
        
        store.dispatch(setError("Test"));
        store.dispatch(clearError());
        const state = store.getState();
        expect(state.error.message).toBe("");
    });
});

describe("MDC Slice", () => {
    it("exports MDC slice correctly", async () => {
        const mdcSlice = await import("../mdcSlice");
        expect(mdcSlice.default).toBeTruthy();
    });

    it("exports MDC actions", async () => {
        const { setMDC, clearMDC } = await import("../mdcSlice");
        expect(typeof setMDC).toBe("function");
        expect(typeof clearMDC).toBe("function");
    });

    it("can create setMDC action", async () => {
        const { setMDC } = await import("../mdcSlice");
        const action = setMDC({ key: "value" });
        expect(action.type).toBe("mdc/setMDC");
        expect(action.payload).toEqual({ key: "value" });
    });

    it("can create clearMDC action", async () => {
        const { clearMDC } = await import("../mdcSlice");
        const action = clearMDC();
        expect(action.type).toBe("mdc/clearMDC");
    });
});

describe("API Configuration", () => {
    it("emptyApi has expected structure", async () => {
        const { emptyApi } = await import("../emptyApi");
        expect(emptyApi).toBeTruthy();
        expect(emptyApi.reducerPath).toBeTruthy();
    });

    it("Configuration class exists", async () => {
        const { Configuration } = await import("../emptyApi");
        expect(Configuration).toBeTruthy();
    });

    it("can create Configuration instance", async () => {
        const { Configuration } = await import("../emptyApi");
        const config = new Configuration();
        expect(config).toBeTruthy();
    });
});

describe("SRAT API", () => {
    it("exports API hooks", async () => {
        const sratApi = await import("../sratApi");
        expect(sratApi).toBeTruthy();
    });

    it("exports query hooks", async () => {
        const {
            useGetApiHealthQuery,
            useGetApiDisksQuery,
            useGetApiSharesQuery,
            useGetApiUsersQuery,
            useGetApiSettingsQuery
        } = await import("../sratApi");
        
        expect(typeof useGetApiHealthQuery).toBe("function");
        expect(typeof useGetApiDisksQuery).toBe("function");
        expect(typeof useGetApiSharesQuery).toBe("function");
        expect(typeof useGetApiUsersQuery).toBe("function");
        expect(typeof useGetApiSettingsQuery).toBe("function");
    });

    it("exports mutation hooks", async () => {
        const {
            usePutApiShareMutation,
            useDeleteApiShareMutation,
            usePutApiUserMutation,
            useDeleteApiUserMutation
        } = await import("../sratApi");
        
        expect(typeof usePutApiShareMutation).toBe("function");
        expect(typeof useDeleteApiShareMutation).toBe("function");
        expect(typeof usePutApiUserMutation).toBe("function");
        expect(typeof useDeleteApiUserMutation).toBe("function");
    });

    it("exports enum types", async () => {
        const { Usage, Type, Telemetry_mode } = await import("../sratApi");
        expect(Usage).toBeTruthy();
        expect(Type).toBeTruthy();
        expect(Telemetry_mode).toBeTruthy();
    });
});

describe("SSE API", () => {
    it("exports SSE API hooks", async () => {
        const { useGetServerEventsQuery } = await import("../sseApi");
        expect(typeof useGetServerEventsQuery).toBe("function");
    });

    it("exports sseApi object", async () => {
        const { sseApi } = await import("../sseApi");
        expect(sseApi).toBeTruthy();
        expect(sseApi.reducerPath).toBeTruthy();
    });
});
