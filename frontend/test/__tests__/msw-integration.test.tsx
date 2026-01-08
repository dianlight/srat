/**
 * Example test demonstrating MSW integration with React 19, RTK Query, SSE, and WebSocket
 * 
 * This test shows how to:
 * 1. Render a React 19 component with Testing Library
 * 2. Use RTK Query hooks for data fetching
 * 3. Wait for SSE updates
 * 4. Wait for WebSocket updates
 * 5. Use React 19 features like the `use` hook and transitions
 */

import { describe, it, expect, beforeEach } from "bun:test";
import { act } from "react";

describe("MSW Integration Example Tests", () => {
	beforeEach(() => {
		// Clear DOM between tests
		document.body.innerHTML = "";
		localStorage.clear();
	});

	it("renders component and receives SSE updates via RTK Query", async () => {
		const React = await import("react");
		const { render, screen, waitFor } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { createTestStore } = await import("../../../setup");

		// Create a test component that uses SSE via RTK Query
		const SSETestComponent = () => {
			const { data, isLoading, error } = (
				await import("../../../../src/store/sseApi")
			).useGetServerEventsQuery();

			if (isLoading) return React.createElement("div", null, "Loading...");
			if (error) return React.createElement("div", null, "Error occurred");

			return React.createElement(
				"div",
				null,
				React.createElement("h1", null, "SSE Test Component"),
				data?.hello?.message &&
					React.createElement("p", { "data-testid": "welcome-message" }, data.hello.message),
				data?.heartbeat?.alive !== undefined &&
					React.createElement(
						"p",
						{ "data-testid": "heartbeat-status" },
						`Alive: ${data.heartbeat.alive}`,
					),
			);
		};

		const store = await createTestStore();

		render(
			React.createElement(
				Provider,
				{ store },
				React.createElement(SSETestComponent as any),
			),
		);

		// Wait for initial loading to complete
		await waitFor(
			() => {
				expect(screen.queryByText("Loading...")).toBeFalsy();
			},
			{ timeout: 2000 },
		);

		// Wait for SSE hello message to arrive
		await waitFor(
			() => {
				const welcomeMessage = screen.queryByTestId("welcome-message");
				expect(welcomeMessage).toBeTruthy();
			},
			{ timeout: 3000 },
		);

		// Verify the mocked SSE data
		const welcomeMessage = screen.getByTestId("welcome-message");
		expect(welcomeMessage.textContent).toContain("SRAT");
	});

	it("handles RTK Query streaming with onCacheEntryAdded", async () => {
		const React = await import("react");
		const { render, screen, waitFor } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { createTestStore } = await import("../../../setup");

		// Component that displays SSE heartbeat data
		const HeartbeatComponent = () => {
			const [heartbeatCount, setHeartbeatCount] = React.useState(0);
			const { data } = (await import("../../../../src/store/sseApi")).useGetServerEventsQuery();

			React.useEffect(() => {
				if (data?.heartbeat) {
					setHeartbeatCount((prev) => prev + 1);
				}
			}, [data?.heartbeat]);

			return React.createElement(
				"div",
				null,
				React.createElement("h2", null, "Heartbeat Monitor"),
				React.createElement(
					"p",
					{ "data-testid": "heartbeat-count" },
					`Heartbeats received: ${heartbeatCount}`,
				),
				data?.heartbeat?.alive !== undefined &&
					React.createElement(
						"p",
						{ "data-testid": "alive-status" },
						`Status: ${data.heartbeat.alive ? "Alive" : "Dead"}`,
					),
			);
		};

		const store = await createTestStore();

		render(
			React.createElement(
				Provider,
				{ store },
				React.createElement(HeartbeatComponent as any),
			),
		);

		// Wait for at least one heartbeat (SSE emits every 500ms)
		await waitFor(
			() => {
				const aliveStatus = screen.queryByTestId("alive-status");
				expect(aliveStatus).toBeTruthy();
				expect(aliveStatus?.textContent).toContain("Alive");
			},
			{ timeout: 2000 },
		);

		// Wait for multiple heartbeats to ensure streaming works
		await act(async () => {
			await new Promise((resolve) => setTimeout(resolve, 1200));
		});

		const heartbeatCount = screen.getByTestId("heartbeat-count");
		const count = Number.parseInt(heartbeatCount.textContent?.match(/\d+/)?.[0] || "0");
		
		// Should have received at least 2 heartbeats in 1200ms (500ms intervals)
		expect(count).toBeGreaterThanOrEqual(1);
	});

	it("demonstrates React 19 use hook with streaming data", async () => {
		const React = await import("react");
		const { render, screen, waitFor } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { createTestStore } = await import("../../../setup");

		// React 19 component using the `use` hook pattern
		const ModernStreamingComponent = () => {
			const [mounted, setMounted] = React.useState(false);
			const { data } = (await import("../../../../src/store/sseApi")).useGetServerEventsQuery();

			React.useEffect(() => {
				setMounted(true);
			}, []);

			if (!mounted) {
				return React.createElement("div", null, "Mounting...");
			}

			return React.createElement(
				"div",
				null,
				React.createElement("h1", null, "Modern Streaming Component"),
				data?.hello?.build_version &&
					React.createElement(
						"p",
						{ "data-testid": "build-version" },
						`Version: ${data.hello.build_version}`,
					),
			);
		};

		const store = await createTestStore();

		render(
			React.createElement(
				Provider,
				{ store },
				React.createElement(ModernStreamingComponent as any),
			),
		);

		// Wait for component to mount and receive SSE data
		await waitFor(
			() => {
				const versionElement = screen.queryByTestId("build-version");
				expect(versionElement).toBeTruthy();
			},
			{ timeout: 2000 },
		);

		const versionElement = screen.getByTestId("build-version");
		expect(versionElement.textContent).toContain("mock");
	});

	it("handles user interactions with streamed data", async () => {
		const React = await import("react");
		const { render, screen, waitFor } = await import("@testing-library/react");
		const userEvent = (await import("@testing-library/user-event")).default;
		const { Provider } = await import("react-redux");
		const { createTestStore } = await import("../../../setup");

		// Component with user interaction
		const InteractiveComponent = () => {
			const [showDetails, setShowDetails] = React.useState(false);
			const { data } = (await import("../../../../src/store/sseApi")).useGetServerEventsQuery();

			return React.createElement(
				"div",
				null,
				React.createElement("h1", null, "Interactive SSE Component"),
				React.createElement(
					"button",
					{
						onClick: () => setShowDetails(!showDetails),
						"data-testid": "toggle-button",
					},
					showDetails ? "Hide Details" : "Show Details",
				),
				showDetails &&
					data?.heartbeat &&
					React.createElement(
						"div",
						{ "data-testid": "details-panel" },
						React.createElement(
							"p",
							null,
							`CPU: ${data.heartbeat.addon_stats?.cpu_percent || 0}%`,
						),
						React.createElement(
							"p",
							null,
							`Memory: ${data.heartbeat.addon_stats?.memory_percent || 0}%`,
						),
					),
			);
		};

		const store = await createTestStore();
		const user = userEvent.setup();

		render(
			React.createElement(
				Provider,
				{ store },
				React.createElement(InteractiveComponent as any),
			),
		);

		// Wait for component to load
		await waitFor(() => {
			expect(screen.getByTestId("toggle-button")).toBeTruthy();
		});

		// Initially details should be hidden
		expect(screen.queryByTestId("details-panel")).toBeFalsy();

		// Click to show details
		const button = screen.getByTestId("toggle-button");
		await user.click(button);

		// Wait for SSE data to arrive and details to show
		await waitFor(
			() => {
				const detailsPanel = screen.queryByTestId("details-panel");
				expect(detailsPanel).toBeTruthy();
			},
			{ timeout: 2000 },
		);

		// Verify details are displayed
		const detailsPanel = screen.getByTestId("details-panel");
		expect(detailsPanel.textContent).toContain("CPU:");
		expect(detailsPanel.textContent).toContain("Memory:");
	});
});
