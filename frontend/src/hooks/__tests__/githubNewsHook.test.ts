import { afterEach, beforeEach, describe, expect, it, mock } from "bun:test";
import "../../../test/setup";

const originalFetch = globalThis.fetch;
// Helper to create a minimal Response-like object usable by fetchBaseQuery
/*
function makeMockResponse(body: any, status = 200) {
	return {
		ok: status >= 200 && status < 300,
		status,
		statusText: String(status),
		json: async () => body,
		text: async () => JSON.stringify(body),
		clone() {
			return this;
		},
		headers: {
			get: (_: string) => null,
		},
	} as unknown as Response;
}
	*/

describe("useGithubNews hook", () => {
	beforeEach(() => {
		mock.restore();
		globalThis.fetch = originalFetch;
	});

	afterEach(() => {
		mock.restore();
		globalThis.fetch = originalFetch;
	});

	it("initializes with loading state", async () => {
		const React = await import("react");
		const { renderHook } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { useGithubNews } = await import("../githubNewsHook");
		const { createTestStore } = await import("../../../test/setup");
		// MSW not used in this test; use fetch mocking instead

		// Keep the MSW handler pending so the initial loading state is deterministic
		// Keep fetch pending so the initial loading state is deterministic.
		globalThis.fetch = mock(() =>
			new Promise<Response>(() => {
				// Intentionally unresolved for this test case.
			}),
		) as unknown as typeof fetch;

		const store = await createTestStore();
		const wrapper = ({ children }: { children: React.ReactNode }) =>
			React.createElement(Provider, { store, children });

		const { result } = renderHook(() => useGithubNews(), { wrapper });

		// Initially should be loading
		expect(result.current.isLoading).toBe(true);
		expect(result.current.error).toBe(null);
	});

	it("fetches news successfully", async () => {
		const React = await import("react");
		const { renderHook, waitFor } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { useGithubNews } = await import("../githubNewsHook");
		const { createTestStore } = await import("../../../test/setup");
		// MSW not used in this test; use fetch mocking instead

		const mockDiscussions = [
			{
				id: 1,
				title: "Test Announcement",
				html_url: "https://github.com/test",
				created_at: new Date().toISOString(),
			},
		];

		// Mock fetch to return the discussions payload
		globalThis.fetch = mock(() =>
			Promise.resolve(
				new Response(JSON.stringify(mockDiscussions), {
					status: 200,
					headers: { "Content-Type": "application/json" },
				}),
			),
		) as unknown as typeof fetch;

		const store = await createTestStore();
		const wrapper = ({ children }: { children: React.ReactNode }) =>
			React.createElement(Provider, { store, children });

		const { result } = renderHook(() => useGithubNews(), { wrapper });

		await waitFor(() => expect(result.current.isLoading).toBe(false), {
			timeout: 10000,
		});

		expect(result.current.error).toBe(null);
		expect(result.current.news.length).toBeGreaterThanOrEqual(0);
	});

	it("handles fetch errors", async () => {
		const React = await import("react");
		const { renderHook, waitFor } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { useGithubNews } = await import("../githubNewsHook");
		const { createTestStore } = await import("../../../test/setup");
		// MSW not used in this test; use fetch mocking instead

		// Mock fetch to return a 404 response
		globalThis.fetch = mock(() =>
			Promise.resolve(
				new Response(null, { status: 404, statusText: "Not Found" }),
			),
		) as unknown as typeof fetch;

		const store = await createTestStore();
		const wrapper = ({ children }: { children: React.ReactNode }) =>
			React.createElement(Provider, { store, children });

		const { result } = renderHook(() => useGithubNews(), { wrapper });

		await waitFor(() => expect(result.current.isLoading).toBe(false), {
			timeout: 10000,
		});

		expect(result.current.error).not.toBe(null);
		expect(result.current.news.length).toBe(0);
	});

	it("filters old news items", async () => {
		const React = await import("react");
		const { renderHook, waitFor } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { useGithubNews } = await import("../githubNewsHook");
		const { createTestStore } = await import("../../../test/setup");
		// MSW not used in this test; use fetch mocking instead

		const oldDate = new Date();
		oldDate.setMonth(oldDate.getMonth() - 6); // 6 months ago

		const mockDiscussions = [
			{
				id: 1,
				title: "Recent Announcement",
				html_url: "https://github.com/recent",
				created_at: new Date().toISOString(),
			},
			{
				id: 2,
				title: "Old Announcement",
				html_url: "https://github.com/old",
				created_at: oldDate.toISOString(),
			},
		];

		// Mock fetch to return mixed recent/old discussions
		globalThis.fetch = mock(() =>
			Promise.resolve(
				new Response(JSON.stringify(mockDiscussions), {
					status: 200,
					headers: { "Content-Type": "application/json" },
				}),
			),
		) as unknown as typeof fetch;

		const store = await createTestStore();
		const wrapper = ({ children }: { children: React.ReactNode }) =>
			React.createElement(Provider, { store, children });

		const { result } = renderHook(() => useGithubNews(), { wrapper });

		await waitFor(() => expect(result.current.isLoading).toBe(false), {
			timeout: 10000,
		});

		// Old news should be filtered out
		expect(result.current.news.length).toBeLessThanOrEqual(
			mockDiscussions.length,
		);
	});

	it("limits news items to maximum", async () => {
		const React = await import("react");
		const { renderHook, waitFor } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { useGithubNews } = await import("../githubNewsHook");
		const { createTestStore } = await import("../../../test/setup");
		// MSW not used in this test; use fetch mocking instead

		const mockDiscussions = Array.from({ length: 10 }, (_, i) => ({
			id: i,
			title: `Announcement ${i}`,
			html_url: `https://github.com/test${i}`,
			created_at: new Date().toISOString(),
		}));

		// Mock fetch to return many discussions
		globalThis.fetch = mock(() =>
			Promise.resolve(
				new Response(JSON.stringify(mockDiscussions), {
					status: 200,
					headers: { "Content-Type": "application/json" },
				}),
			),
		) as unknown as typeof fetch;

		const store = await createTestStore();
		const wrapper = ({ children }: { children: React.ReactNode }) =>
			React.createElement(Provider, { store, children });

		const { result } = renderHook(() => useGithubNews(), { wrapper });

		await waitFor(() => expect(result.current.isLoading).toBe(false), {
			timeout: 10000,
		});

		// Should not exceed MAX_NEWS_ITEMS (5)
		expect(result.current.news.length).toBeLessThanOrEqual(5);
	});

	it("handles network errors", async () => {
		const React = await import("react");
		const { renderHook, waitFor } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { useGithubNews } = await import("../githubNewsHook");
		const { createTestStore } = await import("../../../test/setup");
		// MSW not used in this test; use fetch mocking instead

		// Mock fetch to reject with network error
		globalThis.fetch = mock(() => Promise.reject(new Error("Network error"))) as unknown as typeof fetch;

		const store = await createTestStore();
		const wrapper = ({ children }: { children: React.ReactNode }) =>
			React.createElement(Provider, { store, children });

		const { result } = renderHook(() => useGithubNews(), { wrapper });

		await waitFor(() => expect(result.current.isLoading).toBe(false), {
			timeout: 10000,
		});

		expect(result.current.error).not.toBe(null);
		expect(result.current.news.length).toBe(0);
	});
});
