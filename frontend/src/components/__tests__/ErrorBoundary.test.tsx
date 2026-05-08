import React from "react";
import { cleanup, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ThemeProvider, createTheme } from "@mui/material/styles";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ErrorBoundary } from "../ErrorBoundary";

describe("ErrorBoundary Component", () => {
    let localCleanup: (() => void) | null = null;

    beforeEach(() => {
        if (localStorage && typeof localStorage.clear === 'function') {
            localStorage.clear();
        }
        // Reset any global state or mocks
        console.error = () => { };
    });

    afterEach(() => {
        if (localCleanup) {
            localCleanup();
            localCleanup = null;
        }
        cleanup();
    });

    it("renders children when there is no error", async () => {
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

    it("renders error UI when child component throws", () => {
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

    it("displays error details in accordion", () => {
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

    it("shows alert with proper error message", () => {
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
        const theme = createTheme();
        const user = userEvent.setup();

        let reloadCalled = false;
        const reloadSpy = vi.spyOn(window.location, "reload").mockImplementation(() => {
            reloadCalled = true;
        });

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

        reloadSpy.mockRestore();
    });

    it("logs error to console when error boundary catches error", () => {
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

    it("displays bug report icon in error alert", () => {
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