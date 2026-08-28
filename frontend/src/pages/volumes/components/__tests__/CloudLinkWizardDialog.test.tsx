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

const dropboxProvider = {
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
		// Link identity travels as query params (target ids contain slashes).
		http.get("*/api/rclone/link", () =>
			linkAuthorized
				? HttpResponse.json(authorizedLink)
				: HttpResponse.json({ detail: "rclone link not found" }, { status: 404 }),
		),
		http.put("*/api/rclone/link", async ({ request }) => {
			calls.push(`PUT:${JSON.stringify(await request.json())}`);
			return HttpResponse.json(unlinkedLink);
		}),
		http.post("*/api/rclone/link/auth/start", async ({ request }) => {
			calls.push(`POST:auth/start:${JSON.stringify(await request.json())}`);
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
			http.get("*/api/rclone/providers", () =>
				HttpResponse.json({
					library_available: true,
					broker_available: false,
					oauth_callback_path: "/api/rclone/oauth/callback",
					providers: [dropboxProvider],
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
		expect(calls[1]).toMatch(/^POST:auth\/start:/);
		const putBody = JSON.parse((calls[0] as string).slice(4)) as Record<string, unknown>;
		expect(putBody.provider).toBe("dropbox");

		// The browser-visible origin is forwarded so the OAuth redirect URI
		// resolves for this user (regression: generic "Failed to start
		// authorization" masked every failure before).
		const authBody = JSON.parse(
			(calls[1] as string).replace(/^POST:auth\/start:/, ""),
		) as { public_base_url?: string; auth_mode?: string };
		expect(authBody.public_base_url).toBe(window.location.origin);
		expect(authBody.auth_mode).toBe("custom_app");

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

	it("disables unavailable authorization modes", async () => {
		const { screen, within } = await import("@testing-library/react");
		const user = userEvent.setup();
		await renderWizard();

		await user.click(
			await screen.findByRole("combobox", { name: "Authorization" }),
		);
		const listbox = await screen.findByRole("listbox");
		// Broker not configured on this server → option shown but disabled.
		const brokerOption = within(listbox).getByRole("option", {
			name: /Hosted SRAT OAuth/,
		});
		expect(brokerOption.getAttribute("aria-disabled")).toBe("true");
		// HA-integration reuse is a planned stub and always disabled.
		const haOption = within(listbox).getByRole("option", {
			name: /Reuse Dropbox integration/,
		});
		expect(haOption.getAttribute("aria-disabled")).toBe("true");

		const customOption = within(listbox).getByRole("option", {
			name: /Custom app/,
		});
		expect(customOption.getAttribute("aria-disabled")).not.toBe("true");
	});

	it("allows empty credentials when the hosted OAuth broker is available", async () => {
		const { screen } = await import("@testing-library/react");
		const user = userEvent.setup();

		getMswServer().use(
			http.get("*/api/rclone/providers", () =>
				HttpResponse.json({
					library_available: true,
					broker_available: true,
					oauth_callback_path: "/api/rclone/oauth/callback",
					providers: [dropboxProvider],
				}),
			),
		);
		const { calls } = await renderWizard();

		await user.click(await screen.findByRole("combobox", { name: "Provider" }));
		await user.click(await screen.findByRole("option", { name: "Dropbox" }));
		// No credentials typed: the broker flow makes them optional, so Next
		// must advance to step 1 directly.
		await user.click(screen.getByRole("button", { name: "Next" }));
		expect(
			await screen.findByRole("button", { name: /Connect Dropbox/i }),
		).toBeTruthy();

		await user.click(screen.getByRole("button", { name: /Connect Dropbox/i }));
		const authBody = JSON.parse(
			(calls[1] as string).replace(/^POST:auth\/start:/, ""),
		) as { settings?: Record<string, string>; auth_mode?: string };
		expect(authBody.settings ?? {}).toEqual({});
		expect(authBody.auth_mode).toBe("broker");
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
