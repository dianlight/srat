import { createBrokerApp } from "./app.js";
import {
  D1ClientStore,
  D1InstanceStore,
  D1NonceStore,
  D1SessionStore,
  KVClientStore,
  KVInstanceStore,
  KVNonceStore,
  KVSessionStore,
  MemoryClientStore,
  MemoryInstanceStore,
  MemoryNonceStore,
  MemorySessionStore,
} from "./session.js";
import type { D1DatabaseLike } from "./session.js";
import { getBrokerPublicUrl, isValidBrokerPublicUrl, isProductionEnv } from "./config.js";

type Env = Record<string, string | undefined> & {
  OAUTH_SESSIONS?: {
    get(key: string, opts?: { type: string }): Promise<string | null>;
    put(key: string, value: string, opts?: { expirationTtl?: number }): Promise<void>;
    delete(key: string): Promise<void>;
  };
  OAUTH_SESSIONS_DB?: D1DatabaseLike;
  RATE_LIMITER?: { limit(opts: { key: string }): Promise<{ success: boolean }> };
  BROKER_RATE_LIMITER?: { limit(opts: { key: string }): Promise<{ success: boolean }> };
};

// Module-scoped memory stores reused across requests when KV/D1 absent (local dev / tests).
// Prevents per-request MemoryStore that would lose state between /v1/start and /v1/callback.
const workerMemoryStore = new MemorySessionStore();
const workerMemoryInstanceStore = new MemoryInstanceStore();
const workerMemoryClientStore = new MemoryClientStore();
const workerMemoryNonceStore = new MemoryNonceStore();

function validateEnvAtStartup(env: Record<string, string | undefined>): void {
  const publicUrl = getBrokerPublicUrl(env);
  if (publicUrl && !isValidBrokerPublicUrl(publicUrl)) {
    throw new Error(`[broker] BROKER_PUBLIC_URL invalid at startup: "${publicUrl}" – must be absolute https (loopback http allowed for dev)`);
  }
  const disableAuth = (env.BROKER_DISABLE_AUTH || "").trim().toLowerCase();
  const insecure = disableAuth === "true" || disableAuth === "1" || disableAuth === "yes";
  if (insecure && isProductionEnv(env)) {
    throw new Error("[broker] BROKER_DISABLE_AUTH must not be enabled in production – refusing to start");
  }
  if (insecure) {
    console.warn("[broker] ⚠️ BROKER_DISABLE_AUTH enabled – auth disabled (dev only)");
  }
  const allowlist = env.BROKER_ALLOWED_CALLBACK_PATTERNS?.trim();
  if (allowlist) {
    console.log(`[broker] callback allowlist enabled: ${allowlist}`);
  }
}

// For Cloudflare Workers: `wrangler dev` / deploy uses `fetch` export
// Precedence: D1 (atomic) > KV (eventually consistent) > memory (local dev/tests).
const workerApp = (env: Env) => {
  // Startup validation + store selection warnings
  if (!env.OAUTH_SESSIONS_DB && !env.OAUTH_SESSIONS) {
    console.warn("[broker] no D1 or KV binding – using MemoryStores (single isolate only, data lost on restart)");
  } else if (!env.OAUTH_SESSIONS_DB && env.OAUTH_SESSIONS) {
    console.warn("[broker] OAUTH_SESSIONS_DB (D1) not bound – falling back to KV (eventually consistent, non-atomic consume). Use D1 in production for single-use guarantee.");
  }
  const store = env.OAUTH_SESSIONS_DB
    ? new D1SessionStore(env.OAUTH_SESSIONS_DB)
    : env.OAUTH_SESSIONS
      ? new KVSessionStore(env.OAUTH_SESSIONS)
      : workerMemoryStore;
  const instanceStore = env.OAUTH_SESSIONS_DB
    ? new D1InstanceStore(env.OAUTH_SESSIONS_DB)
    : env.OAUTH_SESSIONS
      ? new KVInstanceStore(env.OAUTH_SESSIONS)
      : workerMemoryInstanceStore;
  const clientStore = env.OAUTH_SESSIONS_DB
    ? new D1ClientStore(env.OAUTH_SESSIONS_DB)
    : env.OAUTH_SESSIONS
      ? new KVClientStore(env.OAUTH_SESSIONS)
      : workerMemoryClientStore;
  const nonceStore = env.OAUTH_SESSIONS_DB
    ? new D1NonceStore(env.OAUTH_SESSIONS_DB)
    : env.OAUTH_SESSIONS
      ? new KVNonceStore(env.OAUTH_SESSIONS)
      : workerMemoryNonceStore;
  return createBrokerApp({ store, instanceStore, clientStore, nonceStore, env });
};

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    // Cheap per-request validation (fail fast on prod misconfig even if worker reuses env)
    try {
      validateEnvAtStartup(env as Record<string, string | undefined>);
    } catch (e) {
      console.error((e as Error).message);
      return new Response(JSON.stringify({ error: (e as Error).message }), { status: 500, headers: { "content-type": "application/json" } });
    }
    const app = workerApp(env);
    return app.fetch(request, env as unknown as Record<string, string>);
  },
};

// For Node / Render / local dev: bun run src/index.ts
if (import.meta.main) {
  const port = parseInt(process.env.PORT || "8080", 10);
  const env: Env = process.env as Env;
  try {
    validateEnvAtStartup(env as Record<string, string | undefined>);
  } catch (e) {
    console.error((e as Error).message);
    process.exit(1);
  }
  if (!isValidBrokerPublicUrl(getBrokerPublicUrl(env as Record<string, string | undefined>)) && getBrokerPublicUrl(env as Record<string, string | undefined>)) {
    console.warn(`[broker] BROKER_PUBLIC_URL "${env.BROKER_PUBLIC_URL}" is not valid https – POST /v1/start will 500 until fixed`);
  }
  const store = new MemorySessionStore();
  const instanceStore = new MemoryInstanceStore();
  const clientStore = new MemoryClientStore();
  const nonceStore = new MemoryNonceStore();
  const app = createBrokerApp({ store, instanceStore, clientStore, nonceStore, env });

  console.log(`[oauth-broker] listening on :${port} (BROKER_PUBLIC_URL=${env.BROKER_PUBLIC_URL || "(unset)"})`);
  Bun.serve({
    port,
    fetch: (req) => app.fetch(req, env as unknown as Record<string, string>),
  });
}
