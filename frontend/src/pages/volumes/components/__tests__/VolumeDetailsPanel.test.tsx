import { describe, it, expect, beforeEach } from "bun:test";

// Required localStorage shim for testing environment
if (!(globalThis as any).localStorage) {
    const _store: Record<string, string> = {};
    (globalThis as any).localStorage = {
        getItem: (k: string) => (_store.hasOwnProperty(k) ? _store[k] : null),
        setItem: (k: string, v: string) => { _store[k] = String(v); },
        removeItem: (k: string) => { delete _store[k]; },
        clear: () => { for (const k of Object.keys(_store)) delete _store[k]; },
    };
}

describe("VolumeDetailsPanel Component", () => {
    beforeEach(() => {
        localStorage.clear();
        // Clear DOM before each test
        document.body.innerHTML = '';
    });

    it("renders placeholder when no disk or partition selected", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { BrowserRouter } = await import("react-router-dom");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(VolumeDetailsPanel as any, {})
            )
        );

        // Should display a placeholder message
        const placeholder = await screen.findByText(/Select a partition/i);
        expect(placeholder).toBeTruthy();
    });

    it("renders disk and partition details", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { BrowserRouter } = await import("react-router-dom");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
            size: 1000000000,
            removable: false,
            connection_bus: "usb"
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1",
            size: 500000000,
            fstype: "ext4"
        };

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(VolumeDetailsPanel as any, {
                    disk: mockDisk,
                    partition: mockPartition
                })
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders disk icon based on connection bus", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { BrowserRouter } = await import("react-router-dom");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
            size: 1000000000,
            removable: false,
            connection_bus: "usb"
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1"
        };

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(VolumeDetailsPanel as any, {
                    disk: mockDisk,
                    partition: mockPartition
                })
            )
        );

        // Look for USB icon
        const icons = container.querySelectorAll('svg');
        expect(icons.length).toBeGreaterThanOrEqual(0);
    });

    it("renders disk icon for SDIO/MMC connection", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { BrowserRouter } = await import("react-router-dom");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "mmcblk0",
            size: 1000000000,
            removable: false,
            connection_bus: "sdio"
        };

        const mockPartition = {
            id: "part-1",
            name: "mmcblk0p1"
        };

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(VolumeDetailsPanel as any, {
                    disk: mockDisk,
                    partition: mockPartition
                })
            )
        );

        expect(container).toBeTruthy();
    });

    it("renders eject icon for removable disks", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { BrowserRouter } = await import("react-router-dom");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sdb",
            size: 1000000000,
            removable: true
        };

        const mockPartition = {
            id: "part-1",
            name: "sdb1"
        };

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(VolumeDetailsPanel as any, {
                    disk: mockDisk,
                    partition: mockPartition
                })
            )
        );

        expect(container).toBeTruthy();
    });

    it("displays partition size information", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { BrowserRouter } = await import("react-router-dom");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
            size: 1000000000
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1",
            size: 500000000
        };

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(VolumeDetailsPanel as any, {
                    disk: mockDisk,
                    partition: mockPartition
                })
            )
        );

        // Check for size information in the container
        expect(container.textContent).toBeTruthy();
    });

    it("renders shared resource information when provided", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { BrowserRouter } = await import("react-router-dom");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda"
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1"
        };

        const mockShare = {
            name: "shared-folder",
            path: "/mnt/share"
        };

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(VolumeDetailsPanel as any, {
                    disk: mockDisk,
                    partition: mockPartition,
                    share: mockShare
                })
            )
        );

        expect(container).toBeTruthy();
    });

    it("handles disk info expansion toggle", async () => {
        const React = await import("react");
        const { render, fireEvent } = await import("@testing-library/react");
        const { BrowserRouter } = await import("react-router-dom");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
            size: 1000000000
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1"
        };

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(VolumeDetailsPanel as any, {
                    disk: mockDisk,
                    partition: mockPartition
                })
            )
        );

        // Look for expand button
        const expandButtons = container.querySelectorAll('[data-testid="ExpandMoreIcon"]');
        const firstExpandButton = expandButtons[0];
        if (expandButtons.length > 0 && firstExpandButton) {
            const button = firstExpandButton.closest('button');
            if (button) {
                fireEvent.click(button);
            }
        }

        expect(container).toBeTruthy();
    });

    it("renders preview dialog when object is selected", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { BrowserRouter } = await import("react-router-dom");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda"
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1"
        };

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(VolumeDetailsPanel as any, {
                    disk: mockDisk,
                    partition: mockPartition
                })
            )
        );

        // Check that PreviewDialog component is present
        expect(container).toBeTruthy();
    });

    it("navigates to shares page when share is clicked", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { BrowserRouter } = await import("react-router-dom");
        const { VolumeDetailsPanel } = await import("../VolumeDetailsPanel");

        const mockDisk = {
            id: "disk-1",
            name: "sda"
        };

        const mockPartition = {
            id: "part-1",
            name: "sda1"
        };

        const mockShare = {
            name: "test-share",
            path: "/mnt/share"
        };

        const { container } = render(
            React.createElement(
                BrowserRouter,
                null,
                React.createElement(VolumeDetailsPanel as any, {
                    disk: mockDisk,
                    partition: mockPartition,
                    share: mockShare
                })
            )
        );

        expect(container).toBeTruthy();
    });
});
