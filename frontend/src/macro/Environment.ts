import packageJson from "../../package.json";

export function getApiUrl(): string {
  return readEnv("API_URL") || "dynamic";
}

/*
export function getServerEventBackend(): string {
	return process.env.SERVER_EVENT_BACKEND || "ws";
}
*/

export function getRollbarClientAccessToken(): string {
  return readEnv("ROLLBAR_CLIENT_ACCESS_TOKEN") || "disabled";
}

function readEnv(key: string): string | undefined {
  return (typeof process !== "undefined" ? process.env : {})[key];
}

export function getCurrentEnv(): string {
  const nodeEnv = readEnv("NODE_ENV") || "development";

  if (nodeEnv !== "production") {
    return nodeEnv;
  }

  const version = readEnv("npm_package_version") || packageJson.version || "";

  if (version.includes("-dev.")) {
    return "development";
  }

  if (version.includes("-rc.")) {
    return "prerelease";
  }

  return "production";
}
