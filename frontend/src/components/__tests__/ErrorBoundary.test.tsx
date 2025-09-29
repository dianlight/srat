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

describe("ErrorBoundary Component", () => {
    beforeEach(() => {
        localStorage.clear();
        // Reset any global state or mocks
        console.error = () => { };
    });

    it("renders children when there is no error", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { ErrorBoundary } = await import("../ErrorBoundary");

        const theme = createTheme();
        const testContent = "Test child content that should render normally";

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(
                    ErrorBoundary as any,
                    {},
                    React.createElement("div", {}, testContent)
                )
            )
        );

        const element = await screen.findByText(testContent);
        expect(element).toBeTruthy();
    });

    it("renders error UI when child component throws", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { ErrorBoundary } = await import("../ErrorBoundary");

        const theme = createTheme();

        // Component that always throws an error
        const ThrowError = () => {
            throw new Error("Test error message");
        };

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(
                    ErrorBoundary as any,
                    {},
                    React.createElement(ThrowError)
                )
            )
        );

        // Check that error UI elements are present
        expect(container.textContent?.includes("Oops! Something went wrong.")).toBeTruthy();
        expect(container.textContent?.includes("Reload Page")).toBeTruthy();
        expect(container.textContent?.includes("Error Details")).toBeTruthy();
    });

    it("displays error details in accordion", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { ErrorBoundary } = await import("../ErrorBoundary");

        const theme = createTheme();

        // Component that throws a specific error
        const ThrowSpecificError = () => {
            throw new Error("Specific test error");
        };

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(
                    ErrorBoundary as any,
                    {},
                    React.createElement(ThrowSpecificError)
                )
            )
        );

        expect(container.textContent?.includes("Error Details")).toBeTruthy();
        expect(container.textContent?.includes("Specific test error")).toBeTruthy();
    });

    it("shows alert with proper error message", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { ErrorBoundary } = await import("../ErrorBoundary");

        const theme = createTheme();

        const ThrowError = () => {
            throw new Error("Test error");
        };

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(
                    ErrorBoundary as any,
                    {},
                    React.createElement(ThrowError)
                )
            )
        );

        expect(container.textContent?.includes("An unexpected error occurred in this section")).toBeTruthy();
    });

    it("handles reload button click functionality", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { fireEvent } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { ErrorBoundary } = await import("../ErrorBoundary");

        const theme = createTheme();

        // Mock window.location.reload
        const originalReload = window.location.reload;
        let reloadCalled = false;
        window.location.reload = () => {
            reloadCalled = true;
        };

        const ThrowError = () => {
            throw new Error("Test error");
        };

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(
                    ErrorBoundary as any,
                    {},
                    React.createElement(ThrowError)
                )
            )
        );

        const reloadButtons = screen.getAllByText("Reload Page");
        fireEvent.click(reloadButtons[0]);

        expect(reloadCalled).toBe(true);

        // Restore original reload function
        window.location.reload = originalReload;
    });

    it("logs error to console when error boundary catches error", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { ErrorBoundary } = await import("../ErrorBoundary");

        const theme = createTheme();

        // Mock console.error to track calls
        const consoleLogs: any[] = [];
        console.error = (...args: any[]) => {
            consoleLogs.push(args);
        };

        const ThrowError = () => {
            throw new Error("Test error for logging");
        };

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(
                    ErrorBoundary as any,
                    {},
                    React.createElement(ThrowError)
                )
            )
        );

        // Check that console.error was called
        expect(consoleLogs.length).toBeGreaterThan(0);
    });

    it("displays bug report icon in error alert", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { ErrorBoundary } = await import("../ErrorBoundary");

        const theme = createTheme();

        const ThrowError = () => {
            throw new Error("Test error");
        };

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(
                    ErrorBoundary as any,
                    {},
                    React.createElement(ThrowError)
                )
            )
        );

        // Check that the Alert component with error severity is rendered
        const alertElement = container.querySelector('[role="alert"]');
        expect(alertElement).toBeTruthy();

        // Check for BugReportIcon
        const bugIcon = container.querySelector('[data-testid="BugReportIcon"]');
        expect(bugIcon).toBeTruthy();
    });
});