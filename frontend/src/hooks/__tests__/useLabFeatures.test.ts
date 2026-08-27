import { beforeEach, describe, expect, it } from "vitest";

async function setup() {
	const React = await import("react");
	const { renderHook, waitFor } = await import("@testing-library/react");
	const { Provider } = await import("react-redux");
	const { getMswServer } = await import("/test/testing");
	const { http, HttpResponse } = await import("msw");
	const { useLabFeatures } = await import("../useLabFeatures");
	return { React, renderHook, waitFor, Provider, getMswServer, http, HttpResponse, useLabFeatures };
}

describe("useLabFeatures hook", () => {
	beforeEach(() => {
		// Each test builds its own store + MSW overrides.
	});

	it("returns the server registry and availability", async () => {
		const { React, renderHook, waitFor, Provider, useLabFeatures } = await setup();
		const { createTestStore } = await import("/test/testing");
		const store = await createTestStore();
		const wrapper = ({ children }: React.PropsWithChildren) =>
			React.createElement(Provider, { store, children });

		const { result } = renderHook(() => useLabFeatures(), { wrapper });

		// Shared mock (customHandlers.ts) returns all features with available: true.
		await waitFor(() => expect(result.current.isLoading).toBe(false));
		expect(result.current.features.length).toBeGreaterThan(0);
		expect(result.current.isAvailable("hdidle")).toBe(true);
	});

	it("isAvailable is fail-closed for unknown keys", async () => {
		const { React, renderHook, waitFor, Provider, useLabFeatures } = await setup();
		const { createTestStore } = await import("/test/testing");
		const store = await createTestStore();
		const wrapper = ({ children }: React.PropsWithChildren) =>
			React.createElement(Provider, { store, children });

		const { result } = renderHook(() => useLabFeatures(), { wrapper });
		await waitFor(() => expect(result.current.isLoading).toBe(false));

		expect(result.current.isAvailable("no_such_feature")).toBe(false);
	});

	it("is fail-closed when the endpoint errors", async () => {
		const { React, renderHook, waitFor, Provider, getMswServer, http, HttpResponse, useLabFeatures } =
			await setup();
		const { createTestStore } = await import("/test/testing");

		const server = getMswServer();
		server.use(http.get(/.*\/api\/lab_features(?:\\?.*)?$/, () => HttpResponse.json(null, { status: 500 })));

		const store = await createTestStore();
		const wrapper = ({ children }: React.PropsWithChildren) =>
			React.createElement(Provider, { store, children });

		const { result } = renderHook(() => useLabFeatures(), { wrapper });
		await waitFor(() => expect(result.current.isLoading).toBe(false));

		expect(result.current.features).toEqual([]);
		expect(result.current.isAvailable("hdidle")).toBe(false);
	});
});
