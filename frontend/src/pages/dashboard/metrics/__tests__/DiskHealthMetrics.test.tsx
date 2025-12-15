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
});
