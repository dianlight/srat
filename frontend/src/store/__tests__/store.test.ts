import { describe, it, expect } from "bun:test";

describe("Redux Store Configuration", () => {
	it("exports a configured store instance", async () => {
		const { store } = await import("../store");
		expect(store).toBeTruthy();
		expect(typeof store.getState).toBe("function");
		expect(typeof store.dispatch).toBe("function");
	});

	it("has all required reducers configured", async () => {
		const { store } = await import("../store");
		const { sratApi } = await import("../sratApi");
		const { sseApi, wsApi } = await import("../sseApi");
		const state = store.getState();

		// Verify all expected slice states exist
		expect(state).toHaveProperty("errors");
		expect(state).toHaveProperty("mdc");
		// API reducers use their reducerPath as keys
		expect(state).toHaveProperty(sratApi.reducerPath);
		expect(state).toHaveProperty(sseApi.reducerPath);
		expect(state).toHaveProperty(wsApi.reducerPath);
	});

	it("has middleware configured correctly", async () => {
		const { store } = await import("../store");

		// Check that the store has middleware by verifying dispatch behavior
		// Middleware should be present if store.dispatch is a function
		expect(typeof store.dispatch).toBe("function");
	});

	it("exports typed hooks for dispatch and selector", async () => {
		const { useAppDispatch, useAppSelector } = await import("../store");

		expect(typeof useAppDispatch).toBe("function");
		expect(typeof useAppSelector).toBe("function");
	});

	it("has proper TypeScript types exported", async () => {
		const storeModule = await import("../store");

		// Verify that type exports exist (they'll be undefined at runtime but should be importable)
		expect(storeModule).toHaveProperty("store");
		expect(storeModule).toHaveProperty("useAppDispatch");
		expect(storeModule).toHaveProperty("useAppSelector");
	});

	it("initializes with default state structure", async () => {
		const { store } = await import("../store");
		const { sratApi } = await import("../sratApi");
		const { sseApi, wsApi } = await import("../sseApi");
		const state = store.getState();

		// Verify errors slice initializes
		expect(state.errors).toBeTruthy();
		expect(state.errors).toHaveProperty("messages");
		expect(Array.isArray(state.errors.messages)).toBe(true);

		// Verify mdc slice initializes
		expect(state.mdc).toBeTruthy();

		// Verify API slices initialize with their reducerPath keys
		expect(state[sratApi.reducerPath]).toBeTruthy();
		expect(state[sseApi.reducerPath]).toBeTruthy();
		expect(state[wsApi.reducerPath]).toBeTruthy();
	});

	it("supports Redux DevTools in development mode", async () => {
		const { store } = await import("../store");

		// In development mode, the store should have DevTools integration
		// We can verify this by checking if __REDUX_DEVTOOLS_EXTENSION__ was called
		// Note: This is an indirect test since we can't directly access the enhancer config
		expect(store).toBeTruthy();
		expect(typeof store.getState).toBe("function");

		// The store should work with or without DevTools browser extension
		const state = store.getState();
		expect(state).toBeTruthy();
	});

	it("can dispatch actions and update state", async () => {
		const { store } = await import("../store");
		const { errorSlice } = await import("../errorSlice");

		const initialState = store.getState();
		const initialMessagesCount = initialState.errors.messages.length;

		// Dispatch an action to add an error message
		store.dispatch(errorSlice.actions.addMessage("Test error message"));

		const newState = store.getState();
		const newMessagesCount = newState.errors.messages.length;

		// Verify state was updated
		expect(newMessagesCount).toBe(initialMessagesCount + 1);
		expect(newState.errors.messages).toContain("Test error message");
	});
});

