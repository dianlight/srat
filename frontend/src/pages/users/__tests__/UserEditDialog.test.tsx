import "../../../../test/setup";
import { describe, it, expect, mock, afterEach } from "bun:test";

describe("UserEditDialog component", () => {
    afterEach(async () => {
        const { cleanup } = await import("@testing-library/react");
        cleanup();
    });

    it("submits new user credentials", async () => {
        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { UserEditDialog } = await import("../UserEditDialog");

        const onClose = mock(() => { });

        render(
            React.createElement(UserEditDialog as any, {
                open: true,
                onClose,
                objectToEdit: { doCreate: true },
            })
        );

        const usernameInput = await screen.findByLabelText(/User Name/i);
        const user = userEvent.setup();
        // Ensure any pre-filled value is cleared before typing
        if ((usernameInput as HTMLInputElement).value) {
            await user.clear(usernameInput as any);
        }
        await user.type(usernameInput as any, "newuser");

        const passwordInput = await screen.findByLabelText(/password/i, {
            selector: 'input[name="password"]',
        });
        if ((passwordInput as HTMLInputElement).value) {
            await user.clear(passwordInput as any);
        }
        await user.type(passwordInput as any, "Secret123!");

        const repeatInput = await screen.findByLabelText(/repeat password/i);
        if ((repeatInput as HTMLInputElement).value) {
            await user.clear(repeatInput as any);
        }
        await user.type(repeatInput as any, "Secret123!");

        const submitButton = await screen.findByRole("button", { name: /create/i });
        await user.click(submitButton as any);

        await waitFor(() => expect(onClose).toHaveBeenCalled());
        const submitted = (onClose.mock.calls as any[])[0]?.[0];
        expect(submitted?.username).toBe("newuser");
        expect(submitted?.password).toBe("Secret123!");
        expect(submitted?.doCreate).toBe(true);
    });

    it("displays read-only username for existing non-admin user and handles cancel", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { UserEditDialog } = await import("../UserEditDialog");

        const onClose = mock(() => { });

        const { container } = render(
            React.createElement(UserEditDialog as any, {
                open: true,
                onClose,
                objectToEdit: {
                    username: "existing",
                    password: "",
                    is_admin: false,
                },
            })
        );

        // Wait for the dialog to be visible
        await screen.findByText("existing");

        // Find the username input by its value
        const usernameInput = await screen.findByDisplayValue("existing");
        expect(usernameInput).toBeTruthy();
        expect((usernameInput as HTMLInputElement).readOnly).toBe(true);

        // Find and click the Cancel button
        const cancelButton = await screen.findByRole("button", { name: /cancel/i });
        const user = userEvent.setup();
        await user.click(cancelButton as any);
        expect(onClose).toHaveBeenCalledWith(undefined);
    });
});
