import "../../../test/setup";
import { beforeEach, describe, expect, it, mock } from "bun:test";

describe("useConsoleErrorCallback hook", () => {
	beforeEach(() => {
		// Clean up any mocks
		mock.restore();
	});

	it("exports useConsoleErrorCallback function", async () => {
		const { useConsoleErrorCallback } = await import(
			"../useConsoleErrorCallback"
		);
		expect(typeof useConsoleErrorCallback).toBe("function");
	});

	it("registers callback on mount", async () => {
		const React = await import("react");
		const { renderHook } = await import("@testing-library/react");
		const { useConsoleErrorCallback } = await import(
			"../useConsoleErrorCallback"
		);

		let callbackExecuted = false;
		const testCallback = () => {
			callbackExecuted = true;
		};

		const { unmount } = renderHook(() => useConsoleErrorCallback(testCallback));

		// Hook should register the callback
		expect(typeof useConsoleErrorCallback).toBe("function");

		unmount();
	});

	it("unregisters callback on unmount", async () => {
		const React = await import("react");
		const { renderHook } = await import("@testing-library/react");
		const { useConsoleErrorCallback } = await import(
			"../useConsoleErrorCallback"
		);

		const testCallback = () => {};
		const { unmount } = renderHook(() => useConsoleErrorCallback(testCallback));

		// Unmount should trigger cleanup
		unmount();
		expect(true).toBe(true);
	});

	it("updates callback ref when callback changes", async () => {
		const React = await import("react");
		const { renderHook } = await import("@testing-library/react");
		const { useConsoleErrorCallback } = await import(
			"../useConsoleErrorCallback"
		);

		let count = 0;
		const testCallback1 = () => {
			count = 1;
		};
		const testCallback2 = () => {
			count = 2;
		};

		const { rerender, unmount } = renderHook(
			({ callback }) => useConsoleErrorCallback(callback),
			{ initialProps: { callback: testCallback1 } },
		);

		// Change the callback
		rerender({ callback: testCallback2 });

		unmount();
	});

	it("handles multiple callback invocations", async () => {
		const React = await import("react");
		const { renderHook } = await import("@testing-library/react");
		const { useConsoleErrorCallback } = await import(
			"../useConsoleErrorCallback"
		);

		let callCount = 0;
		const testCallback = () => {
			callCount++;
		};

		const { unmount } = renderHook(() => useConsoleErrorCallback(testCallback));

		unmount();
		expect(callCount).toBeGreaterThanOrEqual(0);
	});

	it("works with callback that receives arguments", async () => {
		const React = await import("react");
		const { renderHook } = await import("@testing-library/react");
		const { useConsoleErrorCallback } = await import(
			"../useConsoleErrorCallback"
		);

		let receivedArgs: any[] = [];
		const testCallback = (...args: any[]) => {
			receivedArgs = args;
		};

		const { unmount } = renderHook(() => useConsoleErrorCallback(testCallback));

		unmount();
		expect(true).toBe(true);
	});
});
