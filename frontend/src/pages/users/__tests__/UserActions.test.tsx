import "../../../../test/setup";
import { describe, it, expect } from "bun:test";

describe("UserActions component", () => {
    const buildUser = (overrides?: Partial<any>) => ({
        username: "guest",
        is_admin: false,
        ...overrides,
    });

    it("renders edit and delete actions for non-admin users", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        const { UserActions } = await import("../UserActions");

        let editCalls = 0;
        let deleteCalls = 0;

        render(
            React.createElement(UserActions as any, {
                user: buildUser(),
                read_only: false,
                onEdit: () => { editCalls += 1; },
                onDelete: () => { deleteCalls += 1; },
            })
        );

        const editButton = await screen.findByRole("button", { name: /settings/i });
        fireEvent.click(editButton);

        const deleteButton = await screen.findByRole("button", { name: /delete/i });
        fireEvent.click(deleteButton);

        expect(editCalls).toBe(1);
        expect(deleteCalls).toBe(1);
    });

    it("hides delete action for admin or read-only scenarios", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { UserActions } = await import("../UserActions");

        const { rerender } = render(
            React.createElement(UserActions as any, {
                user: buildUser({ is_admin: true }),
                read_only: false,
                onEdit: () => { },
                onDelete: () => { },
            })
        );

        expect(await screen.findByRole("button", { name: /settings/i })).toBeTruthy();
        expect(screen.queryByRole("button", { name: /delete/i })).toBeNull();

        rerender(
            React.createElement(UserActions as any, {
                user: buildUser(),
                read_only: true,
                onEdit: () => { },
                onDelete: () => { },
            })
        );

        expect(screen.queryAllByRole("button")).toHaveLength(0);
    });
});
