import { Hono } from "hono";
import type { ContentfulStatusCode } from "hono/utils/http-status";
import { createHash, timingSafeEqual } from "node:crypto";
import {
  getBrokerPublicUrlOrThrow,
  getInstanceTtlSeconds,
  getProviderOrThrow,
  getSessionTtlSeconds,
  isAllowedSratCallbackUrl,
  isProductionEnv,
  loadProvidersConfig,
  MAX_CALLBACK_URL_LENGTH,
  MAX_INSTANCE_ID_LENGTH,
} from "./config.js";
import type { InstanceStore, SessionStore } from "./session.js";
import { MemoryInstanceStore, MemorySessionStore } from "./session.js";
import type { ProviderConfig } from "./types.js";
import { pickLocale, renderHtmlPage, getMessages } from "./i18n.js";

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

// ---- PKCE S256 helpers ----

/**
 * Base64url-encodes bytes without padding.
 * Uses Buffer when available (Node) with fallback to Web APIs for Workers.
 */
export function base64UrlEncode(bytes: Uint8Array): string {
  // Node / Bun: Buffer is fastest and handles base64url natively
  if (typeof Buffer !== "undefined" && typeof Buffer.from === "function") {
    return (Buffer.from(bytes) as unknown as { toString(enc: string): string }).toString("base64url");
  }
  let binary = "";
  for (const b of bytes) binary += String.fromCharCode(b);
  const b64 = btoa(binary);
  return b64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

/**
 * Generates a PKCE code_verifier per RFC 7636 §4.1:
 * 32 random bytes → 43-char base64url string (within 43–128 allowed range,
 * using unreserved characters A-Z / a-z / 0-9 / - / _ / . / ~).
 */
export function generateCodeVerifier(): string {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  return base64UrlEncode(bytes);
}

/**
 * Derives a PKCE S256 code_challenge from a verifier.
 * code_challenge = BASE64URL-ENCODE(SHA256(ASCII(code_verifier)))
 */
export function pkceChallengeFromVerifier(verifier: string): string {
  const hash = createHash("sha256").update(verifier, "utf8").digest();
  return base64UrlEncode(hash);
}

// ---- Rate limiting (in-memory fallback + optional CF binding) ----
type RateLimitConfig = { windowMs: number; limit: number };
const RATE_LIMITS: Record<string, RateLimitConfig> = {
  "/v1/start": { windowMs: 60_000, limit: 20 },
  "/v1/callback": { windowMs: 60_000, limit: 30 },
  "/v1/session": { windowMs: 60_000, limit: 60 },
  "/v1/instances/register": { windowMs: 60_000, limit: 20 },
  "/v1/instances": { windowMs: 60_000, limit: 20 },
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

// ---- Helpers to reduce per-handler complexity (CodeFactor: Complex Method) ----

type StartBody = { provider?: string; srat_callback_url?: string; instance_id?: string };

function validateStartFields(
  provider: string,
  sratCallbackUrl: string,
  instanceId: string,
): { status: 400; error: string } | null {
  if (!provider) return { status: 400, error: "provider is required" };
  if (!sratCallbackUrl) return { status: 400, error: "srat_callback_url is required" };
  if (!instanceId) return { status: 400, error: "instance_id is required" };
  if (instanceId.length > MAX_INSTANCE_ID_LENGTH) {
    return { status: 400, error: `instance_id too long (max ${MAX_INSTANCE_ID_LENGTH})` };
  }
  if (!isValidInstanceId(instanceId)) {
    return { status: 400, error: "instance_id must be alphanumeric with . _ - (1-128 chars)" };
  }
  if (sratCallbackUrl.length > MAX_CALLBACK_URL_LENGTH) {
    return { status: 400, error: `srat_callback_url too long (max ${MAX_CALLBACK_URL_LENGTH})` };
  }
  if (!isValidSratCallbackUrl(sratCallbackUrl)) {
    return {
      status: 400,
      error: "srat_callback_url must be an absolute https URL (loopback http allowed for dev)",
    };
  }
  return null;
}

type RegisterBody = { instance_id?: string; redirect_url?: string };

function isValidInstanceId(raw: string): boolean {
  return /^[A-Za-z0-9._-]{1,128}$/.test(raw);
}

function validateRegisterFields(instanceId: string, redirectUrl: string): { status: 400; error: string } | null {
  if (!instanceId) return { status: 400, error: "instance_id is required" };
  if (instanceId.length > MAX_INSTANCE_ID_LENGTH) {
    return { status: 400, error: `instance_id too long (max ${MAX_INSTANCE_ID_LENGTH})` };
  }
  if (!isValidInstanceId(instanceId)) {
    return { status: 400, error: "instance_id must be alphanumeric with . _ - (1-128 chars)" };
  }
  if (!redirectUrl) return { status: 400, error: "redirect_url is required" };
  if (redirectUrl.length > MAX_CALLBACK_URL_LENGTH) {
    return { status: 400, error: `redirect_url too long (max ${MAX_CALLBACK_URL_LENGTH})` };
  }
  if (!isValidSratCallbackUrl(redirectUrl)) {
    return { status: 400, error: "redirect_url must be an absolute https URL (loopback http allowed for dev)" };
  }
  return null;
}

export function buildAuthUrl(
  prov: ProviderConfig & { authorize_url: string; token_url: string },
  publicUrl: string,
  sessionId: string,
  codeVerifier?: string,
): URL {
  const verifier = codeVerifier ?? generateCodeVerifier();
  const challenge = pkceChallengeFromVerifier(verifier);
  const redirectUri = `${publicUrl}/v1/callback`;
  const authUrl = new URL(prov.authorize_url);
  authUrl.searchParams.set("client_id", prov.client_id);
  authUrl.searchParams.set("response_type", "code");
  authUrl.searchParams.set("redirect_uri", redirectUri);
  authUrl.searchParams.set("state", sessionId);
  authUrl.searchParams.set("code_challenge", challenge);
  authUrl.searchParams.set("code_challenge_method", "S256");
  for (const [k, v] of Object.entries(prov.auth_params || {})) {
    if (!authUrl.searchParams.has(k)) authUrl.searchParams.set(k, v);
  }
  if (prov.scopes && prov.scopes.length > 0) {
    authUrl.searchParams.set("scope", prov.scopes.join(" "));
  }
  return authUrl;
}

function buildRcloneEnvelope(tokenResp: Record<string, unknown>): {
  envelope: Record<string, unknown>;
  tokenJson: string;
  accountLabel: string;
} {
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
  for (const k of Object.keys(envelope)) if (envelope[k] === undefined) delete envelope[k];
  return { envelope, tokenJson: JSON.stringify(envelope), accountLabel: (envelope.account_id as string) || "" };
}

export async function exchangeCodeForToken(
  prov: ProviderConfig & { authorize_url: string; token_url: string },
  code: string,
  redirectUri: string,
  fetchImpl: typeof fetch,
  codeVerifier?: string,
): Promise<{ tokenResp: Record<string, unknown>; tokenStatus: number } | { error: string; status: ContentfulStatusCode }> {
  const formEntries: Record<string, string> = {
    grant_type: "authorization_code",
    code,
    redirect_uri: redirectUri,
    client_id: prov.client_id,
    client_secret: prov.client_secret,
  };
  if (codeVerifier) {
    formEntries.code_verifier = codeVerifier;
  }
  const form = new URLSearchParams(formEntries);

  try {
    const resp = await fetchImpl(prov.token_url, {
      method: "POST",
      headers: { "content-type": "application/x-www-form-urlencoded" },
      body: form.toString(),
    });
    const tokenStatus = resp.status;
    const text = await resp.text();
    let tokenResp: Record<string, unknown>;
    try {
      tokenResp = JSON.parse(text) as Record<string, unknown>;
    } catch {
      return { error: `invalid token response (status ${tokenStatus})`, status: 502 };
    }
    if (!resp.ok || !tokenResp.access_token) {
      const detail =
        (tokenResp.error_description as string) || (tokenResp.error as string) || `status ${tokenStatus}`;
      console.warn(`[broker] token exchange failed: ${detail} (status ${tokenStatus})`);
      return { error: "token exchange failed", status: 502 };
    }
    return { tokenResp, tokenStatus };
  } catch (e) {
    console.warn(`[broker] token exchange network error: ${(e as Error).message}`);
    return { error: "token exchange failed", status: 502 };
  }
}

function setNoStore(c: { header: (k: string, v: string) => void }): void {
  c.header("Cache-Control", "no-store, no-cache, must-revalidate");
  c.header("Pragma", "no-cache");
  c.header("Expires", "0");
}

/**
 * Creates a Hono application that brokers OAuth authorization flows.
 *
 * @param opts - Optional session store, environment overrides, and token-exchange fetch implementation.
 * @returns The configured OAuth broker application.
 */
export function createBrokerApp(opts?: {
  store?: SessionStore;
  instanceStore?: InstanceStore;
  env?: Record<string, string | undefined>;
  fetchImpl?: typeof fetch;
}) {
  const app = new Hono<AppEnv>();
  const store: SessionStore = opts?.store ?? new MemorySessionStore();
  const instanceStore: InstanceStore = opts?.instanceStore ?? new MemoryInstanceStore();
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
          console.error(
            "[broker] BROKER_DISABLE_AUTH is not allowed in production (BROKER_PUBLIC_URL looks like production) – denying request",
          );
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
    if (
      path.startsWith("/v1/session") ||
      path.startsWith("/v1/callback") ||
      path.startsWith("/v1/start") ||
      path.startsWith("/v1/instances")
    ) {
      if (!c.res.headers.get("Cache-Control")) c.header("Cache-Control", "no-store, no-cache, must-revalidate");
      c.header("Pragma", "no-cache");
      c.header("Expires", "0");
    }
  });

  // ---- Global rate limiting middleware (Both: CF binding + in-memory fallback) ----
  app.use("*", async (c, next) => {
    const path = c.req.path;
    // Match prefix for /v1/session/:id and /v1/instances
    let cfg: RateLimitConfig | undefined;
    if (path === "/v1/start") cfg = RATE_LIMITS["/v1/start"];
    else if (path === "/v1/callback" || path.startsWith("/v1/callback")) cfg = RATE_LIMITS["/v1/callback"];
    else if (path.startsWith("/v1/session")) cfg = RATE_LIMITS["/v1/session"];
    else if (path.startsWith("/v1/instances")) cfg = RATE_LIMITS["/v1/instances/register"] ?? RATE_LIMITS["/v1/instances"];
    if (!cfg) return next();

    const ip = getClientIp(c);
    const key = `${ip}:${path.split("/").slice(0, 3).join("/")}`; // bucket per ip+route prefix

    // Prefer Cloudflare Rate Limiter binding if present (Both mode)
    const envAny = (c.env || {}) as Record<string, unknown>;
    const rlBinding =
      (envAny.RATE_LIMITER as BrokerBindings["RATE_LIMITER"]) ||
      (envAny.BROKER_RATE_LIMITER as BrokerBindings["BROKER_RATE_LIMITER"]);
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

  // Instance registration – must be called before /v1/start, TTL 1h
  app.post("/v1/instances/register", async (c) => {
    if (!requireBearer(c)) {
      return c.json({ error: "unauthorized" }, 401);
    }
    let body: RegisterBody;
    try {
      body = await c.req.json();
    } catch {
      return c.json({ error: "invalid json body" }, 400);
    }
    const instanceId = (body.instance_id || "").trim();
    const redirectUrl = (body.redirect_url || "").trim();
    const fieldErr = validateRegisterFields(instanceId, redirectUrl);
    if (fieldErr) return c.json({ error: fieldErr.error }, fieldErr.status);

    const env = getEnv(c);
    const allowlistRaw = env.BROKER_ALLOWED_CALLBACK_PATTERNS?.trim();
    if (allowlistRaw && !isAllowedSratCallbackUrl(redirectUrl, allowlistRaw)) {
      return c.json({ error: "redirect_url not allowed by broker policy" }, 403);
    }

    const ttl = getInstanceTtlSeconds(env);
    try {
      await instanceStore.set(instanceId, { instanceId, redirectUrl, createdAt: Date.now() }, ttl);
    } catch (e) {
      if ((e as Error).message.includes("store full")) {
        return c.json({ error: "too many pending instances, try again later" }, 429);
      }
      throw e;
    }
    const expiresAt = Date.now() + ttl * 1000;
    c.header("Cache-Control", "no-store, no-cache, must-revalidate");
    return c.json({ instance_id: instanceId, redirect_url: redirectUrl, expires_at: new Date(expiresAt).toISOString(), ttl_seconds: ttl });
  });

  app.post("/v1/start", async (c) => {
    if (!requireBearer(c)) {
      return c.json({ error: "unauthorized" }, 401);
    }
    let body: StartBody;
    try {
      body = await c.req.json();
    } catch {
      return c.json({ error: "invalid json body" }, 400);
    }
    const provider = (body.provider || "").trim();
    const sratCallbackUrl = (body.srat_callback_url || "").trim();
    const instanceId = (body.instance_id || "").trim();

    const fieldErr = validateStartFields(provider, sratCallbackUrl, instanceId);
    if (fieldErr) return c.json({ error: fieldErr.error }, fieldErr.status);

    const env = getEnv(c);
    // Instance must be registered and not expired
    const inst = await instanceStore.get(instanceId);
    if (!inst) {
      return c.json({ error: "instance not registered or expired – register via POST /v1/instances/register first" }, 410);
    }
    // Exact match required per spec
    if (sratCallbackUrl !== inst.redirectUrl) {
      return c.json({ error: "srat_callback_url does not match registered instance redirect_url (exact match required)" }, 403);
    }

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

    const sessionId = crypto.randomUUID();
    const ttl = getSessionTtlSeconds(env);
    const codeVerifier = generateCodeVerifier();
    const authUrl = buildAuthUrl(prov, publicUrl, sessionId, codeVerifier);

    try {
      await store.set(sessionId, { provider, sratCallbackUrl, createdAt: Date.now(), codeVerifier, instanceId }, ttl);
    } catch (e) {
      if ((e as Error).message.includes("session store full")) {
        return c.json({ error: "too many pending sessions, try again later" }, 429);
      }
      throw e;
    }

    c.header("Cache-Control", "no-store, no-cache, must-revalidate");
    return c.json({ auth_url: authUrl.toString(), session_id: sessionId });
  });

  function wantsHtml(c: { req: { header: (n: string) => string | undefined } }): boolean {
    const accept = (c.req.header("accept") || "").toLowerCase();
    // Browser navigations always accept html; API tests send no accept or json.
    // Treat missing accept as html for callback (provider redirect is browser), but keep json for errors when explicitly json.
    if (!accept) return true;
    if (accept.includes("text/html")) return true;
    if (accept.includes("application/json")) return false;
    return true;
  }

  app.get("/v1/callback", async (c) => {
    setNoStore(c);
    const locale = pickLocale(c.req.header("accept-language"));
    const msgs = getMessages(locale);

    const code = c.req.query("code") || "";
    const state = c.req.query("state") || "";
    if (!code || !state) {
      const err = msgs.errorInvalidRequest;
      if (wantsHtml(c)) {
        const html = renderHtmlPage({ locale, success: false, errorMessage: err });
        c.header("Content-Type", "text/html; charset=utf-8");
        return c.html(html, 400 as ContentfulStatusCode);
      }
      return c.json({ error: "missing code or state" }, 400);
    }

    const env = getEnv(c);
    const ttl = getSessionTtlSeconds(env);
    const session = await store.get(state);
    if (!session) {
      const err = msgs.errorSessionExpired;
      if (wantsHtml(c)) {
        const html = renderHtmlPage({ locale, success: false, errorMessage: err });
        c.header("Content-Type", "text/html; charset=utf-8");
        return c.html(html, 410 as ContentfulStatusCode);
      }
      return c.json({ error: "session not found or expired" }, 410);
    }
    if (session.tokenJson) {
      const err = msgs.errorSessionExpired;
      if (wantsHtml(c)) {
        const html = renderHtmlPage({ locale, success: false, errorMessage: err });
        c.header("Content-Type", "text/html; charset=utf-8");
        return c.html(html, 410 as ContentfulStatusCode);
      }
      return c.json({ error: "session not found or expired" }, 410);
    }

    // Validate instance binding if present (hardened flow). Legacy sessions without instanceId are rejected.
    if (!session.instanceId) {
      const err = msgs.errorInstanceNotFound;
      if (wantsHtml(c)) {
        const html = renderHtmlPage({ locale, success: false, errorMessage: err });
        c.header("Content-Type", "text/html; charset=utf-8");
        return c.html(html, 410 as ContentfulStatusCode);
      }
      return c.json({ error: "instance not found or expired" }, 410);
    }
    const inst = await instanceStore.get(session.instanceId);
    if (!inst) {
      const err = msgs.errorInstanceNotFound;
      if (wantsHtml(c)) {
        const html = renderHtmlPage({ locale, success: false, errorMessage: err });
        c.header("Content-Type", "text/html; charset=utf-8");
        return c.html(html, 410 as ContentfulStatusCode);
      }
      return c.json({ error: "instance not found or expired" }, 410);
    }
    if (session.sratCallbackUrl !== inst.redirectUrl) {
      const err = msgs.errorRedirectMismatch;
      if (wantsHtml(c)) {
        const html = renderHtmlPage({ locale, success: false, errorMessage: err });
        c.header("Content-Type", "text/html; charset=utf-8");
        return c.html(html, 403 as ContentfulStatusCode);
      }
      return c.json({ error: "redirect_url mismatch for instance" }, 403);
    }

    const providers = loadProvidersConfig(env);
    let prov: ReturnType<typeof getProviderOrThrow>;
    try {
      prov = getProviderOrThrow(providers, session.provider);
    } catch (e) {
      const err = (e as Error).message;
      if (wantsHtml(c)) {
        const html = renderHtmlPage({ locale, success: false, errorMessage: err });
        c.header("Content-Type", "text/html; charset=utf-8");
        return c.html(html, 400 as ContentfulStatusCode);
      }
      return c.json({ error: err }, 400);
    }

    let publicUrl2: string;
    try {
      publicUrl2 = getBrokerPublicUrlOrThrow(env);
    } catch (e) {
      const err = (e as Error).message;
      if (wantsHtml(c)) {
        const html = renderHtmlPage({ locale, success: false, errorMessage: err });
        c.header("Content-Type", "text/html; charset=utf-8");
        return c.html(html, 500 as ContentfulStatusCode);
      }
      return c.json({ error: err }, 500);
    }
    const redirectUri = `${publicUrl2}/v1/callback`;

    const exchange = await exchangeCodeForToken(prov, code, redirectUri, fetchImpl, session.codeVerifier);
    if ("error" in exchange) {
      const errMsg = msgs.errorTokenFailed;
      if (wantsHtml(c)) {
        const html = renderHtmlPage({ locale, success: false, errorMessage: errMsg });
        c.header("Content-Type", "text/html; charset=utf-8");
        return c.html(html, exchange.status as ContentfulStatusCode);
      }
      if (exchange.error.startsWith("invalid token response")) {
        return c.json({ error: exchange.error }, exchange.status as ContentfulStatusCode);
      }
      return c.json({ error: exchange.error }, exchange.status as ContentfulStatusCode);
    }

    const { tokenJson, accountLabel } = buildRcloneEnvelope(exchange.tokenResp);

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
      ttl,
    );

    // Success: render broker-owned HTML page that auto-redirects to the validated HA redirect_url
    // Keep token server-side (consumed via GET /v1/session/:id); this page is just UX + validation proof.
    if (wantsHtml(c)) {
      const html = renderHtmlPage({ locale, success: true, redirectUrl: session.sratCallbackUrl });
      c.header("Content-Type", "text/html; charset=utf-8");
      // No-store already set, but ensure html
      return c.html(html, 200 as ContentfulStatusCode);
    }
    return c.redirect(session.sratCallbackUrl, 302);
  });

  app.get("/v1/session/:id", async (c) => {
    if (!requireBearer(c)) {
      return c.json({ error: "unauthorized" }, 401);
    }
    const id = c.req.param("id");
    const peek = await store.get(id);
    if (!peek) {
      setNoStore(c);
      return c.json({ error: "expired or already used" }, 404);
    }
    if (!peek.tokenJson) {
      setNoStore(c);
      return c.json({ error: "not yet completed" }, 404);
    }
    const session = await store.consume(id);
    if (!session || !session.tokenJson) {
      setNoStore(c);
      return c.json({ error: "expired or already used" }, 404);
    }
    setNoStore(c);
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
