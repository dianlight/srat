import "../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

describe("CopyButtonBar Component", () => {
	beforeEach(() => {
		document.body.innerHTML = "";
	});

	it("renders compact mode with icon buttons", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { CopyButtonBar } = await import("../CopyButtonBar");

		render(
			React.createElement(CopyButtonBar as any, {
				compact: true,
				plainTextContent: "test content",
			})
		);

		const copyButtons = await screen.findAllByLabelText(/copy as/i);
		expect(copyButtons.length).toBe(2);
	});

	it("renders full mode with labeled buttons", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { CopyButtonBar } = await import("../CopyButtonBar");

		render(
			React.createElement(CopyButtonBar as any, {
				compact: false,
				plainTextContent: "test content",
			})
		);

		// Find buttons by their text content
		const copyButton = await screen.findByText("Copy");
		expect(copyButton).toBeTruthy();

		const markdownButton = await screen.findByText("Copy as Markdown");
		expect(markdownButton).toBeTruthy();
	});

	it("renders with plain text content", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { CopyButtonBar } = await import("../CopyButtonBar");

		const testContent = "username: admin\npassword: 🔒🔒🔒🔒🔒🔒🔒🔒";

		render(
			React.createElement(CopyButtonBar as any, {
				plainTextContent: testContent,
			})
		);

		// Verify the button exists and component renders with correct props
		const copyButton = await screen.findByText("Copy");
		expect(copyButton).toBeTruthy();
	});

	it("renders with markdown content", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { CopyButtonBar } = await import("../CopyButtonBar");

		const testMarkdown = "- **username**: `admin`";
		const testTitle = "Test Data";

		render(
			React.createElement(CopyButtonBar as any, {
				plainTextContent: "plain",
				markdownContent: testMarkdown,
				markdownTitle: testTitle,
			})
		);

		// Verify the button exists and component renders with correct props
		const markdownButton = await screen.findByText("Copy as Markdown");
		expect(markdownButton).toBeTruthy();
	});

	it("renders without markdown title", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const { CopyButtonBar } = await import("../CopyButtonBar");

		const testContent = "test content";

		render(
			React.createElement(CopyButtonBar as any, {
				plainTextContent: testContent,
			})
		);

		const markdownButton = await screen.findByText("Copy as Markdown");
		expect(markdownButton).toBeTruthy();
	});
});
