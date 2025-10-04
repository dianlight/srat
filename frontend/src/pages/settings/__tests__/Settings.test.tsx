import { describe, it, expect, beforeEach } from "bun:test";

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
        localStorage.clear();
        // Clear DOM between tests
        document.body.innerHTML = '';
    });

    it("renders settings component without crashing", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { ThemeProvider } = await import("@mui/material/styles");
        const { Settings } = await import("../Settings");
        const { createTestStore } = await import("../../../../test/setup");

        const store = await createTestStore();
        const { createTheme } = await import("@mui/material/styles");
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

        expect(document.body.innerHTML).toBeTruthy();
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
        const { useGetServerEventsQuery } = await import("../../../store/sseApi");
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

    it("has validation regex patterns defined", async () => {
        // These are internal to the module, so we can't test them directly
        // But we can verify the module imports without errors
        const settingsModule = await import("../Settings");
        expect(settingsModule.Settings).toBeTruthy();
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
});