/* eslint-disable */
import { ThemeProvider, createTheme } from "@mui/material/styles";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { beforeEach, describe, expect, it } from "vitest";
import IssueCard from "../IssueCard";
import { renderWithTestStore } from "/test/testing";

const renderIssueCard = async (
    issue: Record<string, unknown>,
    props: Record<string, unknown> = {}
) => {
    const theme = createTheme();

    const result = await renderWithTestStore(
        React.createElement(
            ThemeProvider,
            { theme },
            React.createElement(IssueCard as any, { issue, ...props })
        )
    );

    return { ...result, screen };
};

describe("IssueCard Component", () => {
    beforeEach(() => {
        localStorage.clear();
    });

    it("renders issue card with basic information", async () => {
        await renderIssueCard({
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

    it("renders other severity labels", async () => {
        await renderIssueCard({
            id: 2,
            title: "Warning Issue",
            description: "This is a warning",
            severity: "warning",
            ignored: false,
        });

        expect(screen.getByText("Warning")).toBeTruthy();
    });

    it("shows resolve button when handler is provided", async () => {
        await renderIssueCard(
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

    it("shows ignored state when requested", async () => {
        await renderIssueCard(
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

    it("hides server-ignored issues when showIgnored is false without local seed", async () => {
        await renderIssueCard(
            {
                id: 10,
                problem_key: "protected_mode",
                title: "Server Ignored Issue",
                description: "This issue is ignored server-side",
                severity: "error",
                ignored: true,
            },
            { showIgnored: false }
        );

        expect(screen.queryByText("Server Ignored Issue")).toBeNull();
    });

    it("hides ignored issues when showIgnored is false", async () => {
        localStorage.setItem("srat_ignored_issues", JSON.stringify(["test_ignored_issue_9"]));

        await renderIssueCard(
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
    });

    it("invokes resolve action when clicked", async () => {
        const user = userEvent.setup();
        let resolved = false;

        await renderIssueCard(
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

    it("shows ignore button when handler is provided", async () => {
        await renderIssueCard(
            {
                id: 10,
                problem_key: "protected_mode",
                title: "Addon in Protected Mode",
                description: "Protected mode is on",
                severity: "error",
                ignored: false,
            },
            { onResolve: () => {}, onIgnore: () => {} }
        );

        expect(screen.getByRole("button", { name: /^ignore$/i })).toBeTruthy();
    });

    it("invokes ignore action with the issue when clicked", async () => {
        const user = userEvent.setup();
        let ignoredIssue: unknown = null;
        const issue = {
            id: 11,
            problem_key: "protected_mode",
            title: "Addon in Protected Mode",
            description: "Protected mode is on",
            severity: "error",
            ignored: false,
        };

        await renderIssueCard(issue, { onIgnore: (value: unknown) => { ignoredIssue = value; } });

        await user.click(screen.getByRole("button", { name: /^ignore$/i }));
        expect(ignoredIssue).toMatchObject({ problem_key: "protected_mode" });
    });

    it("shows re-enable button for ignored issues when handler is provided", async () => {
        const user = userEvent.setup();
        let reenabled = "";
        localStorage.setItem("srat_ignored_issues", JSON.stringify([]));

        await renderIssueCard(
            {
                id: 12,
                problem_key: "protected_mode",
                title: "Addon in Protected Mode",
                description: "Protected mode is on",
                severity: "error",
                ignored: true,
            },
            { showIgnored: true, onReenable: (id: unknown) => { reenabled = String(id); } }
        );

        await user.click(screen.getByRole("button", { name: /re-enable/i }));
        expect(reenabled).toBe("protected_mode");
    });

    it("hides ignore button for ignored issues", async () => {
        await renderIssueCard(
            {
                id: 13,
                problem_key: "protected_mode",
                title: "Addon in Protected Mode",
                description: "Protected mode is on",
                severity: "error",
                ignored: true,
            },
            { showIgnored: true, onIgnore: () => {} }
        );

        expect(screen.queryByRole("button", { name: /^ignore$/i })).toBeNull();
    });
});