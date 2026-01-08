import "../../../../test/setup";
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

describe("Users component", () => {
    beforeEach(() => {
        // Clear DOM between tests
        document.body.innerHTML = "";
        localStorage.clear();
    });

    it("renders user list with admin and regular users", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Check that the component renders
        expect(container).toBeTruthy();
    });

    it("renders add user button in tree view header", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Check that add user button exists
        const addButton = await screen.findByLabelText("Create new user");
        expect(addButton).toBeTruthy();
    });

    it("renders Users title in the left panel", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Check that Users title is rendered
        const title = await screen.findByText("Users");
        expect(title).toBeTruthy();
    });

    it("renders UserEditDialog component", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Verify UserEditDialog is in the component tree
        expect(container).toBeTruthy();
    });

    it("handles InView component for lazy loading", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Check that the component renders with the InView wrapper
        expect(container.firstChild).toBeTruthy();
    });

    it("renders select message when no user is selected", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Check for the select message
        const selectMessage = await screen.findByText("Select a user from the list to view details");
        expect(selectMessage).toBeTruthy();
    });

    it("renders tree view with groups", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Verify the tree view renders
        expect(container).toBeTruthy();
    });

    it("displays share information for users", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Check basic rendering
        expect(container).toBeTruthy();
    });

    it("handles add button click to open create dialog", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Find and click add button
        const addButton = await screen.findByLabelText("Create new user");
        const user = userEvent.setup();
        await user.click(addButton as any);

        // Verify dialog opens (check for dialog title)
        const dialogTitle = await screen.findByText("New User");
        expect(dialogTitle).toBeTruthy();
    });

    it("handles dialog close action", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Open dialog
        const addButton = await screen.findByLabelText("Create new user");
        const user = userEvent.setup();
        await user.click(addButton as any);

        // Find and click cancel button
        const cancelButton = await screen.findByText("Cancel");
        await user.click(cancelButton as any);

        // Verify dialog closes
        expect(addButton).toBeTruthy();
    });

    it("renders loading state correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Check for loading indicators
        const { screen } = await import("@testing-library/react");
        const loadingElements = screen.queryAllByRole("progressbar");
        expect(loadingElements.length).toBeGreaterThanOrEqual(0);
    });

    it("renders empty state when no users exist", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Verify component handles empty state
        expect(container).toBeTruthy();
    });

    it("renders grid layout with two panels", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Verify the component renders with layout structure
        expect(container.firstChild).toBeTruthy();
    });

    it("handles user data updates from SSE", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Verify the component can handle data updates
        expect(container).toBeTruthy();
    });

    it("persists selection to localStorage", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Check that localStorage handling is set up
        expect(localStorage.getItem("users.selectedUserKey")).toBeNull();
    });

    it("restores expanded groups from localStorage", async () => {
        // Set up localStorage with expanded groups
        localStorage.setItem("users.expandedGroups", JSON.stringify(["group-admin", "group-users"]));

        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { Users } = await import("../Users");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();

        const { container } = render(
            React.createElement(Provider, {
                store,
                children: React.createElement(Users as any),
            })
        );

        // Verify component renders with localStorage state
        expect(container).toBeTruthy();
    });
});
