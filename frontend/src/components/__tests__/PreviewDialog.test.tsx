import "../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

describe("PreviewDialog Component", () => {
    beforeEach(() => {
        document.body.innerHTML = "";
    });

    it("renders nested object data and censors sensitive fields with emoticons", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { PreviewDialog } = await import("../PreviewDialog");

        const testObject = {
            password: "supersecret",
            flag: true,
            list: [1, "two"],
            nested: {
                message: "hello",
            },
        };

        render(
            React.createElement(PreviewDialog as any, {
                open: true,
                onClose: () => {
                    /* noop for render test */
                },
                title: "Sensitive Preview",
                objectToDisplay: testObject,
            })
        );

        expect(await screen.findByText("Sensitive Preview")).toBeTruthy();

        const passwordItem = screen.getByRole("treeitem", { name: /password/i });
        // Check for lock emoji censoring instead of asterisks
        expect(passwordItem.textContent).toContain("🔒");
        expect(passwordItem.textContent?.toLowerCase()).toContain("censored");

        const flagItem = screen.getByRole("treeitem", { name: /flag/i });
        expect(flagItem.textContent).toContain("Yes");

        const listItem = screen.getByRole("treeitem", { name: /list/i });
        expect(listItem.textContent).toContain("array[2]");

        const nestedItem = screen.getByRole("treeitem", { name: /nested/i });
        expect(nestedItem.textContent?.toLowerCase()).toContain("object");
    });

    it("calls onClose when Close button is pressed", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { PreviewDialog } = await import("../PreviewDialog");

        let closeCalls = 0;
        const onClose = () => {
            closeCalls += 1;
        };

        render(
            React.createElement(PreviewDialog as any, {
                open: true,
                onClose,
                title: "Inspect Data",
                objectToDisplay: { message: "hello" },
            })
        );

        const closeButton = await screen.findByRole("button", { name: /close/i });
        const user = userEvent.setup();
        await user.click(closeButton as any);
        expect(closeCalls).toBe(1);
    });

    it("shows placeholder when no data is provided", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ObjectTree } = await import("../PreviewDialog");

        render(React.createElement(ObjectTree as any, { object: null }));

        expect(await screen.findByText("No data to display")).toBeTruthy();
    });

    it("renders copy buttons in dialog title and actions", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { PreviewDialog } = await import("../PreviewDialog");

        render(
            React.createElement(PreviewDialog as any, {
                open: true,
                onClose: () => {},
                title: "Test Data",
                objectToDisplay: { test: "value" },
            })
        );

        // Check for title bar copy buttons (icon buttons)
        const copyButtons = await screen.findAllByLabelText(/copy as/i);
        expect(copyButtons.length).toBeGreaterThan(0);

        // Check for action bar buttons
        const copyTextButton = await screen.findByRole("button", { name: /^copy$/i });
        expect(copyTextButton).toBeTruthy();

        const copyMarkdownButton = await screen.findByRole("button", { name: /copy as markdown/i });
        expect(copyMarkdownButton).toBeTruthy();
    });

    it("copies data as plain text when copy button is clicked", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { PreviewDialog } = await import("../PreviewDialog");

        // Mock clipboard API
        const clipboardData: string[] = [];
        Object.assign(navigator, {
            clipboard: {
                writeText: async (text: string) => {
                    clipboardData.push(text);
                },
            },
        });

        const testObject = {
            name: "test",
            value: 42,
        };

        render(
            React.createElement(PreviewDialog as any, {
                open: true,
                onClose: () => {},
                title: "Test Data",
                objectToDisplay: testObject,
            })
        );

        const user = userEvent.setup();
        const copyButton = await screen.findByRole("button", { name: /^copy$/i });
        await user.click(copyButton);

        expect(clipboardData.length).toBe(1);
        expect(clipboardData[0]).toContain("name");
        expect(clipboardData[0]).toContain("test");
        expect(clipboardData[0]).toContain("value");
        expect(clipboardData[0]).toContain("42");
    });

    it("copies data as markdown when markdown button is clicked", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { PreviewDialog } = await import("../PreviewDialog");

        // Mock clipboard API
        const clipboardData: string[] = [];
        Object.assign(navigator, {
            clipboard: {
                writeText: async (text: string) => {
                    clipboardData.push(text);
                },
            },
        });

        const testObject = {
            name: "test",
            value: 42,
        };

        render(
            React.createElement(PreviewDialog as any, {
                open: true,
                onClose: () => {},
                title: "Test Data",
                objectToDisplay: testObject,
            })
        );

        const user = userEvent.setup();
        const markdownButton = await screen.findByRole("button", { name: /copy as markdown/i });
        await user.click(markdownButton);

        expect(clipboardData.length).toBe(1);
        // Check for markdown formatting
        expect(clipboardData[0]).toContain("##");
        expect(clipboardData[0]).toContain("Test Data");
        expect(clipboardData[0]).toContain("**name**");
        expect(clipboardData[0]).toContain("**value**");
        expect(clipboardData[0]).toContain("`test`");
        expect(clipboardData[0]).toContain("`42`");
    });
});