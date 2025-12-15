import { useRollbar, useRollbarConfiguration } from "@rollbar/react";
import { useEffect, useState } from "react";
import type Rollbar from "rollbar";
import packageJson from "../../package.json";
import {
	getCurrentEnv,
	getRollbarClientAccessToken,
} from "../macro/Environment" with { type: "macro" };
import {
	type Settings,
	Telemetry_mode,
	useGetApiSettingsQuery,
} from "../store/sratApi";
import { useGetServerEventsQuery } from "../store/sseApi";

// Extend Rollbar configuration to allow optional Replay settings not present in types
type RollbarConfigWithReplay = Rollbar.Configuration & { replay?: unknown };

/**
 * Hook that provides Rollbar functionality with telemetry mode checking
 * This hook ensures that errors and events are only reported based on the current telemetry mode
 */
export const useRollbarTelemetry = () => {
	const rollbar = useRollbar();
	const {
		data: apiSettings,
		isLoading: apiLoading,
		error: apiError,
	} = useGetApiSettingsQuery();
	const [telemetryMode, setTelemetryMode] = useState<Telemetry_mode>(
		Telemetry_mode.Disabled,
	);
	const { data: evdata, isLoading, error: herror } = useGetServerEventsQuery();
	const [rollbarConfig, setRollbarConfig] = useState<RollbarConfigWithReplay>({
		accessToken: getRollbarClientAccessToken() || "disabled",
		environment: getCurrentEnv(),
		codeVersion: packageJson.version,
		captureUncaught: true,
		captureUnhandledRejections: true,
		replay: {
			enabled: false,
			triggers: [
				{
					type: "occurrence",
					level: ["critical"],
					//				samplingRatio: 1.0,
				},
				{
					type: "occurrence",
					level: ["error", "critical"],
					//samplingRatio: 0.5,
				},
			],
		},
		payload: {
			client: {
				javascript: {
					code_version: packageJson.version,
					source_map_enabled: true,
				},
			},
		},
		enabled: false,
	});
	useRollbarConfiguration(rollbarConfig as Rollbar.Configuration);

	useEffect(() => {
		setTelemetryMode(
			(apiSettings as Settings)?.telemetry_mode || Telemetry_mode.Ask,
		);
	}, [apiSettings]);

	useEffect(() => {
		if (!isLoading && !apiLoading && evdata?.hello && apiSettings) {
			// Configure Telemetry
			const accessToken = getRollbarClientAccessToken();
			const enableRollbar =
				accessToken !== "disabled" &&
				[Telemetry_mode.Errors, Telemetry_mode.All].includes(
					(apiSettings as Settings)?.telemetry_mode || Telemetry_mode.Ask,
				);
			setRollbarConfig({
				accessToken,
				environment: getCurrentEnv(),
				codeVersion: packageJson.version,
				captureUncaught: true,
				captureUnhandledRejections: true,
				replay: {
					enabled: enableRollbar,
					triggers: [
						{
							type: "occurrence",
							level: ["critical"],
							//				samplingRatio: 1.0,
						},
						{
							type: "occurrence",
							level: ["error"],
							//				samplingRatio: 0.5,
						},
					],
				},
				payload: {
					client: {
						javascript: {
							code_version: packageJson.version,
							source_map_enabled: true,
						},
					},
					person: evdata?.hello.machine_id
						? {
								id: evdata.hello.machine_id,
							}
						: undefined,
				},
				enabled: enableRollbar,
			} as RollbarConfigWithReplay);
		}
	}, [isLoading, evdata?.hello, apiSettings, apiLoading]);

	const reportError = (
		error: Error | string,
		extraData?: Record<string, unknown>,
	) => {
		if ([Telemetry_mode.Errors, Telemetry_mode.All].includes(telemetryMode)) {
			rollbar.error(error, extraData);
		}
	};

	const reportEvent = (event: string, data?: Record<string, unknown>) => {
		if ([Telemetry_mode.All].includes(telemetryMode)) {
			const eventData = {
				...data,
				event_type: event,
				timestamp: new Date().toISOString(),
			};

			rollbar.info(`Event: ${event}`, eventData);
			console.debug("Event reported to Rollbar:", event, eventData);
		}
	};

	return {
		reportError,
		reportEvent,
		telemetryMode,
		isLoading: apiLoading || isLoading,
		error: apiError || herror,
	};
};
