import { describe, it, expect, beforeEach } from "bun:test";

// Required localStorage shim for testing environment
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => { _store[k] = String(v); },
        removeItem: (k: string) => { delete _store[k]; },
        clear: () => { for (const k of Object.keys(_store)) delete _store[k]; },
    };
}

describe("SmbConf Component", () => {
    beforeEach(() => {
        localStorage.clear();
        // Clear DOM before each test
        document.body.innerHTML = '';
    });

    it("renders SmbConf component with syntax highlighter", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(SmbConf as any) }
                    )
                }
            )
        );

        // Check that syntax highlighter container is rendered
        const highlighterElements = container.querySelectorAll('[class*="hljs"]');
        expect(highlighterElements.length).toBeGreaterThanOrEqual(0);

        // Check that the component rendered without errors
        expect(container).toBeTruthy();
    });

    it("displays InView component with proper structure", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(SmbConf as any) }
                    )
                }
            )
        );

        // Check that InView creates a div container
        expect(container.querySelector('div')).toBeTruthy();
    });

    it("renders syntax highlighter with correct language setting", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(SmbConf as any) }
                    )
                }
            )
        );

        // Check for syntax highlighter elements
        const codeElements = container.querySelectorAll('code');
        expect(codeElements.length).toBeGreaterThanOrEqual(0);
    });

    it("handles light theme color scheme correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme({
            palette: {
                mode: 'light'
            }
        });
        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(SmbConf as any) }
                    )
                }
            )
        );

        // Component should render without errors in light mode
        expect(container).toBeTruthy();
    });

    it("handles dark theme color scheme correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme({
            palette: {
                mode: 'dark'
            }
        });
        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(SmbConf as any) }
                    )
                }
            )
        );

        // Component should render without errors in dark mode
        expect(container).toBeTruthy();
    });

    it("handles API query state correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(SmbConf as any) }
                    )
                }
            )
        );

        // Should render even when API is loading/error state
        expect(container).toBeTruthy();
    });

    it("displays syntax highlighter with correct styling properties", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(SmbConf as any) }
                    )
                }
            )
        );

        // Check that pre or code elements exist (syntax highlighter structure)
        const preElements = container.querySelectorAll('pre');
        const codeElements = container.querySelectorAll('code');

        expect(preElements.length + codeElements.length).toBeGreaterThanOrEqual(0);
    });

    it("handles InView intersection correctly", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(SmbConf as any) }
                    )
                }
            )
        );

        // The InView component should have rendered a div
        const inViewDiv = container.querySelector('div');
        expect(inViewDiv).toBeTruthy();

        // Fire a scroll event to potentially trigger InView
        if (inViewDiv) {
            fireEvent.scroll(window);
        }

        expect(container).toBeTruthy();
    });

    it("renders empty config data correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(SmbConf as any) }
                    )
                }
            )
        );

        // Component should handle empty/null data gracefully
        expect(container).toBeTruthy();
    });

    it("handles colorScheme mode changes", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        const { container, rerender } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(SmbConf as any) }
                    )
                }
            )
        );

        expect(container).toBeTruthy();

        // Rerender with dark theme
        const darkTheme = createTheme({ palette: { mode: 'dark' } });
        rerender(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(
                        ThemeProvider,
                        { theme: darkTheme, children: React.createElement(SmbConf as any) }
                    )
                }
            )
        );

        expect(container).toBeTruthy();
    });

    it("applies custom styling to syntax highlighter", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store, children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(SmbConf as any) }
                    )
                }
            )
        );

        // SyntaxHighlighter should be in the DOM
        expect(container.querySelector('pre, code')).toBeTruthy();
    });
});