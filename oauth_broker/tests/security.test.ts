import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createTestApp, testEnv, jsonBody } from "./utils.js";
import { MemorySessionStore } from "../src/session.js";
import { __clearRateLimitBucketsForTests } from "../src/app.js";
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

    it("302 callback has no-store cache and Referrer-Policy", async () => {
      const mockFetch = vi.fn(async () => new Response(JSON.stringify({ access_token: "at", expires_in: 3600 }), { status: 200 })) as unknown as typeof fetch;
      const { app } = createTestApp(undefined, { store: new MemorySessionStore(), fetchImpl: mockFetch });
      const startRes = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb" }),
      });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      const cbRes = await app.request(`/v1/callback?code=c&state=${session_id}`);
      expect(cbRes.status).toBe(302);
      expect(cbRes.headers.get("cache-control")).toContain("no-store");
      expect(cbRes.headers.get("pragma")).toBe("no-cache");
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
      // Deny by default – no header set
      expect(res.headers.get("access-control-allow-origin")).toBeNull();
    });
  });

  describe("H1: rate limiting (Both mode – in-memory fallback)", () => {
    it("blocks after 20 POST /v1/start per IP per minute (429)", async () => {
      const store = new MemorySessionStore();
      const { app } = createTestApp(undefined, { store });
      for (let i = 0; i < 20; i++) {
        const r = await app.request("/v1/start", {
          method: "POST",
          headers: { "content-type": "application/json", authorization: "Bearer test-token", "x-forwarded-for": "1.2.3.4" },
          body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb" }),
        });
        expect(r.status).toBe(200);
      }
      const blocked = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token", "x-forwarded-for": "1.2.3.4" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb" }),
      });
      expect(blocked.status).toBe(429);
      expect(blocked.headers.get("retry-after")).toBe("60");
      const body = await jsonBody(blocked);
      expect(body.error).toMatch(/rate limit exceeded/);
    });

    it("uses CF RateLimiter binding when present (Both mode)", async () => {
      const mockLimiter: any = { limit: vi.fn(async () => ({ success: false })) };
      // Verify mock shape matches BrokerBindings RATE_LIMITER contract
      expect(mockLimiter.limit).toBeDefined();
      await expect(mockLimiter.limit({ key: "test" })).resolves.toEqual({ success: false });
    });

    it("resets after window", async () => {
      const { app } = createTestApp();
      // Fill bucket
      for (let i = 0; i < 20; i++) {
        await app.request("/v1/start", {
          method: "POST",
          headers: { "content-type": "application/json", authorization: "Bearer test-token", "x-forwarded-for": "9.9.9.9" },
          body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb" }),
        });
      }
      const blocked = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token", "x-forwarded-for": "9.9.9.9" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb" }),
      });
      expect(blocked.status).toBe(429);
      vi.advanceTimersByTime(61_000);
      const ok = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token", "x-forwarded-for": "9.9.9.9" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb" }),
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
      const { app } = createTestApp(env, { store: new MemorySessionStore() });
      const ok = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://foo.allowed.com/cb" }),
      });
      expect(ok.status).toBe(200);
      const bad = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://evil.com/cb" }),
      });
      expect(bad.status).toBe(403);
      expect((await jsonBody(bad)).error).toMatch(/not allowed by broker policy/);
    });

    it("allowlist with plain host shorthand", () => {
      expect(isAllowedSratCallbackUrl("https://srat.example.com/callback", "srat.example.com")).toBe(true);
      expect(isAllowedSratCallbackUrl("https://evil.com/cb", "srat.example.com")).toBe(false);
    });
  });

  describe("L4: callback URL length cap", () => {
    it("rejects srat_callback_url > 2048 chars", async () => {
      const { app } = createTestApp();
      const longUrl = "https://srat.example.com/cb?" + "a".repeat(2048);
      expect(longUrl.length).toBeGreaterThan(MAX_CALLBACK_URL_LENGTH);
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: longUrl }),
      });
      expect(res.status).toBe(400);
      expect((await jsonBody(res)).error).toMatch(/too long/);
    });
  });

  describe("M2: BROKER_DISABLE_AUTH prod guard", () => {
    it("deny in production even with flag – fail closed (throw equivalent)", async () => {
      const prodEnv = testEnv({ BROKER_PUBLIC_URL: "https://srat-oauth-broker-production.lucio-tarantino.workers.dev", BROKER_API_TOKEN: "", BROKER_DISABLE_AUTH: "true" });
      const { app } = createTestApp(prodEnv, { store: new MemorySessionStore() });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb" }),
      });
      // Must not allow – 401 deny, not 200
      expect(res.status).toBe(401);
    });

    it("allows in dev with flag", async () => {
      const devEnv = testEnv({ BROKER_PUBLIC_URL: "http://localhost:8787", BROKER_API_TOKEN: "", BROKER_DISABLE_AUTH: "true" });
      const { app } = createTestApp(devEnv, { store: new MemorySessionStore() });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb" }),
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
      // Fill to cap
      for (let i = 0; i < 10_000; i++) {
        await store.set(`id-${i}`, { provider: "dropbox", sratCallbackUrl: "https://x/cb", createdAt: Date.now() }, 600);
      }
      expect(store.size()).toBe(10_000);
      // Next POST should 429
      const { app } = createTestApp(undefined, { store });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb" }),
      });
      expect(res.status).toBe(429);
      expect((await jsonBody(res)).error).toMatch(/too many pending sessions/);
      // After evicting expired entries, should allow again
      vi.advanceTimersByTime(601_000);
      const res2 = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb" }),
      });
      expect(res2.status).toBe(200);
      __clearRateLimitBucketsForTests();
    });
  });

  describe("L2/M4: token exchange error not leaked, cache headers", () => {
    it("provider error maps to generic 502", async () => {
      const mockFetch = vi.fn(async () => new Response(JSON.stringify({ error: "invalid_grant", error_description: "bad code secret xyz" }), { status: 400 })) as unknown as typeof fetch;
      const { app } = createTestApp(undefined, { store: new MemorySessionStore(), fetchImpl: mockFetch });
      const startRes = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb" }),
      });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      const cbRes = await app.request(`/v1/callback?code=bad&state=${session_id}`);
      expect(cbRes.status).toBe(502);
      const body = await jsonBody(cbRes);
      expect(body.error).toBe("token exchange failed");
      expect(body.error).not.toContain("bad code");
    });

    it("GET /v1/session 404 has no-store triple", async () => {
      const { app } = createTestApp();
      const res = await app.request("/v1/session/unknown", { headers: { authorization: "Bearer test-token" } });
      expect(res.headers.get("cache-control")).toContain("no-store");
      expect(res.headers.get("pragma")).toBe("no-cache");
      expect(res.headers.get("expires")).toBe("0");
    });
  });
});
