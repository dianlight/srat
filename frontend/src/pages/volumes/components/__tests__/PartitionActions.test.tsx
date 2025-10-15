import "../../../../../test/setup";
import { describe, it, expect } from "bun:test";

describe("PartitionActions component", () => {
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
    const openMenuIfNeeded = async (screen: any, fireEvent: any) => {
        try {
            const menuButton = await screen.findByLabelText(/more actions/i);
            fireEvent.click(menuButton);
            // Wait a bit for menu to open
            await new Promise(resolve => setTimeout(resolve, 50));
        } catch {
            // Menu button not found, likely in desktop mode
        }
    };

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
        const { render, screen, fireEvent } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: false,
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
            })
        );

        await openMenuIfNeeded(screen, fireEvent);

        // Check for either button label or menu item text
        const mountAction = screen.queryByLabelText(/mount partition/i) || await screen.findByText(/mount partition/i);
        expect(mountAction).toBeTruthy();
    });

    it("renders unmount action for mounted partition", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
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
            })
        );

        await openMenuIfNeeded(screen, fireEvent);

        // Check for unmount action (either button or menu item)
        const unmountAction = screen.queryAllByLabelText(/unmount partition/i);
        const unmountText = screen.queryAllByText(/unmount partition/i);
        expect(unmountAction.length > 0 || unmountText.length > 0).toBe(true);
    });

    it("renders force unmount action for mounted partition", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
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
            })
        );

        await openMenuIfNeeded(screen, fireEvent);

        const forceUnmountAction = screen.queryByLabelText(/force unmount partition/i) || await screen.findByText(/force unmount partition/i);
        expect(forceUnmountAction).toBeTruthy();
    });

    it("renders enable automount action when not enabled", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
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

        render(
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

        await openMenuIfNeeded(screen, fireEvent);

        const enableButton = screen.queryByLabelText(/enable mount at startup/i) || await screen.findByText(/enable mount at startup/i);
        expect(enableButton).toBeTruthy();
    });

    it("renders disable automount action when enabled", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
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

        render(
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

        await openMenuIfNeeded(screen, fireEvent);

        const disableButton = screen.queryByLabelText(/disable mount at startup/i) || await screen.findByText(/disable mount at startup/i);
        expect(disableButton).toBeTruthy();
    });

    it("renders create share action for mounted partition under /mnt/", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                    shares: [],
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
            })
        );

        await openMenuIfNeeded(screen, fireEvent);

        const createShareButton = screen.queryByLabelText(/create share/i) || await screen.findByText(/create share/i);
        expect(createShareButton).toBeTruthy();
    });

    it("renders go to share action for partition with shares", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                    shares: [{ name: "TestShare", disabled: false }],
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
            })
        );

        await openMenuIfNeeded(screen, fireEvent);

        const goToShareButton = screen.queryByLabelText(/go to share/i) || await screen.findByText(/go to share/i);
        expect(goToShareButton).toBeTruthy();
    });

    it("calls onMount when mount button is clicked", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
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

        render(
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

        await openMenuIfNeeded(screen, fireEvent);

        // Find the mount action (button or menu item)
        const mountAction = screen.queryByLabelText(/mount partition/i);
        if (mountAction) {
            fireEvent.click(mountAction);
        } else {
            const menuItem = await screen.findByText(/mount partition/i);
            fireEvent.click(menuItem);
        }

        expect(mountCalled).toBe(true);
    });

    it("calls onUnmount when unmount button is clicked", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
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

        render(
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

        await openMenuIfNeeded(screen, fireEvent);

        // Find the unmount action
        const unmountButtons = screen.queryAllByLabelText(/unmount partition/i);
        const firstUnmountButton = unmountButtons[0];
        if (unmountButtons.length > 0 && firstUnmountButton) {
            fireEvent.click(firstUnmountButton);
        } else {
            const menuItems = await screen.findAllByText(/unmount partition/i);
            const firstMenuItem = menuItems[0];
            if (firstMenuItem) {
                fireEvent.click(firstMenuItem);
            }
        }

        expect(unmountCalled).toBe(true);
    });

    it("calls onToggleAutomount when automount button is clicked", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
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

        render(
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

        await openMenuIfNeeded(screen, fireEvent);

        const toggleButton = screen.queryByLabelText(/enable mount at startup/i);
        if (toggleButton) {
            fireEvent.click(toggleButton);
        } else {
            const menuItem = await screen.findByText(/enable mount at startup/i);
            fireEvent.click(menuItem);
        }

        expect(toggleCalled).toBe(true);
    });

    it("calls onCreateShare when create share button is clicked", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                    shares: [],
                },
            ],
        });

        let createShareCalled = false;
        const onCreateShare = () => {
            createShareCalled = true;
        };

        render(
            React.createElement(PartitionActions as any, {
                partition,
                protected_mode: false,
                onToggleAutomount: () => { },
                onMount: () => { },
                onUnmount: () => { },
                onCreateShare,
                onGoToShare: () => { },
            })
        );

        await openMenuIfNeeded(screen, fireEvent);

        const createShareButton = screen.queryByLabelText(/create share/i);
        if (createShareButton) {
            fireEvent.click(createShareButton);
        } else {
            const menuItem = await screen.findByText(/create share/i);
            fireEvent.click(menuItem);
        }

        expect(createShareCalled).toBe(true);
    });

    it("calls onGoToShare when go to share button is clicked", async () => {
        const React = await import("react");
        const { render, screen, fireEvent } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
                    shares: [{ name: "TestShare", disabled: false }],
                },
            ],
        });

        let goToShareCalled = false;
        const onGoToShare = () => {
            goToShareCalled = true;
        };

        render(
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

        await openMenuIfNeeded(screen, fireEvent);

        const goToShareButton = screen.queryByLabelText(/go to share/i);
        if (goToShareButton) {
            fireEvent.click(goToShareButton);
        } else {
            const menuItem = await screen.findByText(/go to share/i);
            fireEvent.click(menuItem);
        }

        expect(goToShareCalled).toBe(true);
    });
});
