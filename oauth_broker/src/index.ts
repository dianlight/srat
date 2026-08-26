import { createBrokerApp } from "./app.js";
import { KVSessionStore, MemorySessionStore } from "./session.js";

type Env = Record<string, string | undefined> & {
  OAUTH_SESSIONS?: {
    get(key: string, opts?: { type: string }): Promise<string | null>;
    put(key: string, value: string, opts?: { expirationTtl?: number }): Promise<void>;
    delete(key: string): Promise<void>;
  };
};

// For Cloudflare Workers: `wrangler dev` / deploy uses `fetch` export
const workerApp = (env: Env) => {
  const store = env.OAUTH_SESSIONS ? new KVSessionStore(env.OAUTH_SESSIONS) : new MemorySessionStore();
  return createBrokerApp({ store, env });
};

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const app = workerApp(env);
    return app.fetch(request, env as unknown as Record<string, string>);
  },
};

// For Node / Render / local dev: bun run src/index.ts
if (import.meta.main) {
  const port = parseInt(process.env.PORT || "8080", 10);
  const env: Env = process.env as Env;
  const store = new MemorySessionStore();
  const app = createBrokerApp({ store, env });

  console.log(`[oauth-broker] listening on :${port} (BROKER_PUBLIC_URL=${env.BROKER_PUBLIC_URL || "(unset)"})`);
  Bun.serve({
    port,
    fetch: (req) => app.fetch(req, env as unknown as Record<string, string>),
  });
}
