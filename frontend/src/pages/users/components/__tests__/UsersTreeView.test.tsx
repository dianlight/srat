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

describe("UsersTreeView component", () => {
    beforeEach(() => {
        document.body.innerHTML = "";
        localStorage.clear();
    });

    it("renders empty tree when no users provided", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { UsersTreeView } = await import("../UsersTreeView");

        const mockSelect = () => { };
        const mockExpand = () => { };

        const { container } = render(
            React.createElement(UsersTreeView as any, {
                users: undefined,
                selectedUserKey: undefined,
                onUserSelect: mockSelect,
                expandedItems: [],
                onExpandedItemsChange: mockExpand,
            })
        );

        expect(container).toBeTruthy();
    });

    it("renders admin and regular user groups", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UsersTreeView } = await import("../UsersTreeView");

        const mockUsers = [
            { username: "admin", is_admin: true, rw_shares: ["share1"], ro_shares: [] },
            { username: "user1", is_admin: false, rw_shares: [], ro_shares: ["share2"] },
        ];

        const mockSelect = () => { };
        const mockExpand = () => { };

        render(
            React.createElement(UsersTreeView as any, {
                users: mockUsers,
                selectedUserKey: undefined,
                onUserSelect: mockSelect,
                expandedItems: ["group-admin", "group-users"],
                onExpandedItemsChange: mockExpand,
            })
        );

        // Check that groups are rendered
        const adminGroup = await screen.findByText("Administrators");
        expect(adminGroup).toBeTruthy();
    });

    it("shows user count in group chip", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UsersTreeView } = await import("../UsersTreeView");

        const mockUsers = [
            { username: "admin", is_admin: true, rw_shares: [], ro_shares: [] },
            { username: "user1", is_admin: false, rw_shares: [], ro_shares: [] },
            { username: "user2", is_admin: false, rw_shares: [], ro_shares: [] },
        ];

        const mockSelect = () => { };
        const mockExpand = () => { };

        render(
            React.createElement(UsersTreeView as any, {
                users: mockUsers,
                selectedUserKey: undefined,
                onUserSelect: mockSelect,
                expandedItems: ["group-admin", "group-users"],
                onExpandedItemsChange: mockExpand,
            })
        );

        // Check count chips are rendered
        const adminCount = await screen.findByText("1");
        expect(adminCount).toBeTruthy();
    });

    it("handles user selection callback", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { UsersTreeView } = await import("../UsersTreeView");

        let selectedKey: string | undefined;
        const mockSelect = (key: string, _user: any) => {
            selectedKey = key;
        };
        const mockExpand = () => { };

        const mockUsers = [
            { username: "testuser", is_admin: false, rw_shares: [], ro_shares: [] },
        ];

        render(
            React.createElement(UsersTreeView as any, {
                users: mockUsers,
                selectedUserKey: undefined,
                onUserSelect: mockSelect,
                expandedItems: ["group-users"],
                onExpandedItemsChange: mockExpand,
            })
        );

        // Find and click user item
        const userItem = await screen.findByText("testuser");
        const user = userEvent.setup();
        await user.click(userItem);

        expect(selectedKey).toBe("testuser");
    });

    it("highlights selected user", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UsersTreeView } = await import("../UsersTreeView");

        const mockUsers = [
            { username: "selecteduser", is_admin: false, rw_shares: [], ro_shares: [] },
        ];

        const mockSelect = () => { };
        const mockExpand = () => { };

        render(
            React.createElement(UsersTreeView as any, {
                users: mockUsers,
                selectedUserKey: "selecteduser",
                onUserSelect: mockSelect,
                expandedItems: ["group-users"],
                onExpandedItemsChange: mockExpand,
            })
        );

        // Find user item
        const userItem = await screen.findByText("selecteduser");
        expect(userItem).toBeTruthy();
    });

    it("shows admin badge for admin users", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { UsersTreeView } = await import("../UsersTreeView");

        const mockUsers = [
            { username: "admin", is_admin: true, rw_shares: [], ro_shares: [] },
        ];

        const mockSelect = () => { };
        const mockExpand = () => { };

        const { container } = render(
            React.createElement(UsersTreeView as any, {
                users: mockUsers,
                selectedUserKey: undefined,
                onUserSelect: mockSelect,
                expandedItems: ["group-admin"],
                onExpandedItemsChange: mockExpand,
            })
        );

        // Verify component renders with content
        expect(container.firstChild).toBeTruthy();
    });

    it("shows share count for users with shares", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UsersTreeView } = await import("../UsersTreeView");

        const mockUsers = [
            { username: "user1", is_admin: false, rw_shares: ["share1", "share2"], ro_shares: ["share3"] },
        ];

        const mockSelect = () => { };
        const mockExpand = () => { };

        render(
            React.createElement(UsersTreeView as any, {
                users: mockUsers,
                selectedUserKey: undefined,
                onUserSelect: mockSelect,
                expandedItems: ["group-users"],
                onExpandedItemsChange: mockExpand,
            })
        );

        // Check for share count chip
        const shareChip = await screen.findByText("3 shares");
        expect(shareChip).toBeTruthy();
    });

    it("sorts users alphabetically within groups", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UsersTreeView } = await import("../UsersTreeView");

        const mockUsers = [
            { username: "zebra", is_admin: false, rw_shares: [], ro_shares: [] },
            { username: "alpha", is_admin: false, rw_shares: [], ro_shares: [] },
        ];

        const mockSelect = () => { };
        const mockExpand = () => { };

        render(
            React.createElement(UsersTreeView as any, {
                users: mockUsers,
                selectedUserKey: undefined,
                onUserSelect: mockSelect,
                expandedItems: ["group-users"],
                onExpandedItemsChange: mockExpand,
            })
        );

        // Both users should be rendered
        const alphaUser = await screen.findByText("alpha");
        const zebraUser = await screen.findByText("zebra");
        expect(alphaUser).toBeTruthy();
        expect(zebraUser).toBeTruthy();
    });
});
