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

        let closeCalled = false;

        render(
            React.createElement(PreviewDialog as any, {
                open: true,
                onClose: () => {
                    closeCalled = true;
                },
                title: "Test",
                objectToDisplay: { key: "value" },
            })
        );

        const user = userEvent.setup();
        const closeButton = await screen.findByRole("button", { name: /close/i });
        await user.click(closeButton);

        expect(closeCalled).toBe(true);
    });

    it("shows placeholder when no data is provided", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { PreviewDialog } = await import("../PreviewDialog");

        render(
            React.createElement(PreviewDialog as any, {
                open: true,
                onClose: () => {},
                title: "Empty Data",
                objectToDisplay: null,
            })
        );

        expect(await screen.findByText(/no data to display/i)).toBeTruthy();
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

        // Check for action bar buttons - find by text content
        const copyTextButton = await screen.findByText("Copy");
        expect(copyTextButton).toBeTruthy();

        const copyMarkdownButton = await screen.findByText("Copy as Markdown");
        expect(copyMarkdownButton).toBeTruthy();
    });

    it("renders with correct data structure", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { PreviewDialog } = await import("../PreviewDialog");

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

        // Verify the button exists and component renders
        const copyButton = await screen.findByText("Copy");
        expect(copyButton).toBeTruthy();
    });

    it("renders markdown button with data", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { PreviewDialog } = await import("../PreviewDialog");

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

        // Verify the markdown button exists
        const markdownButton = await screen.findByText("Copy as Markdown");
        expect(markdownButton).toBeTruthy();
    });
});
