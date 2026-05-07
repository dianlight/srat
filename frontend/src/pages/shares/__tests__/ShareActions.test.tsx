import { ThemeProvider, createTheme } from "@mui/material/styles";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it } from "vitest";
import { Type, Usage } from "../../../store/sratApi";
import { ShareActions } from "../components/ShareActions";

describe("ShareActions component", () => {
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
});
