import '../../../../../test/setup';
import { describe, expect, it } from "bun:test";
import type { DiskHealth } from "../../../../store/sratApi";

describe("DiskHealthMetrics", () => {
    it("orders disks by device name", async () => {
        const React = await import("react");
        const { render, screen, within } = await import("@testing-library/react");
        const { DiskHealthMetrics } = await import("../DiskHealthMetrics");

        const diskHealth: DiskHealth = {
            global: {
                total_iops: 0,
                total_read_latency_ms: 0,
                total_write_latency_ms: 0,
            },
            hdidle_running: false,
            per_disk_io: [
                {
                    device_description: "Second Disk",
                    device_name: "/dev/sdb",
                    read_iops: 2,
                    read_latency_ms: 1,
                    write_iops: 3,
                    write_latency_ms: 1.5,
                },
                {
                    device_description: "First Disk",
                    device_name: "/dev/sda",
                    read_iops: 5,
                    read_latency_ms: 2,
                    write_iops: 4,
                    write_latency_ms: 2.5,
                },
            ],
            per_partition_info: {},
        };

        render(React.createElement(DiskHealthMetrics, { diskHealth }));

        const tables = await screen.findAllByRole("table", { name: "disk health table" });
        expect(tables.length).toBeGreaterThan(0);

        const table = tables[0];
        if (!table) {
            throw new Error("Table not found");
        }
        const tbody = table.querySelector("tbody");
        expect(tbody).toBeTruthy();

        const rows = within(tbody as HTMLElement).getAllByRole("row");
        const deviceOrder = rows.map((row) => {
            const headers = within(row).getAllByRole("rowheader");
            return headers[1]?.textContent;
        });

        expect(deviceOrder).toEqual(["/dev/sda", "/dev/sdb"]);
    });

    it("shows hdidle spin status column when hdidle is running", async () => {
        const React = await import("react");
        const { render, screen, within, cleanup } = await import("@testing-library/react");
        const { DiskHealthMetrics } = await import("../DiskHealthMetrics");

        const diskHealth: DiskHealth = {
            global: {
                total_iops: 0,
                total_read_latency_ms: 0,
                total_write_latency_ms: 0,
            },
            hdidle_running: true,
            per_disk_info: {
                "Test Disk": {
                    device_id: "test-id",
                    hdidle_status: {
                        enabled: true,
                        spun_down: true,
                        supported: true,
                        spin_down_at: "2025-12-15T10:00:00Z",
                    },
                },
            },
            per_disk_io: [
                {
                    device_description: "Test Disk",
                    device_name: "/dev/sda",
                    read_iops: 5,
                    read_latency_ms: 2,
                    write_iops: 4,
                    write_latency_ms: 2.5,
                },
            ],
            per_partition_info: {},
        };

        render(React.createElement(DiskHealthMetrics, { diskHealth }));

        const tables = await screen.findAllByRole("table", { name: "disk health table" });
        const table = tables[tables.length - 1]!; // Get the most recent table
        const headerRow = within(table).getAllByRole("columnheader");
        const headerTexts = headerRow.map(h => h.textContent);
        
        expect(headerTexts).toContain("Spin Status");

        const tbody = table.querySelector("tbody");
        expect(tbody).toBeTruthy();
        
        const rows = within(tbody as HTMLElement).getAllByRole("row");
        expect(rows.length).toBe(1);

        const cells = within(rows[0]!).getAllByRole("cell");
        // Spin status should show pause icon for spun down disk
        const spinStatusCell = cells.find(cell => cell.textContent === "⏸");
        expect(spinStatusCell).toBeTruthy();
        
        cleanup();
    });

    it("hides hdidle spin status column when hdidle is not running", async () => {
        const React = await import("react");
        const { render, screen, within, cleanup } = await import("@testing-library/react");
        const { DiskHealthMetrics } = await import("../DiskHealthMetrics");

        const diskHealth: DiskHealth = {
            global: {
                total_iops: 0,
                total_read_latency_ms: 0,
                total_write_latency_ms: 0,
            },
            hdidle_running: false,
            per_disk_io: [
                {
                    device_description: "Test Disk",
                    device_name: "/dev/sda",
                    read_iops: 5,
                    read_latency_ms: 2,
                    write_iops: 4,
                    write_latency_ms: 2.5,
                },
            ],
            per_partition_info: {},
        };

        render(React.createElement(DiskHealthMetrics, { diskHealth }));

        const tables = await screen.findAllByRole("table", { name: "disk health table" });
        const table = tables[tables.length - 1]!; // Get the most recent table
        const headerRow = within(table).getAllByRole("columnheader");
        const headerTexts = headerRow.map(h => h.textContent);
        
        expect(headerTexts).not.toContain("Spin Status");
        
        cleanup();
    });

    it("shows active spin status for disks that are not spun down", async () => {
        const React = await import("react");
        const { render, screen, within, cleanup } = await import("@testing-library/react");
        const { DiskHealthMetrics } = await import("../DiskHealthMetrics");

        const diskHealth: DiskHealth = {
            global: {
                total_iops: 0,
                total_read_latency_ms: 0,
                total_write_latency_ms: 0,
            },
            hdidle_running: true,
            per_disk_info: {
                "Active Disk": {
                    device_id: "test-id",
                    hdidle_status: {
                        enabled: true,
                        spun_down: false,
                        supported: true,
                        spin_up_at: "2025-12-15T10:30:00Z",
                    },
                },
            },
            per_disk_io: [
                {
                    device_description: "Active Disk",
                    device_name: "/dev/sda",
                    read_iops: 5,
                    read_latency_ms: 2,
                    write_iops: 4,
                    write_latency_ms: 2.5,
                },
            ],
            per_partition_info: {},
        };

        render(React.createElement(DiskHealthMetrics, { diskHealth }));

        const tables = await screen.findAllByRole("table", { name: "disk health table" });
        const table = tables[tables.length - 1]!; // Get the most recent table
        const tbody = table.querySelector("tbody");
        expect(tbody).toBeTruthy();
        
        const rows = within(tbody as HTMLElement).getAllByRole("row");
        const cells = within(rows[0]!).getAllByRole("cell");
        
        // Active disk should show play icon
        const spinStatusCell = cells.find(cell => cell.textContent === "▶");
        expect(spinStatusCell).toBeTruthy();
        
        cleanup();
    });

    it("shows N/A for disks without hdidle status", async () => {
        const React = await import("react");
        const { render, screen, within, cleanup } = await import("@testing-library/react");
        const { DiskHealthMetrics } = await import("../DiskHealthMetrics");

        const diskHealth: DiskHealth = {
            global: {
                total_iops: 0,
                total_read_latency_ms: 0,
                total_write_latency_ms: 0,
            },
            hdidle_running: true,
            per_disk_info: {
                "Test Disk": {
                    device_id: "test-id",
                    // No hdidle_status
                },
            },
            per_disk_io: [
                {
                    device_description: "Test Disk",
                    device_name: "/dev/sda",
                    read_iops: 5,
                    read_latency_ms: 2,
                    write_iops: 4,
                    write_latency_ms: 2.5,
                },
            ],
            per_partition_info: {},
        };

        render(React.createElement(DiskHealthMetrics, { diskHealth }));

        const tables = await screen.findAllByRole("table", { name: "disk health table" });
        const table = tables[tables.length - 1]!; // Get the most recent table
        const tbody = table.querySelector("tbody");
        expect(tbody).toBeTruthy();
        
        const rows = within(tbody as HTMLElement).getAllByRole("row");
        const cells = within(rows[0]!).getAllByRole("cell");
        
        // Should show N/A - find the third cell (after description and device headers)
        const spinStatusCell = cells[0]; // First cell after row headers
        expect(spinStatusCell?.textContent).toBe("N/A");
        
        cleanup();
    });

    it("shows SMART icon when SMART is supported", async () => {
        const React = await import("react");
        const { render, screen, within, cleanup } = await import("@testing-library/react");
        const { DiskHealthMetrics } = await import("../DiskHealthMetrics");

        const diskHealth: DiskHealth = {
            global: {
                total_iops: 0,
                total_read_latency_ms: 0,
                total_write_latency_ms: 0,
            },
            hdidle_running: false,
            per_disk_info: {
                "Test Disk": {
                    device_id: "test-id",
                    smart_info: {
                        supported: true,
                        disk_type: "HDD" as any,
                        rotation_rate: 7200,
                    },
                },
            },
            per_disk_io: [
                {
                    device_description: "Test Disk",
                    device_name: "/dev/sda",
                    read_iops: 5,
                    read_latency_ms: 2,
                    write_iops: 4,
                    write_latency_ms: 2.5,
                },
            ],
            per_partition_info: {},
        };

        render(React.createElement(DiskHealthMetrics, { diskHealth }));

        const tables = await screen.findAllByRole("table", { name: "disk health table" });
        const table = tables[tables.length - 1]!;
        const tbody = table.querySelector("tbody");
        expect(tbody).toBeTruthy();
        
        const rows = within(tbody as HTMLElement).getAllByRole("row");
        const deviceCell = within(rows[0]!).getAllByRole("rowheader")[1];
        
        // Check that SMART icon is present
        expect(deviceCell?.querySelector('svg')).toBeTruthy();
        
        cleanup();
    });

    it("shows SSD type for SMART devices with 0 RPM", async () => {
        const React = await import("react");
        const { render, screen, within, cleanup } = await import("@testing-library/react");
        const { DiskHealthMetrics } = await import("../DiskHealthMetrics");

        const diskHealth: DiskHealth = {
            global: {
                total_iops: 0,
                total_read_latency_ms: 0,
                total_write_latency_ms: 0,
            },
            hdidle_running: false,
            per_disk_info: {
                "SSD Disk": {
                    device_id: "test-id",
                    smart_info: {
                        supported: true,
                        disk_type: "SSD" as any,
                        rotation_rate: 0,
                    },
                },
            },
            per_disk_io: [
                {
                    device_description: "SSD Disk",
                    device_name: "/dev/nvme0n1",
                    read_iops: 50,
                    read_latency_ms: 0.5,
                    write_iops: 40,
                    write_latency_ms: 0.8,
                },
            ],
            per_partition_info: {},
        };

        render(React.createElement(DiskHealthMetrics, { diskHealth }));

        const tables = await screen.findAllByRole("table", { name: "disk health table" });
        const table = tables[tables.length - 1]!;
        const tbody = table.querySelector("tbody");
        expect(tbody).toBeTruthy();
        
        const rows = within(tbody as HTMLElement).getAllByRole("row");
        const deviceCell = within(rows[0]!).getAllByRole("rowheader")[1];
        
        // Check that SMART icon is present for SSD
        expect(deviceCell?.querySelector('svg')).toBeTruthy();
        
        cleanup();
    });

    it("hides SMART icon when SMART is not supported", async () => {
        const React = await import("react");
        const { render, screen, within, cleanup } = await import("@testing-library/react");
        const { DiskHealthMetrics } = await import("../DiskHealthMetrics");

        const diskHealth: DiskHealth = {
            global: {
                total_iops: 0,
                total_read_latency_ms: 0,
                total_write_latency_ms: 0,
            },
            hdidle_running: false,
            per_disk_info: {
                "Test Disk": {
                    device_id: "test-id",
                    smart_info: {
                        supported: false,
                    },
                },
            },
            per_disk_io: [
                {
                    device_description: "Test Disk",
                    device_name: "/dev/sda",
                    read_iops: 5,
                    read_latency_ms: 2,
                    write_iops: 4,
                    write_latency_ms: 2.5,
                },
            ],
            per_partition_info: {},
        };

        render(React.createElement(DiskHealthMetrics, { diskHealth }));

        const tables = await screen.findAllByRole("table", { name: "disk health table" });
        const table = tables[tables.length - 1]!;
        const tbody = table.querySelector("tbody");
        expect(tbody).toBeTruthy();
        
        const rows = within(tbody as HTMLElement).getAllByRole("row");
        const deviceCell = within(rows[0]!).getAllByRole("rowheader")[1];
        
        // Check that SMART icon is NOT present
        expect(deviceCell?.querySelector('svg')).toBeFalsy();
        
        cleanup();
    });
});
