import { http, HttpResponse } from "msw";
import { beforeEach, describe, expect, it } from "vitest";
import userEvent from "@testing-library/user-event";
import { getMswServer, renderWithTestStore } from "/test/testing";

const unlinkedLink = {
	target_kind: "volume",
	target_id: "/mnt/usb",
	provider: "dropbox",
	remote_path: "/",
	status: "unlinked",
	auto_sync: false,
	schedule_minutes: 0,
};

const authorizedLink = {
	...unlinkedLink,
	status: "authorized",
	remote_path: "/srat/USB Disk",
};

/**
 * Renders the wizard dialog directly (open) with MSW handlers that track
 * call order so the provisional PUT-before-auth/start contract can be
 * asserted.
 */
async function renderWizard() {
	const React = await import("react");
	const { CloudLinkWizardDialog } = await import("../rclone/CloudLinkWizardDialog");

	let linkAuthorized = false;
	const calls: string[] = [];

	getMswServer().use(
		http.get(/.*\/api\/rclone\/link\/.*$/, () =>
			linkAuthorized
				? HttpResponse.json(authorizedLink)
				: HttpResponse.json({ detail: "rclone link not found" }, { status: 404 }),
		),
		http.put(/.*\/api\/rclone\/link\/.*$/, async ({ request }) => {
			calls.push(`PUT:${JSON.stringify(await request.json())}`);
			return HttpResponse.json(unlinkedLink);
		}),
		http.post(/.*\/api\/rclone\/link\/.*\/auth\/start$/, () => {
			calls.push("POST:auth/start");
			return HttpResponse.json({
				auth_url: "https://www.dropbox.com/oauth2/authorize?mock=1",
				redirect_uri: "http://localhost/api/rclone/oauth/callback",
				state: "mock-state",
			});
		}),
	);

	await renderWithTestStore(
		React.createElement(CloudLinkWizardDialog as any, {
			open: true,
			onClose: () => {},
			targetKind: "volume",
			targetId: "/mnt/usb",
			targetLabel: "USB Disk",
		}),
	);

	return {
		calls,
		markAuthorized: () => {
			linkAuthorized = true;
		},
	};
}

describe("CloudLinkWizardDialog", () => {
	beforeEach(() => {
		getMswServer().use(
			http.get(/.*\/api\/rclone\/providers$/, () =>
				HttpResponse.json({
					library_available: true,
					providers: [
						{
							name: "dropbox",
							display_name: "Dropbox",
							config_fields: [
								{
									name: "client_id",
									label: "App key",
									description: "Dropbox OAuth app key",
									required: true,
									secret: false,
								},
								{
									name: "client_secret",
									label: "App secret",
									description: undefined,
									required: true,
									secret: true,
								},
							],
						},
					],
				}),
			),
		);
	});

	it("saves a provisional link before starting authorization", async () => {
		const { screen } = await import("@testing-library/react");
		const user = userEvent.setup();
		const { calls } = await renderWizard();

		// Step 0: pick the provider and fill its credentials.
		await user.click(await screen.findByRole("combobox", { name: "Provider" }));
		await user.click(await screen.findByRole("option", { name: "Dropbox" }));
		await user.type(await screen.findByLabelText("App key"), "cid");
		await user.type(await screen.findByLabelText("App secret"), "secret");
		await user.click(screen.getByRole("button", { name: "Next" }));

		// Step 1: connect triggers PUT (row must exist) THEN auth/start.
		await user.click(
			await screen.findByRole("button", { name: /Connect Dropbox/i }),
		);

		expect(calls[0]).toMatch(/^PUT:/);
		expect(calls[1]).toBe("POST:auth/start");
		const putBody = JSON.parse((calls[0] as string).slice(4)) as Record<string, unknown>;
		expect(putBody.provider).toBe("dropbox");

		// The authorization page link is offered while pending.
		expect(
			await screen.findByRole("link", { name: /Open authorization page again/i }),
		).toBeTruthy();
	});

	it("requires credentials before leaving the provider step", async () => {
		const { screen } = await import("@testing-library/react");
		const user = userEvent.setup();
		await renderWizard();

		await user.click(await screen.findByRole("combobox", { name: "Provider" }));
		await user.click(await screen.findByRole("option", { name: "Dropbox" }));
		// No credentials typed: Next must not advance to step 1.
		await user.click(screen.getByRole("button", { name: "Next" }));

		await new Promise((resolve) => setTimeout(resolve, 100));
		expect(screen.queryByText(/Step 2\/3/)).toBeNull();
		expect(screen.queryByText(/Connect Dropbox/i)).toBeNull();
	});

	it("polls until authorized, then finalizes the chosen remote folder", async () => {
		const { screen } = await import("@testing-library/react");
		const user = userEvent.setup();
		const { calls, markAuthorized } = await renderWizard();

		await user.click(await screen.findByRole("combobox", { name: "Provider" }));
		await user.click(await screen.findByRole("option", { name: "Dropbox" }));
		await user.type(await screen.findByLabelText("App key"), "cid");
		await user.type(await screen.findByLabelText("App secret"), "secret");
		await user.click(screen.getByRole("button", { name: "Next" }));
		await user.click(
			await screen.findByRole("button", { name: /Connect Dropbox/i }),
		);

		// Flip the backend state; the 3s poll must pick it up automatically.
		markAuthorized();
		await screen.findByText(
			/Account connected/,
			{},
			{ timeout: 10000 },
		);
		await user.click(screen.getByRole("button", { name: "Next" }));

		// Step 2: adjust the remote folder and create the link.
		const folder = await screen.findByLabelText(/Remote folder/i);
		await user.clear(folder);
		await user.type(folder, "/srat/custom");
		await user.click(screen.getByRole("button", { name: "Create link" }));

		const finalizePut = calls[calls.length - 1] as string;
		expect(finalizePut).toMatch(/^PUT:/);
		const body = JSON.parse(finalizePut.slice(4)) as Record<string, unknown>;
		expect(body.remote_path).toBe("/srat/custom");
		expect(body.provider).toBe("dropbox");
	}, 30000);
});
