import { ThemeProvider, createTheme } from "@mui/material/styles";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { beforeEach, describe, expect, it } from "vitest";
import IssueCard from "../IssueCard";

const renderIssueCard = (issue: Record<string, unknown>, props: Record<string, unknown> = {}) => {
    const theme = createTheme();

    return render(
        React.createElement(
            ThemeProvider,
            { theme },
            React.createElement(IssueCard as any, { issue, ...props })
        )
    );
};

describe("IssueCard Component", () => {
    beforeEach(() => {
        localStorage.clear();
    });

    it("renders issue card with basic information", () => {
        renderIssueCard({
            id: 1,
            title: "Test Issue Title",
            description: "This is a test issue description",
            severity: "error",
            date: "2024-01-15T10:30:00Z",
            ignored: false,
        });

        expect(screen.getByText("Test Issue Title")).toBeTruthy();
        expect(screen.getByText("This is a test issue description")).toBeTruthy();
        expect(screen.getByText("Error")).toBeTruthy();
    });

    it("renders other severity labels", () => {
        renderIssueCard({
            id: 2,
            title: "Warning Issue",
            description: "This is a warning",
            severity: "warning",
            ignored: false,
        });

        expect(screen.getByText("Warning")).toBeTruthy();
    });

    it("shows resolve button when handler is provided", () => {
        renderIssueCard(
            {
                id: 3,
                title: "Issue with Resolve",
                description: "Resolvable issue",
                severity: "info",
                ignored: false,
            },
            { onResolve: () => {} }
        );

        expect(screen.getByRole("button", { name: /resolve/i })).toBeTruthy();
    });

    it("shows ignored state when requested", () => {
        renderIssueCard(
            {
                id: 4,
                title: "Ignored Issue Visible",
                description: "This issue should be visible",
                severity: "warning",
                ignored: true,
            },
            { showIgnored: true }
        );

        expect(screen.getByText("Ignored Issue Visible")).toBeTruthy();
        expect(screen.getByText("Ignored")).toBeTruthy();
    });

    it("hides ignored issues when showIgnored is false", () => {
        localStorage.setItem("srat_ignored_issues", JSON.stringify(["test_ignored_issue_9"]));

        const { container } = renderIssueCard(
            {
                id: 9,
                problem_key: "test_ignored_issue_9",
                title: "Ignored Issue",
                description: "This issue should be hidden",
                severity: "info",
                ignored: true,
            },
            { showIgnored: false }
        );

        expect(screen.queryByText("Ignored Issue")).toBeNull();
        expect(container.innerHTML === "" || !container.textContent?.includes("Ignored Issue")).toBe(true);
    });

    it("invokes resolve action when clicked", async () => {
        const user = userEvent.setup();
        let resolved = false;

        renderIssueCard(
            {
                id: 7,
                title: "Resolvable Issue",
                description: "This issue can be resolved",
                severity: "error",
                ignored: false,
            },
            { onResolve: () => { resolved = true; } }
        );

        await user.click(screen.getByRole("button", { name: /resolve/i }));
        expect(resolved).toBe(true);
    });
});