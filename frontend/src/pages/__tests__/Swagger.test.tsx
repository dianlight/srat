import "../../../test/setup";
import { describe, it, expect, beforeEach, afterEach } from "bun:test";

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

    afterEach(async () => {
        const { cleanup } = await import("@testing-library/react");
        cleanup();
    });

    it("renders overview content and links", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Swagger } = await import("../Swagger");
        const { createTestStore } = await import("../../../test/setup");

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
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Swagger } = await import("../Swagger");
        const { createTestStore } = await import("../../../test/setup");

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
        // happy-dom default origin is http://localhost:3000
        expect(specUrl.startsWith("http://localhost:3000")).toBeTruthy();
    });
});