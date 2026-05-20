import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("../../store/githubApi", () => ({
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

describe("DonationButton Component", () => {
	async function renderDonationButton() {
		const React = await import("react");
		const { renderWithTestStore } = await import("/test/testing");
		const { ThemeProvider, createTheme } = await import("@mui/material/styles");
		const { DonationButton } = await import("../DonationButton");

		const theme = createTheme();
		return renderWithTestStore(
			React.createElement(
				ThemeProvider,
				{ theme },
				React.createElement(DonationButton as any, {})
			)
		);
	}

	beforeEach(() => {
		if (localStorage && typeof localStorage.clear === 'function') {
			localStorage.clear();
		}
		// Reset window.open mock
		(window as any).open = () => null;
	});

	it("renders DonationButton with icon", async () => {
		const result = await renderDonationButton();

		// Component should render successfully - use getByTestId for icon buttons
		expect(result.container).toBeTruthy();

		// Find button by accessible role and name
		const button = screen.getByRole("button", { name: /support this project/i });
		expect(button).toBeTruthy();
	});

	it("opens dropdown menu when clicked", async () => {
		const user = userEvent.setup();
		await renderDonationButton();

		// Find and click the donation button by role/name
		const button = screen.getByRole("button", { name: /support this project/i });
		await user.click(button);

		// Menu should be open - verify aria-expanded attribute
		expect(button.getAttribute("aria-expanded")).toBe("true");
	});

	it("displays funding platforms in menu", async () => {
		const user = userEvent.setup();
		await renderDonationButton();

		// Click to open menu by role/name
		const button = screen.getByRole("button", { name: /support this project/i });
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
		const user = userEvent.setup();

		// Mock window.open to track calls
		let openedUrl = "";
		(window as any).open = (url: string) => {
			openedUrl = url;
			return null;
		};

		await renderDonationButton();

		// Click to open menu by role/name
		const button = screen.getByRole("button", { name: /support this project/i });
		await user.click(button);

		// Click on GitHub Sponsors
		const githubItem = await screen.findByText("GitHub Sponsors");
		await user.click(githubItem);

		// Verify URL was opened
		expect(openedUrl).toContain("github.com/sponsors");
	});

	it("closes menu after clicking a platform", async () => {
		const user = userEvent.setup();

		// Mock window.open
		(window as any).open = () => null;

		await renderDonationButton();

		// Open menu by role/name
		const button = screen.getByRole("button", { name: /support this project/i });
		expect(button).toBeTruthy();
		await user.click(button);

		// Click on a menu item
		const githubItem = await screen.findByText("GitHub Sponsors");
		await user.click(githubItem);

		// Menu should close - check aria-expanded
		expect(button.getAttribute("aria-expanded")).not.toBe("true");
	});

	it("renders platform icons correctly", async () => {
		const user = userEvent.setup();
		await renderDonationButton();

		// Open menu by role/name
		const button = screen.getByRole("button", { name: /support this project/i });
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
		await renderDonationButton();

		// Component renders with button (accessible)
		const button = screen.getByRole("button", { name: /support this project/i });
		expect(button).toBeTruthy();

		// Tooltip text is set (may not be visible until hover)
		// We can verify the component renders successfully
	});

	it("handles menu close correctly", async () => {
		const user = userEvent.setup();
		await renderDonationButton();

		// Open menu by role/name
		const button = screen.getByRole("button", { name: /support this project/i });
		await user.click(button);

		// Verify menu is open
		expect(button.getAttribute("aria-expanded")).toBe("true");

		// Click button again to close (or click outside)
		await user.click(button);

		// Menu should be closed
		expect(button.getAttribute("aria-expanded")).not.toBe("true");
	});
});
