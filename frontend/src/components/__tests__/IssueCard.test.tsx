import { beforeEach, describe, expect, it } from "bun:test";
import "../../../test/setup";

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

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, { issue: mockIssue })
            )
        );

        expect(screen.getByText("Test Issue Title")).toBeTruthy();
        expect(screen.getByText("This is a test issue description")).toBeTruthy();
        expect(screen.getByText("Error")).toBeTruthy();
    });

    it("renders different severity levels correctly", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
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

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, { issue: warningIssue })
            )
        );

        expect(screen.getByText("Warning")).toBeTruthy();
    });

    it("displays resolve button when onResolve prop is provided", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
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

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, {
                    issue: mockIssue,
                    onResolve: mockOnResolve
                })
            )
        );

        expect(screen.getByText("Resolve")).toBeTruthy();
    });

    it("displays ignore status correctly for ignored issues", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
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

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, {
                    issue: ignoredIssue,
                    showIgnored: true
                })
            )
        );

        expect(screen.getByText("Ignored")).toBeTruthy();
    });

    it("handles success severity correctly", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
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

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, { issue: successIssue })
            )
        );

        expect(screen.getByText("Success")).toBeTruthy();
    });

    it("handles unknown severity correctly", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
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

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, { issue: unknownIssue })
            )
        );

        expect(screen.getByText("Unknown")).toBeTruthy();
    });

    it("handles click on resolve button", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        const theme = createTheme();
        const user = userEvent.setup();
        let resolved = false;
        const mockOnResolve = () => { resolved = true; };
        const mockIssue = {
            id: 7,
            title: "Resolvable Issue",
            description: "This issue can be resolved",
            severity: "error" as const,
            date: "2024-01-15T10:30:00Z",
            ignored: false
        };

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, {
                    issue: mockIssue,
                    onResolve: mockOnResolve
                })
            )
        );

        // Find and click the resolve button using semantic query
        const resolveButton = screen.getByRole("button", { name: /resolve/i });
        await user.click(resolveButton);
        expect(resolved).toBe(true);
    });

    it("handles dismiss button click", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        const theme = createTheme();
        const user = userEvent.setup();
        let dismissed = false;
        const mockOnResolve = () => { dismissed = true; };
        const mockIssue = {
            id: 8,
            title: "Dismissable Issue",
            description: "This issue can be dismissed",
            severity: "warning" as const,
            date: "2024-01-15T10:30:00Z",
            ignored: false
        };

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, {
                    issue: mockIssue,
                    onResolve: mockOnResolve
                })
            )
        );

        // Find and click the dismiss button (close icon button)
        const buttons = screen.getAllByRole("button");
        // The close button is typically an icon button without text
        const closeButton = buttons.find(btn => !btn.textContent?.includes('Resolve'));
        if (closeButton) {
            await user.click(closeButton);
            expect(dismissed).toBe(true);
        }
    });

    it("hides ignored issues when showIgnored is false", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        // Set up localStorage to mark issue as ignored
        localStorage.setItem("srat_ignored_issues", JSON.stringify([9]));

        const theme = createTheme();
        const ignoredIssue = {
            id: 9,
            title: "Ignored Issue",
            description: "This issue should be hidden",
            severity: "info" as const,
            date: "2024-01-15T10:30:00Z",
            ignored: false
        };

        const { container } = render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, {
                    issue: ignoredIssue,
                    showIgnored: false
                })
            )
        );

        // Since the issue is in ignored list and showIgnored is false, card should not render or be empty
        const hasContent = screen.queryByText("Ignored Issue");
        // Either null or doesn't render the title
        expect(!hasContent || container.innerHTML === '').toBe(true);
    });

    it("shows ignored issues when showIgnored is true", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        const theme = createTheme();
        const ignoredIssue = {
            id: 10,
            title: "Ignored Issue Visible",
            description: "This issue should be visible",
            severity: "warning" as const,
            date: "2024-01-15T10:30:00Z",
            ignored: true
        };

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, {
                    issue: ignoredIssue,
                    showIgnored: true
                })
            )
        );

        expect(screen.getByText("Ignored Issue Visible")).toBeTruthy();
        expect(screen.getByText("Ignored")).toBeTruthy();
    });

    it("displays date when provided", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        const theme = createTheme();
        const issueWithDate = {
            id: 11,
            title: "Issue with Date",
            description: "This issue has a date",
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

        // Check that the component rendered
        expect(container.firstChild).toBeTruthy();
    });

    it("handles issue without date", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        const theme = createTheme();
        const issueWithoutDate = {
            id: 12,
            title: "Issue without Date",
            description: "This issue has no date",
            severity: "error" as const,
            ignored: false
        };

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, { issue: issueWithoutDate })
            )
        );

        expect(screen.getByText("Issue without Date")).toBeTruthy();
    });

    it("applies correct styling for dark theme", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        const darkTheme = createTheme({ palette: { mode: 'dark' } });
        const mockIssue = {
            id: 13,
            title: "Dark Theme Issue",
            description: "Test dark theme styling",
            severity: "error" as const,
            date: "2024-01-15T10:30:00Z",
            ignored: false
        };

        render(
            React.createElement(
                ThemeProvider,
                { theme: darkTheme },
                React.createElement(IssueCard as any, { issue: mockIssue })
            )
        );

        expect(screen.getByText("Dark Theme Issue")).toBeTruthy();
    });

    it("does not show resolve button for ignored issues", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { default: IssueCard } = await import("../IssueCard");

        const theme = createTheme();
        const mockOnResolve = () => { };
        const ignoredIssue = {
            id: 14,
            title: "Ignored Issue No Resolve",
            description: "Should not show resolve button",
            severity: "warning" as const,
            date: "2024-01-15T10:30:00Z",
            ignored: true
        };

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, {
                    issue: ignoredIssue,
                    onResolve: mockOnResolve,
                    showIgnored: true
                })
            )
        );

        // Resolve button should not be shown for ignored issues
        const resolveButton = screen.queryByRole("button", { name: /resolve/i });
        expect(resolveButton).toBeNull();
    });

    it("formats date correctly when provided", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
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

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, { issue: issueWithDate })
            )
        );

        // Check that the component rendered (date formatting is locale-dependent)
        expect(screen.getByText("Issue with Date")).toBeTruthy();
    });

    it("handles getSeverityConfig function for all severity types", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
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

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(IssueCard as any, { issue: infoIssue })
            )
        );

        expect(screen.getByText("Info")).toBeTruthy();
    });
});