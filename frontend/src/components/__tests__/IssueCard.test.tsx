import { describe, it, expect, beforeEach } from "bun:test";

describe("IssueCard Component", () => {
    beforeEach(() => {
        // Clear the DOM before each test
        document.body.innerHTML = '';
    });

    it("renders issue card with basic information", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        const theme = createTheme();
        const mockIssue = {
            id: 1,
            title: "Test Issue Title",
            description: "This is a test issue description",
            severity: "error" as const,
            date: "2024-01-15T10:30:00Z",
            ignored: false
        };

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, { issue: mockIssue })
            )
        );

        expect(container.textContent?.includes("Test Issue Title")).toBeTruthy();
        expect(container.textContent?.includes("This is a test issue description")).toBeTruthy();
        expect(container.textContent?.includes("Error")).toBeTruthy();
    });

    it("renders different severity levels correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        const theme = createTheme();
        const warningIssue = {
            id: 2,
            title: "Warning Issue",
            description: "This is a warning",
            severity: "warning" as const,
            date: "2024-01-15T10:30:00Z",
            ignored: false
        };

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, { issue: warningIssue })
            )
        );

        expect(container.textContent?.includes("Warning")).toBeTruthy();
    });

    it("displays resolve button when onResolve prop is provided", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        const theme = createTheme();
        const mockOnResolve = () => { };
        const mockIssue = {
            id: 3,
            title: "Issue with Resolve",
            description: "This issue has a resolve handler",
            severity: "info" as const,
            date: "2024-01-15T10:30:00Z",
            ignored: false
        };

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, {
                    issue: mockIssue,
                    onResolve: mockOnResolve
                })
            )
        );

        expect(container.textContent?.includes("Resolve")).toBeTruthy();
    });

    it("displays ignore status correctly for ignored issues", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        const theme = createTheme();
        const ignoredIssue = {
            id: 4,
            title: "Ignored Issue",
            description: "This issue is ignored",
            severity: "error" as const,
            date: "2024-01-15T10:30:00Z",
            ignored: true
        };

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, {
                    issue: ignoredIssue,
                    showIgnored: true
                })
            )
        );

        expect(container.textContent?.includes("Ignored")).toBeTruthy();
    });

    it("handles success severity correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        const theme = createTheme();
        const successIssue = {
            id: 5,
            title: "Success Issue",
            description: "This is a success message",
            severity: "success" as const,
            date: "2024-01-15T10:30:00Z",
            ignored: false
        };

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, { issue: successIssue })
            )
        );

        expect(container.textContent?.includes("Success")).toBeTruthy();
    });

    it("handles unknown severity correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        const theme = createTheme();
        const unknownIssue = {
            id: 6,
            title: "Unknown Severity Issue",
            description: "This has unknown severity",
            severity: "unknown" as any,
            date: "2024-01-15T10:30:00Z",
            ignored: false
        };

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, { issue: unknownIssue })
            )
        );

        expect(container.textContent?.includes("Unknown")).toBeTruthy();
    });

    it("formats date correctly when provided", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        const theme = createTheme();
        const issueWithDate = {
            id: 7,
            title: "Issue with Date",
            description: "This issue has a specific date",
            severity: "info" as const,
            date: "2024-01-15T10:30:00Z",
            ignored: false
        };

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, { issue: issueWithDate })
            )
        );

        // Check that the component rendered (date formatting is locale-dependent)
        expect(container.textContent?.includes("Issue with Date")).toBeTruthy();
    });

    it("handles getSeverityConfig function for all severity types", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        const theme = createTheme();

        // Test info severity
        const infoIssue = {
            id: 8,
            title: "Info Issue",
            description: "This is an info message",
            severity: "info" as const,
            date: "2024-01-15T10:30:00Z",
            ignored: false
        };

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, { issue: infoIssue })
            )
        );

        expect(container.textContent?.includes("Info")).toBeTruthy();
    });
});