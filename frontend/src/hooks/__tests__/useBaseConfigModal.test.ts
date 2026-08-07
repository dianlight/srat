import type { JsonBodyType } from "msw";
import { beforeEach, describe, expect, it } from "vitest";

/**
 * Tests for the base config modal hook, in particular the user-array
 * validation (isValidUsers) added to reject malformed/error-shaped
 * responses instead of treating them as a list of users.
 */

type ApiQueriesState = {
	api?: {
		queries?: Record<
			string,
			{ status?: string; endpointName?: string } | undefined
		>;
	};
};

async function setupApiHandlers(settings: JsonBodyType, users: JsonBodyType) {
	const { getMswServer } = await import("/test/testing");
	const { http, HttpResponse } = await import("msw");

	const server = getMswServer();
	server.use(
		http.get(/.*\/api\/settings(?:\?.*)?$/, () =>
			HttpResponse.json(settings),
		),
		http.get(/.*\/api\/users(?:\?.*)?$/, () => HttpResponse.json(users)),
	);
}

async function renderModalHook() {
	const React = await import("react");
	const { renderHook, waitFor } = await import("@testing-library/react");
	const { Provider } = await import("react-redux");
	const { createTestStore } = await import("/test/testing");
	const { useBaseConfigModal } = await import("../useBaseConfigModal");

	const store = await createTestStore();
	const wrapper = ({ children }: React.PropsWithChildren) =>
		React.createElement(Provider, { store, children });

	const { result } = renderHook(() => useBaseConfigModal(), { wrapper });

	// Wait until both RTK Query requests have resolved so the effect that
	// decides whether to show the modal has run at least once.
	await waitFor(() => {
		const state = store.getState() as ApiQueriesState;
		const queries = Object.values(state.api?.queries ?? {});
		const names = queries.map((q) => q?.endpointName).sort();
		expect(names).toContain("getApiSettings");
		expect(names).toContain("getApiUsers");
		expect(queries.every((q) => q?.status === "fulfilled")).toBe(true);
	});

	return { result, waitFor };
}

describe("useBaseConfigModal hook", () => {
	beforeEach(() => {
		if (typeof localStorage !== "undefined" && localStorage.clear) {
			localStorage.clear();
		}
	});

	it("shows the modal when the admin user still has the default password", async () => {
		await setupApiHandlers(
			{ hostname: "mynas", workgroup: "WORKGROUP" },
			[{ is_admin: true, has_default_password: true, username: "admin" }],
		);

		const { result, waitFor } = await renderModalHook();

		await waitFor(() => expect(result.current.shouldShow).toBe(true));
	});

	it("does not show the modal when users is an array of malformed objects", async () => {
		// A malformed entry that looks admin-like but has no username must
		// not be accepted as a user (the guard validates each element).
		await setupApiHandlers(
			{ hostname: "mynas", workgroup: "WORKGROUP" },
			[{ is_admin: true, has_default_password: true }],
		);

		const { result, waitFor } = await renderModalHook();

		await waitFor(() => expect(result.current.shouldShow).toBe(false));
	});

	it("does not crash when users contains null entries", async () => {
		// Without element validation, `users.find((u) => u.is_admin)` throws
		// on a null entry; the guard must reject the array instead.
		await setupApiHandlers(
			{ hostname: "mynas", workgroup: "WORKGROUP" },
			[null],
		);

		const { result, waitFor } = await renderModalHook();

		await waitFor(() => expect(result.current.shouldShow).toBe(false));
	});

	it("shows the modal on first-time setup when hostname or workgroup are missing", async () => {
		await setupApiHandlers(
			{ hostname: "", workgroup: "" },
			[{ is_admin: true, has_default_password: false, username: "admin" }],
		);

		const { result, waitFor } = await renderModalHook();

		await waitFor(() => expect(result.current.shouldShow).toBe(true));
	});
});
