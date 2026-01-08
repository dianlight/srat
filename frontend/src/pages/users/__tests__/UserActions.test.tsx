import "../../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

describe("UserActions component", () => {
    beforeEach(() => {
        // Clear DOM before each test
        document.body.innerHTML = '';
    });

    const buildUser = (overrides?: Partial<any>) => ({
        username: "guest",
        is_admin: false,
        ...overrides,
    });

    it("renders edit and delete actions for non-admin users", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { UserActions } = await import("../UserActions");

        let editCalls = 0;
        let deleteCalls = 0;

        const user = userEvent.setup();
        const { unmount } = render(
            React.createElement(UserActions as any, {
                user: buildUser(),
                read_only: false,
                onEdit: () => { editCalls += 1; },
                onDelete: () => { deleteCalls += 1; },
            })
        );

        const editButton = screen.getByRole('button', { name: /settings/i });
        expect(editButton).toBeTruthy();
        await user.click(editButton);

        const deleteButton = screen.getByRole('button', { name: /delete/i });
        expect(deleteButton).toBeTruthy();
        await user.click(deleteButton);

        expect(editCalls).toBe(1);
        expect(deleteCalls).toBe(1);
        
        unmount();
    });

    it("hides delete action for admin or read-only scenarios", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserActions } = await import("../UserActions");

        // Test admin user scenario
        const { unmount, rerender } = render(
            React.createElement(UserActions as any, {
                user: buildUser({ is_admin: true }),
                read_only: false,
                onEdit: () => { },
                onDelete: () => { },
            })
        );

        // For admin users: settings button exists, delete button does not
        expect(screen.queryAllByRole('button', { name: /settings/i }).length).toBeGreaterThan(0);
        expect(screen.queryByRole('button', { name: /delete/i })).toBeNull();

        // Test read-only scenario
        rerender(
            React.createElement(UserActions as any, {
                user: buildUser(),
                read_only: true,
                onEdit: () => { },
                onDelete: () => { },
            })
        );

        // For read-only: no buttons should be visible
        expect(screen.queryAllByRole('button')).toHaveLength(0);
        
        unmount();
    });
});
