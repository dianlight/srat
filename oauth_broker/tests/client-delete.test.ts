import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createTestApp, testEnv } from "./utils.js";
import { MemoryClientStore, MemoryInstanceStore, MemoryNonceStore, MemorySessionStore } from "../src/session.js";
import { __clearRateLimitBucketsForTests } from "../src/app.js";
import { generateEd25519KeyPair, signStringToSign, bodyHashBase64Url, buildStringToSign } from "../src/crypto.js";

async function signedHeadersFor(kp: { clientId: string; publicKeyB64Url: string; privateKeyB64Url: string }, method: string, path: string, body: string) {
  const t = String(Math.floor(Date.now() / 1000));
  const nonce = `del-${Math.random().toString(36).slice(2, 8)}-${Date.now()}`;
  const bh = bodyHashBase64Url(body);
  const sts = buildStringToSign(kp.clientId, method, path, t, nonce, bh);
  const sig = await signStringToSign(kp.privateKeyB64Url, kp.publicKeyB64Url, sts);
  return { authorization: `SRAT-Signature client_id="${kp.clientId}", t="${t}", nonce="${nonce}", sig="${sig}"` };
}

describe("DELETE /v1/clients", () => {
  beforeEach(() => {
    __clearRateLimitBucketsForTests();
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
  });
  afterEach(() => vi.useRealTimers());

  it("deletes own client and revokes its instances", async () => {
    const store = new MemorySessionStore();
    const instanceStore = new MemoryInstanceStore();
    const clientStore = new MemoryClientStore();
    const nonceStore = new MemoryNonceStore();
    const env = testEnv();
    const { app } = createTestApp(env, { store, instanceStore, clientStore, nonceStore });
    const kp = await generateEd25519KeyPair();
    const reg = await app.request("/v1/clients", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ client_id: kp.clientId, public_key: kp.publicKeyB64Url }),
    });
    expect(reg.status).toBe(201);

    const instBody = JSON.stringify({ instance_id: "ha-del-1", redirect_url: "https://srat.example/cb" });
    const instHeaders = await signedHeadersFor(kp, "POST", "/v1/instances/register", instBody);
    const instRes = await app.request("/v1/instances/register", {
      method: "POST",
      headers: { "content-type": "application/json", ...instHeaders },
      body: instBody,
    });
    expect(instRes.status).toBe(200);

    const startBody = JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-del-1" });
    const startHeaders = await signedHeadersFor(kp, "POST", "/v1/start", startBody);
    const startRes = await app.request("/v1/start", {
      method: "POST",
      headers: { "content-type": "application/json", ...startHeaders },
      body: startBody,
    });
    expect(startRes.status).toBe(200);

    // Try to delete another client's id should be 403 (both still exist, test before deleting own)
    const kp2 = await generateEd25519KeyPair();
    const reg2 = await app.request("/v1/clients", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ client_id: kp2.clientId, public_key: kp2.publicKeyB64Url }),
    });
    expect(reg2.status).toBe(201);
    const delOtherHeaders = await signedHeadersFor(kp, "DELETE", `/v1/clients/${kp2.clientId}`, "");
    const delOther = await app.request(`/v1/clients/${kp2.clientId}`, {
      method: "DELETE",
      headers: delOtherHeaders,
    });
    expect(delOther.status).toBe(403);

    const delHeaders = await signedHeadersFor(kp, "DELETE", `/v1/clients/${kp.clientId}`, "");
    const delRes = await app.request(`/v1/clients/${kp.clientId}`, {
      method: "DELETE",
      headers: delHeaders,
    });
    expect(delRes.status).toBe(200);
    const delBody = (await delRes.json()) as any;
    expect(delBody.deleted).toBe(true);
    expect(delBody.client_id).toBe(kp.clientId);

    const startBody2 = JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-del-1" });
    const startHeaders2 = await signedHeadersFor(kp, "POST", "/v1/start", startBody2);
    const startRes2 = await app.request("/v1/start", {
      method: "POST",
      headers: { "content-type": "application/json", ...startHeaders2 },
      body: startBody2,
    });
    expect(startRes2.status).toBe(401);

    const delHeaders2 = await signedHeadersFor(kp, "DELETE", `/v1/clients/${kp.clientId}`, "");
    const delRes2 = await app.request(`/v1/clients/${kp.clientId}`, {
      method: "DELETE",
      headers: delHeaders2,
    });
    expect([401, 404]).toContain(delRes2.status);

    const kp3 = await generateEd25519KeyPair();
    const reg3 = await app.request("/v1/clients", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ client_id: kp3.clientId, public_key: kp3.publicKeyB64Url }),
    });
    expect(reg3.status).toBe(201);
    const delNoIdHeaders = await signedHeadersFor(kp3, "DELETE", "/v1/clients", "");
    const delNoId = await app.request("/v1/clients", {
      method: "DELETE",
      headers: delNoIdHeaders,
    });
    expect(delNoId.status).toBe(200);
  });

  it("requires SRAT-Signature for DELETE", async () => {
    const { app } = createTestApp(testEnv(), {});
    const res = await app.request("/v1/clients/some-id", { method: "DELETE" });
    expect(res.status).toBe(401);
  });

  it("rejects invalid client_id format", async () => {
    const kp = await generateEd25519KeyPair();
    const { app } = createTestApp(testEnv(), {});
    await app.request("/v1/clients", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ client_id: kp.clientId, public_key: kp.publicKeyB64Url }),
    });
    const headers = await signedHeadersFor(kp, "DELETE", "/v1/clients/invalid id with spaces", "");
    const res = await app.request("/v1/clients/invalid id with spaces", {
      method: "DELETE",
      headers,
    });
    expect(res.status).toBe(400);
  });
});
