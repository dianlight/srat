import "../../../../test/setup";
import { describe, it, expect, mock } from "bun:test";

describe("UserEditDialog component", () => {

    it("submits new user credentials", async () => {
        const React = await import("react");
        const { render, screen, fireEvent, waitFor } = await import("@testing-library/react");
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
        fireEvent.input(usernameInput, { target: { value: "newuser" } });

        const passwordInput = await screen.findByLabelText(/password/i, {
            selector: 'input[name="password"]',
        });
        fireEvent.input(passwordInput, { target: { value: "Secret123!" } });

        const repeatInput = await screen.findByLabelText(/repeat password/i);
        fireEvent.input(repeatInput, { target: { value: "Secret123!" } });

        const submitButton = await screen.findByRole("button", { name: /create/i });
        fireEvent.click(submitButton);

        await waitFor(() => expect(onClose).toHaveBeenCalled());
        const submitted = (onClose.mock.calls as any[])[0]?.[0];
        expect(submitted?.username).toBe("newuser");
        expect(submitted?.password).toBe("Secret123!");
        expect(submitted?.doCreate).toBe(true);
    });

    it("displays read-only username for existing non-admin user and handles cancel", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        const { UserEditDialog } = await import("../UserEditDialog");

        const onClose = mock(() => { });

        render(
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

        const usernameInput = await screen.findByLabelText(/User Name/i);
        expect((usernameInput as HTMLInputElement).readOnly).toBe(true);
        expect(await screen.findByText("existing")).toBeTruthy();

        const cancelButton = await screen.findByRole("button", { name: /cancel/i });
        fireEvent.click(cancelButton);

        expect(onClose).toHaveBeenCalledWith(undefined);
    });
});
