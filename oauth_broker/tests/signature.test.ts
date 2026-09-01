import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createTestApp, generateTestKeyPair, jsonBody, registerClient } from "./utils.js";
import { __clearRateLimitBucketsForTests } from "../src/app.js";
import { bodyHashBase64Url, buildStringToSign, signStringToSign } from "../src/crypto.js";

describe("SRAT-Signature auth (new contract)", () => {
  let kp: Awaited<ReturnType<typeof generateTestKeyPair>>;
  beforeEach(async () => {
    kp = await generateTestKeyPair();
    __clearRateLimitBucketsForTests();
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
  });
  afterEach(() => vi.useRealTimers());

  async function signedRequest(app: any, method: string, path: string, body: string, key = kp) {
    const t = String(Math.floor(Date.now() / 1000));
    const nonce = `sig-test-${Math.random().toString(36).slice(2, 8)}-${Date.now()}`;
    const bh = bodyHashBase64Url(body);
    const sts = buildStringToSign(key.clientId, method, path, t, nonce, bh);
    const sig = await signStringToSign(key.privateKeyB64Url, key.publicKeyB64Url, sts);
    const header = `SRAT-Signature client_id="${key.clientId}", t="${t}", nonce="${nonce}", sig="${sig}"`;
    return { header, t, nonce };
  }

  it("registers client_id = SHA256(pubkey) and rejects mismatch", async () => {
    const { app, clientStore } = createTestApp();
    const reg = await registerClient(app, kp);
    expect(reg.status).toBe(201);
    const body = (await jsonBody(reg)) as { client_id: string };
    expect(body.client_id).toBe(kp.clientId);
    const stored = await clientStore.get(kp.clientId);
    expect(stored?.publicKey).toBe(kp.publicKeyB64Url);

    // mismatch
    const bad = await app.request("/v1/clients", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ client_id: kp.clientId, public_key: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" }),
    });
    expect(bad.status).toBe(400);
  });

  it("instance register requires valid signature, binds to client_id, rejects different client", async () => {
    const { app } = createTestApp();
    await registerClient(app, kp);
    const kp2 = await generateTestKeyPair();
    await registerClient(app, kp2);

    // missing sig -> 401
    const noAuth = await app.request("/v1/instances/register", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ instance_id: "ha-1", redirect_url: "https://srat.example/cb" }),
    });
    expect(noAuth.status).toBe(401);

    // valid sig
    const body = JSON.stringify({ instance_id: "ha-1", redirect_url: "https://srat.example/cb" });
    const { header } = await signedRequest(app, "POST", "/v1/instances/register", body);
    const ok = await app.request("/v1/instances/register", { method: "POST", headers: { "content-type": "application/json", authorization: header }, body });
    expect(ok.status).toBe(200);
    const okBody = (await jsonBody(ok)) as { client_id: string };
    expect(okBody.client_id).toBe(kp.clientId);

    // different client tries to reuse same instance_id -> 403
    const body2 = JSON.stringify({ instance_id: "ha-1", redirect_url: "https://srat.example/cb" });
    const t2 = String(Math.floor(Date.now() / 1000));
    const nonce2 = `n2-${Date.now()}`;
    const bh2 = bodyHashBase64Url(body2);
    const sts2 = buildStringToSign(kp2.clientId, "POST", "/v1/instances/register", t2, nonce2, bh2);
    const sig2 = await signStringToSign(kp2.privateKeyB64Url, kp2.publicKeyB64Url, sts2);
    const header2 = `SRAT-Signature client_id="${kp2.clientId}", t="${t2}", nonce="${nonce2}", sig="${sig2}"`;
    const conflict = await app.request("/v1/instances/register", { method: "POST", headers: { "content-type": "application/json", authorization: header2 }, body: body2 });
    expect(conflict.status).toBe(403);
  });

  it("replay nonce rejected, clock skew rejected", async () => {
    const { app } = createTestApp();
    await registerClient(app, kp);
    const body = JSON.stringify({ instance_id: "ha-replay", redirect_url: "https://srat.example/cb" });
    const t = String(Math.floor(Date.now() / 1000));
    const nonce = "fixed-nonce-replay-1234567890";
    const bh = bodyHashBase64Url(body);
    const sts = buildStringToSign(kp.clientId, "POST", "/v1/instances/register", t, nonce, bh);
    const sig = await signStringToSign(kp.privateKeyB64Url, kp.publicKeyB64Url, sts);
    const header = `SRAT-Signature client_id="${kp.clientId}", t="${t}", nonce="${nonce}", sig="${sig}"`;
    const first = await app.request("/v1/instances/register", { method: "POST", headers: { "content-type": "application/json", authorization: header }, body });
    expect(first.status).toBe(200);
    const replay = await app.request("/v1/instances/register", { method: "POST", headers: { "content-type": "application/json", authorization: header }, body });
    expect(replay.status).toBe(401);
    expect((await jsonBody(replay)).error).toMatch(/nonce already used/);

    // clock skew: 10 min in future
    const futureT = String(Math.floor(Date.now() / 1000) + 1000);
    const nonce2 = "future-nonce-1234567890";
    const bh2 = bodyHashBase64Url(body);
    const sts2 = buildStringToSign(kp.clientId, "POST", "/v1/instances/register", futureT, nonce2, bh2);
    const sig2 = await signStringToSign(kp.privateKeyB64Url, kp.publicKeyB64Url, sts2);
    const header2 = `SRAT-Signature client_id="${kp.clientId}", t="${futureT}", nonce="${nonce2}", sig="${sig2}"`;
    const skew = await app.request("/v1/instances/register", { method: "POST", headers: { "content-type": "application/json", authorization: header2 }, body });
    expect(skew.status).toBe(401);
    expect((await jsonBody(skew)).error).toMatch(/clock skew/);
  });

  it("POST /v1/start and GET /v1/session require ownership by same client_id", async () => {
    const mockFetch = vi.fn(async () => new Response(JSON.stringify({ access_token: "at", token_type: "bearer", expires_in: 3600, account_id: "acc1" }), { status: 200 })) as unknown as typeof fetch;
    const { app } = createTestApp(undefined, { fetchImpl: mockFetch });
    await registerClient(app, kp);
    // register instance
    const instBody = JSON.stringify({ instance_id: "ha-own", redirect_url: "https://srat.example/cb" });
    const { header: hInst } = await signedRequest(app, "POST", "/v1/instances/register", instBody);
    const reg = await app.request("/v1/instances/register", { method: "POST", headers: { "content-type": "application/json", authorization: hInst }, body: instBody });
    expect(reg.status).toBe(200);

    // start
    const startBody = JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-own" });
    const { header: hStart } = await signedRequest(app, "POST", "/v1/start", startBody);
    const startRes = await app.request("/v1/start", { method: "POST", headers: { "content-type": "application/json", authorization: hStart }, body: startBody });
    expect(startRes.status).toBe(200);
    const { session_id } = (await jsonBody(startRes)) as { session_id: string };

    // callback (browser, no sig)
    const cbRes = await app.request(`/v1/callback?code=c&state=${session_id}`, { headers: { accept: "application/json" } });
    expect(cbRes.status).toBe(302);

    // session fetch with wrong client -> 403
    const kp2 = await generateTestKeyPair();
    await registerClient(app, kp2);
    const { header: hWrong } = await signedRequest(app, "GET", `/v1/session/${session_id}`, "", kp2);
    const wrong = await app.request(`/v1/session/${session_id}`, { headers: { authorization: hWrong } });
    expect(wrong.status).toBe(403);

    // correct client -> 200 and single-use
    const { header: hGood } = await signedRequest(app, "GET", `/v1/session/${session_id}`, "");
    const good = await app.request(`/v1/session/${session_id}`, { headers: { authorization: hGood } });
    expect(good.status).toBe(200);
    // second fetch with same header would replay nonce – must use fresh nonce, but still 404 because already consumed
    const { header: hGood2 } = await signedRequest(app, "GET", `/v1/session/${session_id}`, "");
    const good2b = await app.request(`/v1/session/${session_id}`, { headers: { authorization: hGood2 } });
    expect(good2b.status).toBe(404); // already consumed
  });
});
