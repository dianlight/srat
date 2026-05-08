import { describe, expect, it } from "vitest";

describe("ReportIssueDialog", () => {

async function renderReportIssueDialog(props: any) {
    const React = await import("react");
    const { renderWithTestStore } = await import("/test/testing");
    const { ReportIssueDialog } = await import("../ReportIssueDialog");
    return renderWithTestStore(
        React.createElement(ReportIssueDialog as any, props)
    );
}


	it("renders dialog when open is true", async () => {
                const { screen } = await import("@testing-library/react");
                const mockOnClose = () => { };

                await renderReportIssueDialog({
                        open: true,
                        onClose: mockOnClose,
                });
		const titleElement = await screen.findByText(/Report Issue on GitHub/i);
		expect(titleElement).toBeTruthy();
	});

	it("does not render dialog when open is false", async () => {
                const { screen } = await import("@testing-library/react");
                const mockOnClose = () => { };

                await renderReportIssueDialog({
                        open: false,
                        onClose: mockOnClose,
                });
		const titleElement = screen.queryByText(/Bug Report/i);
		expect(titleElement).toBeFalsy();
	});

	it("has problem type selector with correct options", async () => {
                const { screen } = await import("@testing-library/react");
                const mockOnClose = () => { };

                await renderReportIssueDialog({
                        open: true,
                        onClose: mockOnClose,
                });
		// Check that Problem Type label exists using a more specific query
		const problemTypeSelect = await screen.findByRole("combobox", { name: /Problem Type/i });
		expect(problemTypeSelect).toBeTruthy();
	});

	it("has description textarea", async () => {
                const { screen } = await import("@testing-library/react");
                const mockOnClose = () => { };

                await renderReportIssueDialog({
                        open: true,
                        onClose: mockOnClose,
                });
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
                const { screen } = await import("@testing-library/react");
                const mockOnClose = () => { };

                await renderReportIssueDialog({
                        open: true,
                        onClose: mockOnClose,
                });
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
                const { screen } = await import("@testing-library/react");
                const mockOnClose = () => { };

                await renderReportIssueDialog({
                        open: true,
                        onClose: mockOnClose,
                });
		const cancelButton = await screen.findByRole("button", { name: /Cancel/i });
		expect(cancelButton).toBeTruthy();

		const createButton = await screen.findByRole("button", {
			name: /Create Issue/i,
		});
		expect(createButton).toBeTruthy();
	});

	it("disables Create Issue button when description is empty", async () => {
                const { screen } = await import("@testing-library/react");
                const mockOnClose = () => { };

                await renderReportIssueDialog({
                        open: true,
                        onClose: mockOnClose,
                });
		const createButton = await screen.findByRole("button", {
			name: /Create Issue/i,
		});
		expect(createButton).toBeTruthy();
		expect(createButton.hasAttribute("disabled")).toBe(true);
	});

	it("calls onClose when Cancel button is clicked", async () => {
                const { screen } = await import("@testing-library/react");
                const userEvent = (await import("@testing-library/user-event")).default;
                let closeCalled = false;
                const mockOnClose = () => { closeCalled = true; };
                const user = userEvent.setup();

                await renderReportIssueDialog({
                        open: true,
                        onClose: mockOnClose,
                });
		const cancelButton = await screen.findByRole("button", { name: /Cancel/i });
		await user.click(cancelButton);

		expect(closeCalled).toBe(true);
	});
});
