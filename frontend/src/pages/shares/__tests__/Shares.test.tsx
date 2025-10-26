import "../../../../test/setup";
import { describe, it, expect, beforeEach, afterEach, mock } from "bun:test";

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
    });

    afterEach(() => {
        mock.restore();
    });

    const setupMocks = async () => {
        const ReactModule = await import("react");

        const mutationSpies = {
            create: 0,
            update: 0,
            remove: 0,
        };

        mock.module("../ShareEditDialog", () => ({
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

        mock.module("../components/ShareDetailsPanel", () => ({
            ShareDetailsPanel: ShareDetailsPanelStub,
        }));

        mock.module("../components/SharesTreeView", () => ({
            SharesTreeView: SharesTreeViewStub,
        }));

        mock.module("../components", () => ({
            ShareDetailsPanel: ShareDetailsPanelStub,
            SharesTreeView: SharesTreeViewStub,
            ShareEditForm: () => ReactModule.createElement("div"),
        }));

        mock.module("../../../hooks/shareHook", () => ({
            useShare: () => ({
                shares: { docKey: sampleShare },
                isLoading: false,
                error: null,
            }),
        }));

        mock.module("../../../hooks/volumeHook", () => ({
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

        mock.module("../../../store/sseApi", () => ({
            useGetServerEventsQuery: () => ({
                data: { hello: { read_only: false, protected_mode: false } },
                isLoading: false,
            }),
        }));

        mock.module("material-ui-confirm", () => ({
            useConfirm: () => () => Promise.resolve({ confirmed: true }),
        }));

        mock.module("react-toastify", () => ({
            toast: { info: () => { }, error: () => { } },
        }));

        mock.module("../../../store/errorSlice", () => ({
            addMessage: (payload: string) => ({ type: "error/add", payload }),
        }));

        mock.module("../../../store/store", () => ({
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
            }
        };

        mock.module("react-router", () => ({
            useLocation: () => ({
                pathname: "/shares",
                state: locationState,
            }),
            useNavigate: () => navigateStub,
        }));

        mock.module("../../../store/sratApi", () => ({
            Usage: { None: "None" },
            usePutApiShareByShareNameMutation: () => [
                () => ({
                    unwrap: () => {
                        mutationSpies.update += 1;
                        return Promise.resolve(sampleShare);
                    },
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

        return mutationSpies;
    };

    it("allows creating, updating, and deleting shares via interactions", async () => {
        const mutationSpies = await setupMocks();

        const React = await import("react");
        const { render, screen, fireEvent, waitFor } = await import("@testing-library/react");
        const { Provider } = await import("react-redux");
        const { createTestStore } = await import("../../../../test/setup");
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
        fireEvent.click(formSubmitButton);

        await waitFor(() => expect(mutationSpies.create).toBe(1));

        const selectButton = await screen.findByTestId("select-share");
        fireEvent.click(selectButton);

        const updateButton = await screen.findByTestId("trigger-update");
        fireEvent.click(updateButton);
        await waitFor(() => expect(mutationSpies.update).toBe(1));

        const deleteButton = await screen.findByTestId("trigger-delete");
        fireEvent.click(deleteButton);
        await waitFor(() => expect(mutationSpies.remove).toBe(1));
    });
});
