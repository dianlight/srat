import { describe, it, expect, beforeEach } from "bun:test";

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
        const React = await import("react");
        const rtl = (await import("@testing-library/react")) as any;
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Swagger } = await import("../Swagger");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        rtl.render(
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

        const heading = await rtl.screen.findByText("API Documentation");
        expect(heading).toBeTruthy();

        const jsonLink = await rtl.screen.findByText("JSON");
        const yamlLink = await rtl.screen.findByText("YAML");
        expect(jsonLink).toBeTruthy();
        expect(yamlLink).toBeTruthy();
    });

    it("includes openapi-explorer with normalized spec-url", async () => {
        const React = await import("react");
        const rtl = (await import("@testing-library/react")) as any;
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Swagger } = await import("../Swagger");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        rtl.render(
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

        const explorer = document.querySelector("openapi-explorer");
        expect(explorer).toBeTruthy();
        const specUrl = explorer?.getAttribute("spec-url") || "";
        expect(specUrl).toContain("openapi.yaml");
        // happy-dom default origin is http://localhost:3000
        expect(specUrl.startsWith("http://localhost:3000")).toBeTruthy();
    });
});