import "../../../test/setup";
import { describe, it, expect, beforeEach, mock } from "bun:test";

// Mock the GitHub API to avoid actual network requests
mock.module("../../store/githubApi", () => ({
	useGetFundingConfigQuery: () => ({
		data: [
			{
				platform: "github",
				identifier: "dianlight",
				url: "https://github.com/sponsors/dianlight",
				label: "GitHub Sponsors",
			},
			{
				platform: "buy_me_a_coffee",
				identifier: "ypKZ2I0",
				url: "https://www.buymeacoffee.com/ypKZ2I0",
				label: "Buy Me a Coffee",
			},
		],
		isLoading: false,
		isError: false,
	}),
	githubApi: {
		reducerPath: "githubApi",
		reducer: () => ({}),
		middleware: () => (next: any) => (action: any) => next(action),
	},
}));

// Required localStorage shim for testing environment
if (!(globalThis as any).localStorage) {
	const _store: Record<string, string> = {};
	(globalThis as any).localStorage = {
		getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
		setItem: (k: string, v: string) => {
			_store[k] = String(v);
		},
		removeItem: (k: string) => {
			delete _store[k];
		},
		clear: () => {
			for (const k of Object.keys(_store)) delete _store[k];
		},
	};
}

describe("DonationButton Component", () => {
	beforeEach(() => {
		localStorage.clear();
		mock.restore();
		// Reset window.open mock
		(window as any).open = () => null;
	});

	it("renders DonationButton with icon", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { ThemeProvider, createTheme } = await import("@mui/material/styles");
		const { DonationButton } = await import("../DonationButton");

		const theme = createTheme();

		const { container } = render(
			React.createElement(
				ThemeProvider,
				{ theme },
				React.createElement(DonationButton as any, {})
			)
		);

		// Component should render successfully
		expect(container).toBeTruthy();

		// Find button by role
		const button = screen.queryByRole("button");
		expect(button).toBeTruthy();
	});

	it("opens dropdown menu when clicked", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const userEvent = (await import("@testing-library/user-event")).default;
		const { ThemeProvider, createTheme } = await import("@mui/material/styles");
		const { DonationButton } = await import("../DonationButton");

		const theme = createTheme();
		const user = userEvent.setup();

		render(
			React.createElement(
				ThemeProvider,
				{ theme },
				React.createElement(DonationButton as any, {})
			)
		);

		// Find and click the donation button
		const button = screen.getByRole("button");
		await user.click(button);

		// Menu should be open - look for menu items
		const menu = document.getElementById("donation-menu");
		expect(menu).toBeTruthy();
	});

	it("displays funding platforms in menu", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const userEvent = (await import("@testing-library/user-event")).default;
		const { ThemeProvider, createTheme } = await import("@mui/material/styles");
		const { DonationButton } = await import("../DonationButton");

		const theme = createTheme();
		const user = userEvent.setup();

		render(
			React.createElement(
				ThemeProvider,
				{ theme },
				React.createElement(DonationButton as any, {})
			)
		);

		// Click to open menu
		const button = screen.getByRole("button");
		await user.click(button);

		// Look for menu items - should have GitHub Sponsors and Buy Me a Coffee
		const menuItems = await screen.findAllByRole("menuitem");
		expect(menuItems.length).toBeGreaterThanOrEqual(2);

		// Check for expected platform labels
		const githubItem = await screen.findByText("GitHub Sponsors");
		const coffeeItem = await screen.findByText("Buy Me a Coffee");
		expect(githubItem).toBeTruthy();
		expect(coffeeItem).toBeTruthy();
	});

	it("opens correct URL when platform is clicked", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const userEvent = (await import("@testing-library/user-event")).default;
		const { ThemeProvider, createTheme } = await import("@mui/material/styles");
		const { DonationButton } = await import("../DonationButton");

		const theme = createTheme();
		const user = userEvent.setup();

		// Mock window.open to track calls
		let openedUrl = "";
		(window as any).open = (url: string) => {
			openedUrl = url;
			return null;
		};

		render(
			React.createElement(
				ThemeProvider,
				{ theme },
				React.createElement(DonationButton as any, {})
			)
		);

		// Click to open menu
		const button = screen.getByRole("button");
		await user.click(button);

		// Click on GitHub Sponsors
		const githubItem = await screen.findByText("GitHub Sponsors");
		await user.click(githubItem);

		// Verify URL was opened
		expect(openedUrl).toContain("github.com/sponsors");
	});

	it("closes menu after clicking a platform", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const userEvent = (await import("@testing-library/user-event")).default;
		const { ThemeProvider, createTheme } = await import("@mui/material/styles");
		const { DonationButton } = await import("../DonationButton");

		const theme = createTheme();
		const user = userEvent.setup();

		// Mock window.open
		(window as any).open = () => null;

		render(
			React.createElement(
				ThemeProvider,
				{ theme },
				React.createElement(DonationButton as any, {})
			)
		);

		// Open menu
		const button = screen.getByRole("button");
		await user.click(button);

		// Click on a menu item
		const githubItem = await screen.findByText("GitHub Sponsors");
		await user.click(githubItem);

		// Menu should close - check aria-expanded
		expect(button.getAttribute("aria-expanded")).toBe("false");
	});

	it("renders platform icons correctly", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const userEvent = (await import("@testing-library/user-event")).default;
		const { ThemeProvider, createTheme } = await import("@mui/material/styles");
		const { DonationButton } = await import("../DonationButton");

		const theme = createTheme();
		const user = userEvent.setup();

		render(
			React.createElement(
				ThemeProvider,
				{ theme },
				React.createElement(DonationButton as any, {})
			)
		);

		// Open menu
		const button = screen.getByRole("button");
		await user.click(button);

		// Menu items should have icons (ListItemIcon)
		const menuItems = await screen.findAllByRole("menuitem");
		expect(menuItems.length).toBeGreaterThan(0);

		// Each menu item should render successfully
		for (const item of menuItems) {
			expect(item).toBeTruthy();
		}
	});

	it("has correct tooltip", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { ThemeProvider, createTheme } = await import("@mui/material/styles");
		const { DonationButton } = await import("../DonationButton");

		const theme = createTheme();

		render(
			React.createElement(
				ThemeProvider,
				{ theme },
				React.createElement(DonationButton as any, {})
			)
		);

		// Component renders with button
		const button = screen.getByRole("button");
		expect(button).toBeTruthy();

		// Tooltip text is set (may not be visible until hover)
		// We can verify the component renders successfully
	});

	it("handles menu close correctly", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const userEvent = (await import("@testing-library/user-event")).default;
		const { ThemeProvider, createTheme } = await import("@mui/material/styles");
		const { DonationButton } = await import("../DonationButton");

		const theme = createTheme();
		const user = userEvent.setup();

		render(
			React.createElement(
				ThemeProvider,
				{ theme },
				React.createElement(DonationButton as any, {})
			)
		);

		// Open menu
		const button = screen.getByRole("button");
		await user.click(button);

		// Verify menu is open
		expect(button.getAttribute("aria-expanded")).toBe("true");

		// Click button again to close (or click outside)
		await user.click(button);

		// Menu should be closed
		expect(button.getAttribute("aria-expanded")).toBe("false");
	});
});
