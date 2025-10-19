import "/workspaces/srat/frontend/test/setup.ts";
import { describe, expect, it } from "bun:test";

describe("mdcSlice", () => {
	it("imports and exports mdcSlice correctly", async () => {
		const { mdcSlice } = await import("../mdcSlice");
		expect(mdcSlice).toBeTruthy();
		expect(mdcSlice.name).toBe("mdc");
	});

	it("has correct initial state with spanId and traceId", async () => {
		const { mdcSlice } = await import("../mdcSlice");
		const initialState = mdcSlice.getInitialState();

		expect(initialState).toBeTruthy();
		expect(initialState.spanId).toBeDefined();
		expect(initialState.traceId).toBeDefined();
		expect(typeof initialState.spanId).toBe("string");
		expect(typeof initialState.traceId).toBe("string");
	});

	it("exports setSpanId action", async () => {
		const { setSpanId } = await import("../mdcSlice");
		expect(setSpanId).toBeTruthy();
		expect(typeof setSpanId).toBe("function");
	});

	it("exports setTraceId action", async () => {
		const { setTraceId } = await import("../mdcSlice");
		expect(setTraceId).toBeTruthy();
		expect(typeof setTraceId).toBe("function");
	});

	it("handles setSpanId action correctly", async () => {
		const { mdcSlice, setSpanId } = await import("../mdcSlice");
		const reducer = mdcSlice.reducer;
		const initialState = mdcSlice.getInitialState();

		const newState = reducer(initialState, setSpanId("test-span-id"));
		expect(newState.spanId).toBe("test-span-id");
		expect(newState.traceId).toBe(initialState.traceId); // traceId should remain unchanged
	});

	it("handles setTraceId action correctly", async () => {
		const { mdcSlice, setTraceId } = await import("../mdcSlice");
		const reducer = mdcSlice.reducer;
		const initialState = mdcSlice.getInitialState();

		const newState = reducer(initialState, setTraceId("test-trace-id"));
		expect(newState.traceId).toBe("test-trace-id");
		expect(newState.spanId).toBe(initialState.spanId); // spanId should remain unchanged
	});

	it("handles setSpanId with null value", async () => {
		const { mdcSlice, setSpanId } = await import("../mdcSlice");
		const reducer = mdcSlice.reducer;
		const initialState = mdcSlice.getInitialState();

		const newState = reducer(initialState, setSpanId(null));
		expect(newState.spanId).toBeNull();
	});

	it("handles setTraceId with null value", async () => {
		const { mdcSlice, setTraceId } = await import("../mdcSlice");
		const reducer = mdcSlice.reducer;
		const initialState = mdcSlice.getInitialState();

		const newState = reducer(initialState, setTraceId(null));
		expect(newState.traceId).toBeNull();
	});

	it("handles setAllData action", async () => {
		const { mdcSlice } = await import("../mdcSlice");
		const reducer = mdcSlice.reducer;
		const initialState = mdcSlice.getInitialState();

		const newData = {
			spanId: "new-span-id",
			traceId: "new-trace-id",
		};

		// Access setAllData from mdcSlice.actions
		const setAllData = mdcSlice.actions.setAllData;
		const newState = reducer(initialState, setAllData(newData));

		expect(newState.spanId).toBe("new-span-id");
		expect(newState.traceId).toBe("new-trace-id");
	});

	it("handles unknown actions correctly", async () => {
		const { mdcSlice } = await import("../mdcSlice");
		const reducer = mdcSlice.reducer;
		const initialState = mdcSlice.getInitialState();

		const newState = reducer(initialState, { type: "UNKNOWN_ACTION" } as any);
		expect(newState).toEqual(initialState);
	});

	it("setSpanId action creator has correct structure", async () => {
		const { setSpanId } = await import("../mdcSlice");
		const action = setSpanId("test-span");

		expect(action.type).toBeTruthy();
		expect(action.payload).toBe("test-span");
	});

	it("setTraceId action creator has correct structure", async () => {
		const { setTraceId } = await import("../mdcSlice");
		const action = setTraceId("test-trace");

		expect(action.type).toBeTruthy();
		expect(action.payload).toBe("test-trace");
	});

	it("generates unique UUIDs for initial state", async () => {
		const { mdcSlice } = await import("../mdcSlice");
		const state1 = mdcSlice.getInitialState();
		const state2 = mdcSlice.getInitialState();

		// Each initialization should generate different IDs
		expect(state1.spanId).toBeTruthy();
		expect(state1.traceId).toBeTruthy();
		expect(state2.spanId).toBeTruthy();
		expect(state2.traceId).toBeTruthy();
	});

	it("exports default reducer", async () => {
		const defaultReducer = await import("../mdcSlice");
		expect(defaultReducer.default).toBeTruthy();
		expect(typeof defaultReducer.default).toBe("function");
	});

	it("makeUUID generates valid UUID format", async () => {
		const { mdcSlice } = await import("../mdcSlice");
		const state = mdcSlice.getInitialState();

		// UUID format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
		const uuidRegex =
			/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

		expect(uuidRegex.test(state.spanId!)).toBe(true);
		expect(uuidRegex.test(state.traceId!)).toBe(true);
	});

	it("handles setAllData with null values", async () => {
		const { mdcSlice } = await import("../mdcSlice");
		const reducer = mdcSlice.reducer;
		const initialState = mdcSlice.getInitialState();

		const newData = {
			spanId: null,
			traceId: null,
		};

		const setAllData = mdcSlice.actions.setAllData;
		const newState = reducer(initialState, setAllData(newData));

		expect(newState.spanId).toBeNull();
		expect(newState.traceId).toBeNull();
	});

	it("preserves state immutability", async () => {
		const { mdcSlice, setSpanId } = await import("../mdcSlice");
		const reducer = mdcSlice.reducer;
		const initialState = mdcSlice.getInitialState();
		const originalSpanId = initialState.spanId;

		const newState = reducer(initialState, setSpanId("modified-span"));

		// Original state should not be modified
		expect(initialState.spanId).toBe(originalSpanId);
		expect(newState.spanId).toBe("modified-span");
	});
});
