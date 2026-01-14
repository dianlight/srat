import { describe, it, expect, beforeEach } from "bun:test";

describe("ReportIssueDialog", () => {
	beforeEach(() => {
		// Clear any DOM state
		if (typeof document !== "undefined") {
			document.body.innerHTML = "";
		}
	});

	it("renders dialog when open is true", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { ReportIssueDialog } = await import("../ReportIssueDialog");

		const mockOnClose = () => {};

		render(
			React.createElement(ReportIssueDialog, {
				open: true,
				onClose: mockOnClose,
			}),
		);

		const titleElement = await screen.findByText(/Report Issue on GitHub/i);
		expect(titleElement).toBeTruthy();
	});

	it("does not render dialog when open is false", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { ReportIssueDialog } = await import("../ReportIssueDialog");

		const mockOnClose = () => {};

		render(
			React.createElement(ReportIssueDialog, {
				open: false,
				onClose: mockOnClose,
			}),
		);

		const titleElement = screen.queryByText(/Report Issue on GitHub/i);
		expect(titleElement).toBeFalsy();
	});

	it("has problem type selector with correct options", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const userEvent = (await import("@testing-library/user-event")).default;
		const { ReportIssueDialog } = await import("../ReportIssueDialog");

		const mockOnClose = () => {};
		const user = userEvent.setup();

		render(
			React.createElement(ReportIssueDialog, {
				open: true,
				onClose: mockOnClose,
			}),
		);

		// Check that Problem Type label exists
		const problemTypeLabel = await screen.findByText(/Problem Type/i);
		expect(problemTypeLabel).toBeTruthy();
	});

	it("has description textarea", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { ReportIssueDialog } = await import("../ReportIssueDialog");

		const mockOnClose = () => {};

		render(
			React.createElement(ReportIssueDialog, {
				open: true,
				onClose: mockOnClose,
			}),
		);

		const descriptionLabel = await screen.findByText(/Description/i);
		expect(descriptionLabel).toBeTruthy();

		const textarea = await screen.findByPlaceholderText(
			/Describe the issue in detail/i,
		);
		expect(textarea).toBeTruthy();
	});

	it("has toggle switches for data inclusion options", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { ReportIssueDialog } = await import("../ReportIssueDialog");

		const mockOnClose = () => {};

		render(
			React.createElement(ReportIssueDialog, {
				open: true,
				onClose: mockOnClose,
			}),
		);

		// Check for the three toggle switches
		const contextDataSwitch = await screen.findByText(
			/Contextual data/i,
		);
		expect(contextDataSwitch).toBeTruthy();

		const addonLogsSwitch = await screen.findByText(/Addon config and logs/i);
		expect(addonLogsSwitch).toBeTruthy();

		const sratConfigSwitch = await screen.findByText(/SRAT configuration/i);
		expect(sratConfigSwitch).toBeTruthy();
	});

	it("has Cancel and Create Issue buttons", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { ReportIssueDialog } = await import("../ReportIssueDialog");

		const mockOnClose = () => {};

		render(
			React.createElement(ReportIssueDialog, {
				open: true,
				onClose: mockOnClose,
			}),
		);

		const cancelButton = await screen.findByRole("button", { name: /Cancel/i });
		expect(cancelButton).toBeTruthy();

		const createButton = await screen.findByRole("button", {
			name: /Create Issue/i,
		});
		expect(createButton).toBeTruthy();
	});

	it("disables Create Issue button when description is empty", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { ReportIssueDialog } = await import("../ReportIssueDialog");

		const mockOnClose = () => {};

		render(
			React.createElement(ReportIssueDialog, {
				open: true,
				onClose: mockOnClose,
			}),
		);

		const createButton = await screen.findByRole("button", {
			name: /Create Issue/i,
		});
		expect(createButton).toBeTruthy();
		expect(createButton.hasAttribute("disabled")).toBe(true);
	});

	it("calls onClose when Cancel button is clicked", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const userEvent = (await import("@testing-library/user-event")).default;
		const { ReportIssueDialog } = await import("../ReportIssueDialog");

		let closeCalled = false;
		const mockOnClose = () => {
			closeCalled = true;
		};
		const user = userEvent.setup();

		render(
			React.createElement(ReportIssueDialog, {
				open: true,
				onClose: mockOnClose,
			}),
		);

		const cancelButton = await screen.findByRole("button", { name: /Cancel/i });
		await user.click(cancelButton);

		expect(closeCalled).toBe(true);
	});
}
