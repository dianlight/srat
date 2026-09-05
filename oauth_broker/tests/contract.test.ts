/**
 * Contract tests mirror backend/src/service/rclone/broker_test.go fakeBroker expectations.
 * Ensures the TS broker stays compatible with SRAT Go client (issue-954-rclone).
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { createTestApp, jsonBody, testEnv, registerInstance, signedHeaders, getOrCreateDefaultKeyPair, signedStartRequest, signedSessionRequest, generateTestKeyPair, registerClient } from "./utils.js";
import { MemorySessionStore, MemoryInstanceStore } from "../src/session.js";
import { __clearRateLimitBucketsForTests } from "../src/app.js";

describe("contract: SRAT Go client compatibility (broker_test.go)", () => {
  let store: MemorySessionStore;
  let instanceStore: MemoryInstanceStore;

  beforeEach(() => {
    store = new MemorySessionStore();
    instanceStore = new MemoryInstanceStore();
    __clearRateLimitBucketsForTests();
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2030-01-01T00:00:00Z"));
  });
  afterEach(() => vi.useRealTimers());

  it("BrokerStart happy path: posts provider + srat_callback_url + instance_id, returns auth_url + session_id", async () => {
    const { app } = createTestApp(undefined, { store, instanceStore });
    const httpsCallback = "https://srat.example/api/rclone/oauth/callback?state=st%3D1";
    await registerInstance(app, "ha-contract-1", httpsCallback);
    const res = await signedStartRequest(app, { provider: "dropbox", srat_callback_url: httpsCallback, instance_id: "ha-contract-1" });
    expect(res.status).toBe(200);
    const body = (await jsonBody(res)) as { auth_url: string; session_id: string };
    expect(body.auth_url).toMatch(/dropbox\.com\/oauth2\/authorize/);
    expect(body.session_id).toMatch(/^[0-9a-f-]{36}$/);
    const stored = await store.get(body.session_id);
    expect(stored?.provider).toBe("dropbox");
    expect(stored?.sratCallbackUrl).toBe(httpsCallback);
    expect(stored?.instanceId).toBe("ha-contract-1");
  });

  it("BrokerFetchToken happy path: returns token_json + account_label + client_id/secret", async () => {
    const mockFetch = vi.fn(async () =>
      new Response(
        JSON.stringify({
          access_token: "bt",
          token_type: "bearer",
          refresh_token: "br",
          expires_in: 14400,
          account_id: "acc-broker",
        }),
        { status: 200 }
      )
    ) as unknown as typeof fetch;
    const env = testEnv();
    const { app } = createTestApp(env, { store, instanceStore, fetchImpl: mockFetch });
    await registerInstance(app, "ha-broker", "https://srat.example/cb");

    const startRes = await signedStartRequest(app, { provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-broker" });
    const { session_id } = (await jsonBody(startRes)) as { session_id: string };
    const cbRes = await app.request(`/v1/callback?code=code123&state=${session_id}`, { headers: { accept: "application/json" } });
    expect(cbRes.status).toBe(302);

    const fetchRes = await signedSessionRequest(app, session_id);
    expect(fetchRes.status).toBe(200);
    const token = (await jsonBody(fetchRes)) as {
      token_json: string;
      account_label: string;
      client_id: string;
      client_secret: string;
    };
    expect(JSON.parse(token.token_json).access_token).toBe("bt");
    expect(JSON.parse(token.token_json).refresh_token).toBe("br");
    expect(token.account_label).toBe("acc-broker");
    expect(token.client_id).toBe("dropbox-id");
    expect(token.client_secret).toBe("dropbox-secret");
    const second = await signedSessionRequest(app, session_id);
    expect(second.status).toBe(404);
    const secondBody = await jsonBody(second);
    expect(secondBody.error).toMatch(/expired or already used/);
    const noAuth = await app.request(`/v1/session/${session_id}`, {});
    expect(noAuth.status).toBe(401);
  });

  it("unknown session maps to expiry message", async () => {
    const { app } = createTestApp(undefined, { store, instanceStore });
    const kp = await getOrCreateDefaultKeyPair(app);
    const sig = await signedHeaders(kp, "GET", "/v1/session/other-session", "");
    const res = await app.request("/v1/session/other-session", {
      headers: { ...sig },
    });
    expect(res.status).toBe(404);
    const body = await jsonBody(res);
    expect(body.error).toMatch(/expired or already used/);
  });

  it("invalid payload rejected (Go client expects error for missing fields)", async () => {
    const { app } = createTestApp(undefined, { store, instanceStore });
    const res = await app.request("/v1/start", {
      method: "POST",
      headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "", srat_callback_url: "" }))) }, body: JSON.stringify({ provider: "", srat_callback_url: "" }),
    });
    expect(res.status).toBe(400);
  });

  // Intentional: validates SRAT-Signature success + 401 without sig + dev-bypass — keep BROKER_DISABLE_AUTH, do not migrate to signed
  it("SRAT-Signature auth: valid sig succeeds, missing sig 401, dev bypass allows without sig", async () => {
    const kp = await generateTestKeyPair();
    const { app: appWithSig } = createTestApp(testEnv(), { store, instanceStore });
    await registerClient(appWithSig, kp);
    await registerInstance(appWithSig, "ha-auth", "https://srat.example/cb", kp);
    const body = JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-auth" });
    const headers = await signedHeaders(kp, "POST", "/v1/start", body);
    const ok = await appWithSig.request("/v1/start", {
      method: "POST",
      headers: { "content-type": "application/json", ...headers },
      body,
    });
    expect(ok.status).toBe(200);
    const bad = await appWithSig.request("/v1/start", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body,
    });
    expect(bad.status).toBe(401);

    // Intentional dev-bypass: noTokenEnv with BROKER_DISABLE_AUTH=true must stay — do not migrate to signed (tests dev bypass)
    const noTokenEnv = testEnv({ BROKER_DISABLE_AUTH: "true", BROKER_PUBLIC_URL: "http://localhost:8787" });
    const { app: appNoToken } = createTestApp(noTokenEnv, { store: new MemorySessionStore(), instanceStore: new MemoryInstanceStore() });
    const regDev = await appNoToken.request("/v1/instances/register", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ instance_id: "ha-dev2", redirect_url: "https://srat.example/cb" }),
    });
    expect(regDev.status).toBe(200);
    const devOk = await appNoToken.request("/v1/start", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-dev2" }),
    });
    expect(devOk.status).toBe(200);

    // Intentional fail-closed: empty BROKER_DISABLE_AUTH must stay — do not migrate to signed (tests fail-closed without sig)
    const failClosedEnv = testEnv({ BROKER_DISABLE_AUTH: "" });
    // ensure no disable flag — intentional, do not add BROKER_DISABLE_AUTH
    delete (failClosedEnv as any).BROKER_DISABLE_AUTH;
    const { app: appFailClosed } = createTestApp(failClosedEnv, { store: new MemorySessionStore(), instanceStore: new MemoryInstanceStore() });
    const devFail = await appFailClosed.request("/v1/start", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "x" }),
    });
    expect(devFail.status).toBe(401);
  });

  it("session id path escaping round-trips (valid id)", async () => {
    const validId = "550e8400-e29b-41d4-a716-446655440000"; // UUID, valid charset ^[A-Za-z0-9._-]{1,128}$
    const customStore = new MemorySessionStore();
    const tokenJson = JSON.stringify({ access_token: "at", expiry: new Date().toISOString() });
    await customStore.set(validId, { provider: "dropbox", sratCallbackUrl: "https://srat.example/cb", createdAt: Date.now(), tokenJson, accountLabel: "acc", clientId: "id", clientSecret: "sec" }, 600);
    const { app } = createTestApp(undefined, { store: customStore, instanceStore });
    const kp = await getOrCreateDefaultKeyPair(app);
    const encoded = encodeURIComponent(validId);
    const honoPath = `/v1/session/${encoded.replaceAll("%20", " ")}`;
    const headers = await signedHeaders(kp, "GET", honoPath, "");
    const res = await app.request(`/v1/session/${encoded}`, { headers });
    expect(res.status).toBe(200);
  });

  it("session id rejects invalid charset (slash/space)", async () => {
    const { app } = createTestApp(undefined, { store: new MemorySessionStore(), instanceStore });
    const kp = await getOrCreateDefaultKeyPair(app);
    const invalidId = "a/b c";
    const encoded = encodeURIComponent(invalidId);
    const honoPath = `/v1/session/${encoded.replaceAll("%20", " ")}`;
    const headers = await signedHeaders(kp, "GET", honoPath, "");
    const res = await app.request(`/v1/session/${encoded}`, { headers });
    expect(res.status).toBe(400);
    expect((await res.json() as any).error).toMatch(/invalid session_id/);
  });

  it("expiry wraps as RFC3339 UTC (now + expires_in)", async () => {
    const mockFetch = vi.fn(async () =>
      new Response(JSON.stringify({ access_token: "at", expires_in: 3600, token_type: "bearer" }), { status: 200 })
    ) as unknown as typeof fetch;
    const { app } = createTestApp(undefined, { store, instanceStore, fetchImpl: mockFetch });
    await registerInstance(app, "ha-expiry", "https://srat.example/cb");
    const startRes = await signedStartRequest(app, { provider: "dropbox", srat_callback_url: "https://srat.example/cb", instance_id: "ha-expiry" });
    const { session_id } = (await jsonBody(startRes)) as { session_id: string };
    await app.request(`/v1/callback?code=x&state=${session_id}`, { headers: { accept: "application/json" } });
    const stored = await store.get(session_id);
    const envl = JSON.parse(stored!.tokenJson!);
    expect(envl.expiry).toBe("2030-01-01T01:00:00.000Z");
  });
});
