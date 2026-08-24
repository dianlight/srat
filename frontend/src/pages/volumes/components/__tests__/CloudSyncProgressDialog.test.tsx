import { describe, expect, it } from "vitest";
import userEvent from "@testing-library/user-event";
import { getMswServer, renderWithTestStore } from "/test/testing";
import { http, HttpResponse } from "msw";

describe("CloudSyncProgressDialog", () => {
	it("renders progress, notes as terminal lines and abort while running", async () => {
		const React = await import("react");
		const { screen } = await import("@testing-library/react");
		const { CloudSyncProgressDialog } = await import(
			"../rclone/CloudSyncProgressDialog"
		);
		const user = userEvent.setup();

		let aborted = false;
		getMswServer().use(
			http.post(/.*\/api\/rclone\/link\/.*\/abort$/, () => {
				aborted = true;
				return HttpResponse.json({});
			}),
		);

		await renderWithTestStore(
			React.createElement(CloudSyncProgressDialog as any, {
				open: true,
				onClose: () => {},
				direction: "push",
				targetKind: "volume",
				targetId: "/mnt/usb",
				targetLabel: "USB Disk",
				taskOverride: {
					operation: "sync",
					status: "running",
					progress: 62,
					direction: "push",
					target_kind: "volume",
					target_id: "/mnt/usb",
					message: "Transferring files",
					notes: ["INFO copied photos/2026/img_0042.jpg"],
				},
			}),
		);

		expect(await screen.findByRole("progressbar")).toBeTruthy();
		expect(await screen.findByText("RUNNING")).toBeTruthy();
		expect(await screen.findByText("62%")).toBeTruthy();
		expect(
			await screen.findByText("INFO copied photos/2026/img_0042.jpg"),
		).toBeTruthy();
		expect(screen.getByRole("button", { name: /Abort/i })).toBeTruthy();

		await user.click(screen.getByRole("button", { name: /Abort/i }));
		expect(aborted).toBe(true);
	});

	it("shows the failure alert when the task failed", async () => {
		const React = await import("react");
		const { screen } = await import("@testing-library/react");
		const { CloudSyncProgressDialog } = await import(
			"../rclone/CloudSyncProgressDialog"
		);

		await renderWithTestStore(
			React.createElement(CloudSyncProgressDialog as any, {
				open: true,
				onClose: () => {},
				direction: "push",
				targetKind: "volume",
				targetId: "/mnt/usb",
				targetLabel: "USB Disk",
				taskOverride: {
					operation: "sync",
					status: "failure",
					progress: 10,
					direction: "push",
					target_kind: "volume",
					target_id: "/mnt/usb",
					error: "network unreachable",
				},
			}),
		);

		expect(
			await screen.findByText(/network unreachable/),
		).toBeTruthy();
		expect(screen.queryByRole("button", { name: /Abort/i })).toBeNull();
	});
});
