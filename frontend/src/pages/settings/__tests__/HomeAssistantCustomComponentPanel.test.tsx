import { cleanup } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "bun:test";
import { http, HttpResponse } from "msw";
import "../../../../test/setup";

describe("HomeAssistantCustomComponentPanel", () => {
    beforeEach(() => {
        cleanup();
        localStorage.clear();
        document.body.innerHTML = '';
    });

    afterEach(() => {
        cleanup();
    });

    it("shows custom component actions with status-driven enablement", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { getMswServer } = await import("../../../../test/bun-setup");
        const { HomeAssistantCustomComponentPanel } = await import("../HomeAssistantCustomComponentPanel");
        const { createTestStore } = await import("../../../../test/setup");

        const server = await getMswServer();
        server.use(
            http.get(/.*\/api\/settings\/homeassistant\/custom-component\/status$/, () =>
                HttpResponse.json({
                    component: "srat",
                    install_path: "/homeassistant/custom_components/srat",
                    manifest_path: "/homeassistant/custom_components/srat/manifest.json",
                    installed: true,
                    connected: false,
                    can_install: false,
                    can_upgrade: true,
                    can_uninstall: true,
                    installed_version: "2026.04.1",
                    latest_version: "2026.04.9",
                })
            )
        );

        const store = await createTestStore();
        const theme = createTheme({ components: { MuiDialog: { defaultProps: { slotProps: { transition: { timeout: 0 } } } } } });

        render(
            React.createElement(
                Provider,
                {
                    store, children:
                        React.createElement(
                            ThemeProvider,
                            { theme },
                            React.createElement(HomeAssistantCustomComponentPanel as any, { readOnly: false })
                        )
                }
            )
        );

        const installButton = await screen.findByRole("button", { name: /^install$/i });
        const upgradeButton = await screen.findByRole("button", { name: /^upgrade$/i });
        const uninstallButton = await screen.findByRole("button", { name: /^uninstall$/i });

        expect((installButton as HTMLButtonElement).disabled).toBe(true);
        expect((upgradeButton as HTMLButtonElement).disabled).toBe(false);
        expect((uninstallButton as HTMLButtonElement).disabled).toBe(false);
    });

    it("shows custom component status error feedback", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { getMswServer } = await import("../../../../test/bun-setup");
        const { HomeAssistantCustomComponentPanel } = await import("../HomeAssistantCustomComponentPanel");
        const { createTestStore } = await import("../../../../test/setup");

        const server = await getMswServer();
        server.use(
            http.get(/.*\/api\/settings\/homeassistant\/custom-component\/status$/, () =>
                HttpResponse.json({ message: "status unavailable" }, { status: 500 })
            )
        );

        const store = await createTestStore();
        const theme = createTheme({ components: { MuiDialog: { defaultProps: { slotProps: { transition: { timeout: 0 } } } } } });

        render(
            React.createElement(
                Provider,
                {
                    store, children:
                        React.createElement(
                            ThemeProvider,
                            { theme },
                            React.createElement(HomeAssistantCustomComponentPanel as any, { readOnly: false })
                        )
                }
            )
        );

        expect(await screen.findByText(/status unavailable/i)).toBeTruthy();
    });

    it("shows custom component action failure feedback", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { getMswServer } = await import("../../../../test/bun-setup");
        const { HomeAssistantCustomComponentPanel } = await import("../HomeAssistantCustomComponentPanel");
        const { createTestStore } = await import("../../../../test/setup");

        const server = await getMswServer();
        server.use(
            http.get(/.*\/api\/settings\/homeassistant\/custom-component\/status$/, () =>
                HttpResponse.json({
                    component: "srat",
                    install_path: "/homeassistant/custom_components/srat",
                    manifest_path: "/homeassistant/custom_components/srat/manifest.json",
                    installed: false,
                    connected: false,
                    can_install: true,
                    can_upgrade: false,
                    can_uninstall: false,
                    installed_version: "",
                    latest_version: "2026.04.9",
                })
            ),
            http.post(/.*\/api\/settings\/homeassistant\/custom-component\/install$/, () =>
                HttpResponse.json({ message: "install failed" }, { status: 500 })
            )
        );

        const store = await createTestStore();
        const theme = createTheme({ components: { MuiDialog: { defaultProps: { slotProps: { transition: { timeout: 0 } } } } } });

        render(
            React.createElement(
                Provider,
                {
                    store, children:
                        React.createElement(
                            ThemeProvider,
                            { theme },
                            React.createElement(HomeAssistantCustomComponentPanel as any, { readOnly: false })
                        )
                }
            )
        );

        const user = userEvent.setup();
        const installButton = await screen.findByRole("button", { name: /^install$/i });
        await user.click(installButton);

        // Confirmation dialog should appear; confirm the action
        const confirmButton = await screen.findByRole("button", { name: /^confirm$/i });
        await user.click(confirmButton);

        expect(await screen.findByText(/install failed/i)).toBeTruthy();
    });

    it("shows install confirmation dialog when Install button is clicked", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { getMswServer } = await import("../../../../test/bun-setup");
        const { HomeAssistantCustomComponentPanel } = await import("../HomeAssistantCustomComponentPanel");
        const { createTestStore } = await import("../../../../test/setup");

        const server = await getMswServer();
        server.use(
            http.get(/.*\/api\/settings\/homeassistant\/custom-component\/status$/, () =>
                HttpResponse.json({
                    component: "srat",
                    install_path: "/homeassistant/custom_components/srat",
                    manifest_path: "/homeassistant/custom_components/srat/manifest.json",
                    installed: false,
                    connected: false,
                    can_install: true,
                    can_upgrade: false,
                    can_uninstall: false,
                    installed_version: "",
                    latest_version: "2026.04.9",
                })
            )
        );

        const store = await createTestStore();
        const theme = createTheme({ components: { MuiDialog: { defaultProps: { slotProps: { transition: { timeout: 0 } } } } } });

        render(
            React.createElement(
                Provider,
                { store, children: React.createElement(ThemeProvider, { theme }, React.createElement(HomeAssistantCustomComponentPanel as any, { readOnly: false })) }
            )
        );

        const user = userEvent.setup();
        const installButton = await screen.findByRole("button", { name: /^install$/i });
        await user.click(installButton);

        // Confirmation dialog must appear: verify via action buttons
        await screen.findByRole("button", { name: /^confirm$/i });
        expect(screen.getByRole("button", { name: /^cancel$/i })).toBeTruthy();

        // Verify dialog title and version are in the dialog's text content
        const dialog = screen.getByRole("dialog");
        expect(dialog.textContent).toMatch(/install srat custom component/i);
        expect(dialog.textContent).toMatch(/2026\.04\.9/);
    });

    it("dismisses confirmation dialog when Cancel is clicked", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { getMswServer } = await import("../../../../test/bun-setup");
        const { HomeAssistantCustomComponentPanel } = await import("../HomeAssistantCustomComponentPanel");
        const { createTestStore } = await import("../../../../test/setup");

        const server = await getMswServer();
        server.use(
            http.get(/.*\/api\/settings\/homeassistant\/custom-component\/status$/, () =>
                HttpResponse.json({
                    component: "srat",
                    installed: false,
                    connected: false,
                    can_install: true,
                    can_upgrade: false,
                    can_uninstall: false,
                    installed_version: "",
                    latest_version: "2026.04.9",
                })
            )
        );

        const store = await createTestStore();
        const theme = createTheme({ components: { MuiDialog: { defaultProps: { slotProps: { transition: { timeout: 0 } } } } } });

        render(
            React.createElement(
                Provider,
                { store, children: React.createElement(ThemeProvider, { theme }, React.createElement(HomeAssistantCustomComponentPanel as any, { readOnly: false })) }
            )
        );

        const user = userEvent.setup();
        const installButton = await screen.findByRole("button", { name: /^install$/i });
        await user.click(installButton);

        // Dialog opens; verify it's open
        await screen.findByRole("button", { name: /^cancel$/i });

        // Click Cancel
        const cancelButton = screen.getByRole("button", { name: /^cancel$/i });
        await user.click(cancelButton);

        // Dialog should be dismissed - confirm button gone, no success message
        expect(screen.queryByRole("button", { name: /^confirm$/i })).toBeNull();
        expect(screen.queryByText(/installed successfully/i)).toBeNull();
    });

    it("shows restart dialog after successful install", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { getMswServer } = await import("../../../../test/bun-setup");
        const { HomeAssistantCustomComponentPanel } = await import("../HomeAssistantCustomComponentPanel");
        const { createTestStore } = await import("../../../../test/setup");

        const server = await getMswServer();
        server.use(
            http.get(/.*\/api\/settings\/homeassistant\/custom-component\/status$/, () =>
                HttpResponse.json({
                    component: "srat",
                    install_path: "/homeassistant/custom_components/srat",
                    installed: false,
                    connected: false,
                    can_install: true,
                    can_upgrade: false,
                    can_uninstall: false,
                    installed_version: "",
                    latest_version: "2026.04.9",
                })
            ),
            http.post(/.*\/api\/settings\/homeassistant\/custom-component\/install$/, () =>
                HttpResponse.json({ installed: true })
            )
        );

        const store = await createTestStore();
        const theme = createTheme({ components: { MuiDialog: { defaultProps: { slotProps: { transition: { timeout: 0 } } } } } });

        render(
            React.createElement(
                Provider,
                { store, children: React.createElement(ThemeProvider, { theme }, React.createElement(HomeAssistantCustomComponentPanel as any, { readOnly: false })) }
            )
        );

        const user = userEvent.setup();
        const installButton = await screen.findByRole("button", { name: /^install$/i });
        await user.click(installButton);

        const confirmButton = await screen.findByRole("button", { name: /^confirm$/i });
        await user.click(confirmButton);

        // Restart required dialog should appear after successful install
        await screen.findByRole("button", { name: /restart now/i });
        expect(screen.getByRole("button", { name: /later/i })).toBeTruthy();
        const restartDialog = screen.getByRole("dialog");
        expect(restartDialog.textContent).toMatch(/restart required/i);
    });

    it("dismisses restart dialog when Later is clicked", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { getMswServer } = await import("../../../../test/bun-setup");
        const { HomeAssistantCustomComponentPanel } = await import("../HomeAssistantCustomComponentPanel");
        const { createTestStore } = await import("../../../../test/setup");

        const server = await getMswServer();
        server.use(
            http.get(/.*\/api\/settings\/homeassistant\/custom-component\/status$/, () =>
                HttpResponse.json({
                    component: "srat",
                    installed: false,
                    connected: false,
                    can_install: true,
                    can_upgrade: false,
                    can_uninstall: false,
                    installed_version: "",
                    latest_version: "2026.04.9",
                })
            ),
            http.post(/.*\/api\/settings\/homeassistant\/custom-component\/install$/, () =>
                HttpResponse.json({ installed: true })
            )
        );

        const store = await createTestStore();
        const theme = createTheme({ components: { MuiDialog: { defaultProps: { slotProps: { transition: { timeout: 0 } } } } } });

        render(
            React.createElement(
                Provider,
                { store, children: React.createElement(ThemeProvider, { theme }, React.createElement(HomeAssistantCustomComponentPanel as any, { readOnly: false })) }
            )
        );

        const user = userEvent.setup();
        await user.click(await screen.findByRole("button", { name: /^install$/i }));
        await user.click(await screen.findByRole("button", { name: /^confirm$/i }));

        // Wait for restart dialog to appear, then click Later
        const laterButton = await screen.findByRole("button", { name: /later/i });
        await user.click(laterButton);

        // Dialog should be dismissed
        expect(screen.queryByRole("button", { name: /restart now/i })).toBeNull();
    });

    it("calls POST /api/settings/homeassistant/restart-core when Restart Now is clicked", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { getMswServer } = await import("../../../../test/bun-setup");
        const { HomeAssistantCustomComponentPanel } = await import("../HomeAssistantCustomComponentPanel");
        const { createTestStore } = await import("../../../../test/setup");

        let restartCalled = false;
        const server = await getMswServer();
        server.use(
            http.get(/.*\/api\/settings\/homeassistant\/custom-component\/status$/, () =>
                HttpResponse.json({
                    component: "srat",
                    installed: false,
                    connected: false,
                    can_install: true,
                    can_upgrade: false,
                    can_uninstall: false,
                    installed_version: "",
                    latest_version: "2026.04.9",
                })
            ),
            http.post(/.*\/api\/settings\/homeassistant\/custom-component\/install$/, () =>
                HttpResponse.json({ installed: true })
            ),
            http.post(/.*\/api\/settings\/homeassistant\/restart-core$/, () => {
                restartCalled = true;
                return HttpResponse.json("Home Assistant core restart requested");
            })
        );

        const store = await createTestStore();
        const theme = createTheme({ components: { MuiDialog: { defaultProps: { slotProps: { transition: { timeout: 0 } } } } } });

        render(
            React.createElement(
                Provider,
                { store, children: React.createElement(ThemeProvider, { theme }, React.createElement(HomeAssistantCustomComponentPanel as any, { readOnly: false })) }
            )
        );

        const user = userEvent.setup();
        await user.click(await screen.findByRole("button", { name: /^install$/i }));
        await user.click(await screen.findByRole("button", { name: /^confirm$/i }));

        const restartButton = await screen.findByRole("button", { name: /restart now/i });
        await user.click(restartButton);

        expect(restartCalled).toBe(true);
    });
});
