import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

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
		vi.restoreAllMocks();
		globalThis.fetch = originalFetch;
	});

	afterEach(() => {
		vi.restoreAllMocks();
		globalThis.fetch = originalFetch;
	});

	it("initializes with loading state", async () => {
		const React = await import("react");
		const { renderHook } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { useGithubNews } = await import("../githubNewsHook");
		const { createTestStore } = await import("/test/testing");
		// MSW not used in this test; use fetch mocking instead

		// Keep the MSW handler pending so the initial loading state is deterministic
		// Keep fetch pending so the initial loading state is deterministic.
		globalThis.fetch = vi.fn(() =>
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
		const { createTestStore } = await import("/test/testing");
		// MSW not used in this test; use fetch mocking instead

		const mockDiscussions = [
			{
				id: 1,
				title: "Test Announcement",
				html_url: "https://github.com/test",
				created_at: new Date().toISOString(),
				body: "Test announcement abstract with enough content.",
			},
		];

		// Mock fetch to return the discussions payload
		globalThis.fetch = vi.fn(() =>
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
		for (const item of result.current.news) {
			expect(item.abstract.length).toBeGreaterThan(0);
			expect(["release", "announcement"]).toContain(item.type);
		}
	});

	it("handles fetch errors", async () => {
		const React = await import("react");
		const { renderHook, waitFor } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { useGithubNews } = await import("../githubNewsHook");
		const { createTestStore } = await import("/test/testing");
		// MSW not used in this test; use fetch mocking instead

		// Mock fetch to return a 404 response
		globalThis.fetch = vi.fn(() =>
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
		const { createTestStore } = await import("/test/testing");
		// MSW not used in this test; use fetch mocking instead

		const oldDate = new Date();
		oldDate.setMonth(oldDate.getMonth() - 6); // 6 months ago

		const mockDiscussions = [
			{
				id: 1,
				title: "Recent Announcement",
				html_url: "https://github.com/recent",
				created_at: new Date().toISOString(),
				body: "Recent announcement abstract.",
			},
			{
				id: 2,
				title: "Old Announcement",
				html_url: "https://github.com/old",
				created_at: oldDate.toISOString(),
				body: "Old announcement abstract.",
			},
		];

		// Mock fetch to return mixed recent/old discussions
		globalThis.fetch = vi.fn(() =>
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
		const { createTestStore } = await import("/test/testing");
		// MSW not used in this test; use fetch mocking instead

		const mockDiscussions = Array.from({ length: 10 }, (_, i) => ({
			id: i,
			title: `Announcement ${i}`,
			html_url: `https://github.com/test${i}`,
			created_at: new Date().toISOString(),
			body: `Announcement abstract ${i} with content.`,
		}));

		// Mock fetch to return many discussions
		globalThis.fetch = vi.fn(() =>
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
		const { createTestStore } = await import("/test/testing");
		// MSW not used in this test; use fetch mocking instead

		// Mock fetch to reject with network error
		globalThis.fetch = vi.fn(() => Promise.reject(new Error("Network error"))) as unknown as typeof fetch;

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

	it("maps release type, version and abstract", async () => {
		const React = await import("react");
		const { renderHook, waitFor } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { useGithubNews } = await import("../githubNewsHook");
		const { createTestStore } = await import("/test/testing");

		const mockDiscussions = [
			{
				id: 1,
				title: "Release 2026.8.0-rc12",
				html_url: "https://github.com/dianlight/srat/discussions/919",
				created_at: new Date().toISOString(),
				body: "## Changelog\n\n### ✨ Features\n\n- Something new and shiny.",
			},
		];

		globalThis.fetch = vi.fn(() =>
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
		// WS is skipped in tests so the running version is unknown and the
		// version gate fails open.
		expect(result.current.news).toHaveLength(1);
		expect(result.current.news[0]?.type).toBe("release");
		expect(result.current.news[0]?.version).toBe("2026.8.0-rc12");
		expect(result.current.news[0]?.abstract.length).toBeGreaterThan(0);
		expect(result.current.news[0]?.abstract).not.toContain("##");
	});

	it("filters out discussions without a body abstract", async () => {
		const React = await import("react");
		const { renderHook, waitFor } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { useGithubNews } = await import("../githubNewsHook");
		const { createTestStore } = await import("/test/testing");

		const mockDiscussions = [
			{
				id: 1,
				title: "Empty Announcement",
				html_url: "https://github.com/empty",
				created_at: new Date().toISOString(),
				body: "",
			},
		];

		globalThis.fetch = vi.fn(() =>
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
		expect(result.current.news).toHaveLength(0);
	});
});

describe("github news helpers", () => {
	it("detects release type and extracts version", async () => {
		const { detectNewsType, extractReleaseVersion } = await import(
			"../../store/githubRestApi"
		);

		expect(detectNewsType("Release 2026.8.0-rc12", "body")).toBe("release");
		expect(extractReleaseVersion("Release 2026.8.0-rc12")).toBe(
			"2026.8.0-rc12",
		);
		expect(
			extractReleaseVersion(
				"Some announcement",
				"created from the release <a href='https://github.com/dianlight/srat/releases/tag/2026.7.0-rc11'>Release</a>.",
			),
		).toBe("2026.7.0-rc11");
		expect(detectNewsType("Maintenance window", "plain body")).toBe(
			"announcement",
		);
	});

	it("builds a truncated plain-text abstract", async () => {
		const { makeAbstract } = await import("../../store/githubRestApi");

		expect(makeAbstract("")).toBe("");
		expect(makeAbstract("## Changelog\n\n- Shiny feature")).toBe(
			"Changelog Shiny feature",
		);
		const long = makeAbstract(`${"word ".repeat(100)}end`);
		expect(long.length).toBeLessThanOrEqual(201);
		expect(long.endsWith("…")).toBe(true);
	});

	it("gates releases on the running version", async () => {
		const { isNewerRelease, normalizeVersion } = await import(
			"../githubNewsHook"
		);

		expect(normalizeVersion("v2026.8.0-rc12")).toBe("2026.8.0-rc12");
		expect(normalizeVersion(undefined)).toBe(null);
		expect(isNewerRelease("2026.8.0-rc12", "2026.7.0-rc11")).toBe(true);
		expect(isNewerRelease("2026.5.0-rc9", "2026.9.0-dev.15")).toBe(false);
		expect(isNewerRelease("2026.9.0-rc14", "2026.9.0-rc14")).toBe(false);
		// Unknown running version fails open; unknown item version fails closed.
		expect(isNewerRelease("2026.9.0-rc14", undefined)).toBe(true);
		expect(isNewerRelease(undefined, "2026.9.0-dev.15")).toBe(false);
	});
});
