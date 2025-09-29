import "../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

describe("PreviewDialog Component", () => {
    beforeEach(() => {
        document.body.innerHTML = "";
    });

    it("renders nested object data and censors sensitive fields", async () => {
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
        expect(passwordItem.textContent).toContain("********");
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
        const { render, screen, fireEvent } = await import("@testing-library/react");
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
        fireEvent.click(closeButton);
        expect(closeCalls).toBe(1);
    });

    it("shows placeholder when no data is provided", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ObjectTree } = await import("../PreviewDialog");

        render(React.createElement(ObjectTree as any, { object: null }));

        expect(await screen.findByText("No data to display")).toBeTruthy();
    });
});