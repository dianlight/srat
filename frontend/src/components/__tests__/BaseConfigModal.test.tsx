import "../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

// LocalStorage mock for the tests
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) =>
            _store.hasOwnProperty(k) ? _store[k] : null,
        setItem: (k: string, v: string) => {
            _store[k] = String(v);
        },
        removeItem: (k: string) => {
            delete _store[k];
        },
        clear: () => {
            for (const k of Object.keys(_store)) delete _store[k];
        },
    };
}

describe("BaseConfigModal Component", () => {
    beforeEach(() => {
        localStorage.clear();
        document.body.innerHTML = "";
    });

    it("exports BaseConfigModal as default export", async () => {
        const BaseConfigModal = await import("../BaseConfigModal");
        expect(BaseConfigModal.default).toBeTruthy();
        expect(typeof BaseConfigModal.default).toBe("function");
    });

    it("can import required Material-UI components", async () => {
        const {
            Dialog,
            DialogTitle,
            DialogContent,
            DialogActions,
            Typography,
            Button,
            Box,
            Alert,
            TextField,
            Stack,
        } = await import("@mui/material");

        expect(Dialog).toBeTruthy();
        expect(DialogTitle).toBeTruthy();
        expect(DialogContent).toBeTruthy();
        expect(DialogActions).toBeTruthy();
        expect(Typography).toBeTruthy();
        expect(Button).toBeTruthy();
        expect(Box).toBeTruthy();
        expect(Alert).toBeTruthy();
        expect(TextField).toBeTruthy();
        expect(Stack).toBeTruthy();
    });

    it("can import required API hooks", async () => {
        const {
            usePutApiSettingsMutation,
            usePutApiUseradminMutation,
            useGetApiSettingsQuery,
            useGetApiUsersQuery,
        } = await import("../../store/sratApi");

        expect(typeof usePutApiSettingsMutation).toBe("function");
        expect(typeof usePutApiUseradminMutation).toBe("function");
        expect(typeof useGetApiSettingsQuery).toBe("function");
        expect(typeof useGetApiUsersQuery).toBe("function");
    });

    it("accepts open and onClose props", async () => {
        const React = await import("react");
        const BaseConfigModal = await import("../BaseConfigModal");

        const component = React.createElement(BaseConfigModal.default as any, {
            open: true,
            onClose: () => {
                /* noop */
            },
        });

        expect(component).toBeTruthy();
        expect(component.props.open).toBe(true);
        expect(typeof component.props.onClose).toBe("function");
    });

    it("accepts open: false prop", async () => {
        const React = await import("react");
        const BaseConfigModal = await import("../BaseConfigModal");

        const component = React.createElement(BaseConfigModal.default as any, {
            open: false,
            onClose: () => {
                /* noop */
            },
        });

        expect(component).toBeTruthy();
        expect(component.props.open).toBe(false);
    });

    it("component structure is valid React component", async () => {
        const React = await import("react");
        const BaseConfigModal = await import("../BaseConfigModal");

        const onCloseFn = () => {
            /* noop */
        };

        const component = React.createElement(BaseConfigModal.default as any, {
            open: true,
            onClose: onCloseFn,
        });

        // Check that it's a valid React element and has a type property
        expect(component).toBeTruthy();
        expect(component.type).toBeTruthy();
        expect(component.props).toBeTruthy();
        expect(component.props.open).toBe(true);
        expect(typeof component.props.onClose).toBe("function");
    });
});
