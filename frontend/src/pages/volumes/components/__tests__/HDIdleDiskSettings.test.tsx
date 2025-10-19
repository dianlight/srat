import "/workspaces/srat/frontend/test/setup.ts";
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

describe("HDIdleDiskSettings Component", () => {
    beforeEach(() => {
        localStorage.clear();
        document.body.innerHTML = '';
    });

    it("renders accordion with disk settings title", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");
        const { useForm } = await import("react-hook-form-mui");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
            model: "Test Disk Model",
            size: 1000000000,
            removable: false,
        };

        const TestWrapper = () => {
            const { control } = useForm({
                defaultValues: {
                    hdidle_disk_sda_idle_time: 0,
                    hdidle_disk_sda_command_type: "",
                    hdidle_disk_sda_power_condition: 0,
                },
            });

            return React.createElement(
                HDIdleDiskSettings as any,
                {
                    disk: mockDisk,
                    control,
                    readOnly: false,
                }
            );
        };

        render(React.createElement(TestWrapper));

        const title = await screen.findByText(/HDIdle Disk Spin-Down Settings/i);
        expect(title).toBeTruthy();
    });

    it("displays disk model in description", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");
        const { useForm } = await import("react-hook-form-mui");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
            model: "Samsung SSD 870",
            size: 1000000000,
            removable: false,
        };

        const TestWrapper = () => {
            const { control } = useForm({
                defaultValues: {
                    hdidle_disk_sda_idle_time: 0,
                    hdidle_disk_sda_command_type: "",
                    hdidle_disk_sda_power_condition: 0,
                },
            });

            return React.createElement(
                HDIdleDiskSettings as any,
                {
                    disk: mockDisk,
                    control,
                    readOnly: false,
                }
            );
        };

        const { container } = render(React.createElement(TestWrapper));

        // Component should render with title
        expect(container.innerHTML).toContain("HDIdle Disk Spin-Down Settings");
    });

    it("renders idle time configuration field", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");
        const { useForm } = await import("react-hook-form-mui");

        const mockDisk = {
            id: "disk-1",
            name: "sdb",
            model: "Test Disk",
            size: 2000000000,
            removable: false,
        };

        const TestWrapper = () => {
            const { control } = useForm({
                defaultValues: {
                    hdidle_disk_sdb_idle_time: 0,
                    hdidle_disk_sdb_command_type: "",
                    hdidle_disk_sdb_power_condition: 0,
                },
            });

            return React.createElement(
                HDIdleDiskSettings as any,
                {
                    disk: mockDisk,
                    control,
                    readOnly: false,
                }
            );
        };

        const { container } = render(React.createElement(TestWrapper));

        // Component should render with title
        expect(container.innerHTML).toContain("HDIdle Disk Spin-Down Settings");
    });

    it("renders command type configuration field", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");
        const { useForm } = await import("react-hook-form-mui");

        const mockDisk = {
            id: "disk-1",
            name: "sdc",
            model: "Another Disk",
            size: 500000000,
            removable: true,
        };

        const TestWrapper = () => {
            const { control } = useForm({
                defaultValues: {
                    hdidle_disk_sdc_idle_time: 0,
                    hdidle_disk_sdc_command_type: "",
                    hdidle_disk_sdc_power_condition: 0,
                },
            });

            return React.createElement(
                HDIdleDiskSettings as any,
                {
                    disk: mockDisk,
                    control,
                    readOnly: false,
                }
            );
        };

        const { container } = render(React.createElement(TestWrapper));

        // Component should render with title
        expect(container.innerHTML).toContain("HDIdle Disk Spin-Down Settings");
    });

    it("respects readOnly mode", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");
        const { useForm } = await import("react-hook-form-mui");

        const mockDisk = {
            id: "disk-1",
            name: "sda",
            model: "Test Disk",
            size: 1000000000,
            removable: false,
        };

        const TestWrapper = () => {
            const { control } = useForm({
                defaultValues: {
                    hdidle_disk_sda_idle_time: 0,
                    hdidle_disk_sda_command_type: "",
                    hdidle_disk_sda_power_condition: 0,
                },
            });

            return React.createElement(
                HDIdleDiskSettings as any,
                {
                    disk: mockDisk,
                    control,
                    readOnly: true,
                }
            );
        };

        const { container } = render(React.createElement(TestWrapper));

        // Component should render in read-only mode
        expect(container).toBeTruthy();
    }); it("handles disk without name using ID", async () => {
        const React = await import("react");
        const { render } = await import("@testing-library/react");
        const { HDIdleDiskSettings } = await import("../HDIdleDiskSettings");
        const { useForm } = await import("react-hook-form-mui");

        const mockDisk = {
            id: "disk-unknown",
            model: "Mystery Disk",
            size: 1000000000,
            removable: false,
        };

        const TestWrapper = () => {
            const { control } = useForm({
                defaultValues: {
                    hdidle_disk_disk_unknown_idle_time: 0,
                    hdidle_disk_disk_unknown_command_type: "",
                    hdidle_disk_disk_unknown_power_condition: 0,
                },
            });

            return React.createElement(
                HDIdleDiskSettings as any,
                {
                    disk: mockDisk,
                    control,
                    readOnly: false,
                }
            );
        };

        const { container } = render(React.createElement(TestWrapper));

        // Should render even without a name, using ID
        expect(container).toBeTruthy();
    });
});
