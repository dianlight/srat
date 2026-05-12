import { ThemeProvider, createTheme } from "@mui/material/styles";
import { render, screen } from "@testing-library/react";
import React from "react";
import { Provider } from "react-redux";
import { beforeEach, describe, expect, it } from "vitest";
import { createTestStore } from "/test/testing";
import { Swagger } from "../Swagger";

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

        const { container } = render(
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

        // Custom element without semantic role - use getElementsByTagName
        const explorer = container.getElementsByTagName("openapi-explorer")[0];
        expect(explorer).toBeTruthy();
        const specUrl = explorer?.getAttribute("spec-url") || "";
        expect(specUrl).toContain("openapi.yaml");
        // apiUrl is environment-dependent; assert normalized absolute URL instead of fixed host
        expect(specUrl.startsWith("http://") || specUrl.startsWith("https://")).toBeTruthy();
    });
});