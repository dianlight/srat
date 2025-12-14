import "../../../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

describe("UserDetailsPanel component", () => {
    beforeEach(() => {
        document.body.innerHTML = "";
    });

    it("renders select message when no user provided", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserDetailsPanel } = await import("../UserDetailsPanel");

        render(
            React.createElement(UserDetailsPanel as any, {
                user: undefined,
                userKey: undefined,
            })
        );

        const message = await screen.findByText("Select a user to view details");
        expect(message).toBeTruthy();
    });

    it("renders user details when user is provided", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserDetailsPanel } = await import("../UserDetailsPanel");

        const mockUser = {
            username: "testuser",
            is_admin: false,
            rw_shares: ["share1"],
            ro_shares: ["share2"],
        };

        render(
            React.createElement(UserDetailsPanel as any, {
                user: mockUser,
                userKey: "testuser",
            })
        );

        const username = await screen.findByText("testuser");
        expect(username).toBeTruthy();
    });

    it("displays admin badge for admin users", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserDetailsPanel } = await import("../UserDetailsPanel");

        const mockUser = {
            username: "admin",
            is_admin: true,
            rw_shares: [],
            ro_shares: [],
        };

        render(
            React.createElement(UserDetailsPanel as any, {
                user: mockUser,
                userKey: "admin",
            })
        );

        const adminBadge = await screen.findByText("Administrator");
        expect(adminBadge).toBeTruthy();
    });

    it("displays read/write shares section", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserDetailsPanel } = await import("../UserDetailsPanel");

        const mockUser = {
            username: "testuser",
            is_admin: false,
            rw_shares: ["share1", "share2"],
            ro_shares: [],
        };

        render(
            React.createElement(UserDetailsPanel as any, {
                user: mockUser,
                userKey: "testuser",
            })
        );

        const rwSection = await screen.findByText("Read/Write Shares");
        expect(rwSection).toBeTruthy();

        // Check shares are displayed
        const share1 = await screen.findByText("share1");
        const share2 = await screen.findByText("share2");
        expect(share1).toBeTruthy();
        expect(share2).toBeTruthy();
    });

    it("displays read-only shares section", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserDetailsPanel } = await import("../UserDetailsPanel");

        const mockUser = {
            username: "testuser",
            is_admin: false,
            rw_shares: [],
            ro_shares: ["share3"],
        };

        render(
            React.createElement(UserDetailsPanel as any, {
                user: mockUser,
                userKey: "testuser",
            })
        );

        const roSection = await screen.findByText("Read-Only Shares");
        expect(roSection).toBeTruthy();

        const share3 = await screen.findByText("share3");
        expect(share3).toBeTruthy();
    });

    it("shows no shares message when user has no shares", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserDetailsPanel } = await import("../UserDetailsPanel");

        const mockUser = {
            username: "testuser",
            is_admin: false,
            rw_shares: [],
            ro_shares: [],
        };

        render(
            React.createElement(UserDetailsPanel as any, {
                user: mockUser,
                userKey: "testuser",
            })
        );

        const noRwShares = await screen.findByText("No read/write shares assigned");
        expect(noRwShares).toBeTruthy();
    });

    it("renders edit button when not read-only", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserDetailsPanel } = await import("../UserDetailsPanel");

        const mockUser = {
            username: "testuser",
            is_admin: false,
            rw_shares: [],
            ro_shares: [],
        };

        render(
            React.createElement(UserDetailsPanel as any, {
                user: mockUser,
                userKey: "testuser",
                onEditClick: () => { },
                readOnly: false,
            })
        );

        const editButton = await screen.findByRole("button", { name: /edit user/i });
        expect(editButton).toBeTruthy();
    });

    it("hides edit button when read-only", async () => {
        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const { UserDetailsPanel } = await import("../UserDetailsPanel");

        const mockUser = {
            username: "testuser",
            is_admin: false,
            rw_shares: [],
            ro_shares: [],
        };

        render(
            React.createElement(UserDetailsPanel as any, {
                user: mockUser,
                userKey: "testuser",
                onEditClick: () => { },
                readOnly: true,
            })
        );

        // Wait for render, then check button is not present
        await waitFor(() => {
            const editButtons = screen.queryAllByRole("button", { name: /edit user/i });
            expect(editButtons.length).toBe(0);
        });
    });

    it("renders delete button for non-admin users", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserDetailsPanel } = await import("../UserDetailsPanel");

        const mockUser = {
            username: "testuser",
            is_admin: false,
            rw_shares: [],
            ro_shares: [],
        };

        render(
            React.createElement(UserDetailsPanel as any, {
                user: mockUser,
                userKey: "testuser",
                onDelete: () => { },
                readOnly: false,
            })
        );

        const deleteButton = await screen.findByRole("button", { name: /delete user/i });
        expect(deleteButton).toBeTruthy();
    });

    it("hides delete button for admin users", async () => {
        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const { UserDetailsPanel } = await import("../UserDetailsPanel");

        const mockUser = {
            username: "admin",
            is_admin: true,
            rw_shares: [],
            ro_shares: [],
        };

        render(
            React.createElement(UserDetailsPanel as any, {
                user: mockUser,
                userKey: "admin",
                onDelete: () => { },
                readOnly: false,
            })
        );

        // Wait for render, then check delete button is not present
        await waitFor(() => {
            const deleteButtons = screen.queryAllByRole("button", { name: /delete user/i });
            expect(deleteButtons.length).toBe(0);
        });
    });

    it("displays admin type info text", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserDetailsPanel } = await import("../UserDetailsPanel");

        const mockUser = {
            username: "admin",
            is_admin: true,
            rw_shares: [],
            ro_shares: [],
        };

        render(
            React.createElement(UserDetailsPanel as any, {
                user: mockUser,
                userKey: "admin",
            })
        );

        // Look for admin info text
        const adminInfo = await screen.findByText(/admin user can be renamed/i);
        expect(adminInfo).toBeTruthy();
    });

    it("displays regular user type info text", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserDetailsPanel } = await import("../UserDetailsPanel");

        const mockUser = {
            username: "testuser",
            is_admin: false,
            rw_shares: [],
            ro_shares: [],
        };

        render(
            React.createElement(UserDetailsPanel as any, {
                user: mockUser,
                userKey: "testuser",
            })
        );

        // Look for regular user info text
        const userInfo = await screen.findByText(/regular user account/i);
        expect(userInfo).toBeTruthy();
    });

    it("renders children when isEditing is true", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserDetailsPanel } = await import("../UserDetailsPanel");

        const mockUser = {
            username: "testuser",
            is_admin: false,
            rw_shares: [],
            ro_shares: [],
        };

        render(
            React.createElement(UserDetailsPanel as any, {
                user: mockUser,
                userKey: "testuser",
                isEditing: true,
                children: React.createElement("div", { "data-testid": "edit-form" }, "Edit Form Content"),
            })
        );

        const editForm = await screen.findByTestId("edit-form");
        expect(editForm).toBeTruthy();
    });
});
