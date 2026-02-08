import "../../../test/setup";
import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { createTestStore } from "../../../test/setup";

describe("ReportIssueDialog", () => {
	beforeEach(() => {
		// Clear any DOM state
		if (typeof document !== "undefined") {
			document.body.innerHTML = "";
		}
	});

	afterEach(() => {
		// Clean up DOM state after each test
		if (typeof document !== "undefined") {
			document.body.innerHTML = "";
		}
	});

	it("renders dialog when open is true", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { ReportIssueDialog } = await import("../ReportIssueDialog");
		const store = await createTestStore();

		const mockOnClose = () => { };

		render(
			React.createElement(
				Provider as any,
				{ store },
				React.createElement(ReportIssueDialog as any, {
					open: true,
					onClose: mockOnClose,
				}),
			),
		);

		const titleElement = await screen.findByText(/Report Issue on GitHub/i);
		expect(titleElement).toBeTruthy();
	});

	it("does not render dialog when open is false", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { ReportIssueDialog } = await import("../ReportIssueDialog");
		const store = await createTestStore();

		const mockOnClose = () => { };

		render(
			React.createElement(
				Provider as any,
				{ store },
				React.createElement(ReportIssueDialog as any, {
					open: false,
					onClose: mockOnClose,
				}),
			),
		);

		const titleElement = screen.queryByText(/Bug Report/i);
		expect(titleElement).toBeFalsy();
	});

	it("has problem type selector with correct options", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { ReportIssueDialog } = await import("../ReportIssueDialog");
		const store = await createTestStore();

		const mockOnClose = () => { };

		render(
			React.createElement(
				Provider as any,
				{ store },
				React.createElement(ReportIssueDialog as any, {
					open: true,
					onClose: mockOnClose,
				}),
			),
		);

		// Check that Problem Type label exists using a more specific query
		const problemTypeSelect = await screen.findByRole("combobox", { name: /Problem Type/i });
		expect(problemTypeSelect).toBeTruthy();
	});

	it("has description textarea", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { ReportIssueDialog } = await import("../ReportIssueDialog");
		const store = await createTestStore();

		const mockOnClose = () => { };

		render(
			React.createElement(
				Provider as any,
				{ store },
				React.createElement(ReportIssueDialog as any, {
					open: true,
					onClose: mockOnClose,
				}),
			),
		);

		const titleInput = await screen.findByRole("textbox", { name: /^Title$/i });
		expect(titleInput).toBeTruthy();

		const descriptionEditor = await screen.findByRole("textbox", { name: /Description/i });
		expect(descriptionEditor).toBeTruthy();

		const reproSteps = await screen.findByPlaceholderText(
			/List the steps needed to reproduce the issue/i,
		);
		expect(reproSteps).toBeTruthy();
	});

	it("has toggle switches for data inclusion options", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { ReportIssueDialog } = await import("../ReportIssueDialog");
		const store = await createTestStore();

		const mockOnClose = () => { };

		render(
			React.createElement(
				Provider as any,
				{ store },
				React.createElement(ReportIssueDialog as any, {
					open: true,
					onClose: mockOnClose,
				}),
			),
		);

		// Check for the three toggle switches
		const addonLogsSwitch = await screen.findByText(/Addon logs/i);
		expect(addonLogsSwitch).toBeTruthy();

		const addonConfigSwitch = await screen.findByText(/Addon configuration/i);
		expect(addonConfigSwitch).toBeTruthy();

		const sratConfigSwitch = await screen.findByText(/SRAT configuration/i);
		expect(sratConfigSwitch).toBeTruthy();

		//const databaseDumpSwitch = await screen.findByText(/Database dump/i);
		//expect(databaseDumpSwitch).toBeTruthy();
	});

	it("has Cancel and Create Issue buttons", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { Provider } = await import("react-redux");
		const { ReportIssueDialog } = await import("../ReportIssueDialog");
		const store = await createTestStore();

		const mockOnClose = () => { };

		render(
			React.createElement(
				Provider as any,
				{ store },
				React.createElement(ReportIssueDialog as any, {
					open: true,
					onClose: mockOnClose,
				}),
			),
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
		const { Provider } = await import("react-redux");
		const { ReportIssueDialog } = await import("../ReportIssueDialog");
		const store = await createTestStore();

		const mockOnClose = () => { };

		render(
			React.createElement(
				Provider as any,
				{ store },
				React.createElement(ReportIssueDialog as any, {
					open: true,
					onClose: mockOnClose,
				}),
			),
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
		const { Provider } = await import("react-redux");
		const { ReportIssueDialog } = await import("../ReportIssueDialog");
		const store = await createTestStore();

		let closeCalled = false;
		const mockOnClose = () => {
			closeCalled = true;
		};
		const user = userEvent.setup();

		render(
			React.createElement(
				Provider as any,
				{ store },
				React.createElement(ReportIssueDialog as any, {
					open: true,
					onClose: mockOnClose,
				}),
			),
		);

		const cancelButton = await screen.findByRole("button", { name: /Cancel/i });
		await user.click(cancelButton);

		expect(closeCalled).toBe(true);
	});
});
