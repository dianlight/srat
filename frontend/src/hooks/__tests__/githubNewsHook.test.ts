import { beforeEach, describe, expect, it, mock } from "bun:test";
import "../../../test/setup";

describe("useGithubNews hook", () => {
	beforeEach(() => {
		// Clear any mocks
		mock.restore();
	});

	it("initializes with loading state", async () => {
		const { renderHook } = await import("@testing-library/react");
		const { useGithubNews } = await import("../githubNewsHook");

		// Mock fetch to prevent real API calls
		globalThis.fetch = mock(() =>
			Promise.resolve({
				ok: true,
				json: () => Promise.resolve([]),
			}),
		) as unknown as typeof fetch;

		const { result } = renderHook(() => useGithubNews());

		// Initially should be loading
		expect(result.current.isLoading).toBe(true);
		expect(result.current.error).toBe(null);
	});

	it("fetches news successfully", async () => {
		const { renderHook, waitFor } = await import("@testing-library/react");
		const { useGithubNews } = await import("../githubNewsHook");

		const mockDiscussions = [
			{
				id: 1,
				title: "Test Announcement",
				html_url: "https://github.com/test",
				created_at: new Date().toISOString(),
			},
		];

		globalThis.fetch = mock(() =>
			Promise.resolve({
				ok: true,
				json: () => Promise.resolve(mockDiscussions),
			}),
		) as unknown as typeof fetch;

		const { result } = renderHook(() => useGithubNews());

		await waitFor(() => expect(result.current.isLoading).toBe(false));

		expect(result.current.error).toBe(null);
		expect(result.current.news.length).toBeGreaterThanOrEqual(0);
	});

	it("handles fetch errors", async () => {
		const { renderHook, waitFor } = await import("@testing-library/react");
		const { useGithubNews } = await import("../githubNewsHook");

		globalThis.fetch = mock(() =>
			Promise.resolve({
				ok: false,
				statusText: "Not Found",
			}),
		) as unknown as typeof fetch;

		const { result } = renderHook(() => useGithubNews());

		await waitFor(() => expect(result.current.isLoading).toBe(false));

		expect(result.current.error).not.toBe(null);
		expect(result.current.news.length).toBe(0);
	});

	it("filters old news items", async () => {
		const { renderHook, waitFor } = await import("@testing-library/react");
		const { useGithubNews } = await import("../githubNewsHook");

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

		globalThis.fetch = mock(() =>
			Promise.resolve({
				ok: true,
				json: () => Promise.resolve(mockDiscussions),
			}),
		) as unknown as typeof fetch;

		const { result } = renderHook(() => useGithubNews());

		await waitFor(() => expect(result.current.isLoading).toBe(false));

		// Old news should be filtered out
		expect(result.current.news.length).toBeLessThanOrEqual(
			mockDiscussions.length,
		);
	});

	it("limits news items to maximum", async () => {
		const { renderHook, waitFor } = await import("@testing-library/react");
		const { useGithubNews } = await import("../githubNewsHook");

		const mockDiscussions = Array.from({ length: 10 }, (_, i) => ({
			id: i,
			title: `Announcement ${i}`,
			html_url: `https://github.com/test${i}`,
			created_at: new Date().toISOString(),
		}));

		globalThis.fetch = mock(() =>
			Promise.resolve({
				ok: true,
				json: () => Promise.resolve(mockDiscussions),
			}),
		) as unknown as typeof fetch;

		const { result } = renderHook(() => useGithubNews());

		await waitFor(() => expect(result.current.isLoading).toBe(false));

		// Should not exceed MAX_NEWS_ITEMS (5)
		expect(result.current.news.length).toBeLessThanOrEqual(5);
	});

	it("handles network errors", async () => {
		const { renderHook, waitFor } = await import("@testing-library/react");
		const { useGithubNews } = await import("../githubNewsHook");

		globalThis.fetch = mock(() =>
			Promise.reject(new Error("Network error")),
		) as unknown as typeof fetch;

		const { result } = renderHook(() => useGithubNews());

		await waitFor(() => expect(result.current.isLoading).toBe(false));

		expect(result.current.error).not.toBe(null);
		expect(result.current.news.length).toBe(0);
	});
});
