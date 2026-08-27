import { beforeEach, describe, expect, it } from "vitest";

describe("PartitionActions component", () => {
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
        (window as any).matchMedia = createMatchMedia(false); // Force desktop mode
    });
    const buildPartition = (overrides?: Partial<any>) => ({
        name: "sda1",
        mount_point_data: [
            {
                path: "/mnt/test",
                is_mounted: false,
                is_to_mount_at_startup: false,
            },
        ],
        ...overrides,
    });

    // Helper function to open the mobile menu if it exists
    const openMenuIfNeeded = async (screen: any, user: any) => {
        try {
            const menuButton = await screen.findByLabelText(/more actions/i);
            await user.click(menuButton);
            // Wait a bit for menu to open
            await new Promise((resolve) => setTimeout(resolve, 50));
        } catch {
            // Menu button not found, likely in desktop mode
        }
    };

    it("renders compact menu on phone-sized screens (xs)", async () => {
        // Regression: `isSmallScreen` previously used between("sm","lg"),
        // which excluded xs (phones < 600px) and forced the desktop icon
        // row. A phone-sized matchMedia matches max-width queries but NOT
        // min-width:600px, so `down("lg")` is true while `between("sm","lg")`
        // would have been false.
        (window as any).matchMedia = (query: string) => ({
            matches: query.includes("max-width") && !query.includes("min-width"),
            addListener: () => {},
            removeListener: () => {},
            addEventListener: () => {},
            removeEventListener: () => {},
            dispatchEvent: () => false,
            onchange: null,
            media: query,
        });

        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition();
        render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => {},
                onMount: () => {},
                onUnmount: () => {},
                onCreateShare: () => {},
                onGoToShare: () => {},
            })
        );

        // Compact menu on phones: no inline desktop buttons...
        expect(screen.queryByLabelText("mount partition")).toBeNull();
        // ...and the "more actions" menu button is present instead.
        expect(screen.getByRole("button", { name: /more actions/i })).toBeTruthy();
    });

    it("renders action buttons for non-protected partition", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition();
        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        expect(container).toBeTruthy();
    });

    it("does not render Check Filesystem menu item when callback is missing", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { PartitionActions } = await import("../PartitionActions");

        (window as any).matchMedia = createMatchMedia(true); // Force menu mode
        const user = userEvent.setup();
        const partition = buildPartition({
            filesystem_info: {
                support: {
                    canCheck: true,
                },
            },
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: false,
                    is_to_mount_at_startup: false,
                },
            ],
        });

        render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
                // onCheckFilesystem intentionally omitted
            })
        );

        const menuButton = await screen.findByLabelText(/more actions/i);
        await user.click(menuButton);

        expect(screen.queryByText(/check filesystem/i)).toBeNull();
    });

    it("returns null for protected mode partitions", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition();
        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: true,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        // Component should return null for protected partitions
        expect(container.firstChild).toBeNull();
    });

    it("returns null for hassos- partitions", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({ name: "hassos-data" });
        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        // Component should return null for hassos partitions
        expect(container.firstChild).toBeNull();
    });

    it("returns null for partitions with host mount points", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            host_mount_point_data: [{ path: "/host/path" }],
        });
        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        // Component should return null for host-mounted partitions
        expect(container.firstChild).toBeNull();
    });

    it("renders mount action for unmounted partition", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: false,
                },
            ],
        });

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        // Check that there's exactly one mount partition button within the container
        const mountButtons = within(container).getAllByLabelText("mount partition");
        expect(mountButtons).toHaveLength(1);

        // Check that there's exactly one enable automatic mount button within the container
        const enableButtons = within(container).getAllByLabelText("enable automatic mount");
        expect(enableButtons).toHaveLength(1);
    });

    it("renders unmount action for mounted partition without automount", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                    is_to_mount_at_startup: false,
                },
            ],
        });

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        // Check for unmount action within the container (menu items render in portal)
        const unmountButtons = within(container).getAllByLabelText("unmount partition");
        expect(unmountButtons).toHaveLength(1);
    });

    it("renders unmount action when automount is enabled (#971)", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                    is_to_mount_at_startup: true,
                },
            ],
        });

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        // Unmount must remain available even with automount enabled (#971):
        // hiding it left users with no UI path to unmount a partition that
        // was mounted with automount on (the default of the mount dialog).
        const unmountButtons = within(container).getAllByLabelText("unmount partition");
        expect(unmountButtons).toHaveLength(1);
    });

    it("renders unmount actions for mounted partition with enabled share (#971)", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                    is_to_mount_at_startup: true,
                    share: { disabled: false },
                },
            ],
        });

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        // Enabled share must not remove the ability to unmount (#971).
        const unmountButtons = within(container).getAllByLabelText("unmount partition");
        expect(unmountButtons).toHaveLength(1);
        const forceUnmountButtons = within(container).getAllByLabelText("force unmount partition");
        expect(forceUnmountButtons).toHaveLength(1);
        // Share navigation still offered alongside unmount.
        const goToShareButtons = within(container).getAllByLabelText("go to share");
        expect(goToShareButtons).toHaveLength(1);
    });

    it("renders force unmount action for mounted partition without automount", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                    is_to_mount_at_startup: false,
                },
            ],
        });

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        const forceUnmountButtons = within(container).getAllByLabelText("force unmount partition");
        expect(forceUnmountButtons).toHaveLength(1);
    });

    it("renders enable automount action when not enabled", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: false,
                    is_to_mount_at_startup: false,
                },
            ],
        });

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        const enableButtons = within(container).getAllByLabelText("enable automatic mount");
        expect(enableButtons).toHaveLength(1);
    });

    it("renders disable automount action when enabled", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: false,
                    is_to_mount_at_startup: true,
                },
            ],
        });

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        const disableButtons = within(container).getAllByLabelText("disable automatic mount");
        expect(disableButtons).toHaveLength(1);
    });

    it("renders go to share action for partition with enabled share", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                    share: { disabled: false },
                },
            ],
        });

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        const goToShareButtons = within(container).getAllByLabelText("go to share");
        expect(goToShareButtons).toHaveLength(1);
    });

    it("does not render automount toggle when mounted with enabled share", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                    share: { disabled: false },
                },
            ],
        });

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        // Should not find automount toggle when mounted with enabled share
        const enableButtons = within(container).queryAllByLabelText("enable automatic mount");
        const disableButtons = within(container).queryAllByLabelText("disable automatic mount");
        expect(enableButtons).toHaveLength(0);
        expect(disableButtons).toHaveLength(0);
    });

    it("calls onMount when mount button is clicked", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: false,
                },
            ],
        });

        let mountCalled = false;
        const onMount = () => {
            mountCalled = true;
        };

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount,
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        // Find the mount action within the container
        const mountButtons = within(container).getAllByLabelText("mount partition");
        expect(mountButtons).toHaveLength(1);
        await user.click(mountButtons[0]!);
        await user.click(mountButtons[0]!);

        expect(mountCalled).toBe(true);
    });

    it("calls onUnmount when unmount button is clicked", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                },
            ],
        });

        let unmountCalled = false;
        const onUnmount = () => {
            unmountCalled = true;
        };

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount,
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        // Find the unmount action within the container
        const unmountButtons = within(container).getAllByLabelText("unmount partition");
        expect(unmountButtons).toHaveLength(1);
        await user.click(unmountButtons[0]!);

        expect(unmountCalled).toBe(true);
    });

    it("calls onToggleAutomount when automount button is clicked", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: false,
                    is_to_mount_at_startup: false,
                },
            ],
        });

        let toggleCalled = false;
        const onToggleAutomount = () => {
            toggleCalled = true;
        };

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount,
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        const toggleButtons = within(container).getAllByLabelText("enable automatic mount");
        expect(toggleButtons).toHaveLength(1);
        await user.click(toggleButtons[0]!);

        expect(toggleCalled).toBe(true);
    });

    it("renders create share action for mounted partition without share and automount disabled", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                    is_to_mount_at_startup: false,
                    share: null,
                },
            ],
        });

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        const createShareButtons = within(container).getAllByLabelText("create share");
        expect(createShareButtons).toHaveLength(1);
    });

    it("calls onGoToShare when go to share button is clicked", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                    share: { disabled: false },
                },
            ],
        });

        let goToShareCalled = false;
        const onGoToShare = () => {
            goToShareCalled = true;
        };

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare,
            })
        );

        await openMenuIfNeeded(screen, user);

        const goToShareButtons = within(container).getAllByLabelText("go to share");
        expect(goToShareButtons).toHaveLength(1);
        await user.click(goToShareButtons[0]!);

        expect(goToShareCalled).toBe(true);
    });

    it("does not render more than one mountpoint", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test1",
                    is_mounted: false,
                },
                {
                    path: "/mnt/test2",
                    is_mounted: false,
                },
            ],
        });

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        // Component should return null for partitions with multiple mount points
        expect(container.firstChild).toBeNull();
    });

    it("shows automount toggle for mounted partition without enabled share", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                    is_to_mount_at_startup: false,
                    share: { disabled: true }, // disabled share
                },
            ],
        });

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        // Should have automount toggle when mounted with disabled share
        const enableButtons = within(container).getAllByLabelText("enable automatic mount");
        expect(enableButtons).toHaveLength(1);
    });

    it("shows create share when mounted without share and automount disabled", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                    is_to_mount_at_startup: false,
                    share: null, // no share
                },
            ],
        });

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        // Should have create share button when no share exists
        const createShareButtons = within(container).getAllByLabelText("create share");
        expect(createShareButtons).toHaveLength(1);
    });

    it("calls onSetFilesystemLabel when set label button is clicked", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            filesystem_info: {
                support: {
                    canSetLabel: true,
                },
            },
        });

        let labelCalled = false;
        const onSetFilesystemLabel = () => {
            labelCalled = true;
        };

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
                onSetFilesystemLabel,
            })
        );

        await openMenuIfNeeded(screen, user);

        const labelButtons = within(container).getAllByLabelText("set label");
        expect(labelButtons).toHaveLength(1);
        await user.click(labelButtons[0]!);

        expect(labelCalled).toBe(true);
    });

    it("calls onFormatPartition when format button is clicked", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            filesystem_info: {
                support: {
                    canFormat: true,
                },
            },
        });

        let formatCalled = false;
        const onFormatPartition = () => {
            formatCalled = true;
        };

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
                onFormatPartition,
            })
        );

        await openMenuIfNeeded(screen, user);

        const formatButtons = within(container).getAllByLabelText("format partition");
        expect(formatButtons).toHaveLength(1);
        await user.click(formatButtons[0]!);

        expect(formatCalled).toBe(true);
    });

    it("hides set label and format actions when filesystem support is unavailable", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            filesystem_info: {
                support: {
                    canCheck: false,
                    canSetLabel: false,
                    canFormat: false,
                },
            },
        });

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
                onCheckFilesystem: () => { },
                onSetFilesystemLabel: () => { },
                onFormatPartition: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        expect(within(container).queryAllByLabelText("set label")).toHaveLength(0);
        expect(within(container).queryAllByLabelText("format partition")).toHaveLength(0);
    });

    it("hides set label action when the partition is mounted", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const user = userEvent.setup();
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            filesystem_info: {
                support: {
                    canSetLabel: true,
                },
            },
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                    is_to_mount_at_startup: false,
                },
            ],
        });

        const { container } = render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
                onSetFilesystemLabel: () => { },
                onFormatPartition: () => { },
            })
        );

        await openMenuIfNeeded(screen, user);

        expect(within(container).queryAllByLabelText("set label")).toHaveLength(0);
    });

    it("renders all action icons with consistent size classes (coherence)", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        // Unmounted partition with full filesystem support and all callbacks:
        // enable-automount (FA), mount (FA), check-filesystem (MUI),
        // set-label (MUI), format (MUI) - a mix of FontAwesome and MUI icons.
        const partition = buildPartition({
            filesystem_info: {
                support: {
                    canCheck: true,
                    canSetLabel: true,
                    canFormat: true,
                },
            },
        });

        render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare: () => { },
                onGoToShare: () => { },
                onCheckFilesystem: () => { },
                onSetFilesystemLabel: () => { },
                onFormatPartition: () => { },
            })
        );

        // FA icons carry the registered "partition-action-icon" test id; MUI icons
        // expose their own PascalCase auto-testid. All must share the identical
        // class list, i.e. the same effective size.
        const icons = [
            ...screen.getAllByTestId("partition-action-icon"),
            ...screen.getAllByTestId("FactCheckIcon"),
            ...screen.getAllByTestId("LabelIcon"),
            ...screen.getAllByTestId("DeleteSweepIcon"),
        ];
        expect(icons.length).toBeGreaterThanOrEqual(5);

        const iconClasses = new Set(
            icons.map((svg) => svg.getAttribute("class"))
        );
        expect(iconClasses.size).toBe(1);

        // The shared class must match the small size used by MUI icons.
        const sharedClass = iconClasses.values().next().value ?? "";
        expect(sharedClass).toContain("MuiSvgIcon-fontSizeSmall");
    });
});
