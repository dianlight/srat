import { beforeEach, describe, expect, it } from "bun:test";
import "../../../../../test/setup";

// localStorage shim for testing
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => {
            _store[k] = String(v);
        },
        removeItem: (k: string) => {
            delete _store[k];
        },
        clear: () => {
            for (const k of Object.keys(_store)) delete _store[k];
        },
    };
}

describe("UserEditForm component", () => {
    beforeEach(() => {
        document.body.innerHTML = "";
        localStorage.clear();
    });

    it("renders empty form for new user", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserEditForm } = await import("../UserEditForm");

        const mockSubmit = () => { };

        render(
            React.createElement(UserEditForm as any, {
                userData: { username: "", password: "", doCreate: true },
                onSubmit: mockSubmit,
            })
        );

        const usernameField = await screen.findByLabelText(/username/i);
        expect(usernameField).toBeTruthy();
    });

    it("renders form with user data for existing user", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserEditForm } = await import("../UserEditForm");

        const mockSubmit = () => { };

        render(
            React.createElement(UserEditForm as any, {
                userData: { username: "testuser", password: "", doCreate: false, is_admin: false },
                onSubmit: mockSubmit,
            })
        );

        const usernameField = await screen.findByLabelText(/username/i);
        expect(usernameField).toBeTruthy();
        expect((usernameField as HTMLInputElement).value).toBe("testuser");
    });

    it("shows username readonly for existing non-admin users", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserEditForm } = await import("../UserEditForm");

        const mockSubmit = () => { };

        render(
            React.createElement(UserEditForm as any, {
                userData: { username: "testuser", password: "", doCreate: false, is_admin: false },
                onSubmit: mockSubmit,
            })
        );

        const usernameField = await screen.findByLabelText(/username/i);
        expect((usernameField as HTMLInputElement).readOnly).toBe(true);
    });

    it("allows username edit for admin users", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserEditForm } = await import("../UserEditForm");

        const mockSubmit = () => { };

        render(
            React.createElement(UserEditForm as any, {
                userData: { username: "admin", password: "", doCreate: false, is_admin: true },
                onSubmit: mockSubmit,
            })
        );

        const usernameField = await screen.findByLabelText(/username/i);
        expect((usernameField as HTMLInputElement).readOnly).toBe(false);
    });

    it("renders password fields", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserEditForm } = await import("../UserEditForm");

        const mockSubmit = () => { };

        render(
            React.createElement(UserEditForm as any, {
                userData: { username: "", password: "", doCreate: true },
                onSubmit: mockSubmit,
            })
        );

        // Find all text inputs (username, password, repeat password)
        const inputs = await screen.findAllByRole("textbox");
        // The username field should be present
        expect(inputs.length).toBeGreaterThanOrEqual(1);

        // Check the form contains password inputs (type=password doesn't have textbox role)
        const passwordInputs = document.body.getElementsByTagName('input');
        let passwordCount = 0;
        for (let i = 0; i < passwordInputs.length; i++) {
            if (passwordInputs[i]?.type === 'password') {
                passwordCount++;
            }
        }
        expect(passwordCount).toBe(2);
    });

    it("renders cancel button when onCancel provided", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserEditForm } = await import("../UserEditForm");

        const mockSubmit = () => { };
        const mockCancel = () => { };

        render(
            React.createElement(UserEditForm as any, {
                userData: { username: "", password: "", doCreate: true },
                onSubmit: mockSubmit,
                onCancel: mockCancel,
            })
        );

        const cancelButton = await screen.findByText(/cancel/i);
        expect(cancelButton).toBeTruthy();
    });

    it("shows Create User button for new users", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserEditForm } = await import("../UserEditForm");

        const mockSubmit = () => { };

        render(
            React.createElement(UserEditForm as any, {
                userData: { username: "", password: "", doCreate: true },
                onSubmit: mockSubmit,
            })
        );

        const createButton = await screen.findByText(/create user/i);
        expect(createButton).toBeTruthy();
    });

    it("shows Save Changes button for existing users", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserEditForm } = await import("../UserEditForm");

        const mockSubmit = () => { };

        render(
            React.createElement(UserEditForm as any, {
                userData: { username: "testuser", password: "", doCreate: false, is_admin: false },
                onSubmit: mockSubmit,
            })
        );

        const saveButton = await screen.findByText(/save changes/i);
        expect(saveButton).toBeTruthy();
    });

    it("displays admin account info text for admin users", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserEditForm } = await import("../UserEditForm");

        const mockSubmit = () => { };

        render(
            React.createElement(UserEditForm as any, {
                userData: { username: "admin", password: "", doCreate: false, is_admin: true },
                onSubmit: mockSubmit,
            })
        );

        const adminInfo = await screen.findByText(/administrator account/i);
        expect(adminInfo).toBeTruthy();
    });

    it("displays new user info text for new users", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserEditForm } = await import("../UserEditForm");

        const mockSubmit = () => { };

        render(
            React.createElement(UserEditForm as any, {
                userData: { username: "", password: "", doCreate: true },
                onSubmit: mockSubmit,
            })
        );

        const newUserInfo = await screen.findByText(/new user/i);
        expect(newUserInfo).toBeTruthy();
    });

    it("displays regular user info text for existing non-admin users", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserEditForm } = await import("../UserEditForm");

        const mockSubmit = () => { };

        render(
            React.createElement(UserEditForm as any, {
                userData: { username: "testuser", password: "", doCreate: false, is_admin: false },
                onSubmit: mockSubmit,
            })
        );

        const userInfo = await screen.findByText(/regular user/i);
        expect(userInfo).toBeTruthy();
    });

    it("disables form when disabled prop is true", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserEditForm } = await import("../UserEditForm");

        const mockSubmit = () => { };

        render(
            React.createElement(UserEditForm as any, {
                userData: { username: "", password: "", doCreate: true },
                onSubmit: mockSubmit,
                disabled: true,
            })
        );

        const usernameField = await screen.findByLabelText(/username/i);
        expect((usernameField as HTMLInputElement).disabled).toBe(true);
    });

    it("does not require password fields in edit mode", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserEditForm } = await import("../UserEditForm");

        render(
            React.createElement(UserEditForm as any, {
                userData: { username: "testuser", password: "", doCreate: false, is_admin: false },
                onSubmit: () => { },
            })
        );

        const passwordField = await screen.findByLabelText(/^password$/i);
        const repeatPasswordField = await screen.findByLabelText(/repeat password/i);

        expect((passwordField as HTMLInputElement).required).toBe(false);
        expect((repeatPasswordField as HTMLInputElement).required).toBe(false);
    });

    it("requires matching repeat password in edit mode when password is set", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { UserEditForm } = await import("../UserEditForm");

        let submitCount = 0;

        render(
            React.createElement(UserEditForm as any, {
                userData: { username: "testuser", password: "", doCreate: false, is_admin: false },
                onSubmit: () => {
                    submitCount += 1;
                },
            })
        );

        const user = userEvent.setup();
        const passwordField = await screen.findByLabelText(/^password$/i);
        const repeatPasswordField = await screen.findByLabelText(/repeat password/i);

        await user.type(passwordField, "secret1");
        await user.type(repeatPasswordField, "secret2");

        const saveButton = await screen.findByRole("button", { name: /save changes/i });
        await user.click(saveButton);

        expect(submitCount).toBe(0);
    });

    it("submits in edit mode when password and repeat password match", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { UserEditForm } = await import("../UserEditForm");

        let submitCount = 0;

        render(
            React.createElement(UserEditForm as any, {
                userData: { username: "testuser", password: "", doCreate: false, is_admin: false },
                onSubmit: () => {
                    submitCount += 1;
                },
            })
        );

        const user = userEvent.setup();
        const passwordField = await screen.findByLabelText(/^password$/i);
        const repeatPasswordField = await screen.findByLabelText(/repeat password/i);

        await user.type(passwordField, "secret1");
        await user.type(repeatPasswordField, "secret1");

        const saveButton = await screen.findByRole("button", { name: /save changes/i });
        await user.click(saveButton);

        expect(submitCount).toBe(1);
    });
});
