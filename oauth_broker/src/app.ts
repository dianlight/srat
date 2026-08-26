import { Hono } from "hono";
import { timingSafeEqual } from "node:crypto";
import { getBrokerPublicUrl, getProviderOrThrow, getSessionTtlSeconds, loadProvidersConfig } from "./config.js";
import type { SessionStore } from "./session.js";
import { MemorySessionStore } from "./session.js";

export type BrokerBindings = {
  OAUTH_SESSIONS?: {
    get(key: string, opts?: { type: string }): Promise<string | null>;
    put(key: string, value: string, opts?: { expirationTtl?: number }): Promise<void>;
    delete(key: string): Promise<void>;
  };
};

type AppEnv = {
  Bindings: BrokerBindings & Record<string, unknown>;
  Variables: Record<string, never>;
};

function isLoopbackHttp(url: URL): boolean {
  if (url.protocol !== "http:") return false;
  const host = url.hostname.toLowerCase();
  return host === "localhost" || host === "127.0.0.1" || host === "::1" || host === "[::1]";
}

export function isValidSratCallbackUrl(raw: string): boolean {
  try {
    const u = new URL(raw);
    if (u.protocol === "https:") return true;
    if (isLoopbackHttp(u)) return true;
    return false;
  } catch {
    return false;
  }
}

function bearerTokenFromHeader(authHeader: string | null): string {
  if (!authHeader) return "";
  const m = authHeader.match(/^Bearer\s+(.+)$/i);
  return m ? m[1].trim() : "";
}

function constantTimeEqual(a: string, b: string): boolean {
  if (a.length !== b.length) {
    // Still do a timing-safe compare on padded buffers to avoid length leak
    const ba = Buffer.from(a);
    const bb = Buffer.from(b);
    // Use equal length buffers for timingSafeEqual; if lengths differ, pad
    const maxLen = Math.max(ba.length, bb.length);
    const padA = Buffer.alloc(maxLen);
    const padB = Buffer.alloc(maxLen);
    ba.copy(padA);
    bb.copy(padB);
    // Always false when lengths differ, but we still call timingSafeEqual
    timingSafeEqual(padA, padB);
    return false;
  }
  return timingSafeEqual(Buffer.from(a), Buffer.from(b));
}

export function createBrokerApp(opts?: {
  store?: SessionStore;
  env?: Record<string, string | undefined>;
  fetchImpl?: typeof fetch;
}) {
  const app = new Hono<AppEnv>();
  const store: SessionStore = opts?.store ?? new MemorySessionStore();
  const envOverride = opts?.env;
  const fetchImpl: typeof fetch = opts?.fetchImpl ?? fetch;

  function getEnv(c: { env: Record<string, unknown> }): Record<string, string | undefined> {
    // c.env is Workers env (wrangler vars + secrets); precedence: request env > override > process.env
    const workerEnv = (c.env || {}) as Record<string, string | undefined>;
    return { ...(process.env as Record<string, string | undefined>), ...(envOverride || {}), ...workerEnv };
  }

  function requireBearer(c: { req: { header: (n: string) => string | undefined }; env: Record<string, unknown> }): boolean {
    const env = getEnv(c);
    const expected = (env.BROKER_API_TOKEN || "").trim();
    if (!expected) return true; // if no token configured, allow (dev)
    const got = bearerTokenFromHeader(c.req.header("authorization") || null);
    if (!got) return false;
    return constantTimeEqual(got, expected);
  }

  // Healthz — public, no auth
  app.get("/v1/healthz", (c) => {
    const env = getEnv(c);
    const providers = loadProvidersConfig(env);
    return c.json({ status: "ok", providers: Object.keys(providers) });
  });

  // Also support /healthz without prefix for Render healthCheckPath
  app.get("/healthz", (c) => {
    const env = getEnv(c);
    const providers = loadProvidersConfig(env);
    return c.json({ status: "ok", providers: Object.keys(providers) });
  });

  app.post("/v1/start", async (c) => {
    if (!requireBearer(c)) {
      return c.json({ error: "unauthorized" }, 401);
    }
    let body: { provider?: string; srat_callback_url?: string };
    try {
      body = await c.req.json();
    } catch {
      return c.json({ error: "invalid json body" }, 400);
    }
    const provider = (body.provider || "").trim();
    const sratCallbackUrl = (body.srat_callback_url || "").trim();
    if (!provider) return c.json({ error: "provider is required" }, 400);
    if (!sratCallbackUrl) return c.json({ error: "srat_callback_url is required" }, 400);
    if (!isValidSratCallbackUrl(sratCallbackUrl)) {
      return c.json({ error: "srat_callback_url must be an absolute https URL (loopback http allowed for dev)" }, 400);
    }

    const env = getEnv(c);
    const providers = loadProvidersConfig(env);
    let prov: ReturnType<typeof getProviderOrThrow>;
    try {
      prov = getProviderOrThrow(providers, provider);
    } catch (e) {
      return c.json({ error: (e as Error).message }, 400);
    }

    const publicUrl = getBrokerPublicUrl(env);
    if (!publicUrl) return c.json({ error: "BROKER_PUBLIC_URL is not configured" }, 500);
    const redirectUri = `${publicUrl}/v1/callback`;

    const sessionId = crypto.randomUUID();
    const ttl = getSessionTtlSeconds(env);

    // Build auth_url: provider authorize URL + query (client_id, response_type=code, redirect_uri, state=session_id, + auth_params + scope)
    const authUrl = new URL(prov.authorize_url);
    authUrl.searchParams.set("client_id", prov.client_id);
    authUrl.searchParams.set("response_type", "code");
    authUrl.searchParams.set("redirect_uri", redirectUri);
    authUrl.searchParams.set("state", sessionId);
    for (const [k, v] of Object.entries(prov.auth_params || {})) {
      if (!authUrl.searchParams.has(k)) authUrl.searchParams.set(k, v);
    }
    if (prov.scopes && prov.scopes.length > 0) {
      authUrl.searchParams.set("scope", prov.scopes.join(" "));
    }

    await store.set(sessionId, { provider, sratCallbackUrl, createdAt: Date.now() }, ttl);

    return c.json({ auth_url: authUrl.toString(), session_id: sessionId });
  });

  app.get("/v1/callback", async (c) => {
    const code = c.req.query("code") || "";
    const state = c.req.query("state") || "";
    if (!code || !state) {
      return c.json({ error: "missing code or state" }, 400);
    }

    const env = getEnv(c);
    const ttl = getSessionTtlSeconds(env);
    const session = await store.get(state);
    if (!session) {
      return c.json({ error: "session not found or expired" }, 410);
    }
    // If already completed, treat as gone (single-use guard)
    if (session.tokenJson) {
      return c.json({ error: "session not found or expired" }, 410);
    }

    const providers = loadProvidersConfig(env);
    let prov: ReturnType<typeof getProviderOrThrow>;
    try {
      prov = getProviderOrThrow(providers, session.provider);
    } catch (e) {
      return c.json({ error: (e as Error).message }, 400);
    }

    const publicUrl = getBrokerPublicUrl(env);
    const redirectUri = `${publicUrl}/v1/callback`;

    // Exchange code with provider token URL
    const form = new URLSearchParams({
      grant_type: "authorization_code",
      code,
      redirect_uri: redirectUri,
      client_id: prov.client_id,
      client_secret: prov.client_secret,
    });

    let tokenResp: Record<string, unknown>;
    let tokenStatus = 200;
    try {
      const resp = await fetchImpl(prov.token_url, {
        method: "POST",
        headers: { "content-type": "application/x-www-form-urlencoded" },
        body: form.toString(),
      });
      tokenStatus = resp.status;
      const text = await resp.text();
      try {
        tokenResp = JSON.parse(text) as Record<string, unknown>;
      } catch {
        return c.json({ error: `invalid token response (status ${tokenStatus})` }, 502);
      }
      if (!resp.ok || !tokenResp.access_token) {
        const msg = (tokenResp.error as string) || (tokenResp.error_description as string) || `token exchange failed (status ${tokenStatus})`;
        return c.json({ error: msg }, 502);
      }
    } catch (e) {
      return c.json({ error: `token exchange failed: ${(e as Error).message}` }, 502);
    }

    // Wrap in rclone envelope: {"access_token","token_type","refresh_token","expires_in","expiry","account_id"}
    const expiresIn = Number(tokenResp.expires_in || 0);
    const expiry = new Date(Date.now() + expiresIn * 1000).toISOString();
    const envelope: Record<string, unknown> = {
      access_token: tokenResp.access_token,
      token_type: tokenResp.token_type || "Bearer",
      refresh_token: tokenResp.refresh_token,
      expires_in: expiresIn || undefined,
      expiry,
      account_id: tokenResp.account_id || tokenResp.accountId || tokenResp.user_id || undefined,
    };
    // Remove undefined keys for cleanliness
    for (const k of Object.keys(envelope)) if (envelope[k] === undefined) delete envelope[k];

    const tokenJson = JSON.stringify(envelope);
    const accountLabel = (envelope.account_id as string) || "";

    await store.set(
      state,
      {
        ...session,
        tokenJson,
        accountLabel,
        clientId: prov.client_id,
        clientSecret: prov.client_secret,
        completedAt: Date.now(),
      },
      ttl
    );

    // 302 redirect back to srat_callback_url (which already carries SRAT's state)
    return c.redirect(session.sratCallbackUrl, 302);
  });

  app.get("/v1/session/:id", async (c) => {
    if (!requireBearer(c)) {
      return c.json({ error: "unauthorized" }, 401);
    }
    const id = c.req.param("id");
    const session = await store.get(id);
    if (!session) {
      c.header("Cache-Control", "no-store");
      return c.json({ error: "expired or already used" }, 404);
    }
    if (!session.tokenJson) {
      // Not yet completed — 404 without consuming (early polling cannot destroy flow)
      c.header("Cache-Control", "no-store");
      return c.json({ error: "not yet completed" }, 404);
    }
    // Completed — consume single-use
    await store.delete(id);
    c.header("Cache-Control", "no-store");
    return c.json({
      token_json: session.tokenJson,
      account_label: session.accountLabel || "",
      client_id: session.clientId || "",
      client_secret: session.clientSecret || "",
    });
  });

  // Catch-all for unknown routes
  app.notFound((c) => c.json({ error: "not found" }, 404));

  return app;
}
