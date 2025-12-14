import "../../../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

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
        const container = document.body;
        const passwordInputs = container.querySelectorAll('input[type="password"]');
        expect(passwordInputs.length).toBe(2);
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
});
