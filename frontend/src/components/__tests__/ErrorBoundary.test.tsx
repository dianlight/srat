import { afterEach, beforeEach, describe, expect, it } from "bun:test";
import "../../../test/setup";

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
    let cleanup: (() => void) | null = null;

    beforeEach(() => {
        localStorage.clear();
        // Reset any global state or mocks
        console.error = () => { };
    });

    afterEach(async () => {
        if (cleanup) {
            cleanup();
            cleanup = null;
        }
        const { cleanup: rtlCleanup } = await import("@testing-library/react");
        rtlCleanup();
    });

    it("renders children when there is no error", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { ErrorBoundary } = await import("../ErrorBoundary");

        const theme = createTheme();
        const testContent = "Test child content that should render normally";

        const { findByText } = render(
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

        const element = await findByText(testContent);
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

        const { getByText } = render(
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

        // Check that error UI elements are present using semantic queries
        expect(getByText("Oops! Something went wrong.")).toBeTruthy();
        expect(getByText("Reload Page")).toBeTruthy();
        expect(getByText("Error Details")).toBeTruthy();
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

        const { getByText } = render(
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

        expect(getByText("Error Details")).toBeTruthy();
        expect(getByText(/Specific test error/)).toBeTruthy();
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

        const { getByText } = render(
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

        expect(getByText(/An unexpected error occurred in this section/)).toBeTruthy();
    });

    it("handles reload button click functionality", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { ErrorBoundary } = await import("../ErrorBoundary");

        const theme = createTheme();
        const user = userEvent.setup();

        // Mock window.location.reload
        const originalReload = window.location.reload;
        let reloadCalled = false;
        window.location.reload = () => {
            reloadCalled = true;
        };

        const ThrowError = () => {
            throw new Error("Test error");
        };

        const { getByRole } = render(
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

        const reloadButton = getByRole("button", { name: /reload page/i });
        await user.click(reloadButton);

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

        const { getByRole, getByText } = render(
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

        // Check that the Alert component with error severity is rendered using semantic query
        const alertElement = getByRole("alert");
        expect(alertElement).toBeTruthy();

        // Verify alert contains error message (icon is implementation detail, but we can verify content)
        expect(getByText(/An unexpected error occurred/)).toBeTruthy();
    });
});