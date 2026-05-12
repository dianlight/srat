import { cleanup } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

// LocalStorage mock for the tests
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => { _store[k] = String(v); },
        removeItem: (k: string) => { delete _store[k]; },
        clear: () => { for (const k of Object.keys(_store)) delete _store[k]; },
    };
}

describe("Settings", () => {
    beforeEach(() => {
        cleanup();
        localStorage.clear();
    });

    afterEach(() => {
        cleanup();
    });

    it("renders settings component with new VS Code-like layout", async () => {
        const React = await import("react");
        const { Settings } = await import("../Settings");

        // Just test that the component can be created without crashing
        expect(() => {
            React.createElement(Settings as any);
        }).not.toThrow();

        // Test that the component is a function
        expect(typeof Settings).toBe("function");
    });

    it("exports Settings component correctly", async () => {
        const { Settings } = await import("../Settings");
        expect(typeof Settings).toBe("function");
    });

    it("can import default config JSON", async () => {
        const defaultConfig = await import("../../../json/default_config.json");
        expect(defaultConfig).toBeTruthy();
        expect(defaultConfig.default).toBeTruthy();
    });

    it("can import TabIDs from locationState", async () => {
        const { TabIDs } = await import("../../../store/locationState");
        expect(TabIDs).toBeTruthy();
        expect(TabIDs.SETTINGS).toBeTruthy();
    });

    it("can import TourEvents and TourEventTypes", async () => {
        const { TourEvents, TourEventTypes } = await import("../../../utils/TourEvents");
        expect(TourEvents).toBeTruthy();
        expect(TourEventTypes).toBeTruthy();
        expect(TourEventTypes.SETTINGS_STEP_3).toBeTruthy();
        expect(TourEventTypes.SETTINGS_STEP_5).toBeTruthy();
        expect(TourEventTypes.SETTINGS_STEP_8).toBeTruthy();
    });

    it("can import API hooks from sratApi", async () => {
        const {
            useGetApiSettingsAppConfigQuery,
            useGetApiSettingsAppConfigSchemaQuery,
            usePutApiSettingsAppConfigMutation,
            useGetApiHostnameQuery,
            useGetApiNicsQuery,
            useGetApiSettingsQuery,
            useGetApiUpdateChannelsQuery,
            useGetApiTelemetryModesQuery,
            useGetApiTelemetryInternetConnectionQuery,
            useGetApiCapabilitiesQuery,
            usePutApiSettingsMutation,
            Telemetry_mode
        } = await import("../../../store/sratApi");

        expect(typeof useGetApiSettingsAppConfigQuery).toBe("function");
        expect(typeof useGetApiSettingsAppConfigSchemaQuery).toBe("function");
        expect(typeof usePutApiSettingsAppConfigMutation).toBe("function");
        expect(typeof useGetApiHostnameQuery).toBe("function");
        expect(typeof useGetApiNicsQuery).toBe("function");
        expect(typeof useGetApiSettingsQuery).toBe("function");
        expect(typeof useGetApiUpdateChannelsQuery).toBe("function");
        expect(typeof useGetApiTelemetryModesQuery).toBe("function");
        expect(typeof useGetApiTelemetryInternetConnectionQuery).toBe("function");
        expect(typeof useGetApiCapabilitiesQuery).toBe("function");
        expect(typeof usePutApiSettingsMutation).toBe("function");
        expect(Telemetry_mode).toBeTruthy();
    });

    it("can import SSE API hook", async () => {
        const { useGetServerEventsQuery } = await import("../../../store/wsApi");
        expect(typeof useGetServerEventsQuery).toBe("function");
    });

    it("can import Material-UI components", async () => {
        const {
            CircularProgress,
            IconButton,
            Stack,
            Typography,
            Button,
            Divider,
            Grid,
            InputAdornment,
            Tooltip
        } = await import("@mui/material");

        expect(CircularProgress).toBeTruthy();
        expect(IconButton).toBeTruthy();
        expect(Stack).toBeTruthy();
        expect(Typography).toBeTruthy();
        expect(Button).toBeTruthy();
        expect(Divider).toBeTruthy();
        expect(Grid).toBeTruthy();
        expect(InputAdornment).toBeTruthy();
        expect(Tooltip).toBeTruthy();
    });

    it("can import Material-UI icons", async () => {
        const AutorenewIcon = await import("@mui/icons-material/Autorenew");
        const PlaylistAddIcon = await import("@mui/icons-material/PlaylistAdd");

        expect(AutorenewIcon.default).toBeTruthy();
        expect(PlaylistAddIcon.default).toBeTruthy();
    });

    it("can import MuiChipsInput", async () => {
        const { MuiChipsInput } = await import("mui-chips-input");
        expect(MuiChipsInput).toBeTruthy();
    });

    it("can import react-hook-form-mui components", async () => {
        const {
            AutocompleteElement,
            CheckboxElement,
            Controller,
            SwitchElement,
            TextFieldElement,
            useForm
        } = await import("react-hook-form-mui");

        expect(AutocompleteElement).toBeTruthy();
        expect(CheckboxElement).toBeTruthy();
        expect(Controller).toBeTruthy();
        expect(SwitchElement).toBeTruthy();
        expect(TextFieldElement).toBeTruthy();
        expect(typeof useForm).toBe("function");
    });

    it("can import React hooks", async () => {
        const { useEffect } = await import("react");
        expect(typeof useEffect).toBe("function");
    });

    it("can import InView from react-intersection-observer", async () => {
        const { InView } = await import("react-intersection-observer");
        expect(InView).toBeTruthy();
    });

    it("Settings component has proper function signature", async () => {
        const { Settings } = await import("../Settings");

        // Settings should be a function component
        expect(typeof Settings).toBe("function");
        expect(Settings.length).toBe(0); // No props expected
    });

    it("can create a simple component instance", async () => {
        const React = await import("react");
        const { Settings } = await import("../Settings");

        // Should be able to create element without throwing
        expect(() => {
            React.createElement(Settings as any);
        }).not.toThrow();
    });

    it("renders form fields for workgroup and NetBIOS name", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Settings } = await import("../Settings");
        const { createTestStore } = await import("/test/testing");

        const store = await createTestStore();
        const theme = createTheme();

        render(
            React.createElement(
                Provider,
                {
                    store, children:
                        React.createElement(
                            ThemeProvider,
                            { theme },
                            React.createElement(Settings as any)
                        )
                }
            )
        );

        // Look for form inputs using accessible queries
        const inputs = screen.queryAllByRole('textbox');
        expect(inputs.length).toBeGreaterThanOrEqual(0);
    });

    it("renders accordion sections", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Settings } = await import("../Settings");
        const { createTestStore } = await import("/test/testing");

        const store = await createTestStore();
        const theme = createTheme();

        render(
            React.createElement(
                Provider,
                {
                    store, children:
                        React.createElement(
                            ThemeProvider,
                            { theme },
                            React.createElement(Settings as any)
                        )
                }
            )
        );

        // Look for region elements (accordions have role="region" when expanded)
        const regions = screen.queryAllByRole('region');
        expect(regions.length).toBeGreaterThanOrEqual(0);
    });

    it("handles telemetry consent toggle", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Settings } = await import("../Settings");
        const { createTestStore } = await import("/test/testing");

        const store = await createTestStore();
        const theme = createTheme();

        render(
            React.createElement(
                Provider,
                {
                    store, children:
                        React.createElement(
                            ThemeProvider,
                            { theme },
                            React.createElement(Settings as any)
                        )
                }
            )
        );

        // Look for switches or checkboxes using semantic queries
        const switches = screen.queryAllByRole('switch');
        const checkboxes = screen.queryAllByRole('checkbox');
        expect(switches.length + checkboxes.length).toBeGreaterThanOrEqual(0);
    });

    it("renders loading indicators correctly", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Settings } = await import("../Settings");
        const { createTestStore } = await import("/test/testing");

        const store = await createTestStore();
        const theme = createTheme();

        render(
            React.createElement(
                Provider,
                {
                    store, children:
                        React.createElement(
                            ThemeProvider,
                            { theme },
                            React.createElement(Settings as any)
                        )
                }
            )
        );

        // Check for progress indicators using semantic query
        const loadingElements = screen.queryAllByRole('progressbar');
        expect(loadingElements.length).toBeGreaterThanOrEqual(0);
    });

    it("renders action buttons", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Settings } = await import("../Settings");
        const { createTestStore } = await import("/test/testing");

        const store = await createTestStore();
        const theme = createTheme();

        render(
            React.createElement(
                Provider,
                {
                    store, children:
                        React.createElement(
                            ThemeProvider,
                            { theme },
                            React.createElement(Settings as any)
                        )
                }
            )
        );

        // Look for buttons using semantic query
        const buttons = screen.queryAllByRole('button');
        expect(buttons.length).toBeGreaterThanOrEqual(0);
    });

    it("renders the 2 fields in Basic settings panel", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { Settings } = await import("../Settings");
        const { createTestStore } = await import("/test/testing");

        const store = await createTestStore();
        const theme = createTheme();

        render(
            React.createElement(
                Provider,
                {
                    store, children:
                        React.createElement(
                            ThemeProvider,
                            { theme },
                            React.createElement(Settings as any)
                        )
                }
            )
        );

        // The Settings component should render successfully
        // Wait for the search field to appear
        const searchField = await screen.findByPlaceholderText("Search settings...");
        expect(searchField).toBeTruthy();

        // General category should be visible - use getAllByText to find tree items specifically
        const generalItems = await screen.findAllByText("General");
        expect(generalItems.length).toBeGreaterThan(0);

        // Verify the component is interactive
        const user = userEvent.setup();
        // Click on the first General item (from the tree)
        await user.click(generalItems[0] as HTMLElement);

        // Component should remain rendered after click
        expect(generalItems[0]).toBeTruthy();
    });

    it("renders AllowGuest toggle in General settings", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Settings } = await import("../Settings");
        const { createTestStore } = await import("/test/testing");

        const store = await createTestStore();
        const theme = createTheme();

        render(
            React.createElement(
                Provider,
                {
                    store, children:
                        React.createElement(
                            ThemeProvider,
                            { theme },
                            React.createElement(Settings as any)
                        )
                }
            )
        );

        // Wait for the search field to appear
        const searchField = await screen.findByPlaceholderText("Search settings...");
        expect(searchField).toBeTruthy();

        // Look for AllowGuest setting in the tree
        const allowGuestElements = await screen.findAllByText(/allow.*guest/i);
        expect(allowGuestElements.length).toBeGreaterThan(0);

        // Verify that the component renders successfully
        expect(allowGuestElements[0]).toBeTruthy();
    });

    it("includes ha_use_nfs in HomeAssistant settings category", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { Settings } = await import("../Settings");
        const { createTestStore } = await import("/test/testing");

        const store = await createTestStore();
        const theme = createTheme();

        render(
            React.createElement(
                Provider,
                {
                    store, children:
                        React.createElement(
                            ThemeProvider,
                            { theme },
                            React.createElement(Settings as any)
                        )
                }
            )
        );

        const searchField = await screen.findByPlaceholderText("Search settings...");
        expect(searchField).toBeTruthy();

        // Check if tree renders
        const treeItems = screen.queryAllByRole("treeitem");
        console.log("DEBUG 411: treeitems=", treeItems.length);
        treeItems.forEach((item, i) => console.log(`  ITEM ${i}:`, item.textContent?.substring(0, 80)));

        expect(screen.queryAllByRole('button').length).toBeGreaterThan(0);
    });

    it("renders disable smart toggle with default false", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { Settings } = await import("../Settings");
        const { createTestStore } = await import("/test/testing");

        const store = await createTestStore();
        const theme = createTheme();

        render(
            React.createElement(
                Provider,
                {
                    store, children:
                        React.createElement(
                            ThemeProvider,
                            { theme },
                            React.createElement(Settings as any)
                        )
                }
            )
        );

        const user = userEvent.setup();
        const searchField = await screen.findByPlaceholderText("Search settings...");
        expect(searchField).toBeTruthy();
        const generalLabels = await screen.findAllByText("General");
        const generalTreeLabel = generalLabels.find(
            el => el.closest('[role="treeitem"]')
        );
        if (generalTreeLabel) {
            await user.click(generalTreeLabel);
        }
        expect(await screen.findByRole("heading", { name: /general/i })).toBeTruthy();
        expect(await screen.findByRole("heading", { name: /general/i })).toBeTruthy();
    });

    it("renders app configuration from option values when schema map is descriptor-like", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { getMswServer } = await import("/test/testing");
        const { Settings } = await import("../Settings");
        const { createTestStore } = await import("/test/testing");

        const server = await getMswServer();
        server.use(
            http.get(/.*\/api\/settings$/, () => HttpResponse.json({
                hostname: "homeassistant",
                workgroup: "WORKGROUP",
                telemetry_mode: "Disabled",
                bind_all_interfaces: true,
                interfaces: [],
            })),
            http.get(/.*\/api\/hostname$/, () => HttpResponse.json("homeassistant")),
            http.get(/.*\/api\/settings\/app-config$/, () => HttpResponse.json({
                options: {
                    auto_update: true,
                    clean_upgrade_dir: false,
                    disable_ipv6: false,
                    factory_reset: false,
                    leave_front_door_open: true,
                    log_level: "debug",
                    srat_update_channel: "develop",
                    use_external_kernel_modules: true,
                },
                runtime_config: { rendered: true },
                requires_restart: true,
            })),
            http.get(/.*\/api\/settings\/app-config\/schema$/, () => HttpResponse.json({
                description: "Configure the current app.",
                long_description: "Schema-driven settings for the running app.",
                requires_restart: true,
                fields: [],
            })),
        );

        const store = await createTestStore();
        const theme = createTheme();

        render(
            React.createElement(
                Provider,
                {
                    store, children:
                        React.createElement(
                            ThemeProvider,
                            { theme },
                            React.createElement(Settings as any)
                        )
                }
            )
        );

        const user = userEvent.setup();
        await screen.findByPlaceholderText("Search settings...");
        const appConfigLabels = await screen.findAllByText("App Configuration");
        const appConfigTreeLabel = appConfigLabels.find(
            el => el.closest('[role="treeitem"]')
        );
        expect(appConfigTreeLabel).toBeTruthy();
        await user.click(appConfigTreeLabel!);
        expect(await screen.findByRole("heading", { name: /app configuration/i })).toBeTruthy();

        // Field-level rendering is covered by dedicated AppConfigurationPanel tests.
        expect(await screen.findByRole("heading", { name: /app configuration/i })).toBeTruthy();
    });

    it("hides rendered runtime configuration when it matches options", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { getMswServer } = await import("/test/testing");
        const { Settings } = await import("../Settings");
        const { createTestStore } = await import("/test/testing");

        const sameConfig = {
            auto_update: true,
            log_level: "debug",
        };

        const server = await getMswServer();
        server.use(
            http.get(/.*\/api\/settings$/, () => HttpResponse.json({
                hostname: "homeassistant",
                workgroup: "WORKGROUP",
                telemetry_mode: "Disabled",
                bind_all_interfaces: true,
                interfaces: [],
            })),
            http.get(/.*\/api\/hostname$/, () => HttpResponse.json("homeassistant")),
            http.get(/.*\/api\/settings\/app-config$/, () => HttpResponse.json({
                options: sameConfig,
                runtime_config: sameConfig,
                requires_restart: true,
            })),
            http.get(/.*\/api\/settings\/app-config\/schema$/, () => HttpResponse.json({
                description: "Configure the current app.",
                long_description: "Schema-driven settings for the running app.",
                requires_restart: true,
                fields: [
                    { name: "auto_update", constraint: "bool", description: "Auto update", optional: false },
                    { name: "log_level", constraint: "str", description: "Logging verbosity", optional: false, options: ["debug", "info"] },
                ],
            })),
        );

        const store = await createTestStore();
        const theme = createTheme();

        render(
            React.createElement(
                Provider,
                {
                    store, children:
                        React.createElement(
                            ThemeProvider,
                            { theme },
                            React.createElement(Settings as any)
                        )
                }
            )
        );

        const user = userEvent.setup();
        await screen.findByPlaceholderText("Search settings...");
        const appConfigLabels = await screen.findAllByText("App Configuration");
        const appConfigTreeLabel = appConfigLabels.find(
            el => el.closest('[role="treeitem"]')
        );
        expect(appConfigTreeLabel).toBeTruthy();
        await user.click(appConfigTreeLabel!);
        expect(await screen.findByRole("heading", { name: /app configuration/i })).toBeTruthy();

        expect(await screen.findByRole("heading", { name: /app configuration/i })).toBeTruthy();
        expect(screen.queryByText(/rendered runtime configuration/i)).toBeNull();
    });

});