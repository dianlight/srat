import "../../../../../test/setup";
import { describe, it, expect, beforeEach } from "bun:test";

describe("PartitionActions component", () => {
    const createMatchMedia = (matches: boolean) => () => ({
        matches,
        addListener: () => { },
        removeListener: () => { },
        addEventListener: () => { },
        removeEventListener: () => { },
        dispatchEvent: () => false,
        onchange: null,
        media: "",
    }) as any;

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
        const { render, screen, fireEvent, within } = await import("@testing-library/react");
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

        await openMenuIfNeeded(screen, fireEvent);

        // Check that there's exactly one mount partition button within the container
        const mountButtons = within(container).getAllByLabelText("mount partition");
        expect(mountButtons).toHaveLength(1);

        // Check that there's exactly one enable mount at startup button within the container
        const enableButtons = within(container).getAllByLabelText("enable mount at startup");
        expect(enableButtons).toHaveLength(1);
    });

    it("renders unmount action for mounted partition", async () => {
        const React = await import("react");
        const { render, screen, fireEvent, within } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
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

        await openMenuIfNeeded(screen, fireEvent);

        // Check for unmount action within the container (menu items render in portal)
        const unmountButtons = within(container).getAllByLabelText("unmount partition");
        expect(unmountButtons).toHaveLength(1);
    });

    it("renders force unmount action for mounted partition", async () => {
        const React = await import("react");
        const { render, screen, fireEvent, within } = await import("@testing-library/react");
        const { PartitionActions } = await import("../PartitionActions");

        const partition = buildPartition({
            mount_point_data: [
                {
                    path: "/mnt/test",
                    is_mounted: true,
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

        await openMenuIfNeeded(screen, fireEvent);

        const forceUnmountButtons = within(container).getAllByLabelText("force unmount partition");
        expect(forceUnmountButtons).toHaveLength(1);
    });

    it("renders enable automount action when not enabled", async () => {
        const React = await import("react");
        const { render, screen, fireEvent, within } = await import("@testing-library/react");
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

        await openMenuIfNeeded(screen, fireEvent);

        const enableButtons = within(container).getAllByLabelText("enable mount at startup");
        expect(enableButtons).toHaveLength(1);
    });

    it("renders disable automount action when enabled", async () => {
        const React = await import("react");
        const { render, screen, fireEvent, within } = await import("@testing-library/react");
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

        await openMenuIfNeeded(screen, fireEvent);

        const disableButtons = within(container).getAllByLabelText("disable mount at startup");
        expect(disableButtons).toHaveLength(1);
    });

    it("renders create share action for mounted partition under /mnt/", async () => {
        const React = await import("react");
        const { render, screen, fireEvent, within } = await import("@testing-library/react");
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

        await openMenuIfNeeded(screen, fireEvent);

        const createShareButtons = within(container).getAllByLabelText("create share");
        expect(createShareButtons).toHaveLength(1);
    });

    it("renders go to share action for partition with shares", async () => {
        const React = await import("react");
        const { render, screen, fireEvent, within } = await import("@testing-library/react");
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

        await openMenuIfNeeded(screen, fireEvent);

        const goToShareButtons = within(container).getAllByLabelText("go to share");
        expect(goToShareButtons).toHaveLength(1);
    });

    it("calls onMount when mount button is clicked", async () => {
        const React = await import("react");
        const { render, screen, fireEvent, within } = await import("@testing-library/react");
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

        await openMenuIfNeeded(screen, fireEvent);

        // Find the mount action within the container
        const mountButtons = within(container).getAllByLabelText("mount partition");
        expect(mountButtons).toHaveLength(1);
        fireEvent.click(mountButtons[0]!);

        expect(mountCalled).toBe(true);
    });

    it("calls onUnmount when unmount button is clicked", async () => {
        const React = await import("react");
        const { render, screen, fireEvent, within } = await import("@testing-library/react");
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

        await openMenuIfNeeded(screen, fireEvent);

        // Find the unmount action within the container
        const unmountButtons = within(container).getAllByLabelText("unmount partition");
        expect(unmountButtons).toHaveLength(1);
        fireEvent.click(unmountButtons[0]!);

        expect(unmountCalled).toBe(true);
    });

    it("calls onToggleAutomount when automount button is clicked", async () => {
        const React = await import("react");
        const { render, screen, fireEvent, within } = await import("@testing-library/react");
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

        await openMenuIfNeeded(screen, fireEvent);

        const toggleButtons = within(container).getAllByLabelText("enable mount at startup");
        expect(toggleButtons).toHaveLength(1);
        fireEvent.click(toggleButtons[0]!);

        expect(toggleCalled).toBe(true);
    });

    it("calls onCreateShare when create share button is clicked", async () => {
        const React = await import("react");
        const { render, screen, fireEvent, within } = await import("@testing-library/react");
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

        const { container } = render(
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

        const createShareButtons = within(container).getAllByLabelText("create share");
        expect(createShareButtons).toHaveLength(1);
        fireEvent.click(createShareButtons[0]!);

        expect(createShareCalled).toBe(true);
    });

    it("calls onGoToShare when go to share button is clicked", async () => {
        const React = await import("react");
        const { render, screen, fireEvent, within } = await import("@testing-library/react");
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

        await openMenuIfNeeded(screen, fireEvent);

        const goToShareButtons = within(container).getAllByLabelText("go to share");
        expect(goToShareButtons).toHaveLength(1);
        fireEvent.click(goToShareButtons[0]!);

        expect(goToShareCalled).toBe(true);
    });
});
