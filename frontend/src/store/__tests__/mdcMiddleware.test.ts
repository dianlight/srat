import "../../../test/setup";
import { describe, expect, it } from "bun:test";

describe("mdcMiddleware", () => {
	it("exports mdcMiddleware as a function", async () => {
		const { mdcMiddleware } = await import("../mdcMiddleware");
		expect(typeof mdcMiddleware).toBe("function");
	});

	it("exports default middleware", async () => {
		const module = await import("../mdcMiddleware");
		expect(typeof module.default).toBe("function");
	});

	it("middleware has correct structure", async () => {
		const { mdcMiddleware } = await import("../mdcMiddleware");

		// Middleware should be a function that returns a function
		const mockStoreApi = {
			dispatch: () => {},
			getState: () => ({}),
		};

		const result = mdcMiddleware(mockStoreApi as any);
		expect(typeof result).toBe("function");
	});

	it("middleware chain returns a function", async () => {
		const { mdcMiddleware } = await import("../mdcMiddleware");

		const mockStoreApi = {
			dispatch: () => {},
			getState: () => ({}),
		};

		const next = () => {};
		const chain = mdcMiddleware(mockStoreApi as any)(next as any);
		expect(typeof chain).toBe("function");
	});

	it("middleware passes through action", async () => {
		const { mdcMiddleware } = await import("../mdcMiddleware");

		let nextCalled = false;
		const mockStoreApi = {
			dispatch: () => {},
			getState: () => ({}),
		};

		const next = (action: any) => {
			nextCalled = true;
			return action;
		};

		const action = { type: "TEST_ACTION" };
		const chain = mdcMiddleware(mockStoreApi as any)(next as any);
		const result = chain(action);

		expect(nextCalled).toBe(true);
		expect(result).toEqual(action);
	});

	it("handles action with X-Request-Id header", async () => {
		const { mdcMiddleware } = await import("../mdcMiddleware");

		let dispatchedAction: any = null;
		const mockStoreApi = {
			dispatch: (action: any) => {
				dispatchedAction = action;
			},
			getState: () => ({}),
		};

		const next = (action: any) => action;
		const chain = mdcMiddleware(mockStoreApi as any)(next as any);

		const action = {
			type: "TEST_ACTION",
			meta: {
				arg: {
					originalArgs: {
						"X-Request-Id": "req-123",
						"X-Span-Id": "span-456",
						"X-Trace-Id": "trace-789",
					},
				},
			},
		};

		chain(action);

		// Should have dispatched MDC action
		expect(dispatchedAction).toBeTruthy();
	});

	it("handles action with response headers", async () => {
		const { mdcMiddleware } = await import("../mdcMiddleware");

		let dispatchedAction: any = null;
		const mockStoreApi = {
			dispatch: (action: any) => {
				dispatchedAction = action;
			},
			getState: () => ({}),
		};

		const next = (action: any) => action;
		const chain = mdcMiddleware(mockStoreApi as any)(next as any);

		const action = {
			type: "TEST_ACTION",
			meta: {
				baseQueryMeta: {
					response: {
						headers: {
							"X-Span-Id": "span-123",
							"X-Trace-Id": "trace-456",
						},
					},
				},
			},
		};

		chain(action);

		// Should have dispatched MDC action
		expect(dispatchedAction).toBeTruthy();
	});

	it("handles action with Headers object", async () => {
		const { mdcMiddleware } = await import("../mdcMiddleware");

		const mockStoreApi = {
			dispatch: () => {},
			getState: () => ({}),
		};

		const next = (action: any) => action;
		const chain = mdcMiddleware(mockStoreApi as any)(next as any);

		const mockHeaders = {
			get: (key: string) => {
				if (key === "X-Span-Id") return "span-789";
				if (key === "X-Trace-Id") return "trace-012";
				return null;
			},
		};

		const action = {
			type: "TEST_ACTION",
			meta: {
				baseQueryMeta: {
					response: {
						headers: mockHeaders,
					},
				},
			},
		};

		const result = chain(action);
		expect(result).toBeTruthy();
	});

	it("handles action without MDC data", async () => {
		const { mdcMiddleware } = await import("../mdcMiddleware");

		const mockStoreApi = {
			dispatch: () => {},
			getState: () => ({}),
		};

		const next = (action: any) => action;
		const chain = mdcMiddleware(mockStoreApi as any)(next as any);

		const action = { type: "SIMPLE_ACTION" };
		const result = chain(action);

		expect(result).toEqual(action);
	});

	it("handles malformed action gracefully", async () => {
		const { mdcMiddleware } = await import("../mdcMiddleware");

		const mockStoreApi = {
			dispatch: () => {},
			getState: () => ({}),
		};

		const next = (action: any) => action;
		const chain = mdcMiddleware(mockStoreApi as any)(next as any);

		const action = null;
		/*const _result =*/ chain(action as any);

		// Should not throw
		expect(true).toBe(true);
	});

	it("handles lowercase header keys", async () => {
		const { mdcMiddleware } = await import("../mdcMiddleware");

		const mockStoreApi = {
			dispatch: () => {},
			getState: () => ({}),
		};

		const next = (action: any) => action;
		const chain = mdcMiddleware(mockStoreApi as any)(next as any);

		const action = {
			type: "TEST_ACTION",
			meta: {
				baseQueryMeta: {
					headers: {
						"x-span-id": "span-abc",
						"x-trace-id": "trace-def",
					},
				},
			},
		};

		const result = chain(action);
		expect(result).toBeTruthy();
	});

	it("handles uppercase header keys", async () => {
		const { mdcMiddleware } = await import("../mdcMiddleware");

		const mockStoreApi = {
			dispatch: () => {},
			getState: () => ({}),
		};

		const next = (action: any) => action;
		const chain = mdcMiddleware(mockStoreApi as any)(next as any);

		const action = {
			type: "TEST_ACTION",
			meta: {
				baseQueryMeta: {
					headers: {
						"X-SPAN-ID": "span-xyz",
						"X-TRACE-ID": "trace-uvw",
					},
				},
			},
		};

		const result = chain(action);
		expect(result).toBeTruthy();
	});

	it("handles missing headers object", async () => {
		const { mdcMiddleware } = await import("../mdcMiddleware");

		const mockStoreApi = {
			dispatch: () => {},
			getState: () => ({}),
		};

		const next = (action: any) => action;
		const chain = mdcMiddleware(mockStoreApi as any)(next as any);

		const action = {
			type: "TEST_ACTION",
			meta: {
				baseQueryMeta: {
					response: {},
				},
			},
		};

		const result = chain(action);
		expect(result).toBeTruthy();
	});

	it("handles partial MDC data", async () => {
		const { mdcMiddleware } = await import("../mdcMiddleware");

		let dispatchedAction: any = null;
		const mockStoreApi = {
			dispatch: (action: any) => {
				dispatchedAction = action;
			},
			getState: () => ({}),
		};

		const next = (action: any) => action;
		const chain = mdcMiddleware(mockStoreApi as any)(next as any);

		const action = {
			type: "TEST_ACTION",
			meta: {
				arg: {
					originalArgs: {
						"X-Span-Id": "span-only",
					},
				},
			},
		};

		chain(action);

		// Should still dispatch even with partial data
		expect(dispatchedAction).toBeTruthy();
	});
});
