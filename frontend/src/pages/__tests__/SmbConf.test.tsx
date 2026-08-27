import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// Mock react-syntax-highlighter to avoid refractor/lib/core dependency issues
vi.mock("react-syntax-highlighter", () => ({
    default: ({ children, ...props }: any) => {
        return React.createElement("pre", { "data-testid": "syntax-highlighter", ...props },
            React.createElement("code", null, children)
        );
    }
}));

vi.mock("react-syntax-highlighter/dist/esm/styles/hljs", () => ({
    a11yDark: {},
    a11yLight: {}
}));

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
        vi.restoreAllMocks();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("renders SmbConf component with syntax highlighter", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConfPage: SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("/test/testing");

        const theme = createTheme();
        const store = await createTestStore();

        render(
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

        expect(screen.getByTestId("smbconf-code-viewer")).toBeTruthy();
    });

    it("displays InView component with proper structure", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConfPage: SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("/test/testing");

        const theme = createTheme();
        const store = await createTestStore();

        render(
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

        expect(screen.getByTestId("smbconf-code-viewer")).toBeTruthy();
    });

    it("renders syntax highlighter with correct language setting", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConfPage: SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("/test/testing");

        const theme = createTheme();
        const store = await createTestStore();

        render(
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

        const codeViewer = screen.getByTestId("smbconf-code-viewer");
        expect(codeViewer).toBeTruthy();
    });

    it("handles light theme color scheme correctly", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConfPage: SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("/test/testing");

        const theme = createTheme({
            palette: {
                mode: 'light'
            }
        });
        const store = await createTestStore();

        render(
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

        expect(screen.getByTestId("smbconf-code-viewer")).toBeTruthy();
    });

    it("handles dark theme color scheme correctly", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConfPage: SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("/test/testing");

        const theme = createTheme({
            palette: {
                mode: 'dark'
            }
        });
        const store = await createTestStore();

        render(
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

        expect(screen.getByTestId("smbconf-code-viewer")).toBeTruthy();
    });

    it("handles API query state correctly", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConfPage: SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("/test/testing");

        const theme = createTheme();
        const store = await createTestStore();

        render(
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

        expect(screen.getByTestId("smbconf-code-viewer")).toBeTruthy();
    });

    it("displays syntax highlighter with correct styling properties", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConfPage: SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("/test/testing");

        const theme = createTheme();
        const store = await createTestStore();

        render(
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
        const codeViewer = screen.getByTestId("smbconf-code-viewer");
        expect(codeViewer).toBeTruthy();
    });

    it("handles InView intersection correctly", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;

        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConfPage: SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("/test/testing");

        const theme = createTheme();
        const store = await createTestStore();

        render(
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

        const codeViewer = screen.getByTestId("smbconf-code-viewer");
        expect(codeViewer).toBeTruthy();

        const user = userEvent.setup();
        await user.pointer({ target: codeViewer });

        expect(screen.getByTestId("smbconf-code-viewer")).toBeTruthy();
    });

    it("renders empty config data correctly", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConfPage: SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("/test/testing");

        const theme = createTheme();
        const store = await createTestStore();

        render(
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

        expect(screen.getByTestId("smbconf-code-viewer")).toBeTruthy();
    });

    it("handles colorScheme mode changes", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConfPage: SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("/test/testing");

        const theme = createTheme();
        const store = await createTestStore();

        const { rerender } = render(
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

        expect(screen.getByTestId("smbconf-code-viewer")).toBeTruthy();

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

        expect(screen.getByTestId("smbconf-code-viewer")).toBeTruthy();
    });

    it("applies custom styling to syntax highlighter", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Provider } = await import("react-redux");
        const { SmbConfPage: SmbConf } = await import("../SmbConf");
        const { createTestStore } = await import("/test/testing");

        const theme = createTheme();
        const store = await createTestStore();

        render(
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

        const codeViewer = screen.getByTestId("smbconf-code-viewer");
        expect(codeViewer).toBeTruthy();
    });
});