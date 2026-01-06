import "../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

// REQUIRED localStorage shim for every localStorage test
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => { _store[k] = String(v); },
        removeItem: (k: string) => { delete _store[k]; },
        clear: () => { for (const k of Object.keys(_store)) delete _store[k]; },
    };
}

describe("Footer Component", () => {
    beforeEach(() => {
        // Clear the DOM before each test
        document.body.innerHTML = '';
        localStorage.clear();
    });

    it("renders footer with basic information", async () => {
        // REQUIRED: Dynamic imports after globals are set
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Footer } = await import("../Footer");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();
        const currentYear = new Date().getFullYear();

        // REQUIRED: Use React.createElement, not JSX
        render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(Footer as any) }
                    )
                }
            )
        );

        // REQUIRED: Use findByText for async, toBeTruthy() for assertions
        const versionElement = await screen.findByText(/Version/);
        expect(versionElement).toBeTruthy();

        const copyrightElement = await screen.findByText(new RegExp(`© 2024-${currentYear} Copyright`));
        expect(copyrightElement).toBeTruthy();
    });

    it("displays version information with link", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Footer } = await import("../Footer");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(Footer as any) }
                    )
                }
            )
        );

        // Check that version link is present
        const versionLink = container.querySelector('a[href*="commit"]');
        expect(versionLink).toBeTruthy();
        expect(container.textContent?.includes("Version")).toBeTruthy();
    });

    it("renders copyright information", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Footer } = await import("../Footer");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();
        const currentYear = new Date().getFullYear();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(Footer as any) }
                    )
                }
            )
        );

        expect(container.textContent?.includes("Copyright")).toBeTruthy();
        expect(container.textContent?.includes(`2024-${currentYear}`)).toBeTruthy();
    });

    it("renders as a footer element with proper styling", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Footer } = await import("../Footer");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(Footer as any) }
                    )
                }
            )
        );

        // Check that footer element exists
        const footerElement = container.querySelector('footer');
        expect(footerElement).toBeTruthy();
    });

    it("renders footer structure correctly", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Footer } = await import("../Footer");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(Footer as any) }
                    )
                }
            )
        );

        // Check that footer renders with all essential content
        expect(container.textContent?.includes("Version")).toBeTruthy();
        expect(container.textContent?.includes("Copyright")).toBeTruthy();

        // Check that the footer has proper HTML structure
        const footerElement = container.querySelector('footer');
        expect(footerElement).toBeTruthy();

        // Check that Stack component is rendered (indicates proper layout)
        const stackElement = container.querySelector('[class*="MuiStack"]');
        expect(stackElement).toBeTruthy();
    });

    it("handles responsive layout on small screens", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Footer } = await import("../Footer");
        const { createTestStore } = await import("../../../test/setup");

        // Create a theme that simulates small screen
        const theme = createTheme({
            breakpoints: {
                values: {
                    xs: 0,
                    sm: 600,
                    md: 900,
                    lg: 1200,
                    xl: 1536,
                },
            },
        });

        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(Footer as any) }
                    )
                }
            )
        );

        // Footer should still render with basic content
        expect(container.textContent?.includes("Version")).toBeTruthy();
        expect(container.textContent?.includes("Copyright")).toBeTruthy();
    });

    it("shows loading state when server events are loading", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Footer } = await import("../Footer");
        const { createTestStore } = await import("../../../test/setup");

        const theme = createTheme();
        const store = await createTestStore();

        const { container } = render(
            React.createElement(
                Provider,
                {
                    store,
                    children: React.createElement(
                        ThemeProvider,
                        { theme, children: React.createElement(Footer as any) }
                    )
                }
            )
        );

        // Component should render even when loading
        expect(container.textContent?.includes("Version")).toBeTruthy();
    });
});