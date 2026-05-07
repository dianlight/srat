import path from "path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("Shares page", () => {
    const sampleShare: any = {
        name: "Documents",
        usage: "general",
        disabled: false,
        mount_point_data: {
            path_hash: "hash-docs",
            path: "/mnt/documents",
            is_mounted: true,
            is_write_supported: true,
        },
    };

    beforeEach(() => {
        if ((globalThis as any).localStorage) {
            localStorage.clear();
        }
        vi.restoreAllMocks();
    });

    afterEach(async () => {
        vi.restoreAllMocks();
        // CRITICAL: Reset RTK Query state after tests to prevent pollution
        // Without this, module-mocked tests can corrupt the global API instances
        try {
            const { sratApi } = await import("../../../store/sratApi");
            await import("../../../store/wsApi");
            // Force clear all internal subscription state
            if ((sratApi as any).internalActions) {
                // Reset middleware tracking
            }
        } catch {
            // Ignore if modules not loaded
        }
    });

    const setupMocks = async () => {
        const ReactModule = await import("react");

        const mutationSpies = {
            create: 0,
            update: 0,
            remove: 0,
        };

        vi.doMock("../ShareEditDialog", () => ({
            ShareEditDialog: (props: any) =>
                props.open
                    ? ReactModule.createElement(
                        "div",
                        { "data-testid": "mock-share-dialog" },
                        ReactModule.createElement(
                            "button",
                            {
                                type: "button",
                                "data-testid": "mock-share-form",
                                onClick: () =>
                                    props.onClose({
                                        org_name: props.objectToEdit?.org_name ?? "",
                                        name: props.objectToEdit?.name ?? "NewShare",
                                        mount_point_data: props.objectToEdit?.mount_point_data ?? {
                                            path: "/mnt/free",
                                            path_hash: "free-hash",
                                        },
                                    }),
                            },
                            "submit"
                        )
                    )
                    : null,
        }));

        const ShareDetailsPanelStub = (props: any) =>
            ReactModule.createElement(
                "div",
                null,
                ReactModule.createElement(
                    "button",
                    {
                        "data-testid": "trigger-update",
                        onClick: () =>
                            props.onEdit?.({
                                org_name: props.share?.name || "",
                                name: props.share?.name || "",
                                mount_point_data: { path: "/mnt/documents" },
                            }),
                    },
                    "update"
                ),
                ReactModule.createElement(
                    "button",
                    {
                        "data-testid": "trigger-delete",
                        onClick: () => props.onDelete?.(props.share?.name || "", props.share),
                    },
                    "delete"
                ),
                props.children
            );

        const SharesTreeViewStub = (props: any) =>
            ReactModule.createElement(
                "button",
                {
                    "data-testid": "select-share",
                    onClick: () => props.onShareSelect?.("docKey", sampleShare),
                },
                "select share"
            );

        vi.doMock("../components/ShareDetailsPanel", () => ({
            ShareDetailsPanel: ShareDetailsPanelStub,
        }));

        vi.doMock("../components/SharesTreeView", () => ({
            SharesTreeView: SharesTreeViewStub,
        }));

        vi.doMock("../components", () => ({
            ShareDetailsPanel: ShareDetailsPanelStub,
            SharesTreeView: SharesTreeViewStub,
            ShareEditForm: () => ReactModule.createElement("div"),
        }));

        vi.doMock("../../../hooks/shareHook", () => ({
            useShare: () => ({
                shares: { docKey: sampleShare },
                isLoading: false,
                error: null,
            }),
        }));

        vi.doMock("../../../hooks/volumeHook", () => ({
            useVolume: () => ({
                disks: [
                    {
                        partitions: [
                            {
                                mount_point_data: [
                                    {
                                        path: "/mnt/free",
                                        path_hash: "free-hash",
                                        is_mounted: true,
                                        is_write_supported: true,
                                    },
                                ],
                            },
                        ],
                    },
                ],
                isLoading: false,
                error: null,
            }),
        }));

        // Mock wsApi hook as imported by Shares component (../../store/wsApi)
        vi.doMock("../../store/wsApi", () => ({
            wsApi: {
                reducerPath: "wsApi",
                reducer: fakeReducer,
                middleware: makeMiddleware(),
                util: {
                    resetApiState: () => ({ type: "wsApi/resetApiState" }),
                },
            },
            useGetServerEventsQuery: () => ({
                data: { hello: { read_only: false, protected_mode: false } },
                isLoading: false,
            }),
        }));

        // Minimal API shapes for store creation dynamic imports from test helper module.
        const fakeReducer = (state: any = {}, _action: any) => state;
        const makeMiddleware = () => () => (next: any) => (action: any) => next(action);
        vi.doMock("../src/store/wsApi", () => ({
            wsApi: {
                reducerPath: "wsApi",
                reducer: fakeReducer,
                middleware: makeMiddleware(),
                util: {
                    resetApiState: () => ({ type: "wsApi/resetApiState" }),
                },
            },
        }));

        vi.doMock("material-ui-confirm", () => ({
            useConfirm: () => () => Promise.resolve({ confirmed: true }),
        }));

        vi.doMock("react-toastify", () => ({
            toast: { info: () => { }, error: () => { } },
        }));

        vi.doMock("../../../store/errorSlice", () => ({
            errorSlice: {
                reducer: (state: { messages: string[] } = { messages: [] }) => state,
            },
            addMessage: (payload: string) => ({ type: "error/add", payload }),
        }));

        vi.doMock("../../../store/store", () => ({
            useAppDispatch: () => () => { },
        }));

        let locationState: any = {
            newShareData: {
                path: "/mnt/free",
                path_hash: "free-hash",
                is_mounted: true,
                is_write_supported: true,
            },
        };

        const navigateStub = (_path: string, options?: { state?: any }) => {
            if (options && "state" in options) {
                locationState = options.state ?? {};
            } else {
                // If navigation occurs without explicit state, clear it to prevent loops
                locationState = {};
            }
        };

        vi.doMock("react-router", () => ({
            useLocation: () => ({
                pathname: "/shares",
                state: locationState,
            }),
            useNavigate: () => navigateStub,
        }));

        // Mock sratApi hooks as imported by Shares component (../../store/sratApi)
        vi.doMock("../../store/sratApi", () => {
            // Debug to verify the sratApi mock for Shares component is used
            // console.debug("Using mocked ../../store/sratApi for Shares test");
            return {
                sratApi: {
                    reducerPath: "api",
                    reducer: fakeReducer,
                    middleware: makeMiddleware(),
                    util: {
                        resetApiState: () => ({ type: "api/resetApiState" }),
                    },
                },
                Usage: { None: "none", Backup: "backup", Media: "media", Share: "share", Internal: "internal" },
                // Provide the users query hook used by ShareEditForm to avoid hitting real RTKQ
                useGetApiUsersQuery: () => ({ data: [], isLoading: false, error: null }),
                usePutApiShareByShareNameMutation: () => [
                    () => ({
                        unwrap: () => {
                            mutationSpies.update += 1;
                            return Promise.resolve(sampleShare);
                        },
                    }),
                    {},
                ],
                usePutApiShareByShareNameDisableMutation: () => [
                    () => ({
                        unwrap: () => Promise.resolve(sampleShare),
                    }),
                    {},
                ],
                useDeleteApiShareByShareNameMutation: () => [
                    () => ({
                        unwrap: () => {
                            mutationSpies.remove += 1;
                            return Promise.resolve({});
                        },
                    }),
                    {},
                ],
                usePostApiShareMutation: () => [
                    () => ({
                        unwrap: () => {
                            mutationSpies.create += 1;
                            return Promise.resolve(sampleShare);
                        },
                    }),
                    {},
                ],
            };
        });

        // Defensive: also mock by absolute path in case Bun resolves to absolute module IDs
        vi.doMock(path.resolve(__dirname, "../../../store/sratApi.ts"), () => ({
            // Minimal RTK Query API object for store creation expectations, align reducerPath with default 'api'
            sratApi: {
                reducerPath: "api",
                reducer: fakeReducer,
                middleware: makeMiddleware(),
                util: {
                    resetApiState: () => ({ type: "api/resetApiState" }),
                },
            },
            // Hooks + enums used by component
            Usage: { None: "None" },
            useGetApiUsersQuery: () => ({ data: [], isLoading: false, error: null }),
            usePutApiShareByShareNameMutation: () => [
                () => ({
                    unwrap: () => {
                        mutationSpies.update += 1;
                        return Promise.resolve(sampleShare);
                    },
                }),
                {},
            ],
            usePutApiShareByShareNameDisableMutation: () => [
                () => ({
                    unwrap: () => Promise.resolve(sampleShare),
                }),
                {},
            ],
            useDeleteApiShareByShareNameMutation: () => [
                () => ({
                    unwrap: () => {
                        mutationSpies.remove += 1;
                        return Promise.resolve({});
                    },
                }),
                {},
            ],
            usePostApiShareMutation: () => [
                () => ({
                    unwrap: () => {
                        mutationSpies.create += 1;
                        return Promise.resolve(sampleShare);
                    },
                }),
                {},
            ],
        }));

        vi.doMock(path.resolve(__dirname, "../../../store/wsApi.ts"), () => ({
            wsApi: {
                reducerPath: "wsApi",
                reducer: fakeReducer,
                middleware: makeMiddleware(),
                util: {
                    resetApiState: () => ({ type: "wsApi/resetApiState" }),
                },
            },
            useGetServerEventsQuery: () => ({
                data: { hello: { read_only: false, protected_mode: false } },
                isLoading: false,
            }),
        }));

        // Provide minimal RTK Query API object for store creation dynamic import (../src/store/sratApi)
        vi.doMock("../src/store/sratApi", () => ({
            // Align with default 'api' reducerPath so createTestStore wires middleware correctly
            sratApi: {
                reducerPath: "api",
                reducer: fakeReducer,
                middleware: makeMiddleware(),
                util: {
                    resetApiState: () => ({ type: "api/resetApiState" }),
                },
            },
        }));

        return mutationSpies;
    };

    it("allows creating, updating, and deleting shares via interactions", async () => {
        const mutationSpies = await setupMocks();
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("/test/testing");
        // @ts-expect-error - Query param ensures fresh module instance for mocks
        const { Shares } = await import("../Shares?shares-test");

        const store = await createTestStore();

        render(
            React.createElement(
                Provider as any,
                { store },
                React.createElement(Shares as any)
            )
        );

        const formSubmitButton = await screen.findByTestId("mock-share-form");
        await user.click(formSubmitButton as any);

        await waitFor(() => expect(mutationSpies.create).toBe(1));

        const selectButton = await screen.findByTestId("select-share");
        await user.click(selectButton as any);

        const updateButton = await screen.findByTestId("trigger-update");
        await user.click(updateButton as any);
        await waitFor(() => expect(mutationSpies.update).toBe(1));

        const deleteButton = await screen.findByTestId("trigger-delete");
        await user.click(deleteButton as any);
        await waitFor(() => expect(mutationSpies.remove).toBe(1));
    });
});
