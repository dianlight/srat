import { createBrokerApp } from "../src/app.js";
import { MemorySessionStore } from "../src/session.js";

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
  opts: { store?: MemorySessionStore; fetchImpl?: typeof fetch } = {}
) {
  const store = opts.store ?? new MemorySessionStore();
  const app = createBrokerApp({ store, env, fetchImpl: opts.fetchImpl });
  return { app, store, env };
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
