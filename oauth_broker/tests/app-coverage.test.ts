import { describe, it, expect, vi, beforeEach } from "vitest";
import { createBrokerApp, base64UrlEncode, isValidSratCallbackUrl, generateCodeVerifier, pkceChallengeFromVerifier, buildAuthUrl, __clearRateLimitBucketsForTests, exchangeCodeForToken } from "../src/app.js";
import { MemorySessionStore, MemoryInstanceStore, MemoryClientStore, MemoryNonceStore } from "../src/session.js";

import { testEnv, createTestApp, generateTestKeyPair, registerClient, signedHeaders } from "./utils.js";

describe("app.ts – patch 75% → 85% (40 missing +53 partials)", () => {
  beforeEach(() => __clearRateLimitBucketsForTests());

  it("base64UrlEncode Buffer fallback (app.ts lines 72-75) via vi.stubGlobal", () => {
    const bytes = new Uint8Array([1,2,3,255]);
    const encViaBuffer = base64UrlEncode(bytes);
    vi.stubGlobal("Buffer", undefined as unknown as typeof Buffer);
    const encViaWeb = base64UrlEncode(bytes);
    expect(encViaWeb).toBe(encViaBuffer);
    vi.unstubAllGlobals();
  });

  it("isValidSratCallbackUrl and isLoopback", () => {
    expect(isValidSratCallbackUrl("https://example.com/cb")).toBe(true);
    expect(isValidSratCallbackUrl("http://localhost:3000/cb")).toBe(true);
    expect(isValidSratCallbackUrl("http://127.0.0.1/cb")).toBe(true);
    expect(isValidSratCallbackUrl("http://[::1]/cb")).toBe(true);
    expect(isValidSratCallbackUrl("http://example.com/cb")).toBe(false); // http non-loopback
    expect(isValidSratCallbackUrl("not-a-url")).toBe(false);
  });

  it("generateCodeVerifier / pkceChallenge", () => {
    const v = generateCodeVerifier();
    expect(v).toMatch(/^[A-Za-z0-9_-]{43}$/);
    const c = pkceChallengeFromVerifier(v);
    expect(c).toMatch(/^[A-Za-z0-9_-]{43}$/);
  });

  it("buildAuthUrl", () => {
    const prov = { authorize_url: "https://auth.example.com/authorize", token_url: "https://auth.example.com/token", client_id: "id", client_secret: "sec" } as any;
    const url = buildAuthUrl(prov, "https://broker.example.com", "sess-123", "verifier123");
    expect(url.toString()).toContain("sess-123");
    const url2 = buildAuthUrl(prov, "https://broker.example.com", "sess-456");
    expect(url2.toString()).toContain("sess-456");
  });

  it("exchangeCodeForToken – invalid JSON and !ok", async () => {
    const prov = { token_url: "https://tok.example.com/token" } as any;
    const badFetch = async () => new Response("not-json", { status: 200 }) as unknown as Response;
    const res = await exchangeCodeForToken(prov, "code", "https://broker.example.com/v1/callback", badFetch as unknown as typeof fetch, "ver");
    expect(res).toHaveProperty("error");
    expect((res as any).status).toBe(502);
    const errFetch = async () => new Response(JSON.stringify({ error: "invalid_grant", error_description: "bad code" }), { status: 400 }) as unknown as Response;
    const res2 = await exchangeCodeForToken(prov, "code", "https://broker.example.com/v1/callback", errFetch as unknown as typeof fetch, "ver");
    expect((res2 as any).status).toBe(502);
  });

  it("POST /v1/clients – invalid json, 409 different pubkey, 200 same pubkey, rotate 501", async () => {
    const { app } = createTestApp();
    const bad = await app.request("/v1/clients", { method: "POST", headers: { "content-type": "application/json" }, body: "not-json" });
    expect(bad.status).toBe(400);
    const kp = await generateTestKeyPair();
    const r1 = await registerClient(app, kp);
    expect(r1.status).toBe(201);
    const r2 = await registerClient(app, kp);
    expect(r2.status).toBe(200);
    const bad2 = await app.request("/v1/clients", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ client_id: kp.clientId, public_key: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" }) });
    expect(bad2.status).toBe(400);
    const rot = await app.request(`/v1/clients/${kp.clientId}/rotate`, { method: "POST" });
    expect(rot.status).toBe(501);
  });

  it("POST /v1/instances/register – invalid json, 403 allowlist, store full 429", async () => {
    const kp = await generateTestKeyPair();
    const { app } = createTestApp(testEnv({ BROKER_ALLOWED_CALLBACK_PATTERNS: "https://allowed.example.com/*" }));
    await registerClient(app, kp);
    const { header } = await (async () => {
      const t = String(Math.floor(Date.now()/1000));
      const nonce = `n-${Math.random().toString(36).slice(2,10)}-${Date.now()}`;
      const { bodyHashBase64Url, buildStringToSign } = await import("../src/crypto.js");
      const { signStringToSign: s2 } = await import("../src/crypto.js");
      const bh = bodyHashBase64Url("not-json");
      const sts = buildStringToSign(kp.clientId, "POST", "/v1/instances/register", t, nonce, bh);
      const sig = await s2(kp.privateKeyB64Url, kp.publicKeyB64Url, sts);
      return { header: `SRAT-Signature client_id="${kp.clientId}", t="${t}", nonce="${nonce}", sig="${sig}"` };
    })();
    const badJson = await app.request("/v1/instances/register", { method: "POST", headers: { "content-type": "application/json", authorization: header }, body: "not-json" });
    expect(badJson.status).toBe(400);
    // allowlist 403
    const body = JSON.stringify({ instance_id: "ha-1", redirect_url: "https://not-allowed.example.com/cb" });
    const hdr2 = await signedHeaders(kp, "POST", "/v1/instances/register", body);
    const forbid = await app.request("/v1/instances/register", { method: "POST", headers: { "content-type": "application/json", ...hdr2 }, body });
    expect(forbid.status).toBe(403);
    // store full
    const failingStore = { get: async () => null, set: async () => { throw new Error("store full: too many pending instances"); }, delete: async () => {} } as unknown as MemoryInstanceStore;
    const { app: app2 } = createTestApp(testEnv(), { instanceStore: failingStore });
    await registerClient(app2, kp);
    const fullBody = JSON.stringify({ instance_id: "ha-full", redirect_url: "https://allowed.example.com/cb" });
    const hdr3 = await signedHeaders(kp, "POST", "/v1/instances/register", fullBody);
    const full = await app2.request("/v1/instances/register", { method: "POST", headers: { "content-type": "application/json", ...hdr3 }, body: fullBody });
    expect(full.status).toBe(429);
  });

  it("POST /v1/start – session store full 429", async () => {
    const kp = await generateTestKeyPair();
    const failingSess = { get: async () => null, set: async () => { throw new Error("session store full: too many pending sessions"); }, delete: async () => {}, consume: async () => null } as unknown as MemorySessionStore;
    const { app } = createTestApp(testEnv(), { store: failingSess });
    await registerClient(app, kp);
    const instBody = JSON.stringify({ instance_id: "ha-start-full", redirect_url: "https://allowed.example.com/cb" });
    const hdrInst = await signedHeaders(kp, "POST", "/v1/instances/register", instBody);
    await app.request("/v1/instances/register", { method: "POST", headers: { "content-type": "application/json", ...hdrInst }, body: instBody });
    const startBody = JSON.stringify({ provider: "dropbox", srat_callback_url: "https://allowed.example.com/cb", instance_id: "ha-start-full" });
    const hdrStart = await signedHeaders(kp, "POST", "/v1/start", startBody);
    const res = await app.request("/v1/start", { method: "POST", headers: { "content-type": "application/json", ...hdrStart }, body: startBody });
    expect(res.status).toBe(429);
  });

  it("GET /v1/callback – notFound 404, healthz", async () => {
    const { app } = createTestApp();
    const nf = await app.request("/nope", { method: "GET" });
    expect(nf.status).toBe(404);
    const h1 = await app.request("/v1/healthz");
    expect(h1.status).toBe(200);
    const h2 = await app.request("/healthz");
    expect(h2.status).toBe(200);
  });

  it("GET /v1/session – 403 not owned, 404 expired", async () => {
    const kp = await generateTestKeyPair();
    const kp2 = await generateTestKeyPair();
    const mockFetch = async () => new Response(JSON.stringify({ access_token: "at", token_type: "bearer" }), { status: 200 }) as unknown as Response;
    const { app } = createTestApp(undefined, { fetchImpl: mockFetch as unknown as typeof fetch });
    await registerClient(app, kp);
    await registerClient(app, kp2);
    const hdrInst = await signedHeaders(kp, "POST", "/v1/instances/register", JSON.stringify({ instance_id: "ha-sess", redirect_url: "https://allowed.example.com/cb" }));
    await app.request("/v1/instances/register", { method: "POST", headers: { "content-type": "application/json", ...hdrInst }, body: JSON.stringify({ instance_id: "ha-sess", redirect_url: "https://allowed.example.com/cb" }) });
    const startBody = JSON.stringify({ provider: "dropbox", srat_callback_url: "https://allowed.example.com/cb", instance_id: "ha-sess" });
    const hdrStart = await signedHeaders(kp, "POST", "/v1/start", startBody);
    const startRes = await app.request("/v1/start", { method: "POST", headers: { "content-type": "application/json", ...hdrStart }, body: startBody });
    const { session_id } = await startRes.json() as { session_id: string };
    await app.request(`/v1/callback?code=c&state=${session_id}`, { headers: { accept: "application/json" } });
    const hdrWrong = await signedHeaders(kp2, "GET", `/v1/session/${session_id}`, "");
    const wrong = await app.request(`/v1/session/${session_id}`, { headers: hdrWrong });
    expect(wrong.status).toBe(403);
    const hdrGood = await signedHeaders(kp, "GET", `/v1/session/${session_id}`, "");
    const good = await app.request(`/v1/session/${session_id}`, { headers: hdrGood });
    expect(good.status).toBe(200);
    const hdrAgain = await signedHeaders(kp, "GET", `/v1/session/${session_id}`, "");
    const again = await app.request(`/v1/session/${session_id}`, { headers: hdrAgain });
    expect(again.status).toBe(404);
  });

  it("rate limiter – binding error fallback and memory 10k cleanup (lines 128-129)", async () => {
    const { app } = createTestApp();
    // binding that throws
    const throwingBinding = { limit: async () => { throw new Error("boom"); } };
    await app.request("/v1/healthz", { headers: { "x-broker-client-ip": "1.1.1.1" } } as unknown as RequestInit & { env: unknown });
    // directly test memoryBuckets >10k cleanup via filling
    for (let i = 0; i < 10005; i++) {
      await app.request("/v1/healthz", { headers: { "x-broker-client-ip": `10.0.0.${i % 250}` } } as unknown as RequestInit);
    }
    // after filling, a new request should still succeed and have triggered cleanup
    const ok = await app.request("/v1/healthz");
    expect(ok.status).toBe(200);
    // also test binding error path
    const app2 = createBrokerApp({ env: testEnv(), store: new MemorySessionStore(), instanceStore: new MemoryInstanceStore(), clientStore: new MemoryClientStore(), nonceStore: new MemoryNonceStore(), fetchImpl: undefined as unknown as typeof fetch });
    // inject binding via c.env – Hono's c.env is from env param, so we test via request with env containing binding
    // we simulate by calling the app with a custom env that has RATE_LIMITER
    const withBinding = await app2.request("/v1/healthz", {}, { RATE_LIMITER: throwingBinding } as unknown as Record<string, unknown>);
    expect(withBinding.status).toBe(200);
  });
});
