import { http, HttpResponse } from "msw";
import { beforeEach, describe, expect, it } from "vitest";
import userEvent from "@testing-library/user-event";
import { getMswServer, renderWithTestStore } from "/test/testing";
import { rcloneMockState } from "../../../../mocks/customHandlers";

const linkedLink = {
	target_kind: "volume",
	target_id: "/mnt/usb",
	provider: "dropbox",
	remote_path: "/srat/usb",
	status: "authorized",
	auto_sync: false,
	schedule_minutes: 0,
	last_sync_at: "2026-08-01T10:00:00Z",
	last_sync_result: "success",
};

describe("CloudSyncPanel", () => {
	beforeEach(() => {
		rcloneMockState.link = null;
	});

	it("shows the unlinked placeholder with a Link button when no link exists", async () => {
		const React = await import("react");
		const { screen } = await import("@testing-library/react");
		const { CloudSyncPanel } = await import("../rclone/CloudSyncPanel");

		await renderWithTestStore(
			React.createElement(CloudSyncPanel as any, {
				targetKind: "volume",
				targetId: "/mnt/usb",
				targetLabel: "USB Disk",
			}),
		);

		expect(
			await screen.findByText("This volume is not linked to a cloud provider."),
		).toBeTruthy();
		expect(screen.getByText(/Link to cloud/i)).toBeTruthy();
	});

	it("opens the wizard and lists registered providers", async () => {
		const React = await import("react");
		const { screen } = await import("@testing-library/react");
		const { CloudSyncPanel } = await import("../rclone/CloudSyncPanel");
		const user = userEvent.setup();

		await renderWithTestStore(
			React.createElement(CloudSyncPanel as any, {
				targetKind: "volume",
				targetId: "/mnt/usb",
				targetLabel: "USB Disk",
			}),
		);

		await user.click(await screen.findByText(/Link to cloud/i));

		expect(
			await screen.findByText(/Link “USB Disk” to cloud/),
		).toBeTruthy();
		expect(await screen.findByText("Provider")).toBeTruthy();
		expect(
			await screen.findByRole("button", { name: "Next" }),
		).toBeTruthy();
	});

	it("renders provider, status and sync actions when linked", async () => {
		const React = await import("react");
		const { screen } = await import("@testing-library/react");
		const { CloudSyncPanel } = await import("../rclone/CloudSyncPanel");

		getMswServer().use(
			http.get("*/api/rclone/link", () =>
				HttpResponse.json(linkedLink),
			),
		);

		await renderWithTestStore(
			React.createElement(CloudSyncPanel as any, {
				targetKind: "volume",
				targetId: "/mnt/usb",
				targetLabel: "USB Disk",
			}),
		);

		expect(await screen.findByText("Dropbox")).toBeTruthy();
		expect(await screen.findByText(/Remote folder: \/srat\/usb/)).toBeTruthy();
		expect(await screen.findByText(/Last sync:/)).toBeTruthy();
		expect(screen.getByText("Push local → remote")).toBeTruthy();
		expect(screen.getByText("Pull remote → local")).toBeTruthy();
		expect(screen.getByText("Sync both ways")).toBeTruthy();
		expect(screen.getByText("Unlink")).toBeTruthy();
	});

	it("computes differences when Refresh diff is clicked", async () => {
		const React = await import("react");
		const { screen } = await import("@testing-library/react");
		const { CloudSyncPanel } = await import("../rclone/CloudSyncPanel");
		const user = userEvent.setup();

		getMswServer().use(
			http.get("*/api/rclone/link", () =>
				HttpResponse.json(linkedLink),
			),
		);

		await renderWithTestStore(
			React.createElement(CloudSyncPanel as any, {
				targetKind: "volume",
				targetId: "/mnt/usb",
				targetLabel: "USB Disk",
			}),
		);

		await user.click(await screen.findByText(/Refresh diff/i));

		expect(await screen.findByText("Differences (3)")).toBeTruthy();
		expect(await screen.findByText("Local only (1)")).toBeTruthy();
		expect(await screen.findByText(/^photos\//)).toBeTruthy();
		expect(await screen.findByText(/^backup\.zip/)).toBeTruthy();
		expect(await screen.findByText(/^notes\.txt/)).toBeTruthy();
	});

	it("opens confirm dialog and starts a push sync", async () => {
		const React = await import("react");
		const { screen } = await import("@testing-library/react");
		const { CloudSyncPanel } = await import("../rclone/CloudSyncPanel");
		const user = userEvent.setup();

		getMswServer().use(
			http.get("*/api/rclone/link", () =>
				HttpResponse.json(linkedLink),
			),
		);
		let syncCalled = false;
		getMswServer().use(
			http.post("*/api/rclone/link/sync", async ({ request }) => {
				syncCalled = true;
				const body = (await request.json()) as Record<string, unknown>;
				expect(body).toEqual({ direction: "push", dry_run: false });
				return HttpResponse.json({});
			}),
		);

		await renderWithTestStore(
			React.createElement(CloudSyncPanel as any, {
				targetKind: "volume",
				targetId: "/mnt/usb",
				targetLabel: "USB Disk",
			}),
		);

		await user.click(await screen.findByText("Push local → remote"));
		expect(
			await screen.findByText("Confirm synchronization"),
		).toBeTruthy();
		await user.click(screen.getByRole("button", { name: "Start" }));

		expect(syncCalled).toBe(true);
	});

	it("starts a dry-run sync when the checkbox is enabled", async () => {
		const React = await import("react");
		const { screen } = await import("@testing-library/react");
		const { CloudSyncPanel } = await import("../rclone/CloudSyncPanel");
		const user = userEvent.setup();

		getMswServer().use(
			http.get("*/api/rclone/link", () =>
				HttpResponse.json(linkedLink),
			),
		);
		let syncBody: Record<string, unknown> | null = null;
		getMswServer().use(
			http.post("*/api/rclone/link/sync", async ({ request }) => {
				syncBody = (await request.json()) as Record<string, unknown>;
				return HttpResponse.json({});
			}),
		);

		await renderWithTestStore(
			React.createElement(CloudSyncPanel as any, {
				targetKind: "volume",
				targetId: "/mnt/usb",
				targetLabel: "USB Disk",
			}),
		);

		await user.click(await screen.findByText("Pull remote → local"));
		await user.click(
			screen.getByRole("checkbox", { name: /Dry run/i }),
		);
		await user.click(screen.getByRole("button", { name: "Start" }));

		expect(syncBody).toEqual({ direction: "pull", dry_run: true });
	});

	it("unlinks after confirmation without deleting remote files", async () => {
		const React = await import("react");
		const { screen } = await import("@testing-library/react");
		const { CloudSyncPanel } = await import("../rclone/CloudSyncPanel");
		const user = userEvent.setup();

		rcloneMockState.link = linkedLink;
		let deleted = false;
		getMswServer().use(
			http.get("*/api/rclone/link", () => {
				return rcloneMockState.link
					? HttpResponse.json(rcloneMockState.link)
					: HttpResponse.json({ detail: "not found" }, { status: 404 });
			}),
			http.delete("*/api/rclone/link", () => {
				deleted = true;
				rcloneMockState.link = null;
				return new HttpResponse(null, { status: 204 });
			}),
		);

		await renderWithTestStore(
			React.createElement(CloudSyncPanel as any, {
				targetKind: "volume",
				targetId: "/mnt/usb",
				targetLabel: "USB Disk",
			}),
		);

		await user.click(await screen.findByText("Unlink"));
		expect(
			await screen.findByText(/Remote files are NOT deleted/),
		).toBeTruthy();
		await user.click(screen.getByRole("button", { name: "Unlink" }));

		expect(deleted).toBe(true);
	});
});
