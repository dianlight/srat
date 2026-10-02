/* eslint-disable */
import { ThemeProvider, createTheme } from "@mui/material/styles";
import { act, render, screen, waitFor } from "@testing-library/react";
import React from "react";
import { Provider } from "react-redux";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestStore } from "/test/testing";
import { Swagger } from "../Swagger";

vi.mock("prismjs", () => ({ default: {} }));
vi.mock("openapi-explorer", () => ({}));

// Minimal localStorage shim for bun:test
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

describe("Swagger page", () => {
    beforeEach(() => {
        localStorage.clear();
    });

    it("renders overview content and links", async () => {
        const theme = createTheme();
        const store = await createTestStore();

        render(
            React.createElement(
                Provider as any,
                { store },
                React.createElement(
                    ThemeProvider,
                    { theme },
                    React.createElement(Swagger as any),
                ),
            ),
        );

        const heading = await screen.findByText("API Documentation");
        expect(heading).toBeTruthy();

        const jsonLink = await screen.findByText("JSON");
        const yamlLink = await screen.findByText("YAML");
        expect(jsonLink).toBeTruthy();
        expect(yamlLink).toBeTruthy();
    });

    it("includes openapi-explorer with normalized spec-url", async () => {
        const theme = createTheme();
        const store = await createTestStore();

        render(
            React.createElement(
                Provider as any,
                { store },
                React.createElement(
                    ThemeProvider,
                    { theme },
                    React.createElement(Swagger as any),
                ),
            ),
        );

        // Custom element without semantic role - query via data-testid
        const explorer = screen.getByTestId("openapi-explorer");
        expect(explorer).toBeTruthy();
        const specUrl = explorer.getAttribute("spec-url") || "";
        expect(specUrl).toContain("openapi.yaml");
        // apiUrl is environment-dependent; assert normalized absolute URL instead of fixed host
        expect(specUrl.startsWith("http://") || specUrl.startsWith("https://")).toBeTruthy();
    });

    it("logs timeout debug when custom element never becomes defined", async () => {
        const debugSpy = vi.spyOn(console, "debug").mockImplementation(() => {});
        (globalThis as any).__TEST__ = false;
        vi.useFakeTimers();
        try {
            expect((window as any).customElements).toBeTruthy();

            const theme = createTheme();
            const store = await createTestStore();

            render(
                React.createElement(
                    Provider as any,
                    { store },
                    React.createElement(
                        ThemeProvider,
                        { theme },
                        React.createElement(Swagger as any),
                    ),
                ),
            );

            await act(async () => {
                await vi.advanceTimersByTimeAsync(2100);
            });

            expect(debugSpy).toHaveBeenCalledWith(
                "openapi-explorer timeout, marking as loaded anyway",
            );
            expect(screen.getByText("JSON")).toBeTruthy();
        } finally {
            vi.useRealTimers();
            (globalThis as any).__TEST__ = true;
            debugSpy.mockRestore();
        }
    });

    it("logs ready debug when custom element becomes defined", async () => {
        const debugSpy = vi.spyOn(console, "debug").mockImplementation(() => {});
        (globalThis as any).__TEST__ = false;
        try {
            // Define the element so whenDefined resolves immediately.
            (window as any).customElements.define(
                "openapi-explorer",
                class extends HTMLElement {},
            );

            const theme = createTheme();
            const store = await createTestStore();

            render(
                React.createElement(
                    Provider as any,
                    { store },
                    React.createElement(
                        ThemeProvider,
                        { theme },
                        React.createElement(Swagger as any),
                    ),
                ),
            );

            await waitFor(() =>
                expect(debugSpy).toHaveBeenCalledWith(
                    "openapi-explorer custom element is ready",
                ),
            );
        } finally {
            (globalThis as any).__TEST__ = true;
            debugSpy.mockRestore();
        }
    });
});