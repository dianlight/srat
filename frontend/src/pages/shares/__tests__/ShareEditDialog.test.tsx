import "../../../../test/setup";
import { describe, it, expect, beforeEach, mock } from "bun:test";

describe("ShareEditDialog", () => {
    beforeEach(() => {
        mock.restore();
    });

    const setupMockForm = async (submitCallback?: (data: any) => void) => {
        const React = await import("react");
        mock.module("../components/ShareEditForm", () => {
            return {
                ShareEditForm: (props: any) =>
                    React.createElement(
                        "button",
                        {
                            type: "button",
                            "data-testid": "mock-share-form",
                            onClick: () => {
                                submitCallback?.({ name: "Submitted" });
                                props.onSubmit({ name: "Submitted" });
                            },
                        },
                        "submit"
                    ),
            };
        });
    };

    it("renders dialog and handles cancel", async () => {
        await setupMockForm();

        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        // @ts-expect-error - Query param ensures unmocked module instance
        const { ShareEditDialog } = await import("../ShareEditDialog?share-edit-dialog-test");

        let closeCalls = 0;
        render(
            React.createElement(ShareEditDialog as any, {
                open: true,
                onClose: () => { closeCalls += 1; },
            })
        );

        expect(await screen.findByText(/Create New Share/i)).toBeTruthy();

        const cancelButton = screen.getByRole("button", { name: /cancel/i });
        fireEvent.click(cancelButton);
        expect(closeCalls).toBe(1);
    });

    it("calls delete handler when delete button clicked", async () => {
        let deleteCalls = 0;
        await setupMockForm();

        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        // @ts-expect-error - Query param ensures unmocked module instance
        const { ShareEditDialog } = await import("../ShareEditDialog?share-edit-dialog-test");

        render(
            React.createElement(ShareEditDialog as any, {
                open: true,
                onClose: () => { },
                objectToEdit: { org_name: "share1", name: "share1" },
                onDeleteSubmit: () => { deleteCalls += 1; },
            })
        );

        const deleteButton = await screen.findByRole("button", { name: /delete/i });
        fireEvent.click(deleteButton);
        expect(deleteCalls).toBe(1);
    });

    it("submits form result via ShareEditForm", async () => {
        let received: any = null;
        await setupMockForm((data) => {
            received = data;
        });

        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        // @ts-expect-error - Query param ensures unmocked module instance
        const { ShareEditDialog } = await import("../ShareEditDialog?share-edit-dialog-test");

        let closePayload: any = null;
        render(
            React.createElement(ShareEditDialog as any, {
                open: true,
                onClose: (data?: any) => {
                    closePayload = data;
                },
            })
        );

        const submitButton = await screen.findByTestId("mock-share-form");
        fireEvent.click(submitButton);

        expect(received).toBeTruthy();
        expect(closePayload?.name).toBe("Submitted");
    });
});
