import { cleanup } from "@testing-library/react";
import { afterEach, describe, expect, it, mock } from "bun:test";
import { http, HttpResponse } from "msw";
import { withTestHandlers } from "../../../test/bun-setup";
import "../../../test/setup";

const putSettingsUrl = /.*\/api\/settings(?:\?.*)?$/;

afterEach(() => {
	cleanup();
});

async function renderTelemetryModal(
	props: {
		onClose?: () => void;
		hasInternet?: boolean;
		telemetryMode?: string;
	} = {},
) {
	const React = await import("react");
	const { render } = await import("@testing-library/react");
	const { Provider } = await import("react-redux");
	const { createTestStore } = await import("../../../test/setup");
	const { sratApi } = await import("../../store/sratApi");
	const TelemetryModal = (await import("../TelemetryModal")).default;

	const store = await createTestStore();

	// Seed RTK Query cache for GET endpoints so tests don't depend on MSW timing
	(store.dispatch as any)(
		sratApi.util.upsertQueryData(
			"getApiTelemetryInternetConnection",
			undefined,
			props.hasInternet ?? true,
		),
	);
	(store.dispatch as any)(
		sratApi.util.upsertQueryData("getApiSettings", undefined, {
			telemetry_mode: props.telemetryMode ?? "all",
			hostname: "test-host",
		} as any),
	);

	return render(
		React.createElement(Provider, {
			store,
			children: React.createElement(TelemetryModal as any, {
				open: true,
				onClose: props.onClose ?? (() => {}),
			}),
		}),
	);
}

describe("TelemetryModal Component", () => {
	it("renders dialog when internet connection is available", async () => {
		const { screen } = await import("@testing-library/react");

		await renderTelemetryModal();
		expect(
			await screen.findByRole("heading", {
				name: /help improve srat/i,
			}),
		).toBeTruthy();
	});

	it("renders all three radio options", async () => {
		const { screen } = await import("@testing-library/react");

		await renderTelemetryModal();
		const radios = await screen.findAllByRole("radio");
		expect(radios.length).toBe(3);
	});

	it("renders Continue button", async () => {
		const { screen } = await import("@testing-library/react");

		await renderTelemetryModal();
		const button = await screen.findByRole("button", {
			name: /continue/i,
		});
		expect(button).toBeTruthy();
	});

	it("calls onClose after successful submission", async () => {
		const { screen } = await import("@testing-library/react");
		const userEvent = (await import("@testing-library/user-event")).default;
		const user = userEvent.setup();
		const onClose = mock(() => {});

		await withTestHandlers(
			[
				http.put(putSettingsUrl, () =>
					HttpResponse.json({ telemetry_mode: "all" }),
				),
			],
			async () => {
				await renderTelemetryModal({ onClose });
				const button = await screen.findByRole("button", {
					name: /continue/i,
				});
				await user.click(button);
				expect(onClose).toHaveBeenCalled();
			},
		);
	});
});
