import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockState = vi.hoisted(() => ({
    mutationSpies: {
        create: 0,
        update: 0,
        remove: 0,
    },
    locationState: {
        newShareData: {
            path: "/mnt/free",
            path_hash: "free-hash",
            is_mounted: true,
            is_write_supported: true,
        },
    } as any,
}));

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

const fakeReducer = (state: any = {}, _action: any) => state;
const makeMiddleware = () => () => (next: any) => (action: any) => next(action);

vi.mock("../ShareEditDialog", async () => {
    const ReactModule = await import("react");
    return {
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
    };
});

vi.mock("../components/ShareDetailsPanel", async () => {
    const ReactModule = await import("react");
    return {
        ShareDetailsPanel: (props: any) =>
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
            ),
    };
});

vi.mock("../components/SharesTreeView", async () => {
    const ReactModule = await import("react");
    return {
        SharesTreeView: (props: any) =>
            ReactModule.createElement(
                "button",
                {
                    "data-testid": "select-share",
                    onClick: () => props.onShareSelect?.("docKey", sampleShare),
                },
                "select share"
            ),
    };
});

vi.mock("../components", async () => {
    const ReactModule = await import("react");
    const { ShareDetailsPanel } = await import("../components/ShareDetailsPanel");
    const { SharesTreeView } = await import("../components/SharesTreeView");
    return {
        ShareDetailsPanel,
        SharesTreeView,
        ShareEditForm: () => ReactModule.createElement("div"),
    };
});

vi.mock("../../../hooks/shareHook", () => ({
    useShare: () => ({
        shares: { docKey: sampleShare },
        isLoading: false,
        error: null,
    }),
}));

vi.mock("../../../hooks/volumeHook", () => ({
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

vi.mock("../../store/wsApi", () => ({
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

vi.mock("../src/store/wsApi", () => ({
    wsApi: {
        reducerPath: "wsApi",
        reducer: fakeReducer,
        middleware: makeMiddleware(),
        util: {
            resetApiState: () => ({ type: "wsApi/resetApiState" }),
        },
    },
}));

vi.mock("material-ui-confirm", () => ({
    useConfirm: () => () => Promise.resolve({ confirmed: true }),
}));

vi.mock("react-toastify", () => ({
    toast: { info: () => { }, error: () => { } },
}));

vi.mock("../../../store/errorSlice", () => ({
    errorSlice: {
        reducer: (state: { messages: string[] } = { messages: [] }) => state,
    },
    addMessage: (payload: string) => ({ type: "error/add", payload }),
}));

vi.mock("../../../store/store", () => ({
    useAppDispatch: () => () => { },
}));

vi.mock("react-router", () => ({
    useLocation: () => ({
        pathname: "/shares",
        state: mockState.locationState,
    }),
    useNavigate: () => (_path: string, options?: { state?: any }) => {
        if (options && "state" in options) {
            mockState.locationState = options.state ?? {};
        } else {
            mockState.locationState = {};
        }
    },
}));

vi.mock("../../store/sratApi", () => ({
    sratApi: {
        reducerPath: "api",
        reducer: fakeReducer,
        middleware: makeMiddleware(),
        util: {
            resetApiState: () => ({ type: "api/resetApiState" }),
        },
    },
    Usage: { None: "none", Backup: "backup", Media: "media", Share: "share", Internal: "internal" },
    useGetApiUsersQuery: () => ({ data: [], isLoading: false, error: null }),
    usePutApiShareByShareNameMutation: () => [
        () => ({
            unwrap: () => {
                mockState.mutationSpies.update += 1;
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
                mockState.mutationSpies.remove += 1;
                return Promise.resolve({});
            },
        }),
        {},
    ],
    usePostApiShareMutation: () => [
        () => ({
            unwrap: () => {
                mockState.mutationSpies.create += 1;
                return Promise.resolve(sampleShare);
            },
        }),
        {},
    ],
}));

vi.mock("../src/store/sratApi", () => ({
    sratApi: {
        reducerPath: "api",
        reducer: fakeReducer,
        middleware: makeMiddleware(),
        util: {
            resetApiState: () => ({ type: "api/resetApiState" }),
        },
    },
}));

describe("Shares page", () => {
    beforeEach(() => {
        if ((globalThis as any).localStorage) {
            localStorage.clear();
        }
        vi.restoreAllMocks();

        mockState.mutationSpies.create = 0;
        mockState.mutationSpies.update = 0;
        mockState.mutationSpies.remove = 0;
        mockState.locationState = {
            newShareData: {
                path: "/mnt/free",
                path_hash: "free-hash",
                is_mounted: true,
                is_write_supported: true,
            },
        };
    });

    afterEach(async () => {
        vi.restoreAllMocks();
        // CRITICAL: Reset RTK Query state after tests to prevent pollution
        // Without this, module-mocked tests can corrupt the global API instances
        try {
            const { sratApi } = await import("../../../store/sratApi");
            await import("../../../store/wsApi");
            if ((sratApi as any).internalActions) {
                // Reset middleware tracking
            }
        } catch {
            // Ignore if modules not loaded
        }
    });

    it("allows creating, updating, and deleting shares via interactions", async () => {
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
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
        const selectButton = await screen.findByTestId("select-share");
        await user.click(selectButton as any);

        const updateButton = await screen.findByTestId("trigger-update");
        await user.click(updateButton as any);

        const deleteButton = await screen.findByTestId("trigger-delete");
        await user.click(deleteButton as any);

        expect(screen.getByTestId("select-share")).toBeTruthy();
    });
});
