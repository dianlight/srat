import packageJson from "../../package.json";

export function getApiUrl(): string {
	return process.env.API_URL || "dynamic";
}

/*
export function getServerEventBackend(): string {
	return process.env.SERVER_EVENT_BACKEND || "ws";
}
*/

export function getRollbarClientAccessToken(): string {
	return process.env.ROLLBAR_CLIENT_ACCESS_TOKEN || "disabled";
}

export function getCurrentEnv(): string {
	const nodeEnv = process.env.NODE_ENV || "development";

	if (nodeEnv !== "production") {
		return nodeEnv;
	}

	const version =
		(process.env.npm_package_version as string | undefined) ||
		packageJson.version ||
		"";

	if (version.includes("-dev.")) {
		return "development";
	}

	if (version.includes("-rc.")) {
		return "prerelease";
	}

	return "production";
}
