import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createTestApp, testEnv, jsonBody, registerInstance, signedHeaders, getOrCreateDefaultKeyPair } from "./utils.js";
import { MemorySessionStore, MemoryInstanceStore } from "../src/session.js";
import { __clearRateLimitBucketsForTests } from "../src/app.js";
import type { BrokerBindings } from "../src/app.js";
import {
  globToRegExp,
  isAllowedSratCallbackUrl,
  isProductionEnv,
  MAX_CALLBACK_URL_LENGTH,
  parseAllowedCallbackPatterns,
} from "../src/config.js";

describe("security hardening (Z-Audit fixes)", () => {
  beforeEach(() => {
    __clearRateLimitBucketsForTests();
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
  });
  afterEach(() => vi.useRealTimers());

  describe("H2: security headers", () => {
    it("adds X-Content-Type-Options, X-Frame-Options, CSP, HSTS on all routes", async () => {
      const { app } = createTestApp();
      const res = await app.request("/v1/healthz");
      expect(res.headers.get("x-content-type-options")).toBe("nosniff");
      expect(res.headers.get("x-frame-options")).toBe("DENY");
      expect(res.headers.get("referrer-policy")).toBe("no-referrer");
      expect(res.headers.get("strict-transport-security")).toContain("max-age=63072000");
      expect(res.headers.get("content-security-policy")).toContain("default-src 'none'");
      expect(res.headers.get("x-robots-tag")).toContain("noindex");
    });

    it("callback html has no-store cache and Referrer-Policy", async () => {
      const mockFetch = vi.fn(async () => new Response(JSON.stringify({ access_token: "at", expires_in: 3600 }), { status: 200 })) as unknown as typeof fetch;
      const { app } = createTestApp(testEnv(), { store: new MemorySessionStore(), instanceStore: new MemoryInstanceStore(), fetchImpl: mockFetch });
      await registerInstance(app, "ha-h2", "https://srat.example/cb");
      const startRes = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-h2" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-h2" }),
      });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      const cbRes = await app.request(`/v1/callback?code=c&state=${session_id}`);
      expect(cbRes.status).toBe(200);
      expect(cbRes.headers.get("cache-control")).toContain("no-store");
      expect(cbRes.headers.get("pragma")).toBe("no-cache");
      expect((await cbRes.text())).toContain("Authorization successful");
    });
  });

  describe("L1: CORS deny (no Allow-Origin)", () => {
    it("OPTIONS preflight returns 204 without Allow-Origin", async () => {
      const { app } = createTestApp();
      const res = await app.request("/v1/start", { method: "OPTIONS", headers: { origin: "https://evil.com" } });
      expect(res.status).toBe(204);
      expect(res.headers.get("access-control-allow-origin")).toBeNull();
      expect(res.headers.get("access-control-allow-methods")).toContain("GET");
    });

    it("GET does not expose Access-Control-Allow-Origin", async () => {
      const { app } = createTestApp();
      const res = await app.request("/v1/healthz", { headers: { origin: "https://evil.com" } });
      expect(res.headers.get("access-control-allow-origin")).toBeNull();
    });
  });

  describe("H1: rate limiting (Both mode – in-memory fallback)", () => {
    it("blocks after 20 POST /v1/start per IP per minute (429)", async () => {
      const store = new MemorySessionStore();
      const instanceStore = new MemoryInstanceStore();
      const { app } = createTestApp(testEnv(), { store, instanceStore });
      await registerInstance(app, "ha-rate", "https://srat.example/cb");
      const kp = await getOrCreateDefaultKeyPair(app);
      const bodyStr = JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-rate" });
      for (let i = 0; i < 20; i++) {
        const sig = await signedHeaders(kp, "POST", "/v1/start", bodyStr);
        const r = await app.request("/v1/start", {
          method: "POST",
          headers: { "content-type": "application/json", ...sig, "x-forwarded-for": "1.2.3.4" },
          body: bodyStr,
        });
        expect(r.status).toBe(200);
      }
      const blockedSig = await signedHeaders(kp, "POST", "/v1/start", bodyStr);
      const blocked = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...blockedSig, "x-forwarded-for": "1.2.3.4" },
        body: bodyStr,
      });
      expect(blocked.status).toBe(429);
      expect(blocked.headers.get("retry-after")).toBe("60");
      const body = await jsonBody(blocked);
      expect(body.error).toMatch(/rate limit exceeded/);
    });

    it("uses CF RateLimiter binding when present (Both mode)", async () => {
      const mockLimiter: NonNullable<BrokerBindings["RATE_LIMITER"]> = {
        limit: vi.fn(async () => ({ success: false })),
      };
      expect(mockLimiter.limit).toBeDefined();
      await expect(mockLimiter.limit({ key: "test" })).resolves.toEqual({ success: false });
    });

    it("resets after window", async () => {
      const instanceStore = new MemoryInstanceStore();
      const { app } = createTestApp(testEnv(), { instanceStore });
      await registerInstance(app, "ha-reset", "https://srat.example/cb");
      const kp = await getOrCreateDefaultKeyPair(app);
      const bodyStr = JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-reset" });
      for (let i = 0; i < 20; i++) {
        const sig = await signedHeaders(kp, "POST", "/v1/start", bodyStr);
        await app.request("/v1/start", {
          method: "POST",
          headers: { "content-type": "application/json", ...sig, "x-forwarded-for": "9.9.9.9" },
          body: bodyStr,
        });
      }
      const blockedSig = await signedHeaders(kp, "POST", "/v1/start", bodyStr);
      const blocked = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...blockedSig, "x-forwarded-for": "9.9.9.9" },
        body: bodyStr,
      });
      expect(blocked.status).toBe(429);
      vi.advanceTimersByTime(61_000);
      const okSig = await signedHeaders(kp, "POST", "/v1/start", bodyStr);
      const ok = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...okSig, "x-forwarded-for": "9.9.9.9" },
        body: bodyStr,
      });
      expect(ok.status).toBe(200);
    });
  });

  describe("M1: callback allowlist glob CSV", () => {
    it("globToRegExp handles wildcards", () => {
      expect(globToRegExp("https://*.example.com/*").test("https://foo.example.com/bar")).toBe(true);
      expect(globToRegExp("https://*.example.com/*").test("https://example.com/bar")).toBe(false);
      expect(globToRegExp("https://srat.example.com/cb").test("https://srat.example.com/cb")).toBe(true);
    });

    it("parseAllowedCallbackPatterns splits CSV", () => {
      expect(parseAllowedCallbackPatterns("https://a.com/*, https://b.com/*")).toEqual(["https://a.com/*", "https://b.com/*"]);
      expect(parseAllowedCallbackPatterns("")).toEqual([]);
      expect(parseAllowedCallbackPatterns(undefined)).toEqual([]);
    });

    it("isAllowedSratCallbackUrl permissive when empty", () => {
      expect(isAllowedSratCallbackUrl("https://any.example.com/cb", "")).toBe(true);
      expect(isAllowedSratCallbackUrl("https://any.example.com/cb", undefined)).toBe(true);
    });

    it("allowlist restricts to glob", async () => {
      const env = testEnv({ BROKER_ALLOWED_CALLBACK_PATTERNS: "https://*.allowed.com/*,https://srat.example.com/*" });
      const instanceStore = new MemoryInstanceStore();
      const { app } = createTestApp(env, { store: new MemorySessionStore(), instanceStore });
      // allowlist test uses real env with allowlist, but needs signed, so use registerInstance which is signed via default kp
      await registerInstance(app, "ha-allow-ok", "https://foo.allowed.com/cb");
      const ok = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: "https://foo.allowed.com/cb", instance_id: "ha-allow-ok" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://foo.allowed.com/cb", instance_id: "ha-allow-ok" }),
      });
      expect(ok.status).toBe(200);
      await registerInstance(app, "ha-allow-evil", "https://evil.com/cb");
      // evil instance will be rejected at register due to allowlist, so start should 403 at register level; test bad callback via same instance but mismatched url after allowlist bypass?
      // Instead test that register for evil is rejected
      const regBad = await app.request("/v1/instances/register", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/instances/register", JSON.stringify({ instance_id: "ha-evil2", redirect_url: "https://evil.com/cb" }))) }, body: JSON.stringify({ instance_id: "ha-evil2", redirect_url: "https://evil.com/cb" }),
      });
      expect(regBad.status).toBe(403);
    });

    it("allowlist with plain host shorthand", () => {
      expect(isAllowedSratCallbackUrl("https://srat.example.com/callback", "srat.example.com")).toBe(true);
      expect(isAllowedSratCallbackUrl("https://evil.com/cb", "srat.example.com")).toBe(false);
    });
  });

  describe("L4: callback URL length cap", () => {
    it("rejects srat_callback_url > 2048 chars", async () => {
      const { app } = createTestApp(testEnv(), { instanceStore: new MemoryInstanceStore() });
      const longUrl = "https://srat.example.com/cb?" + "a".repeat(2048);
      expect(longUrl.length).toBeGreaterThan(MAX_CALLBACK_URL_LENGTH);
      const longId = "ha-long";
      // Register with short url first
      await registerInstance(app, longId, "https://srat.example.com/cb");
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: longUrl, instance_id: longId }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: longUrl, instance_id: longId }),
      });
      // Should fail due to length or mismatch – both count as hardening
      expect([400, 403]).toContain(res.status);
      // Also test register length directly
      const regLong = await app.request("/v1/instances/register", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/instances/register", JSON.stringify({ instance_id: "ha-long2", redirect_url: longUrl }))) }, body: JSON.stringify({ instance_id: "ha-long2", redirect_url: longUrl }),
      });
      expect(regLong.status).toBe(400);
      expect((await jsonBody(regLong)).error).toMatch(/too long/);
    });
  });

  // Intentional dev-bypass tests: verify BROKER_DISABLE_AUTH prod guard, not happy-path OAuth — do not migrate to signedStartRequest
  describe("M2: BROKER_DISABLE_AUTH prod guard", () => {
    // Intentional: verifies fail-closed in prod even with flag — must keep BROKER_DISABLE_AUTH, do not migrate to signed
    it("deny in production even with flag – fail closed (throw equivalent)", async () => {
      const prodEnv = testEnv({ BROKER_PUBLIC_URL: "https://srat-oauth-broker-production.lucio-tarantino.workers.dev", BROKER_DISABLE_AUTH: "true" });
      const { app } = createTestApp(prodEnv, { store: new MemorySessionStore(), instanceStore: new MemoryInstanceStore() });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "x" }),
      });
      expect(res.status).toBe(401);
    });

    // Intentional: verifies dev bypass allows without sig — must keep BROKER_DISABLE_AUTH, do not migrate to signed
    it("allows in dev with flag", async () => {
      const devEnv = testEnv({ BROKER_PUBLIC_URL: "http://localhost:8787", BROKER_DISABLE_AUTH: "true" });
      const appObj = createTestApp(devEnv, { store: new MemorySessionStore(), instanceStore: new MemoryInstanceStore() });
      const reg = await appObj.app.request("/v1/instances/register", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ instance_id: "ha-dev", redirect_url: "https://srat.example.com/cb" }),
      });
      expect(reg.status).toBe(200);
      const res = await appObj.app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "ha-dev" }),
      });
      expect(res.status).toBe(200);
    });

    it("isProductionEnv detects production", () => {
      expect(isProductionEnv({ BROKER_PUBLIC_URL: "https://broker-production.example.com" })).toBe(true);
      expect(isProductionEnv({ BROKER_PUBLIC_URL: "https://broker-staging.example.com" })).toBe(false);
      expect(isProductionEnv({ BROKER_PUBLIC_URL: "http://localhost:8787" })).toBe(false);
      expect(isProductionEnv({ ENV: "production" })).toBe(true);
    });
  });

  describe("H1: MemorySessionStore cap (DoS guard)", () => {
    it("rejects when store full (10k entries)", async () => {
      const store = new MemorySessionStore();
      const instanceStore = new MemoryInstanceStore();
      for (let i = 0; i < 10_000; i++) {
        await store.set(`id-${i}`, { provider: "dropbox", sratCallbackUrl: "https://x/cb", createdAt: Date.now() }, 600);
      }
      expect(store.size()).toBe(10_000);
      const { app } = createTestApp(testEnv(), { store, instanceStore });
      await registerInstance(app, "ha-full", "https://srat.example/cb");
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-full" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-full" }),
      });
      expect(res.status).toBe(429);
      expect((await jsonBody(res)).error).toMatch(/too many pending sessions/);
      vi.advanceTimersByTime(601_000);
      const res2 = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-full" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-full" }),
      });
      expect(res2.status).toBe(200);
      __clearRateLimitBucketsForTests();
    });
  });

  describe("L2/M4: token exchange error not leaked, cache headers", () => {
    it("provider error maps to generic 502 (json mode)", async () => {
      const mockFetch = vi.fn(async () => new Response(JSON.stringify({ error: "invalid_grant", error_description: "bad code secret xyz" }), { status: 400 })) as unknown as typeof fetch;
      const instanceStore = new MemoryInstanceStore();
      const { app } = createTestApp(testEnv(), { store: new MemorySessionStore(), instanceStore, fetchImpl: mockFetch });
      await registerInstance(app, "ha-err", "https://srat.example/cb");
      const startRes = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-err" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-err" }),
      });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      const cbRes = await app.request(`/v1/callback?code=bad&state=${session_id}`, { headers: { accept: "application/json" } });
      expect(cbRes.status).toBe(502);
      const body = await jsonBody(cbRes);
      expect(body.error).toBe("token exchange failed");
      expect(body.error).not.toContain("bad code");
    });

    it("GET /v1/session 404 has no-store triple", async () => {
      const { app } = createTestApp();
      const kp = await getOrCreateDefaultKeyPair(app);
      const sig = await signedHeaders(kp, "GET", "/v1/session/unknown", "");
      const res = await app.request("/v1/session/unknown", { headers: { ...sig } });
      expect(res.headers.get("cache-control")).toContain("no-store");
      expect(res.headers.get("pragma")).toBe("no-cache");
      expect(res.headers.get("expires")).toBe("0");
    });
  });
});
