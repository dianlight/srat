export function getApiUrl(): string {
    return process.env.API_URL || "dynamic";
}

export function getServerEventBackend(): string {
    return process.env.SERVER_EVENT_BACKEND || "ws";
}

export function getRollbarClientAccessToken(): string {
    return process.env.ROLLBAR_CLIENT_ACCESS_TOKEN || "disabled";
}

export function getNodeEnv(): string {
    return process.env.NODE_ENV || "development";
}