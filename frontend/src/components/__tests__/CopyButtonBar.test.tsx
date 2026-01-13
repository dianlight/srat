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

		const copyButton = await screen.findByRole("button", { name: /^copy$/i });
		expect(copyButton).toBeTruthy();

		const markdownButton = await screen.findByRole("button", { name: /copy as markdown/i });
		expect(markdownButton).toBeTruthy();
	});

	it("copies plain text when button is clicked", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const userEvent = (await import("@testing-library/user-event")).default;
		const { CopyButtonBar } = await import("../CopyButtonBar");

		// Mock clipboard API
		const clipboardData: string[] = [];
		Object.assign(navigator, {
			clipboard: {
				writeText: async (text: string) => {
					clipboardData.push(text);
				},
			},
		});

		const testContent = "username: admin\npassword: 🔒🔒🔒🔒🔒🔒🔒🔒";

		render(
			React.createElement(CopyButtonBar as any, {
				plainTextContent: testContent,
			})
		);

		const user = userEvent.setup();
		const copyButton = await screen.findByRole("button", { name: /^copy$/i });
		await user.click(copyButton);

		expect(clipboardData.length).toBe(1);
		expect(clipboardData[0]).toBe(testContent);
	});

	it("copies markdown when markdown button is clicked", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const userEvent = (await import("@testing-library/user-event")).default;
		const { CopyButtonBar } = await import("../CopyButtonBar");

		// Mock clipboard API
		const clipboardData: string[] = [];
		Object.assign(navigator, {
			clipboard: {
				writeText: async (text: string) => {
					clipboardData.push(text);
				},
			},
		});

		const testMarkdown = "- **username**: `admin`";
		const testTitle = "Test Data";

		render(
			React.createElement(CopyButtonBar as any, {
				plainTextContent: "plain",
				markdownContent: testMarkdown,
				markdownTitle: testTitle,
			})
		);

		const user = userEvent.setup();
		const markdownButton = await screen.findByRole("button", { name: /copy as markdown/i });
		await user.click(markdownButton);

		expect(clipboardData.length).toBe(1);
		expect(clipboardData[0]).toContain(`## ${testTitle}`);
		expect(clipboardData[0]).toContain(testMarkdown);
	});

	it("uses plain text as markdown if no markdown content provided", async () => {
		const React = await import("react");
		const { render, screen } = await import("@testing-library/react");
		const userEvent = (await import("@testing-library/user-event")).default;
		const { CopyButtonBar } = await import("../CopyButtonBar");

		// Mock clipboard API
		const clipboardData: string[] = [];
		Object.assign(navigator, {
			clipboard: {
				writeText: async (text: string) => {
					clipboardData.push(text);
				},
			},
		});

		const testContent = "test content";

		render(
			React.createElement(CopyButtonBar as any, {
				plainTextContent: testContent,
			})
		);

		const user = userEvent.setup();
		const markdownButton = await screen.findByRole("button", { name: /copy as markdown/i });
		await user.click(markdownButton);

		expect(clipboardData.length).toBe(1);
		expect(clipboardData[0]).toContain(testContent);
	});
});
