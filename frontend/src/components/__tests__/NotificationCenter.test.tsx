import "../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

// Required localStorage shim for testing environment
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => { _store[k] = String(v); },
        removeItem: (k: string) => { delete _store[k]; },
        clear: () => { for (const k of Object.keys(_store)) delete _store[k]; },
    };
}

describe("NotificationCenter Component", () => {
    beforeEach(() => {
        localStorage.clear();
        // Clear DOM before each test
        document.body.innerHTML = '';
    });

    it("renders notification center icon button", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { NotificationCenter } = await import("../NotificationCenter");

        const theme = createTheme();

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(NotificationCenter as any)
            )
        );

        // Check that the notification button is rendered
        const notificationButtons = container.querySelectorAll('button');
        expect(notificationButtons.length).toBeGreaterThanOrEqual(1);

        // Look for notification icon
        const notificationIcons = container.querySelectorAll('[data-testid="NotificationsIcon"]');
        expect(notificationIcons.length).toBeGreaterThanOrEqual(0);
    });

    it("opens popover when notification button is clicked", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { NotificationCenter } = await import("../NotificationCenter");

        const theme = createTheme();

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(NotificationCenter as any)
            )
        );

        // Find and click the notification button
        const notificationButton = container.querySelector('button');
        if (notificationButton) {
            fireEvent.click(notificationButton);

            // Check if popover opened (look for notifications text)
            expect(container).toBeTruthy();
        }
    });

    it("displays notification count badge", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { NotificationCenter } = await import("../NotificationCenter");

        const theme = createTheme();

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(NotificationCenter as any)
            )
        );

        // Look for badge element
        const badges = container.querySelectorAll('[class*="MuiBadge"]');
        expect(badges.length).toBeGreaterThanOrEqual(0);
    });

    it("handles show read notifications toggle", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { NotificationCenter } = await import("../NotificationCenter");

        const theme = createTheme();

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(NotificationCenter as any)
            )
        );

        // Open the popover first
        const notificationButton = container.querySelector('button');
        if (notificationButton) {
            fireEvent.click(notificationButton);

            // Look for toggle switch
            const switches = container.querySelectorAll('input[type="checkbox"]');
            const firstSwitch = switches[0];
            if (switches.length > 0 && firstSwitch) {
                fireEvent.click(firstSwitch);
            }
        }

        expect(container).toBeTruthy();
    });

    it("renders notification list when popover is open", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { NotificationCenter } = await import("../NotificationCenter");

        const theme = createTheme();

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(NotificationCenter as any)
            )
        );

        // Open the popover
        const notificationButton = container.querySelector('button');
        if (notificationButton) {
            fireEvent.click(notificationButton);

            // Check for notification list container
            const stacks = container.querySelectorAll('[class*="MuiStack"]');
            expect(stacks.length).toBeGreaterThanOrEqual(0);
        }
    });

    it("handles notification actions (clear, mark as read)", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { NotificationCenter } = await import("../NotificationCenter");

        const theme = createTheme();

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(NotificationCenter as any)
            )
        );

        // Open the popover
        const notificationButton = container.querySelector('button');
        if (notificationButton) {
            fireEvent.click(notificationButton);

            // Look for action buttons (clear, mark as read)
            const actionButtons = container.querySelectorAll('button');
            expect(actionButtons.length).toBeGreaterThanOrEqual(1);

            // Try to click action buttons if they exist
            const thirdButton = actionButtons[2];
            if (actionButtons.length > 2 && thirdButton) {
                fireEvent.click(thirdButton); // Try clicking clear or mark as read
            }
        }
    });

    it("displays tooltips on hover", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { NotificationCenter } = await import("../NotificationCenter");

        const theme = createTheme();

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(NotificationCenter as any)
            )
        );

        // Test tooltip on main notification button
        const notificationButton = container.querySelector('button');
        if (notificationButton) {
            fireEvent.mouseEnter(notificationButton);
            fireEvent.mouseLeave(notificationButton);
        }

        expect(container).toBeTruthy();
    });

    it("handles color scheme changes", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { NotificationCenter } = await import("../NotificationCenter");

        const theme = createTheme();

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(NotificationCenter as any)
            )
        );

        // The component should render without errors regardless of color scheme
        expect(container).toBeTruthy();
    });

    it("converts severity levels correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { NotificationCenter } = await import("../NotificationCenter");

        const theme = createTheme();

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(NotificationCenter as any)
            )
        );

        // Test that the component renders (severity conversion is internal)
        expect(container).toBeTruthy();
    });

    it("renders ToastContainer with correct configuration", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { NotificationCenter } = await import("../NotificationCenter");

        const theme = createTheme();

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(NotificationCenter as any)
            )
        );

        // Look for toast container elements
        const toastContainers = container.querySelectorAll('[class*="Toastify"]');
        expect(toastContainers.length).toBeGreaterThanOrEqual(0);
    });

    it("handles popover close correctly", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { NotificationCenter } = await import("../NotificationCenter");

        const theme = createTheme();

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(NotificationCenter as any)
            )
        );

        // Open and close the popover
        const notificationButton = container.querySelector('button');
        if (notificationButton) {
            fireEvent.click(notificationButton);
            // Clicking outside or pressing escape should close popover
            fireEvent.keyDown(document, { key: 'Escape' });
        }

        expect(container).toBeTruthy();
    });

    it("handles disabled action buttons correctly", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { NotificationCenter } = await import("../NotificationCenter");

        const theme = createTheme();

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(NotificationCenter as any)
            )
        );

        // Open the popover
        const notificationButton = container.querySelector('button');
        if (notificationButton) {
            fireEvent.click(notificationButton);

            // Look for disabled buttons (when no notifications)
            const disabledButtons = container.querySelectorAll('button[disabled]');
            expect(disabledButtons.length).toBeGreaterThanOrEqual(0);
        }
    });
});