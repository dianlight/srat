import { Hono } from "hono";
import type { ContentfulStatusCode } from "hono/utils/http-status";
import { createHash } from "node:crypto";
import {
  CLOCK_SKEW_SECONDS,
  getBrokerPublicUrlOrThrow,
  getInstanceTtlSeconds,
  getProviderOrThrow,
  getSessionTtlSeconds,
  isAllowedSratCallbackUrl,
  isProductionEnv,
  loadProvidersConfig,
  MAX_CALLBACK_URL_LENGTH,
  MAX_INSTANCE_ID_LENGTH,
  NONCE_TTL_SECONDS,
} from "./config.js";
import {
  bodyHashBase64Url,
  buildStringToSign,
  computeClientId,
  isValidClientId,
  isValidNonce,
  isValidPublicKeyB64Url,
  parseSratSignature,
  verifyEd25519,
} from "./crypto.js";
import type { ClientStore, InstanceStore, NonceStore, SessionStore } from "./session.js";
import { MemoryClientStore, MemoryInstanceStore, MemoryNonceStore, MemorySessionStore } from "./session.js";
import type { ProviderConfig } from "./types.js";
import { getMessages, pickLocale, renderHtmlPage } from "./i18n.js";

export type BrokerBindings = {
  OAUTH_SESSIONS?: {
    get(key: string, opts?: { type: string }): Promise<string | null>;
    put(key: string, value: string, opts?: { expirationTtl?: number }): Promise<void>;
    delete(key: string): Promise<void>;
  };
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

export function base64UrlEncode(bytes: Uint8Array): string {
  if (typeof Buffer !== "undefined" && typeof Buffer.from === "function") {
    return (Buffer.from(bytes) as unknown as { toString(enc: string): string }).toString("base64url");
  }
  let binary = "";
  for (const b of bytes) binary += String.fromCharCode(b);
  const b64 = btoa(binary);
  return b64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function generateCodeVerifier(): string {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  return base64UrlEncode(bytes);
}

export function pkceChallengeFromVerifier(verifier: string): string {
  const hash = createHash("sha256").update(verifier, "utf8").digest();
  return base64UrlEncode(hash);
}

// ---- Rate limiting ----
type RateLimitConfig = { windowMs: number; limit: number };
const RATE_LIMITS: Record<string, RateLimitConfig> = {
  "/v1/start": { windowMs: 60_000, limit: 20 },
  "/v1/callback": { windowMs: 60_000, limit: 30 },
  "/v1/session": { windowMs: 60_000, limit: 60 },
  "/v1/instances/register": { windowMs: 60_000, limit: 20 },
  "/v1/instances": { windowMs: 60_000, limit: 20 },
  "/v1/clients": { windowMs: 60_000, limit: 20 },
};

const memoryBuckets = new Map<string, number[]>();
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
  if (memoryBuckets.size > 10_000) {
    for (const [k, v] of memoryBuckets) {
      if (v.length === 0 || v[v.length - 1]! < windowStart) memoryBuckets.delete(k);
    }
  }
  return true;
}

// ---- Validation helpers ----
type StartBody = { provider?: string; srat_callback_url?: string; instance_id?: string };

function validateStartFields(provider: string, sratCallbackUrl: string, instanceId: string): { status: 400; error: string } | null {
  if (!provider) return { status: 400, error: "provider is required" };
  if (!sratCallbackUrl) return { status: 400, error: "srat_callback_url is required" };
  if (!instanceId) return { status: 400, error: "instance_id is required" };
  if (instanceId.length > MAX_INSTANCE_ID_LENGTH) return { status: 400, error: `instance_id too long (max ${MAX_INSTANCE_ID_LENGTH})` };
  if (!isValidInstanceId(instanceId)) return { status: 400, error: "instance_id must be alphanumeric with . _ - (1-128 chars)" };
  if (sratCallbackUrl.length > MAX_CALLBACK_URL_LENGTH) return { status: 400, error: `srat_callback_url too long (max ${MAX_CALLBACK_URL_LENGTH})` };
  if (!isValidSratCallbackUrl(sratCallbackUrl)) return { status: 400, error: "srat_callback_url must be an absolute https URL (loopback http allowed for dev)" };
  return null;
}

type RegisterBody = { instance_id?: string; redirect_url?: string };
type ClientRegisterBody = { client_id?: string; public_key?: string };

function isValidInstanceId(raw: string): boolean {
  return /^[A-Za-z0-9._-]{1,128}$/.test(raw);
}

function validateRegisterFields(instanceId: string, redirectUrl: string): { status: 400; error: string } | null {
  if (!instanceId) return { status: 400, error: "instance_id is required" };
  if (instanceId.length > MAX_INSTANCE_ID_LENGTH) return { status: 400, error: `instance_id too long (max ${MAX_INSTANCE_ID_LENGTH})` };
  if (!isValidInstanceId(instanceId)) return { status: 400, error: "instance_id must be alphanumeric with . _ - (1-128 chars)" };
  if (!redirectUrl) return { status: 400, error: "redirect_url is required" };
  if (redirectUrl.length > MAX_CALLBACK_URL_LENGTH) return { status: 400, error: `redirect_url too long (max ${MAX_CALLBACK_URL_LENGTH})` };
  if (!isValidSratCallbackUrl(redirectUrl)) return { status: 400, error: "redirect_url must be an absolute https URL (loopback http allowed for dev)" };
  return null;
}

function validateClientRegisterFields(clientId: string, publicKey: string): { status: 400; error: string } | null {
  if (!clientId) return { status: 400, error: "client_id is required" };
  if (!isValidClientId(clientId)) return { status: 400, error: "client_id must be 43-char base64url (SHA256 of public key)" };
  if (!publicKey) return { status: 400, error: "public_key is required" };
  if (!isValidPublicKeyB64Url(publicKey)) return { status: 400, error: "public_key must be 32-byte base64url Ed25519 key" };
  const computed = computeClientId(publicKey);
  if (computed !== clientId) return { status: 400, error: "client_id does not match SHA256(public_key)" };
  return null;
}

export function buildAuthUrl(prov: ProviderConfig & { authorize_url: string; token_url: string }, publicUrl: string, sessionId: string, codeVerifier?: string): URL {
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
  if (prov.scopes && prov.scopes.length > 0) authUrl.searchParams.set("scope", prov.scopes.join(" "));
  return authUrl;
}

function buildRcloneEnvelope(tokenResp: Record<string, unknown>): { envelope: Record<string, unknown>; tokenJson: string; accountLabel: string } {
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
  const formEntries: Record<string, string> = { grant_type: "authorization_code", code, redirect_uri: redirectUri, client_id: prov.client_id, client_secret: prov.client_secret };
  if (codeVerifier) formEntries.code_verifier = codeVerifier;
  const form = new URLSearchParams(formEntries);
  try {
    const resp = await fetchImpl(prov.token_url, { method: "POST", headers: { "content-type": "application/x-www-form-urlencoded" }, body: form.toString() });
    const tokenStatus = resp.status;
    const text = await resp.text();
    let tokenResp: Record<string, unknown>;
    try {
      tokenResp = JSON.parse(text) as Record<string, unknown>;
    } catch {
      return { error: `invalid token response (status ${tokenStatus})`, status: 502 };
    }
    if (!resp.ok || !tokenResp.access_token) {
      const detail = (tokenResp.error_description as string) || (tokenResp.error as string) || `status ${tokenStatus}`;
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

export function createBrokerApp(opts?: {
  store?: SessionStore;
  instanceStore?: InstanceStore;
  clientStore?: ClientStore;
  nonceStore?: NonceStore;
  env?: Record<string, string | undefined>;
  fetchImpl?: typeof fetch;
}) {
  const app = new Hono<AppEnv>();
  const store: SessionStore = opts?.store ?? new MemorySessionStore();
  const instanceStore: InstanceStore = opts?.instanceStore ?? new MemoryInstanceStore();
  const clientStore: ClientStore = opts?.clientStore ?? new MemoryClientStore();
  const nonceStore: NonceStore = opts?.nonceStore ?? new MemoryNonceStore();
  const envOverride = opts?.env;
  const fetchImpl: typeof fetch = opts?.fetchImpl ?? fetch;

  function getEnv(c: { env: Record<string, unknown> }): Record<string, string | undefined> {
    const workerEnv = (c.env || {}) as Record<string, string | undefined>;
    return { ...(process.env as Record<string, string | undefined>), ...envOverride, ...workerEnv };
  }

  function isAuthDisabled(env: Record<string, string | undefined>): boolean {
    const raw = (env.BROKER_DISABLE_AUTH || "").trim().toLowerCase();
    const enabled = raw === "true" || raw === "1" || raw === "yes";
    if (enabled && isProductionEnv(env)) {
      console.error("[broker] BROKER_DISABLE_AUTH is not allowed in production – denying request");
      return false;
    }
    if (enabled) console.warn("[broker] BROKER_DISABLE_AUTH enabled – auth disabled (dev only)");
    return enabled;
  }

  async function verifySratSignature(c: { req: { header: (n: string) => string | undefined; method: string; path: string } }, rawBody: string): Promise<{ ok: true; clientId: string } | { ok: false; error: string; status: 401 | 400 }> {
    const env = getEnv(c as unknown as { env: Record<string, unknown> });
    if (isAuthDisabled(env)) return { ok: true, clientId: "dev-bypass" };
    const header = c.req.header("authorization") || c.req.header("Authorization");
    const parsed = parseSratSignature(header);
    if (!parsed) return { ok: false, error: "missing or malformed SRAT-Signature (expected SRAT-Signature client_id=\"...\", t=\"...\", nonce=\"...\", sig=\"...\")", status: 401 };
    const { clientId, t, nonce, sig } = parsed;
    if (!isValidClientId(clientId)) return { ok: false, error: "invalid client_id", status: 401 };
    if (!isValidNonce(nonce)) return { ok: false, error: "invalid nonce", status: 400 };
    const tNum = Number(t);
    if (!Number.isFinite(tNum) || tNum <= 0) return { ok: false, error: "invalid t (unix seconds)", status: 400 };
    const nowSec = Math.floor(Date.now() / 1000);
    if (Math.abs(nowSec - tNum) > CLOCK_SKEW_SECONDS) return { ok: false, error: "t outside clock skew window", status: 401 };
    const client = await clientStore.get(clientId);
    if (!client) return { ok: false, error: "unknown client_id – register via POST /v1/clients first", status: 401 };
    // BodyHash on raw wire bytes (rawBody is c.req.text() untouched); empty string for GET.
    // Do not re-serialize JSON – signing must use exact bytes sent.
    const bodyHash = bodyHashBase64Url(rawBody);
    const path = c.req.path;
    const method = c.req.method;
    const stringToSign = buildStringToSign(clientId, method, path, t, nonce, bodyHash);
    const valid = await verifyEd25519(client.publicKey, stringToSign, sig);
    if (!valid) return { ok: false, error: "invalid signature", status: 401 };
    // Atomic replay gate: set() returns false if nonce already present (D1 PK or Memory map).
    // This is the sole replay check – has() before verify was removed to avoid TOCTOU.
    const inserted = await nonceStore.set(nonce, NONCE_TTL_SECONDS);
    if (!inserted) return { ok: false, error: "nonce already used (replay)", status: 401 };
    return { ok: true, clientId };
  }

  // ---- Global middleware: CORS + security headers ----
  app.use("*", async (c, next) => {
    if (c.req.method === "OPTIONS") {
      return new Response(null, { status: 204, headers: { "Access-Control-Allow-Methods": "GET, POST, OPTIONS", "Access-Control-Allow-Headers": "Authorization, Content-Type", "Access-Control-Max-Age": "86400" } });
    }
    await next();
    c.header("X-Content-Type-Options", "nosniff");
    c.header("X-Frame-Options", "DENY");
    c.header("Referrer-Policy", "no-referrer");
    c.header("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload");
    c.header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'");
    c.header("X-Robots-Tag", "noindex, nofollow");
    c.header("Permissions-Policy", "camera=(), microphone=(), geolocation=()");
    const path = c.req.path;
    if (path.startsWith("/v1/session") || path.startsWith("/v1/callback") || path.startsWith("/v1/start") || path.startsWith("/v1/instances") || path.startsWith("/v1/clients")) {
      if (!c.res.headers.get("Cache-Control")) c.header("Cache-Control", "no-store, no-cache, must-revalidate");
      c.header("Pragma", "no-cache");
      c.header("Expires", "0");
    }
  });

  app.use("*", async (c, next) => {
    const path = c.req.path;
    let cfg: RateLimitConfig | undefined;
    if (path === "/v1/start") cfg = RATE_LIMITS["/v1/start"];
    else if (path === "/v1/callback" || path.startsWith("/v1/callback")) cfg = RATE_LIMITS["/v1/callback"];
    else if (path.startsWith("/v1/session")) cfg = RATE_LIMITS["/v1/session"];
    else if (path.startsWith("/v1/instances")) cfg = RATE_LIMITS["/v1/instances/register"] ?? RATE_LIMITS["/v1/instances"];
    else if (path.startsWith("/v1/clients")) cfg = RATE_LIMITS["/v1/clients"];
    if (!cfg) return next();
    const ip = getClientIp(c);
    const key = `${ip}:${path.split("/").slice(0, 3).join("/")}`;
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
    if (!isAllowedWithMemoryBucket(key, cfg)) {
      c.header("Retry-After", "60");
      return c.json({ error: "rate limit exceeded, retry after 60s" }, 429);
    }
    return next();
  });

  app.get("/v1/healthz", (c) => {
    const env = getEnv(c);
    const providers = loadProvidersConfig(env);
    return c.json({ status: "ok", providers: Object.keys(providers) });
  });
  app.get("/healthz", (c) => {
    const env = getEnv(c);
    const providers = loadProvidersConfig(env);
    return c.json({ status: "ok", providers: Object.keys(providers) });
  });

  // ---- Client registration (public, no sig – self-certifying) ----
  app.post("/v1/clients", async (c) => {
    let body: ClientRegisterBody;
    try {
      body = await c.req.json();
    } catch {
      return c.json({ error: "invalid json body" }, 400);
    }
    const clientId = (body.client_id || "").trim();
    const publicKey = (body.public_key || "").trim();
    const fieldErr = validateClientRegisterFields(clientId, publicKey);
    if (fieldErr) return c.json({ error: fieldErr.error }, fieldErr.status);
    const existing = await clientStore.get(clientId);
    if (existing) {
      if (existing.publicKey !== publicKey) return c.json({ error: "client_id already registered with different public_key (rotate via POST /v1/clients/:id/rotate)" }, 409);
      c.header("Cache-Control", "no-store, no-cache, must-revalidate");
      return c.json({ client_id: clientId, public_key: publicKey, created_at: new Date(existing.createdAt).toISOString() });
    }
    const rec = { clientId, publicKey, createdAt: Date.now() };
    await clientStore.set(clientId, rec);
    c.header("Cache-Control", "no-store, no-cache, must-revalidate");
    return c.json({ client_id: clientId, public_key: publicKey, created_at: new Date(rec.createdAt).toISOString() }, 201);
  });

  // Rotate placeholder – for now return 501, actual rotate via signature will be added later
  app.post("/v1/clients/:id/rotate", async (c) => {
    return c.json({ error: "not implemented – delete and re-register for now" }, 501);
  });

  // Delete client (manual revocation) – deletes client and its instances
  app.delete("/v1/clients/:id", async (c) => {
    const rawBody = "";
    const auth = await verifySratSignature(c, rawBody);
    if (!auth.ok) return c.json({ error: auth.error }, auth.status);
    const id = c.req.param("id");
    if (!isValidClientId(id)) return c.json({ error: "invalid client_id" }, 400);
    const existing = await clientStore.get(id);
    if (!existing) return c.json({ error: "client not found" }, 404);
    if (id !== auth.clientId) return c.json({ error: "client_id mismatch: can only delete own client" }, 403);
    await clientStore.delete(id);
    if (instanceStore.deleteByClientId) {
      await instanceStore.deleteByClientId(id);
    }
    c.header("Cache-Control", "no-store, no-cache, must-revalidate");
    return c.json({ deleted: true, client_id: id });
  });

  app.delete("/v1/clients", async (c) => {
    const rawBody = "";
    const auth = await verifySratSignature(c, rawBody);
    if (!auth.ok) return c.json({ error: auth.error }, auth.status);
    const id = auth.clientId;
    const existing = await clientStore.get(id);
    if (!existing) return c.json({ error: "client not found" }, 404);
    await clientStore.delete(id);
    if (instanceStore.deleteByClientId) {
      await instanceStore.deleteByClientId(id);
    }
    c.header("Cache-Control", "no-store, no-cache, must-revalidate");
    return c.json({ deleted: true, client_id: id });
  });

  // Instance registration – signed, bound to client_id
  app.post("/v1/instances/register", async (c) => {
    const rawBody = await c.req.text();
    const auth = await verifySratSignature(c, rawBody);
    if (!auth.ok) return c.json({ error: auth.error }, auth.status);
    let body: RegisterBody;
    try {
      body = rawBody ? (JSON.parse(rawBody) as RegisterBody) : ({} as RegisterBody);
    } catch {
      return c.json({ error: "invalid json body" }, 400);
    }
    const instanceId = (body.instance_id || "").trim();
    const redirectUrl = (body.redirect_url || "").trim();
    const fieldErr = validateRegisterFields(instanceId, redirectUrl);
    if (fieldErr) return c.json({ error: fieldErr.error }, fieldErr.status);
    const env = getEnv(c);
    const allowlistRaw = env.BROKER_ALLOWED_CALLBACK_PATTERNS?.trim();
    if (allowlistRaw && !isAllowedSratCallbackUrl(redirectUrl, allowlistRaw)) return c.json({ error: "redirect_url not allowed by broker policy" }, 403);
    // Enforce that instance is owned by signer: if instance exists and owned by different client, reject
    const existingInst = await instanceStore.get(instanceId);
    if (existingInst && existingInst.clientId !== auth.clientId) return c.json({ error: "instance_id already registered to different client_id" }, 403);
    const ttl = getInstanceTtlSeconds(env);
    try {
      await instanceStore.set(instanceId, { instanceId, redirectUrl, createdAt: Date.now(), clientId: auth.clientId }, ttl);
    } catch (e) {
      if ((e as Error).message.includes("store full")) return c.json({ error: "too many pending instances, try again later" }, 429);
      throw e;
    }
    const expiresAt = Date.now() + ttl * 1000;
    c.header("Cache-Control", "no-store, no-cache, must-revalidate");
    return c.json({ instance_id: instanceId, redirect_url: redirectUrl, client_id: auth.clientId, expires_at: new Date(expiresAt).toISOString(), ttl_seconds: ttl });
  });

  // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: start validates auth/instance/allowlist/provider and creates PKCE session
  app.post("/v1/start", async (c) => {
    const rawBody = await c.req.text();
    const auth = await verifySratSignature(c, rawBody);
    if (!auth.ok) return c.json({ error: auth.error }, auth.status);
    let body: StartBody;
    try {
      body = rawBody ? (JSON.parse(rawBody) as StartBody) : ({} as StartBody);
    } catch {
      return c.json({ error: "invalid json body" }, 400);
    }
    const provider = (body.provider || "").trim();
    const sratCallbackUrl = (body.srat_callback_url || "").trim();
    const instanceId = (body.instance_id || "").trim();
    const fieldErr = validateStartFields(provider, sratCallbackUrl, instanceId);
    if (fieldErr) return c.json({ error: fieldErr.error }, fieldErr.status);
    const env = getEnv(c);
    const inst = await instanceStore.get(instanceId);
    if (!inst) return c.json({ error: "instance not registered or expired – register via POST /v1/instances/register first" }, 410);
    if (inst.clientId !== auth.clientId) return c.json({ error: "instance not owned by authenticated client_id" }, 403);
    if (sratCallbackUrl !== inst.redirectUrl) return c.json({ error: "srat_callback_url does not match registered instance redirect_url (exact match required)" }, 403);
    const allowlistRaw = env.BROKER_ALLOWED_CALLBACK_PATTERNS?.trim();
    if (allowlistRaw && !isAllowedSratCallbackUrl(sratCallbackUrl, allowlistRaw)) return c.json({ error: "srat_callback_url not allowed by broker policy" }, 403);
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
      await store.set(sessionId, { provider, sratCallbackUrl, createdAt: Date.now(), codeVerifier, instanceId, ownerClientId: auth.clientId }, ttl);
    } catch (e) {
      if ((e as Error).message.includes("session store full")) return c.json({ error: "too many pending sessions, try again later" }, 429);
      throw e;
    }
    c.header("Cache-Control", "no-store, no-cache, must-revalidate");
    return c.json({ auth_url: authUrl.toString(), session_id: sessionId });
  });

  function wantsHtml(c: { req: { header: (n: string) => string | undefined } }): boolean {
    const accept = (c.req.header("accept") || "").toLowerCase();
    if (!accept) return true;
    if (accept.includes("text/html")) return true;
    if (accept.includes("application/json")) return false;
    return true;
  }

  function callbackErrorResponse(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any -- Hono Context type varies by route; helper is intentionally generic
    c: any,
    locale: string,
    status: ContentfulStatusCode,
    htmlMessage: string,
    jsonError: string,
  ): Response | Promise<Response> {
    if (wantsHtml(c)) {
      const html = renderHtmlPage({ locale: locale as unknown as import("./i18n.js").Locale, success: false, errorMessage: htmlMessage });
      c.header("Content-Type", "text/html; charset=utf-8");
      return c.html(html, status);
    }
    return c.json({ error: jsonError }, status);
  }

  // biome-ignore lint/complexity/noExcessiveCognitiveComplexity: OAuth callback orchestrates session/instance/token states with HTML/JSON branching
  app.get("/v1/callback", async (c) => {
    setNoStore(c);
    const locale = pickLocale(c.req.header("accept-language"));
    const msgs = getMessages(locale);
    const code = c.req.query("code") || "";
    const state = c.req.query("state") || "";
    if (!code || !state) {
      return callbackErrorResponse(c, locale, 400, msgs.errorInvalidRequest, "missing code or state");
    }
    const env = getEnv(c);
    const ttl = getSessionTtlSeconds(env);
    const session = await store.get(state);
    if (!session) {
      return callbackErrorResponse(c, locale, 410, msgs.errorSessionExpired, "session not found or expired");
    }
    if (session.tokenJson) {
      return callbackErrorResponse(c, locale, 410, msgs.errorSessionExpired, "session not found or expired");
    }
    if (!session.instanceId) {
      return callbackErrorResponse(c, locale, 410, msgs.errorInstanceNotFound, "instance not found or expired");
    }
    const inst = await instanceStore.get(session.instanceId);
    if (!inst) {
      return callbackErrorResponse(c, locale, 410, msgs.errorInstanceNotFound, "instance not found or expired");
    }
    if (session.sratCallbackUrl !== inst.redirectUrl) {
      return callbackErrorResponse(c, locale, 403, msgs.errorRedirectMismatch, "redirect_url mismatch for instance");
    }
    const providers = loadProvidersConfig(env);
    let prov: ReturnType<typeof getProviderOrThrow>;
    try {
      prov = getProviderOrThrow(providers, session.provider);
    } catch (e) {
      const err = (e as Error).message;
      return callbackErrorResponse(c, locale, 400, err, err);
    }
    let publicUrl2: string;
    try {
      publicUrl2 = getBrokerPublicUrlOrThrow(env);
    } catch (e) {
      const err = (e as Error).message;
      return callbackErrorResponse(c, locale, 500, err, err);
    }
    const redirectUri = `${publicUrl2}/v1/callback`;
    const exchange = await exchangeCodeForToken(prov, code, redirectUri, fetchImpl, session.codeVerifier);
    if ("error" in exchange) {
      return callbackErrorResponse(c, locale, exchange.status as ContentfulStatusCode, msgs.errorTokenFailed, exchange.error);
    }
    const { tokenJson, accountLabel } = buildRcloneEnvelope(exchange.tokenResp);
    await store.set(state, { ...session, tokenJson, accountLabel, clientId: prov.client_id, clientSecret: prov.client_secret, completedAt: Date.now() }, ttl);
    if (wantsHtml(c)) {
      const html = renderHtmlPage({ locale, success: true, redirectUrl: session.sratCallbackUrl });
      c.header("Content-Type", "text/html; charset=utf-8");
      return c.html(html, 200 as ContentfulStatusCode);
    }
    return c.redirect(session.sratCallbackUrl, 302);
  });

  // Canonical path for SRAT-Signature: Hono's c.req.path decodes %20 to space but keeps %2F.
  // Documented in README and contract.test.ts path-escaping test. Clients must sign the Hono-decoded
  // path (encoded.replaceAll("%20"," ")) not the raw encoded URL.
  app.get("/v1/session/:id", async (c) => {
    const rawBody = ""; // GET has no body
    const auth = await verifySratSignature(c, rawBody);
    if (!auth.ok) return c.json({ error: auth.error }, auth.status);
    const id = c.req.param("id");
    // Harden: session_id is always a UUID (36 chars) from crypto.randomUUID(); validate charset to
    // make encoding a non-issue and reject user-derived ids with %/non-ASCII. Same charset as instance_id.
    if (!isValidInstanceId(id)) {
      return c.json({ error: "invalid session_id" }, 400);
    }
    const peek = await store.get(id);
    if (!peek) {
      setNoStore(c);
      return c.json({ error: "expired or already used" }, 404);
    }
    if (!peek.tokenJson) {
      setNoStore(c);
      return c.json({ error: "not yet completed" }, 404);
    }
    if (peek.ownerClientId && peek.ownerClientId !== auth.clientId) {
      setNoStore(c);
      return c.json({ error: "session not owned by authenticated client_id" }, 403);
    }
    const session = await store.consume(id);
    if (!session || !session.tokenJson) {
      setNoStore(c);
      return c.json({ error: "expired or already used" }, 404);
    }
    setNoStore(c);
    return c.json({ token_json: session.tokenJson, account_label: session.accountLabel || "", client_id: session.clientId || "", client_secret: session.clientSecret || "" });
  });

  app.notFound((c) => {
    c.header("Cache-Control", "no-store");
    return c.json({ error: "not found" }, 404);
  });

  return app;
}
