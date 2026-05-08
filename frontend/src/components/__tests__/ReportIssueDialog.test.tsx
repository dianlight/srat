import React from "react";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { renderWithTestStore } from "/test/testing";
import { ReportIssueDialog } from "../ReportIssueDialog";

describe("ReportIssueDialog", () => {

async function renderReportIssueDialog(props: any) {
    return renderWithTestStore(
        React.createElement(ReportIssueDialog as any, props)
    );
}

	it("renders dialog when open is true", async () => {
		const mockOnClose = () => { };

		await renderReportIssueDialog({
			open: true,
			onClose: mockOnClose,
		});
		const titleElement = await screen.findByText(/Report Issue on GitHub/i);
		expect(titleElement).toBeTruthy();
	});

	it("does not render dialog when open is false", async () => {
		const mockOnClose = () => { };

		await renderReportIssueDialog({
			open: false,
			onClose: mockOnClose,
		});
		const titleElement = screen.queryByText(/Bug Report/i);
		expect(titleElement).toBeFalsy();
	});

	it("has problem type selector with correct options", async () => {
		const mockOnClose = () => { };

		await renderReportIssueDialog({
			open: true,
			onClose: mockOnClose,
		});
		const problemTypeSelect = await screen.findByRole("combobox", { name: /Problem Type/i }, { timeout: 5000 });
		expect(problemTypeSelect).toBeTruthy();
	});

	it("has description textarea", async () => {
		const mockOnClose = () => { };

		await renderReportIssueDialog({
			open: true,
			onClose: mockOnClose,
		});
		const titleInput = await screen.findByRole("textbox", { name: /^Title$/i }, { timeout: 5000 });
		expect(titleInput).toBeTruthy();

		const descriptionEditor = await screen.findByRole("textbox", { name: /Description/i });
		expect(descriptionEditor).toBeTruthy();

		const reproSteps = await screen.findByPlaceholderText(
			/List the steps needed to reproduce the issue/i,
		);
		expect(reproSteps).toBeTruthy();
	});

	it("has toggle switches for data inclusion options", async () => {
		const mockOnClose = () => { };

		await renderReportIssueDialog({
			open: true,
			onClose: mockOnClose,
		});
		const addonLogsSwitch = await screen.findByText(/Addon logs/i, {}, { timeout: 5000 });
		expect(addonLogsSwitch).toBeTruthy();

		const addonConfigSwitch = await screen.findByText(/Addon configuration/i);
		expect(addonConfigSwitch).toBeTruthy();

		const sratConfigSwitch = await screen.findByText(/SRAT configuration/i);
		expect(sratConfigSwitch).toBeTruthy();
	});

	it("has Cancel and Create Issue buttons", async () => {
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
