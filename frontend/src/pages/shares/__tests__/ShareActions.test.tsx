import "../../../../test/setup";
import { describe, it, expect, beforeEach, afterEach } from "bun:test";

describe("ShareActions component", () => {
    const createMatchMedia = (matches: boolean) => () => ({
        matches,
        addListener: () => { },
        removeListener: () => { },
        addEventListener: () => { },
        removeEventListener: () => { },
        dispatchEvent: () => false,
        onchange: null,
        media: "",
    }) as any;

    beforeEach(() => {
        (window as any).matchMedia = createMatchMedia(false);
    });

    afterEach(async () => {
        // Reset matchMedia and cleanup rendered components between reruns
        const { cleanup } = await import("@testing-library/react");
        cleanup();
    });

    const buildShare = () => ({
        name: "Public",
        usage: "general",
        disabled: false,
        mount_point_data: {
            path_hash: "hash",
            invalid: false,
        },
        users: [],
        ro_users: [],
    });

    it("renders desktop action buttons and triggers callbacks", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { ShareActions } = await import("../components/ShareActions");

        let viewCalls = 0;
        let disableCalls = 0;

        const theme = createTheme();
        const share = buildShare();

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(ShareActions as any, {
                    shareKey: "shareKey",
                    shareProps: { ...share, mount_point_data: { ...share.mount_point_data, path: "/mnt/test" } },
                    read_only: false,
                    protected_mode: false,
                    onViewVolumeSettings: () => { viewCalls += 1; },
                    onEnable: () => { /* not used */ },
                    onDisable: () => { disableCalls += 1; },
                })
            )
        );

        const viewVolumeButton = (await screen.findAllByRole("button", { name: /view volume mount settings/i }))[0];
        const disableButton = (await screen.findAllByRole("button", { name: /disable share/i }))[0];

        const user = userEvent.setup();
        if (viewVolumeButton) await user.click(viewVolumeButton as any);
        if (disableButton) await user.click(disableButton as any);

        expect(viewCalls).toBe(1);
        expect(disableCalls).toBe(1);
    });

    it("renders compact menu on small screens", async () => {
        (window as any).matchMedia = createMatchMedia(true);

        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { ShareActions } = await import("../components/ShareActions");

        let enableCalls = 0;

        const theme = createTheme();
        const share = buildShare();

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(ShareActions as any, {
                    shareKey: "shareKey",
                    shareProps: { ...share, disabled: true },
                    read_only: false,
                    protected_mode: false,
                    onViewVolumeSettings: () => { },
                    onEnable: () => { enableCalls += 1; },
                    onDisable: () => { },
                })
            )
        );

        const menuButton = await screen.findByRole("button", { name: /more actions/i });
        const user = userEvent.setup();
        await user.click(menuButton as any);

        const enableOption = await screen.findByText(/enable share/i);
        await user.click(enableOption as any);

        expect(enableCalls).toBe(1);
    });
});
