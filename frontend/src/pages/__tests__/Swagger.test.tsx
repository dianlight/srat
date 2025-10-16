import { describe, it, expect, beforeEach } from "bun:test";

// Minimal localStorage shim for bun:test
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => { _store[k] = String(v); },
        removeItem: (k: string) => { delete _store[k]; },
        clear: () => { for (const k of Object.keys(_store)) delete _store[k]; },
    };
}

describe("Swagger Component", () => {
    beforeEach(() => {
        localStorage.clear();
    });

    it("renders Swagger component with API documentation interface", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { CssBaseline } = await import("@mui/material");
        const { Swagger } = await import("../Swagger");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme },
                        React.createElement(CssBaseline),
                        React.createElement(Swagger as any)
                    ),
                },
            )
        );

        // Check that the main Box container is rendered
        const boxContainer = document.querySelector('[class*="MuiBox-root"]');
        expect(boxContainer).toBeTruthy();
    });

    it("renders openapi-explorer web component", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Swagger } = await import("../Swagger");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme },
                        React.createElement(Swagger as any)
                    ),
                },
            )
        );

        // Check that openapi-explorer custom element is present
        const openapiExplorer = document.querySelector('openapi-explorer');
        expect(openapiExplorer).toBeTruthy();
    });

    it("renders API documentation title in overview slot", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Swagger } = await import("../Swagger");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme },
                        React.createElement(Swagger as any)
                    ),
                },
            )
        );

        // Check that API Documentation title appears - use getAllByText since there may be multiple instances
        const titleElements = screen.getAllByText("API Documentation");
        expect(titleElements.length).toBeGreaterThan(0);
        const firstTitle = titleElements[0];
        if (firstTitle) {
            expect(firstTitle.tagName.toLowerCase()).toBe("h1");
        }
    });

    it("renders JSON and YAML download links", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Swagger } = await import("../Swagger");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme },
                        React.createElement(Swagger as any)
                    ),
                },
            )
        );

        // Check that JSON link appears - use getAllByText since there may be multiple instances
        const jsonLinks = screen.getAllByText("JSON");
        expect(jsonLinks.length).toBeGreaterThan(0);
        const firstJson = jsonLinks[0];
        if (firstJson) {
            expect(firstJson.tagName.toLowerCase()).toBe("a");
        }

        // Check that YAML link appears - use getAllByText since there may be multiple instances  
        const yamlLinks = screen.getAllByText("YAML");
        expect(yamlLinks.length).toBeGreaterThan(0);
        const firstYaml = yamlLinks[0];
        if (firstYaml) {
            expect(firstYaml.tagName.toLowerCase()).toBe("a");
        }
    });

    it("configures openapi-explorer with correct spec-url attribute", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Swagger } = await import("../Swagger");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme },
                        React.createElement(Swagger as any)
                    ),
                },
            )
        );

        // Check that openapi-explorer has spec-url attribute
        const openapiExplorer = document.querySelector('openapi-explorer');
        expect(openapiExplorer).toBeTruthy();
        expect(openapiExplorer?.getAttribute('spec-url')).toContain('openapi.yaml');
    });

    it("renders overview slot with proper slot attribute", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Swagger } = await import("../Swagger");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme },
                        React.createElement(Swagger as any)
                    ),
                },
            )
        );

        // Check that overview slot div is present
        const overviewSlot = document.querySelector('div[slot="overview"]');
        expect(overviewSlot).toBeTruthy();
    });

    it("uses normalized URLs for API endpoints", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Swagger } = await import("../Swagger");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme },
                        React.createElement(Swagger as any)
                    ),
                },
            )
        );

        // Check that JSON link has proper href - use getAllByText since there may be multiple instances
        const jsonLinks = screen.getAllByText("JSON");
        expect(jsonLinks.length).toBeGreaterThan(0);
        const firstJsonLink = jsonLinks[0];
        if (firstJsonLink) {
            const jsonHref = firstJsonLink.getAttribute('href');
            expect(jsonHref).toContain('openapi.json');
            expect(jsonHref).toContain('http://localhost:3000'); // Should be normalized URL
        }

        // Check that YAML link has proper href - use getAllByText since there may be multiple instances  
        const yamlLinks = screen.getAllByText("YAML");
        expect(yamlLinks.length).toBeGreaterThan(0);
        const firstYamlLink = yamlLinks[0];
        if (firstYamlLink) {
            const yamlHref = firstYamlLink.getAttribute('href');
            expect(yamlHref).toContain('openapi.yaml');
            expect(yamlHref).toContain('http://localhost:3000'); // Should be normalized URL
        }
    });

    it("renders component structure correctly", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Swagger } = await import("../Swagger");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme },
                        React.createElement(Swagger as any)
                    ),
                },
            )
        );

        // Check that component hierarchy is correct
        const boxContainer = document.querySelector('[class*="MuiBox-root"]');
        expect(boxContainer).toBeTruthy();

        const openapiExplorer = boxContainer?.querySelector('openapi-explorer');
        expect(openapiExplorer).toBeTruthy();

        const overviewSlot = openapiExplorer?.querySelector('div[slot="overview"]');
        expect(overviewSlot).toBeTruthy();
    });

    it("handles theme integration correctly", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Swagger } = await import("../Swagger");
        const { createTestStore } = await import("../../../test/setup");

        const darkTheme = createTheme({ palette: { mode: 'dark' } });
        const store = await createTestStore();

        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme: darkTheme },
                        React.createElement(Swagger as any)
                    ),
                },
            )
        );

        // Check that component still renders correctly with dark theme - use getAllByText since there may be multiple instances
        const titleElements = screen.getAllByText("API Documentation");
        expect(titleElements.length).toBeGreaterThan(0);

        const jsonLinks = screen.getAllByText("JSON");
        expect(jsonLinks.length).toBeGreaterThan(0);
    });
});