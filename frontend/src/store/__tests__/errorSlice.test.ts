import "../../../test/setup";
import { beforeEach, describe, expect, it } from "bun:test";

describe("errorSlice", () => {
	beforeEach(() => {
		// Clear any global state before each test
	});

	it("imports errorSlice reducer correctly", async () => {
		const { default: errorReducer } = await import("../errorSlice");
		expect(typeof errorReducer).toBe("function");
	});

	it("exports ErrorState interface and slice components", async () => {
		const { errorSlice, addMessage, clearMessages } = await import(
			"../errorSlice"
		);

		expect(errorSlice).toBeTruthy();
		expect(errorSlice.name).toBe("errors");
		expect(typeof addMessage).toBe("function");
		expect(typeof clearMessages).toBe("function");
	});

	it("has correct initial state", async () => {
		const { default: errorReducer } = await import("../errorSlice");

		const initialState = errorReducer(undefined, { type: "@@redux/INIT" });
		expect(initialState).toEqual({
			messages: [],
		});
	});

	it("handles addMessage action correctly", async () => {
		const { default: errorReducer, addMessage } = await import("../errorSlice");

		const initialState = { messages: [] as string[] };
		const testMessage = "Test error message";

		const action = addMessage(testMessage);
		const newState = errorReducer(initialState, action);

		expect(newState.messages).toEqual([testMessage]);
		expect(newState.messages.length).toBe(1);
	});

	it("handles multiple addMessage actions correctly", async () => {
		const { default: errorReducer, addMessage } = await import("../errorSlice");

		let state = { messages: [] as string[] };
		const firstMessage = "First error";
		const secondMessage = "Second error";

		// Add first message
		state = errorReducer(state, addMessage(firstMessage));
		expect(state.messages).toEqual([firstMessage]);

		// Add second message
		state = errorReducer(state, addMessage(secondMessage));
		expect(state.messages).toEqual([firstMessage, secondMessage]);
		expect(state.messages.length).toBe(2);
	});

	it("handles clearMessages action correctly", async () => {
		const { default: errorReducer, clearMessages } = await import(
			"../errorSlice"
		);

		// Start with state containing messages
		let state = { messages: ["Error 1", "Error 2", "Error 3"] as string[] };

		// Clear all messages
		state = errorReducer(state, clearMessages());

		expect(state.messages).toEqual([]);
		expect(state.messages.length).toBe(0);
	});

	it("handles clearMessages action on empty state", async () => {
		const { default: errorReducer, clearMessages } = await import(
			"../errorSlice"
		);

		const initialState = { messages: [] as string[] };
		const newState = errorReducer(initialState, clearMessages());

		expect(newState.messages).toEqual([]);
		expect(newState.messages.length).toBe(0);
	});

	it("maintains immutability with addMessage", async () => {
		const { default: errorReducer, addMessage } = await import("../errorSlice");

		const originalState = { messages: ["Existing error"] as string[] };
		const testMessage = "New error";

		const newState = errorReducer(originalState, addMessage(testMessage));

		// Check that original state is not mutated
		expect(originalState.messages).toEqual(["Existing error"]);
		// Check that new state contains both messages
		expect(newState.messages).toEqual(["Existing error", "New error"]);
		// Verify they are different objects
		expect(newState).not.toBe(originalState);
	});

	it("maintains immutability with clearMessages", async () => {
		const { default: errorReducer, clearMessages } = await import(
			"../errorSlice"
		);

		const originalState = { messages: ["Error 1", "Error 2"] as string[] };
		const newState = errorReducer(originalState, clearMessages());

		// Check that original state is not mutated
		expect(originalState.messages).toEqual(["Error 1", "Error 2"]);
		// Check that new state is cleared
		expect(newState.messages).toEqual([]);
		// Verify they are different objects
		expect(newState).not.toBe(originalState);
	});

	it("handles unknown actions correctly", async () => {
		const { default: errorReducer } = await import("../errorSlice");

		const state = { messages: ["Test error"] as string[] };
		const unknownAction = { type: "UNKNOWN_ACTION", payload: "test" };

		const newState = errorReducer(state, unknownAction);

		// State should remain unchanged for unknown actions
		expect(newState).toEqual(state);
	});

	it("action creators have correct type and payload structure", async () => {
		const { addMessage, clearMessages } = await import("../errorSlice");

		// Test addMessage action creator
		const testMessage = "Test error message";
		const addAction = addMessage(testMessage);
		expect(addAction.type).toBe("errors/addMessage");
		expect(addAction.payload).toBe(testMessage);

		// Test clearMessages action creator
		const clearAction = clearMessages();
		expect(clearAction.type).toBe("errors/clearMessages");
		expect(clearAction.payload).toBeUndefined();
	});

	it("reducer handles state correctly with different payload types", async () => {
		const { default: errorReducer, addMessage } = await import("../errorSlice");

		const initialState = { messages: [] as string[] };

		// Test with string message
		const stringMessage = "String error message";
		let state = errorReducer(initialState, addMessage(stringMessage));
		expect(state.messages).toEqual([stringMessage]);

		// Test with number converted to string (TypeScript should prevent this, but test runtime behavior)
		const numberMessage = "123";
		state = errorReducer(state, addMessage(numberMessage));
		expect(state.messages).toEqual([stringMessage, numberMessage]);
	});

	it("slice name matches expected value", async () => {
		const { errorSlice } = await import("../errorSlice");
		expect(errorSlice.name).toBe("errors");
	});
});
