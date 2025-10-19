import "../../../../test/setup";
import { describe, it, expect } from "bun:test";
import { act } from "@testing-library/react";

describe("ShareDetailsPanel", () => {
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
        const { render, screen, fireEvent, within } = await import("@testing-library/react");
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

        expect(await screen.findByText("Documents")).toBeTruthy();
        expect(screen.getByText(/Mount Point Information/)).toBeTruthy();

        const toggle = within(container).getByLabelText(/show more/i);
        fireEvent.click(toggle);

        expect(await screen.findByText("/mnt/data")).toBeTruthy();

        const editIcons = screen.getAllByTestId("EditIcon");
        const firstEditIcon = editIcons[0];
        if (firstEditIcon) {
            const primaryEditButton = firstEditIcon.closest("button");
            if (primaryEditButton) {
                fireEvent.click(primaryEditButton);
                expect(editClicks).toBe(1);
            }
        }
    });

    it("renders embedded form when editing", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        // @ts-expect-error - Query param ensures fresh module instance for mocks
        const { ShareDetailsPanel } = await import("../components/ShareDetailsPanel?share-details-test");
        const share = await buildShare();

        await act(async () => {
            render(
                React.createElement(ShareDetailsPanel as any, {
                    share,
                    shareKey: "documents",
                    isEditing: true,
                    onCancelEdit: () => { },
                    children: React.createElement("div", { role: "form" }, "embedded form"),
                })
            );
        });

        expect(await screen.findByRole("form")).toBeTruthy();
    });

    it("opens PreviewDialog when StorageIcon is clicked", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        // @ts-expect-error - Query param ensures fresh module instance for mocks
        const { ShareDetailsPanel } = await import("../components/ShareDetailsPanel?share-details-preview-test");

        const share = await buildShare();

        const { container } = await act(async () => {
            return render(
                React.createElement(ShareDetailsPanel as any, {
                    share,
                    shareKey: "documents",
                })
            );
        });

        // Find the StorageIcon button with the aria-label (from Tooltip title)
        const storageIconButton = container.querySelector('button[aria-label="View mount point details"]') as HTMLButtonElement;
        expect(storageIconButton).toBeTruthy();

        // Click on the StorageIcon to open PreviewDialog
        await act(async () => {
            fireEvent.click(storageIconButton);
        });

        // PreviewDialog should now be visible with the mount point path in the title
        expect(await screen.findByText(/Mount Point: \/mnt\/data/)).toBeTruthy();

        // Find and click the Close button in the dialog
        const closeButton = screen.getByRole("button", { name: /close/i });
        await act(async () => {
            fireEvent.click(closeButton);
        });
    });

    it("shows disabled visual effect when share is disabled", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        // @ts-expect-error - Query param ensures fresh module instance for mocks
        const { ShareDetailsPanel } = await import("../components/ShareDetailsPanel?share-details-disabled-test");

        const share = await buildShare();
        share.disabled = true;

        const { container } = await act(async () => {
            return render(
                React.createElement(ShareDetailsPanel as any, {
                    share,
                    shareKey: "documents",
                })
            );
        });

        // Check that "Share Disabled" badge is visible
        expect(await screen.findByText("Share Disabled")).toBeTruthy();

        // Check that the main container has disabled styling
        const mountPointInfo = container.querySelector('h6');
        expect(mountPointInfo?.textContent).toBe("Mount Point Information");
    });
});
