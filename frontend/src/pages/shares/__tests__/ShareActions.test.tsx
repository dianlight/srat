import "../../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

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
        const { render, fireEvent, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { ShareActions } = await import("../components/ShareActions");

        let editCalls = 0;
        let viewCalls = 0;
        let deleteCalls = 0;
        let disableCalls = 0;

        const theme = createTheme();
        const share = buildShare();

        render(
            React.createElement(
                ThemeProvider,
                { theme },
                React.createElement(ShareActions as any, {
                    shareKey: "shareKey",
                    shareProps: share,
                    read_only: false,
                    protected_mode: false,
                    onEdit: () => { editCalls += 1; },
                    onViewVolumeSettings: () => { viewCalls += 1; },
                    onDelete: () => { deleteCalls += 1; },
                    onEnable: () => { /* not used */ },
                    onDisable: () => { disableCalls += 1; },
                })
            )
        );

        const settingsButton = (await screen.findAllByRole("button", { name: /settings/i }))[0];
        const viewVolumeButton = (await screen.findAllByRole("button", { name: /view volume mount settings/i }))[0];
        const deleteButton = (await screen.findAllByRole("button", { name: /delete share/i }))[0];
        const disableButton = (await screen.findAllByRole("button", { name: /disable share/i }))[0];

        fireEvent.click(settingsButton);
        fireEvent.click(viewVolumeButton);
        fireEvent.click(deleteButton);
        fireEvent.click(disableButton);

        expect(editCalls).toBe(1);
        expect(viewCalls).toBe(1);
        expect(deleteCalls).toBe(1);
        expect(disableCalls).toBe(1);
    });

    it("renders compact menu on small screens", async () => {
        (window as any).matchMedia = createMatchMedia(true);

        const React = await import("react");
        const { render, fireEvent, screen } = await import("@testing-library/react");
        const { ThemeProvider, createTheme } = await import("@mui/material/styles");
        const { ShareActions } = await import("../components/ShareActions");

        let menuDeleteCalls = 0;

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
                    onEdit: () => { },
                    onViewVolumeSettings: () => { },
                    onDelete: () => { menuDeleteCalls += 1; },
                    onEnable: () => { },
                    onDisable: () => { },
                })
            )
        );

        const menuButton = await screen.findByRole("button", { name: /more actions/i });
        fireEvent.click(menuButton);

        const deleteOption = await screen.findByText(/delete share/i);
        fireEvent.click(deleteOption);

        expect(menuDeleteCalls).toBe(1);
    });
});
