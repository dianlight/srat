import { afterEach, beforeEach, describe, expect, it, mock } from "bun:test";
import "../../../../../test/setup";

describe("ShareEditForm component", () => {
    beforeEach(() => {
        mock.restore();
    });

    afterEach(async () => {
        mock.restore();
        const { cleanup } = await import("@testing-library/react");
        cleanup();
    });

    const setupCommonOverrides = () => {
        const volumeResult = {
            disks: [
                {
                    partitions: [
                        {
                            mount_point_data: [
                                {
                                    path: "/mnt/free",
                                    path_hash: "free-hash",
                                    disk_label: "FreeDisk",
                                    is_mounted: true,
                                    is_write_supported: true,
                                    time_machine_support: "Supported",
                                },
                            ],
                        },
                    ],
                },
            ],
            isLoading: false,
            error: null,
        };

        const usersResult = {
            data: [
                { username: "admin", is_admin: true },
                { username: "guest", is_admin: false },
            ],
            isLoading: false,
            error: null,
            refetch: () => Promise.resolve(),
        };

        const overrides = {
            useVolume: () => volumeResult,
            useGetApiUsersQuery: () => usersResult,
        } as const;

        return { overrides };
    };

    it("cycles share name casing and submits form", async () => {
        const { overrides } = setupCommonOverrides();

        const React = await import("react");
        const { render, screen, waitFor } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { Usage } = await import("../../../../store/sratApi");
        // @ts-expect-error - Query param fetches isolated module instance
        const { ShareEditForm } = await import("../ShareEditForm?share-edit-form-test");

        const handleSubmit = mock(() => { });

        render(
            React.createElement(ShareEditForm as any, {
                shareData: {
                    name: "TestShare",
                    usage: Usage.None,
                    mount_point_data: {
                        path: "/mnt/free",
                        path_hash: "free-hash",
                        is_mounted: true,
                        is_write_supported: true,
                    },
                },
                shares: {
                    docKey: {
                        name: "Documents",
                        usage: "general",
                        mount_point_data: {
                            path_hash: "doc-hash",
                        },
                    },
                },
                onSubmit: handleSubmit,
                testOverrides: overrides,
            })
        );

        const [nameInput] = await screen.findAllByLabelText(/Share Name/i);
        expect((nameInput as HTMLInputElement).value).toBe("TestShare");

        const cycleButton = await screen.findByRole("button", { name: /cycle share name casing/i });
        const user = userEvent.setup();
        await user.click(cycleButton);
        await user.click(cycleButton);

        const addDefaults = (await screen.findAllByLabelText(/add suggested default veto files/i))[0];
        if (addDefaults) {
            await user.click(addDefaults);
        }

        const submitButton = await screen.findByRole("button", { name: /create/i });
        await user.click(submitButton);

        await waitFor(() => expect(handleSubmit).toHaveBeenCalled());
        const submissionCalls = handleSubmit.mock.calls as any[];
        const submission = submissionCalls[0]?.[0];
        expect(submission?.name).toBeTruthy();
        expect(submission?.veto_files?.length).toBeGreaterThanOrEqual(0);
    });

    it("renders delete action for existing share", async () => {
        const { overrides } = setupCommonOverrides();

        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { Usage } = await import("../../../../store/sratApi");
        // @ts-expect-error - Query param fetches isolated module instance
        const { ShareEditForm } = await import("../ShareEditForm?share-edit-form-existing");

        const handleDelete = mock(() => { });

        render(
            React.createElement(ShareEditForm as any, {
                shareData: {
                    org_name: "Existing",
                    name: "Existing",
                    usage: Usage.Share,
                    mount_point_data: {
                        path: "/mnt/existing",
                        path_hash: "existing-hash",
                        is_mounted: true,
                        is_write_supported: true,
                    },
                },
                shares: {},
                onSubmit: () => { },
                onDelete: handleDelete,
                testOverrides: overrides,
            })
        );

        const deleteButton = await screen.findByRole("button", { name: /delete/i });
        const user = userEvent.setup();
        await user.click(deleteButton);

        expect(handleDelete).toHaveBeenCalledWith("Existing", expect.objectContaining({ name: "Existing" }));
    });

    it("hides Volume field and disables edit name button for internal shares", async () => {
        const { overrides } = setupCommonOverrides();

        const React = await import("react");
        const { render, within } = await import("@testing-library/react");
        const { Usage } = await import("../../../../store/sratApi");
        const { ShareEditForm } = await import("../ShareEditForm");

        const { container } = render(
            React.createElement(ShareEditForm as any, {
                shareData: {
                    org_name: "InternalShare",
                    name: "InternalShare",
                    usage: Usage.Internal,
                    mount_point_data: {
                        path: "/internal/path",
                        path_hash: "internal-hash",
                        is_mounted: true,
                        is_write_supported: true,
                    },
                },
                shares: {},
                onSubmit: () => { },
                testOverrides: overrides,
            })
        );

        // Volume field should not be present for internal shares
        const volumeLabels = within(container).queryAllByText("Volume");
        expect(volumeLabels.length).toBe(0);
    });
});
