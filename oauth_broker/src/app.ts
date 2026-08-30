import { Hono } from "hono";
import { createHash, timingSafeEqual } from "node:crypto";
import {
  getBrokerPublicUrlOrThrow,
  getProviderOrThrow,
  getSessionTtlSeconds,
  isAllowedSratCallbackUrl,
  isProductionEnv,
  loadProvidersConfig,
  MAX_CALLBACK_URL_LENGTH,
} from "./config.js";
import type { SessionStore } from "./session.js";
import { MemorySessionStore } from "./session.js";

export type BrokerBindings = {
  OAUTH_SESSIONS?: {
    get(key: string, opts?: { type: string }): Promise<string | null>;
    put(key: string, value: string, opts?: { expirationTtl?: number }): Promise<void>;
    delete(key: string): Promise<void>;
  };
  // Optional Cloudflare Rate Limiter binding (Workers Rate Limiting)
  // Create via: wrangler ratelimit create broker-ratelimit --max-keys 10000 --period 60
  RATE_LIMITER?: {
    limit(opts: { key: string }): Promise<{ success: boolean }>;
  };
  BROKER_RATE_LIMITER?: {
    limit(opts: { key: string }): Promise<{ success: boolean }>;
  };
};

type AppEnv = {
  Bindings: BrokerBindings & Record<string, unknown>;
  Variables: Record<string, never>;
};

/**
 * Determines whether a URL uses HTTP and targets a loopback host.
 *
 * @param url - The URL to check
 * @returns `true` if the URL uses HTTP and targets localhost or a loopback address, `false` otherwise.
 */
function isLoopbackHttp(url: URL): boolean {
  if (url.protocol !== "http:") return false;
  const host = url.hostname.toLowerCase();
  return host === "localhost" || host === "127.0.0.1" || host === "::1" || host === "[::1]";
}

/**
 * Determines whether an SRAT callback URL is valid.
 *
 * @param raw - The callback URL to validate
 * @returns `true` if `raw` is an absolute HTTPS URL or a loopback HTTP URL, `false` otherwise
 */
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

/**
 * Extracts a bearer token from an authorization header.
 *
 * @param authHeader - The authorization header value, or `null` when absent
 * @returns The trimmed bearer token, or an empty string when the header does not contain one
 */
function bearerTokenFromHeader(authHeader: string | null): string {
  if (!authHeader) return "";
  const m = authHeader.match(/^Bearer\s+(.+)$/i);
  return m ? m[1].trim() : "";
}

/**
 * Compares two strings using timing-safe equality.
 *
 * @param a - The first string to compare
 * @param b - The second string to compare
 * @returns `true` if the strings are equal, `false` otherwise.
 */
function constantTimeEqual(a: string, b: string): boolean {
  const da = createHash("sha256").update(a, "utf8").digest();
  const db = createHash("sha256").update(b, "utf8").digest();
  return timingSafeEqual(da, db);
}

// ---- Rate limiting (in-memory fallback + optional CF binding) ----
type RateLimitConfig = { windowMs: number; limit: number };
const RATE_LIMITS: Record<string, RateLimitConfig> = {
  "/v1/start": { windowMs: 60_000, limit: 20 },
  "/v1/callback": { windowMs: 60_000, limit: 30 },
  "/v1/session": { windowMs: 60_000, limit: 60 },
};

// Shared in-memory buckets for fallback (per isolate). Map key: `${ip}:${route}`
const memoryBuckets = new Map<string, number[]>();

/** For tests: clear rate-limit buckets between isolated app instances. */
export function __clearRateLimitBucketsForTests(): void {
  memoryBuckets.clear();
}

function getClientIp(c: { req: { header: (n: string) => string | undefined } }): string {
  const h = c.req.header.bind(c.req);
  return (
    h("cf-connecting-ip") ||
    h("x-real-ip") ||
    (h("x-forwarded-for") || "").split(",")[0]?.trim() ||
    h("x-broker-client-ip") ||
    "anon"
  );
}

function isAllowedWithMemoryBucket(key: string, cfg: RateLimitConfig): boolean {
  const now = Date.now();
  const bucket = memoryBuckets.get(key) || [];
  const windowStart = now - cfg.windowMs;
  const recent = bucket.filter((t) => t > windowStart);
  if (recent.length >= cfg.limit) {
    memoryBuckets.set(key, recent);
    return false;
  }
  recent.push(now);
  memoryBuckets.set(key, recent);
  // Prevent unbounded growth – prune stale buckets opportunistically
  if (memoryBuckets.size > 10_000) {
    for (const [k, v] of memoryBuckets) {
      if (v.length === 0 || v[v.length - 1]! < windowStart) memoryBuckets.delete(k);
    }
  }
  return true;
}

/**
 * Creates a Hono application that brokers OAuth authorization flows.
 *
 * @param opts - Optional session store, environment overrides, and token-exchange fetch implementation.
 * @returns The configured OAuth broker application.
 */
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
    return { ...(process.env as Record<string, string | undefined>), ...envOverride, ...workerEnv };
  }

  function requireBearer(c: { req: { header: (n: string) => string | undefined }; env: Record<string, unknown> }): boolean {
    const env = getEnv(c);
    const expected = (env.BROKER_API_TOKEN || "").trim();
    if (!expected) {
      // Fail closed: allow unauthenticated only when explicitly opted in for local dev
      const allowInsecure = (env.BROKER_DISABLE_AUTH || "").trim().toLowerCase();
      const insecureEnabled = allowInsecure === "true" || allowInsecure === "1" || allowInsecure === "yes";
      if (insecureEnabled) {
        if (isProductionEnv(env)) {
          // Throw-equivalent: fail closed in production even if flag set – never allow
          console.error("[broker] BROKER_DISABLE_AUTH is not allowed in production (BROKER_PUBLIC_URL looks like production) – denying request");
          return false;
        }
        console.warn("[broker] BROKER_DISABLE_AUTH enabled – auth disabled (dev only)");
        return true;
      }
      return false;
    }
    const got = bearerTokenFromHeader(c.req.header("authorization") || null);
    if (!got) return false;
    return constantTimeEqual(got, expected);
  }

  // ---- Global middleware: CORS (deny by default) + security headers ----
  app.use("*", async (c, next) => {
    // CORS preflight – deny cross-origin by not returning Allow-Origin
    if (c.req.method === "OPTIONS") {
      // Intentionally no Access-Control-Allow-Origin -> browser will block
      return new Response(null, {
        status: 204,
        headers: {
          "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
          "Access-Control-Allow-Headers": "Authorization, Content-Type",
          "Access-Control-Max-Age": "86400",
        },
      });
    }
    await next();
    // Security headers on all responses
    c.header("X-Content-Type-Options", "nosniff");
    c.header("X-Frame-Options", "DENY");
    c.header("Referrer-Policy", "no-referrer");
    c.header("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload");
    c.header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'");
    c.header("X-Robots-Tag", "noindex, nofollow");
    c.header("Permissions-Policy", "camera=(), microphone=(), geolocation=()");
    // Extra cache hardening for sensitive routes
    const path = c.req.path;
    if (path.startsWith("/v1/session") || path.startsWith("/v1/callback") || path.startsWith("/v1/start")) {
      if (!c.res.headers.get("Cache-Control")) c.header("Cache-Control", "no-store, no-cache, must-revalidate");
      c.header("Pragma", "no-cache");
      c.header("Expires", "0");
    }
  });

  // ---- Global rate limiting middleware (Both: CF binding + in-memory fallback) ----
  app.use("*", async (c, next) => {
    const path = c.req.path;
    // Match prefix for /v1/session/:id
    let cfg: RateLimitConfig | undefined;
    if (path === "/v1/start") cfg = RATE_LIMITS["/v1/start"];
    else if (path === "/v1/callback" || path.startsWith("/v1/callback")) cfg = RATE_LIMITS["/v1/callback"];
    else if (path.startsWith("/v1/session")) cfg = RATE_LIMITS["/v1/session"];
    if (!cfg) return next();

    const ip = getClientIp(c);
    const key = `${ip}:${path.split("/").slice(0, 3).join("/")}`; // bucket per ip+route prefix

    // Prefer Cloudflare Rate Limiter binding if present (Both mode)
    const envAny = (c.env || {}) as Record<string, unknown>;
    const rlBinding = (envAny.RATE_LIMITER as BrokerBindings["RATE_LIMITER"]) || (envAny.BROKER_RATE_LIMITER as BrokerBindings["BROKER_RATE_LIMITER"]);
    if (rlBinding && typeof rlBinding.limit === "function") {
      try {
        const res = await rlBinding.limit({ key });
        if (!res.success) {
          c.header("Retry-After", "60");
          return c.json({ error: "rate limit exceeded, retry after 60s" }, 429);
        }
        return next();
      } catch (e) {
        console.warn(`[broker] rate limiter binding error, falling back to memory: ${(e as Error).message}`);
      }
    }

    // Fallback: in-memory sliding window
    if (!isAllowedWithMemoryBucket(key, cfg)) {
      c.header("Retry-After", "60");
      return c.json({ error: "rate limit exceeded, retry after 60s" }, 429);
    }
    return next();
  });

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
    if (sratCallbackUrl.length > MAX_CALLBACK_URL_LENGTH) {
      return c.json({ error: `srat_callback_url too long (max ${MAX_CALLBACK_URL_LENGTH})` }, 400);
    }
    if (!isValidSratCallbackUrl(sratCallbackUrl)) {
      return c.json({ error: "srat_callback_url must be an absolute https URL (loopback http allowed for dev)" }, 400);
    }

    const env = getEnv(c);
    // Optional allowlist: BROKER_ALLOWED_CALLBACK_PATTERNS (glob CSV, e.g. "https://*.srat.example/*")
    const allowlistRaw = env.BROKER_ALLOWED_CALLBACK_PATTERNS?.trim();
    if (allowlistRaw && !isAllowedSratCallbackUrl(sratCallbackUrl, allowlistRaw)) {
      return c.json({ error: "srat_callback_url not allowed by broker policy" }, 403);
    }

    const providers = loadProvidersConfig(env);
    let prov: ReturnType<typeof getProviderOrThrow>;
    try {
      prov = getProviderOrThrow(providers, provider);
    } catch (e) {
      return c.json({ error: (e as Error).message }, 400);
    }

    let publicUrl: string;
    try {
      publicUrl = getBrokerPublicUrlOrThrow(env);
    } catch (e) {
      return c.json({ error: (e as Error).message }, 500);
    }
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

    try {
      await store.set(sessionId, { provider, sratCallbackUrl, createdAt: Date.now() }, ttl);
    } catch (e) {
      // MemoryStore full or other store error
      if ((e as Error).message.includes("session store full")) {
        return c.json({ error: "too many pending sessions, try again later" }, 429);
      }
      throw e;
    }

    c.header("Cache-Control", "no-store, no-cache, must-revalidate");
    return c.json({ auth_url: authUrl.toString(), session_id: sessionId });
  });

  app.get("/v1/callback", async (c) => {
    // Ensure browser/edge never caches the code/state exchange
    c.header("Cache-Control", "no-store, no-cache, must-revalidate");
    c.header("Pragma", "no-cache");
    c.header("Expires", "0");

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

    let publicUrl2: string;
    try {
      publicUrl2 = getBrokerPublicUrlOrThrow(env);
    } catch (e) {
      return c.json({ error: (e as Error).message }, 500);
    }
    const redirectUri = `${publicUrl2}/v1/callback`;

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
        // Log detailed provider error server-side without leaking to browser verbatim (L2)
        const detail = (tokenResp.error_description as string) || (tokenResp.error as string) || `status ${tokenStatus}`;
        console.warn(`[broker] token exchange failed for provider ${session.provider}: ${detail} (status ${tokenStatus})`);
        return c.json({ error: "token exchange failed" }, 502);
      }
    } catch (e) {
      // Network failure – generic to browser, detail server-side
      console.warn(`[broker] token exchange network error: ${(e as Error).message}`);
      return c.json({ error: "token exchange failed" }, 502);
    }

    // Wrap in rclone envelope: {"access_token","token_type","refresh_token","expires_in","expiry","account_id"}
    const expiresIn = Number(tokenResp.expires_in || 0);
    const expiry = expiresIn > 0 ? new Date(Date.now() + expiresIn * 1000).toISOString() : undefined;
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
    // Extra no-store already set above; ensure no leakage via Referrer
    return c.redirect(session.sratCallbackUrl, 302);
  });

  app.get("/v1/session/:id", async (c) => {
    if (!requireBearer(c)) {
      return c.json({ error: "unauthorized" }, 401);
    }
    const id = c.req.param("id");
    const peek = await store.get(id);
    if (!peek) {
      c.header("Cache-Control", "no-store, no-cache, must-revalidate");
      c.header("Pragma", "no-cache");
      c.header("Expires", "0");
      return c.json({ error: "expired or already used" }, 404);
    }
    if (!peek.tokenJson) {
      // Not yet completed — 404 without consuming (early polling cannot destroy flow)
      c.header("Cache-Control", "no-store, no-cache, must-revalidate");
      c.header("Pragma", "no-cache");
      c.header("Expires", "0");
      return c.json({ error: "not yet completed" }, 404);
    }
    // Completed — consume single-use atomically; only one concurrent caller receives the token
    const session = await store.consume(id);
    if (!session || !session.tokenJson) {
      c.header("Cache-Control", "no-store, no-cache, must-revalidate");
      c.header("Pragma", "no-cache");
      c.header("Expires", "0");
      return c.json({ error: "expired or already used" }, 404);
    }
    c.header("Cache-Control", "no-store, no-cache, must-revalidate");
    c.header("Pragma", "no-cache");
    c.header("Expires", "0");
    return c.json({
      token_json: session.tokenJson,
      account_label: session.accountLabel || "",
      client_id: session.clientId || "",
      client_secret: session.clientSecret || "",
    });
  });

  // Catch-all for unknown routes
  app.notFound((c) => {
    c.header("Cache-Control", "no-store");
    return c.json({ error: "not found" }, 404);
  });

  return app;
}
