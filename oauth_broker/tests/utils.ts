import { createBrokerApp } from "../src/app.js";
import { MemorySessionStore } from "../src/session.js";

export function testEnv(overrides: Record<string, string | undefined> = {}): Record<string, string | undefined> {
  return {
    BROKER_PUBLIC_URL: "https://broker.example.com",
    BROKER_API_TOKEN: "test-token",
    SESSION_TTL: "600",
    DROPBOX_CLIENT_ID: "dropbox-id",
    DROPBOX_CLIENT_SECRET: "dropbox-secret",
    ...overrides,
  };
}

export function createTestApp(
  env: Record<string, string | undefined> = testEnv(),
  opts: { store?: MemorySessionStore; fetchImpl?: typeof fetch } = {}
) {
  const store = opts.store ?? new MemorySessionStore();
  const app = createBrokerApp({ store, env, fetchImpl: opts.fetchImpl });
  return { app, store, env };
}

export async function jsonBody(resp: Response): Promise<Record<string, unknown>> {
  const text = await resp.text();
  try {
    return JSON.parse(text) as Record<string, unknown>;
  } catch {
    return { _raw: text };
  }
}
