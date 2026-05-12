import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const shareEditFormMockState = vi.hoisted(() => ({
    submitCallback: undefined as ((data: any) => void) | undefined,
    testId: "mock-share-form",
}));

vi.mock("../components/ShareEditForm", async () => {
    const React = await import("react");
    return {
        ShareEditForm: (props: any) =>
            React.createElement(
                "button",
                {
                    type: "button",
                    "data-testid": shareEditFormMockState.testId,
                    onClick: () => {
                        shareEditFormMockState.submitCallback?.({ name: "Submitted" });
                        props.onSubmit({ name: "Submitted" });
                    },
                },
                "submit"
            ),
    };
});

describe("ShareEditDialog", () => {
    beforeEach(() => {
        vi.resetModules();
        vi.restoreAllMocks();
        shareEditFormMockState.submitCallback = undefined;
        shareEditFormMockState.testId = "mock-share-form";
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("renders dialog and handles cancel", async () => {
        shareEditFormMockState.submitCallback = undefined;
        shareEditFormMockState.testId = "mock-share-form";

        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        // @ts-expect-error - Query param ensures unmocked module instance
        const { ShareEditDialog } = await import("../ShareEditDialog?share-edit-dialog-test-cancel");

        const user = userEvent.setup();
        let closeCalls = 0;
        render(
            React.createElement(ShareEditDialog as any, {
                open: true,
                onClose: () => { closeCalls += 1; },
            })
        );

        expect(await screen.findByText(/Create New Share/i)).toBeTruthy();

        const cancelButton = screen.getByRole("button", { name: /cancel/i });
        await user.click(cancelButton);
        expect(closeCalls).toBe(1);
    });

    it("calls delete handler when delete button clicked", async () => {
        let deleteCalls = 0;
        shareEditFormMockState.submitCallback = undefined;
        shareEditFormMockState.testId = "mock-share-form";

        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        // @ts-expect-error - Query param ensures unmocked module instance
        const { ShareEditDialog } = await import("../ShareEditDialog?share-edit-dialog-test-delete");

        const user = userEvent.setup();
        render(
            React.createElement(ShareEditDialog as any, {
                open: true,
                onClose: () => { },
                objectToEdit: { org_name: "share1", name: "share1" },
                onDeleteSubmit: () => { deleteCalls += 1; },
            })
        );

        const deleteButton = await screen.findByRole("button", { name: /delete/i });
        await user.click(deleteButton);
        expect(deleteCalls).toBe(1);
    });

    it("submits form result via ShareEditForm", async () => {
        let received: any = null;
        shareEditFormMockState.submitCallback = (data) => {
            received = data;
        };
        shareEditFormMockState.testId = "submit-form-test";

        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        // @ts-expect-error - Query param ensures unmocked module instance
        const { ShareEditDialog } = await import("../ShareEditDialog?share-edit-dialog-test-submit");

        const user = userEvent.setup();
        let closePayload: any = null;
        render(
            React.createElement(ShareEditDialog as any, {
                open: true,
                onClose: (data?: any) => {
                    closePayload = data;
                },
            })
        );

        // Find the mock form button with unique test ID
        const submitButton = await screen.findByTestId("submit-form-test");
        await user.click(submitButton);

        expect(received).toBeTruthy();
        expect(closePayload?.name).toBe("Submitted");
    });
});
