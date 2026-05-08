import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { describe, expect, it, vi } from "vitest";
import { renderWithTestStore, withTestHandlers } from "/test/testing";

const putSettingsUrl = /.*\/api\/settings(?:\?.*)?$/;

async function renderTelemetryModal(
	props: {
		onClose?: () => void;
		hasInternet?: boolean;
		telemetryMode?: string;
	} = {},
) {
	const React = await import("react");
	const { sratApi } = await import("../../store/sratApi");
	const TelemetryModal = (await import("../TelemetryModal")).default;

	const result = await renderWithTestStore(
		React.createElement(TelemetryModal as any, {
			open: true,
			onClose: props.onClose ?? (() => {}),
		}),
		{
			seedStore: (store) => {
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
			},
		},
	);

	return { ...result, screen };
}

describe("TelemetryModal Component", () => {
	it("renders dialog when internet connection is available", async () => {
		await renderTelemetryModal();
		expect(
			await screen.findByRole("heading", {
				name: /help improve srat/i,
			}),
		).toBeTruthy();
	});

	it("renders all three radio options", async () => {
		await renderTelemetryModal();
		const radios = await screen.findAllByRole("radio");
		expect(radios.length).toBe(3);
	});

	it("renders Continue button", async () => {
		await renderTelemetryModal();
		const button = await screen.findByRole("button", {
			name: /continue/i,
		});
		expect(button).toBeTruthy();
	});

	it("calls onClose after successful submission", async () => {
		const userEvent = (await import("@testing-library/user-event")).default;
		const user = userEvent.setup();
		const onClose = vi.fn(() => {});

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
