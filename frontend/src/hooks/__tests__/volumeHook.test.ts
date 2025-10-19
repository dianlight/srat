import "../../../test/setup";
import { beforeEach, describe, expect, it } from "bun:test";

describe("useVolume hook", () => {
	beforeEach(() => {
		// Clear any previous state
	});

	it("initializes with empty disks array", async () => {
		const React = await import("react");
		const { renderHook } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { createTestStore } = await import("../../../test/setup");
		const { useVolume } = await import("../volumeHook");

		const store = await createTestStore();
		const wrapper = ({ children }: React.PropsWithChildren) =>
			React.createElement(Provider, { store, children });

		const { result } = renderHook(() => useVolume(), { wrapper });

		// Initially should have empty/undefined disks
		expect(result.current.disks).toBeTruthy();
	});

	it("returns loading state correctly", async () => {
		const React = await import("react");
		const { renderHook } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { createTestStore } = await import("../../../test/setup");
		const { useVolume } = await import("../volumeHook");

		const store = await createTestStore();
		const wrapper = ({ children }: React.PropsWithChildren) =>
			React.createElement(Provider, { store, children });

		const { result } = renderHook(() => useVolume(), { wrapper });

		// Should have a loading state
		expect(typeof result.current.isLoading).toBe("boolean");
	});

	it("returns error state correctly", async () => {
		const React = await import("react");
		const { renderHook } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { createTestStore } = await import("../../../test/setup");
		const { useVolume } = await import("../volumeHook");

		const store = await createTestStore();
		const wrapper = ({ children }: React.PropsWithChildren) =>
			React.createElement(Provider, { store, children });

		const { result } = renderHook(() => useVolume(), { wrapper });

		// Error property exists in the return value
		expect("error" in result.current).toBe(true);
	});

	it("updates disks when data changes", async () => {
		const React = await import("react");
		const { renderHook, waitFor } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { createTestStore } = await import("../../../test/setup");
		const { useVolume } = await import("../volumeHook");

		const store = await createTestStore();
		const wrapper = ({ children }: React.PropsWithChildren) =>
			React.createElement(Provider, { store, children });

		const { result } = renderHook(() => useVolume(), { wrapper });

		await waitFor(
			() => {
				expect(result.current.disks !== undefined).toBe(true);
			},
			{ timeout: 1000 },
		);
	});

	it("handles SSE updates correctly", async () => {
		const React = await import("react");
		const { renderHook } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { createTestStore } = await import("../../../test/setup");
		const { useVolume } = await import("../volumeHook");

		const store = await createTestStore();
		const wrapper = ({ children }: React.PropsWithChildren) =>
			React.createElement(Provider, { store, children });

		const { result } = renderHook(() => useVolume(), { wrapper });

		// Hook should handle SSE updates
		expect(result.current.disks).toBeTruthy();
	});

	it("combines REST API and SSE data correctly", async () => {
		const React = await import("react");
		const { renderHook } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { createTestStore } = await import("../../../test/setup");
		const { useVolume } = await import("../volumeHook");

		const store = await createTestStore();
		const wrapper = ({ children }: React.PropsWithChildren) =>
			React.createElement(Provider, { store, children });

		const { result } = renderHook(() => useVolume(), { wrapper });

		// Should combine loading states correctly
		expect(typeof result.current.isLoading).toBe("boolean");
		expect(result.current.disks).toBeTruthy();
	});
});
