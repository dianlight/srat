import "../../../../test/setup";
import { describe, it, expect, afterEach } from "bun:test";
import { act } from "@testing-library/react";

describe("ShareDetailsPanel", () => {
    afterEach(async () => {
        const { cleanup } = await import("@testing-library/react");
        cleanup();
    });
    const buildShare = async () => {
        const { Time_machine_support, Usage } = await import("../../../store/sratApi");
        return {
            name: "Documents",
            usage: Usage.Backup ?? "backup",
            timemachine: true,
            timemachine_max_size: "250GB",
            recycle_bin_enabled: true,
            guest_ok: true,
            disabled: false,
            users: [{ username: "admin", is_admin: true }],
            ro_users: [{ username: "guest", is_admin: false }],
            veto_files: ["Thumbs.db"],
            mount_point_data: {
                disk_label: "DATA",
                disk_size: 1024 * 1024 * 1024,
                path: "/mnt/data",
                device_id: "sda1",
                fstype: "ext4",
                is_write_supported: true,
                is_mounted: true,
                warnings: "Check usage",
                time_machine_support: Time_machine_support.Supported ?? "Supported",
            },
        } as any;
    };

    it("renders share information and triggers toggle actions", async () => {
        const React = await import("react");
        const { render, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        // @ts-expect-error - Query param ensures fresh module instance for mocks
        const { ShareDetailsPanel } = await import("../components/ShareDetailsPanel?share-details-test");

        const share = await buildShare();

        let editClicks = 0;
        const onEditClick = () => { editClicks += 1; };

        const { container } = await act(async () => {
            return render(
                React.createElement(ShareDetailsPanel as any, {
                    share,
                    shareKey: "documents",
                    onEditClick,
                })
            );
        });

        expect(await within(container).findByText("Documents")).toBeTruthy();
        expect(within(container).getByText(/Mount Point Information/)).toBeTruthy();

        const toggle = within(container).getByLabelText(/show more/i);
        const user = userEvent.setup();
        await user.click(toggle as any);

        expect(await within(container).findByText("/mnt/data")).toBeTruthy();

        const editIcons = within(container).getAllByTestId("EditIcon");
        const firstEditIcon = editIcons[0];
        if (firstEditIcon) {
            const primaryEditButton = firstEditIcon.closest("button");
            if (primaryEditButton) {
                await user.click(primaryEditButton as any);
                expect(editClicks).toBe(1);
            }
        }
    });

    it("renders embedded form when editing", async () => {
        const React = await import("react");
        const { render, within } = await import("@testing-library/react");
        // @ts-expect-error - Query param ensures fresh module instance for mocks
        const { ShareDetailsPanel } = await import("../components/ShareDetailsPanel?share-details-test");
        const share = await buildShare();

        const { container } = await act(async () => {
            return render(
                React.createElement(ShareDetailsPanel as any, {
                    share,
                    shareKey: "documents",
                    isEditing: true,
                    onCancelEdit: () => { },
                    children: React.createElement("div", { role: "form" }, "embedded form"),
                })
            );
        });

        expect(await within(container).findByRole("form")).toBeTruthy();
    });

    it("opens PreviewDialog when StorageIcon is clicked", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        // @ts-expect-error - Query param ensures fresh module instance for mocks
        const { ShareDetailsPanel } = await import("../components/ShareDetailsPanel?share-details-preview-test");

        const share = await buildShare();

        await act(async () => {
            return render(
                React.createElement(ShareDetailsPanel as any, {
                    share,
                    shareKey: "documents",
                })
            );
        });

        // Find the StorageIcon button using getByRole with aria-label
        const storageIconButton = screen.getByRole('button', { name: /view mount point details/i });
        expect(storageIconButton).toBeTruthy();

        // Click on the StorageIcon to open PreviewDialog
        await act(async () => {
            const user = userEvent.setup();
            await user.click(storageIconButton as any);
        });

        // PreviewDialog is rendered in a portal, so we need to search document
        const { screen: globalScreen } = await import("@testing-library/react");
        expect(await globalScreen.findByText(/Mount Point: \/mnt\/data/)).toBeTruthy();

        // Find and click the Close button in the dialog
        const closeButton = globalScreen.getByRole("button", { name: /close/i });
        await act(async () => {
            const user = userEvent.setup();
            await user.click(closeButton as any);
        });
    });

    it("shows disabled visual effect when share is disabled", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        // @ts-expect-error - Query param ensures fresh module instance for mocks
        const { ShareDetailsPanel } = await import("../components/ShareDetailsPanel?share-details-disabled-test");

        const share = await buildShare();
        share.disabled = true;

        await act(async () => {
            return render(
                React.createElement(ShareDetailsPanel as any, {
                    share,
                    shareKey: "documents",
                })
            );
        });

        // Check that a Disabled badge is visible
        const disabledChip = await screen.findByText(/Disabled/i);
        expect(disabledChip).toBeTruthy();

        // Check that Mount Point Information is visible using semantic query
        const mountPointInfo = screen.getByText("Mount Point Information");
        expect(mountPointInfo).toBeTruthy();
    });
});
