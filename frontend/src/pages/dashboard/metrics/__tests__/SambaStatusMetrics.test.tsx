import { describe, expect, it } from "bun:test";
import '../../../../../test/setup';
import type { SambaStatus } from "../../../../store/sratApi";

describe("SambaStatusMetrics", () => {
    it("renders tcons as sub-rows of matching sessions by session_id", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const userEvent = (await import("@testing-library/user-event")).default;
        const { SambaStatusMetrics } = await import("../SambaStatusMetrics");

        const sambaStatus: SambaStatus = {
            smb_conf: "",
            timestamp: "2026-03-13T00:00:00Z",
            version: "4.x",
            sessions: {
                sessionOne: {
                    auth_time: "2026-03-13T00:00:00Z",
                    channels: {},
                    creation_time: "2026-03-13T00:00:00Z",
                    encryption: { cipher: "AES", degree: "128" },
                    gid: 0,
                    groupname: "root",
                    hostname: "host-a",
                    remote_machine: "remote-a",
                    server_id: { pid: "1", task_id: "1", unique_id: "u1", vnn: "0" },
                    session_dialect: "SMB3",
                    session_id: "S-1",
                    signing: { cipher: "HMAC", degree: "256" },
                    uid: 0,
                    username: "alice",
                },
            },
            tcons: {
                tconOne: {
                    connected_at: "2026-03-13T00:00:00Z",
                    device: "disk1",
                    encryption: { cipher: "AES", degree: "128" },
                    machine: "remote-a",
                    server_id: { pid: "1", task_id: "1", unique_id: "u1", vnn: "0" },
                    service: "public",
                    session_id: "S-1",
                    share: "public",
                    signing: { cipher: "HMAC", degree: "256" },
                    tcon_id: "10",
                },
                orphanTcon: {
                    connected_at: "2026-03-13T00:00:00Z",
                    device: "disk2",
                    encryption: { cipher: "AES", degree: "128" },
                    machine: "remote-b",
                    server_id: { pid: "2", task_id: "2", unique_id: "u2", vnn: "0" },
                    service: "private",
                    session_id: "S-999",
                    share: "private",
                    signing: { cipher: "HMAC", degree: "256" },
                    tcon_id: "99",
                },
            },
        };

        render(React.createElement(SambaStatusMetrics, { sambaStatus }));
        const user = userEvent.setup();

        expect(await screen.findByText("Samba Sessions")).toBeTruthy();
        expect(screen.getByText("S-1")).toBeTruthy();

        await user.click(screen.getByRole("button", { name: /expand session s-1/i }));

        expect(screen.getByRole("table", { name: /samba tcons subtable s-1/i })).toBeTruthy();
        expect(screen.getByText("10")).toBeTruthy();
        expect(screen.queryByText("99")).toBeNull();
    });

    it("shows unavailable message when samba status is missing", async () => {
        const React = await import("react");
        const { render, screen } = await import("@testing-library/react");
        const { SambaStatusMetrics } = await import("../SambaStatusMetrics");

        render(React.createElement(SambaStatusMetrics, { sambaStatus: undefined }));

        expect(await screen.findByText("Samba status data not available.")).toBeTruthy();
    });
});
