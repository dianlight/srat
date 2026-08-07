import { ThemeProvider, createTheme } from "@mui/material/styles";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it } from "vitest";
import { Type, Usage } from "../../../store/sratApi";
import { ShareActions } from "../components/ShareActions";

describe("ShareActions component", () => {
    const theme = createTheme();
    const createMatchMedia = (matches: boolean) => () => (({
        matches,
        addListener: () => { },
        removeListener: () => { },
        addEventListener: () => { },
        removeEventListener: () => { },
        dispatchEvent: () => false,
        onchange: null,
        media: ""
    }) as any);

    beforeEach(() => {
        (window as any).matchMedia = createMatchMedia(false);
    });

    const buildShare = () => ({
        name: "Public",
        usage: Usage.Share,
        disabled: false,
        mount_point_data: {
            path_hash: "hash",
            invalid: false,
            path: "/mnt/test",
            type: Type.Host,
        },
        users: [],
        ro_users: [],
    });

    it("renders desktop action buttons and triggers callbacks", async () => {
        let viewCalls = 0;
        let disableCalls = 0;

        const theme = createTheme();
        const share = buildShare();

        render(
            <ThemeProvider theme={theme}>
                <ShareActions
                    shareKey="shareKey"
                    shareProps={{ ...share, mount_point_data: { ...share.mount_point_data, path: "/mnt/test" } }}
                    protected_mode={false}
                    onViewVolumeSettings={() => {
                        viewCalls += 1;
                    }}
                    onEnable={() => {
                        // not used in this test
                    }}
                    onDisable={() => {
                        disableCalls += 1;
                    }}
                    onDelete={() => {}}
                />
            </ThemeProvider>,
        );

        const viewVolumeButton = screen.getByRole("button", { name: /view volume mount settings/i });
        const disableButton = screen.getByRole("button", { name: /disable share/i });

        const user = userEvent.setup();
        if (viewVolumeButton) await user.click(viewVolumeButton as any);
        if (disableButton) await user.click(disableButton as any);

        expect(viewCalls).toBe(1);
        expect(disableCalls).toBe(1);
    });

    it("renders compact menu on small screens", async () => {
        (window as any).matchMedia = createMatchMedia(true);

        let enableCalls = 0;

        const theme = createTheme();
        const share = buildShare();

        render(
            <ThemeProvider theme={theme}>
                <ShareActions
                    shareKey="shareKey"
                    shareProps={{ ...share, disabled: true }}
                    protected_mode={false}
                    onViewVolumeSettings={() => {}}
                    onEnable={() => {
                        enableCalls += 1;
                    }}
                    onDisable={() => {}}
                    onDelete={() => {}}
                />
            </ThemeProvider>,
        );

        const menuButton = screen.getAllByRole("button", { name: /more actions/i })[0];
        const user = userEvent.setup();
        await user.click(menuButton as any);

        const enableOption = await screen.findByRole("menuitem", { name: /enable share/i });
        await user.click(enableOption as any);

        expect(enableCalls).toBe(1);
    });

    it("offers disable (not delete) for legacy shares with invalid mount data but valid status", async () => {
        const legacyShare = {
            ...buildShare(),
            name: "addons",
            usage: Usage.Internal,
            mount_point_data: {
                path_hash: "hash",
                invalid: true,
                invalid_error: "legacy volume no longer mounted",
                path: "",
                type: Type.Host,
            },
            status: { is_valid: true },
        };

        render(
            <ThemeProvider theme={theme}>
                <ShareActions
                    shareKey="shareKey"
                    shareProps={legacyShare}
                    protected_mode={false}
                    onViewVolumeSettings={() => {}}
                    onEnable={() => {}}
                    onDisable={() => {}}
                    onDelete={() => {}}
                />
            </ThemeProvider>,
        );

        // No delete action for a valid-status share
        expect(screen.queryByRole("button", { name: /delete share/i })).toBeNull();
        // Disable action is offered (share is not disabled)
        expect(screen.getByRole("button", { name: /disable share/i })).toBeTruthy();
    });

    it("offers only delete for shares with status.is_valid === false", async () => {
        const brokenShare = {
            ...buildShare(),
            mount_point_data: {
                path_hash: "hash",
                invalid: false,
                path: "/mnt/test",
                type: Type.Host,
            },
            status: { is_valid: false },
        };

        render(
            <ThemeProvider theme={theme}>
                <ShareActions
                    shareKey="shareKey"
                    shareProps={brokenShare}
                    protected_mode={false}
                    onViewVolumeSettings={() => {}}
                    onEnable={() => {}}
                    onDisable={() => {}}
                    onDelete={() => {}}
                />
            </ThemeProvider>,
        );

        // Delete is the only offered action for status-invalid shares
        expect(screen.getByRole("button", { name: /delete share/i })).toBeTruthy();
        expect(screen.queryByRole("button", { name: /disable share/i })).toBeNull();
    });
});
