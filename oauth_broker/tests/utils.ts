import { createBrokerApp } from "../src/app.js";
import { MemoryInstanceStore, MemorySessionStore } from "../src/session.js";

/**
 * Creates the default environment configuration used by tests.
 *
 * @param overrides - Environment values that replace the defaults
 * @returns The test environment configuration
 */
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

/**
 * Creates a broker application configured for testing.
 *
 * @param env - Environment variables used to configure the broker application
 * @param opts - Optional session store and fetch implementation overrides
 * @returns The broker application, session store, and environment configuration
 */
export function createTestApp(
  env: Record<string, string | undefined> = testEnv(),
  opts: { store?: MemorySessionStore; instanceStore?: MemoryInstanceStore; fetchImpl?: typeof fetch } = {}
) {
  const store = opts.store ?? new MemorySessionStore();
  const instanceStore = opts.instanceStore ?? new MemoryInstanceStore();
  const app = createBrokerApp({ store, instanceStore, env, fetchImpl: opts.fetchImpl });
  return { app, store, instanceStore, env };
}

export async function registerInstance(app: any, instanceId: string, redirectUrl: string) {
  return app.request("/v1/instances/register", {
    method: "POST",
    headers: { "content-type": "application/json", authorization: "Bearer test-token" },
    body: JSON.stringify({ instance_id: instanceId, redirect_url: redirectUrl }),
  });
}

/**
 * Parses a response body as JSON and preserves the raw text when parsing fails.
 *
 * @param resp - The response whose body should be parsed
 * @returns The parsed JSON object, or an object containing the raw response text under `_raw`
 */
export async function jsonBody(resp: Response): Promise<Record<string, unknown>> {
  const text = await resp.text();
  try {
    return JSON.parse(text) as Record<string, unknown>;
  } catch {
    return { _raw: text };
  }
}
