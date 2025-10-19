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

        const { container } = render(
            React.createElement(UserActions as any, {
                user: buildUser(),
                read_only: false,
                onEdit: () => { editCalls += 1; },
                onDelete: () => { deleteCalls += 1; },
            })
        );

        const editButton = container.querySelector('button[aria-label="settings"]') as HTMLButtonElement;
        expect(editButton).toBeTruthy();
        fireEvent.click(editButton);

        const deleteButton = container.querySelector('button[aria-label="delete"]') as HTMLButtonElement;
        expect(deleteButton).toBeTruthy();
        fireEvent.click(deleteButton);

        expect(editCalls).toBe(1);
        expect(deleteCalls).toBe(1);
    });

    it("hides delete action for admin or read-only scenarios", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { UserActions } = await import("../UserActions");

        const { container, rerender } = render(
            React.createElement(UserActions as any, {
                user: buildUser({ is_admin: true }),
                read_only: false,
                onEdit: () => { },
                onDelete: () => { },
            })
        );

        expect(container.querySelector('button[aria-label="settings"]')).toBeTruthy();
        expect(container.querySelector('button[aria-label="delete"]')).toBeNull();

        rerender(
            React.createElement(UserActions as any, {
                user: buildUser(),
                read_only: true,
                onEdit: () => { },
                onDelete: () => { },
            })
        );

        expect(container.querySelectorAll('button')).toHaveLength(0);
    });
});
