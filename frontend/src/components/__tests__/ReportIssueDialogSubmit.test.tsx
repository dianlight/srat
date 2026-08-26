import React from "react";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { http, HttpResponse } from "msw";
import { getMswServer, renderWithTestStore } from "/test/testing";
import { toast } from "react-toastify";
import { ReportIssueDialog } from "../ReportIssueDialog";

const { mockTrigger } = vi.hoisted(() => ({ mockTrigger: vi.fn() }));

vi.mock("../../store/sratApi", async (importOriginal) => {
    const actual =
        await importOriginal<typeof import("../../store/sratApi")>();
    return {
        ...actual,
        usePostApiIssuesReportMutation: () => [
            mockTrigger,
            { isLoading: false },
        ],
    };
});

describe("ReportIssueDialog submit paths", () => {
    beforeEach(() => {
        mockTrigger.mockReset();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    async function renderReportIssueDialog() {
        const server = getMswServer();
        server.use(
            http.get(/.*\/api\/issues\/template(?:\?.*)?$/, () =>
                HttpResponse.json({
                    title: "Bug Report",
                    description: "Describe the bug",
                    problem_type: "bug",
                })
            ),
        );
        return renderWithTestStore(
            React.createElement(ReportIssueDialog as any, {
                open: true,
                onClose: () => {},
            })
        );
    }

    async function fillForm(user: ReturnType<typeof userEvent.setup>) {
        const titleInput = await screen.findByRole(
            "textbox",
            { name: /^Title$/i },
            { timeout: 5000 }
        );
        await user.type(titleInput, "Test issue title");

        const descriptionEditor = await screen.findByRole("textbox", {
            name: /Description/i,
        });
        await user.type(descriptionEditor, "Test issue description");
    }

    it("shows error toast when the report mutation throws", async () => {
        const user = userEvent.setup();
        mockTrigger.mockImplementation(() => {
            throw new Error("boom");
        });
        const toastErrorSpy = vi.spyOn(toast, "error");

        await renderReportIssueDialog();
        await fillForm(user);

        const createButton = await screen.findByRole("button", {
            name: /Create Issue/i,
        });
        expect(createButton.hasAttribute("disabled")).toBe(false);
        await user.click(createButton);

        await vi.waitFor(() => {
            expect(toastErrorSpy).toHaveBeenCalledWith(
                "Failed to generate issue report. Please try again."
            );
        });
    });

    it("shows popup-blocked dialog with issue link when window.open is blocked", async () => {
        const user = userEvent.setup();
        const githubUrl =
            "https://github.com/dianlight/srat/issues/new?title=Test";
        mockTrigger.mockImplementation(() => ({
            unwrap: () =>
                Promise.resolve({
                    github_url: githubUrl,
                    issue_title: "Test",
                }),
        }));
        const openSpy = vi.spyOn(window, "open").mockReturnValue(null);

        await renderReportIssueDialog();
        await fillForm(user);

        const createButton = await screen.findByRole("button", {
            name: /Create Issue/i,
        });
        await user.click(createButton);

        expect(await screen.findByText("Popup blocked")).toBeTruthy();
        const link = screen.getByRole("link", { name: /Open issue link/i });
        expect(link).toHaveAttribute("href", githubUrl);
        expect(openSpy).toHaveBeenCalledWith(githubUrl, "_blank");
    });
});
