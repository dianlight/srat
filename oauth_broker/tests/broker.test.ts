import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { createTestApp, jsonBody, testEnv, registerInstance, signedHeaders, getOrCreateDefaultKeyPair, signedStartRequest, signedSessionRequest } from "./utils.js";
import { MemorySessionStore, MemoryInstanceStore } from "../src/session.js";
import { getSessionTtlSeconds, loadProvidersConfig } from "../src/config.js";
import { isValidSratCallbackUrl, __clearRateLimitBucketsForTests } from "../src/app.js";

describe("isValidSratCallbackUrl", () => {
  it("accepts https absolute", () => {
    expect(isValidSratCallbackUrl("https://srat.example.com/api/rclone/oauth/callback?state=abc")).toBe(true);
  });
  it("accepts loopback http for dev", () => {
    expect(isValidSratCallbackUrl("http://localhost:3000/callback?state=1")).toBe(true);
    expect(isValidSratCallbackUrl("http://127.0.0.1/callback")).toBe(true);
    expect(isValidSratCallbackUrl("http://[::1]/callback")).toBe(true);
  });
  it("rejects non-https non-loopback", () => {
    expect(isValidSratCallbackUrl("http://srat.example.com/callback")).toBe(false);
    expect(isValidSratCallbackUrl("not-a-url")).toBe(false);
    expect(isValidSratCallbackUrl("ftp://example.com")).toBe(false);
  });
});

describe("config", () => {
  it("session TTL parses variants", () => {
    expect(getSessionTtlSeconds({ SESSION_TTL: "600" })).toBe(600);
    expect(getSessionTtlSeconds({ SESSION_TTL: "10m" })).toBe(600);
    expect(getSessionTtlSeconds({ SESSION_TTL: "600s" })).toBe(600);
    expect(getSessionTtlSeconds({})).toBe(600);
  });

  it("loadProvidersConfig merges DROPBOX_* and BROKER_PROVIDERS_JSON", () => {
    const env = {
      DROPBOX_CLIENT_ID: "id1",
      DROPBOX_CLIENT_SECRET: "sec1",
      BROKER_PROVIDERS_JSON: JSON.stringify({
        gdrive: {
          client_id: "g-id",
          client_secret: "g-sec",
          authorize_url: "https://accounts.google.com/o/oauth2/v2/auth",
          token_url: "https://oauth2.googleapis.com/token",
          scopes: ["https://www.googleapis.com/auth/drive"],
          auth_params: { access_type: "offline" },
        },
      }),
    };
    const cfg = loadProvidersConfig(env);
    expect(cfg.dropbox.client_id).toBe("id1");
    expect(cfg.dropbox.authorize_url).toBe("https://www.dropbox.com/oauth2/authorize");
    expect(cfg.gdrive.authorize_url).toBe("https://accounts.google.com/o/oauth2/v2/auth");
  });

  it("file + env both: env wins on overlap", () => {
    const env = testEnv({ DROPBOX_CLIENT_ID: "from-env", DROPBOX_CLIENT_SECRET: "sec" });
    const cfg = loadProvidersConfig(env);
    expect(cfg.dropbox.client_id).toBe("from-env");
  });
});

describe("broker endpoints", () => {
  let store: MemorySessionStore;
  let instanceStore: MemoryInstanceStore;

  beforeEach(() => {
    store = new MemorySessionStore();
    instanceStore = new MemoryInstanceStore();
    __clearRateLimitBucketsForTests();
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  async function prepareRegisteredApp(cbUrl = "https://srat.example.com/cb", id = "ha-instance-1") {
    const { app } = createTestApp(undefined, { store, instanceStore });
    const reg = await registerInstance(app, id, cbUrl);
    expect(reg.status).toBe(200);
    return { app, instanceId: id, cbUrl };
  }

  describe("GET /v1/healthz", () => {
    it("returns ok with providers list", async () => {
      const { app } = createTestApp();
      const res = await app.request("/v1/healthz");
      expect(res.status).toBe(200);
      const body = await jsonBody(res);
      expect(body.status).toBe("ok");
      expect((body.providers as string[])).toContain("dropbox");
    });
  });

  describe("POST /v1/instances/register", () => {
    it("registers instance with exact redirect_url and TTL", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      const res = await app.request("/v1/instances/register", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/instances/register", JSON.stringify({ instance_id: "my-ha-123", redirect_url: "https://srat.example.com/cb" }))) }, body: JSON.stringify({ instance_id: "my-ha-123", redirect_url: "https://srat.example.com/cb" }),
      });
      expect(res.status).toBe(200);
      const body = await jsonBody(res);
      expect(body.instance_id).toBe("my-ha-123");
      expect(body.redirect_url).toBe("https://srat.example.com/cb");
      expect(body.ttl_seconds).toBe(3600);
      const stored = await instanceStore.get("my-ha-123");
      expect(stored?.redirectUrl).toBe("https://srat.example.com/cb");
    });

    it("rejects missing instance_id", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      const res = await app.request("/v1/instances/register", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/instances/register", JSON.stringify({ redirect_url: "https://srat.example.com/cb" }))) }, body: JSON.stringify({ redirect_url: "https://srat.example.com/cb" }),
      });
      expect(res.status).toBe(400);
    });

    it("rejects invalid redirect_url (non-https)", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      const res = await app.request("/v1/instances/register", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/instances/register", JSON.stringify({ instance_id: "id1", redirect_url: "http://evil.com/cb" }))) }, body: JSON.stringify({ instance_id: "id1", redirect_url: "http://evil.com/cb" }),
      });
      expect(res.status).toBe(400);
    });

    it("requires SRAT-Signature (401 without sig)", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      const res = await app.request("/v1/instances/register", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ instance_id: "id1", redirect_url: "https://srat.example.com/cb" }),
      });
      expect(res.status).toBe(401);
    });

    it("expires after TTL (1h)", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      await registerInstance(app, "exp-id", "https://srat.example.com/cb");
      expect(await instanceStore.get("exp-id")).not.toBeNull();
      vi.advanceTimersByTime(3601_000);
      expect(await instanceStore.get("exp-id")).toBeNull();
      // start should now 410
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "exp-id" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "exp-id" }),
      });
      expect(res.status).toBe(410);
    });
  });

  describe("POST /v1/start", () => {
    it("happy path with instance binding returns auth_url + session_id", async () => {
      const { app, cbUrl, instanceId } = await prepareRegisteredApp("https://srat.example.com/api/rclone/oauth/callback?state=st%3D1", "ha-1");
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: cbUrl, instance_id: instanceId }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: cbUrl, instance_id: instanceId }),
      });
      expect(res.status).toBe(200);
      const body = await jsonBody(res);
      expect(body.auth_url).toBeDefined();
      expect(body.session_id).toBeDefined();
      const authUrl = new URL(body.auth_url as string);
      expect(authUrl.hostname).toBe("www.dropbox.com");
      expect(authUrl.searchParams.get("client_id")).toBe("dropbox-id");
      expect(authUrl.searchParams.get("response_type")).toBe("code");
      expect(authUrl.searchParams.get("redirect_uri")).toBe("https://broker.example.com/v1/callback");
      expect(authUrl.searchParams.get("token_access_type")).toBe("offline");
      expect(authUrl.searchParams.get("state")).toBe(body.session_id);
      // session stores instanceId
      const sess = await store.get(body.session_id as string);
      expect(sess?.instanceId).toBe(instanceId);
    });

    it("hard fail 400 when instance_id missing", async () => {
      const { app } = await prepareRegisteredApp();
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb" }),
      });
      expect(res.status).toBe(400);
      expect((await jsonBody(res)).error).toMatch(/instance_id is required/i);
    });

    it("410 when instance not registered", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "unknown" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "unknown" }),
      });
      expect(res.status).toBe(410);
    });

    it("403 when srat_callback_url does not exactly match registered redirect_url", async () => {
      const { app } = await prepareRegisteredApp("https://srat.example.com/cb", "ha-1");
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/other", instance_id: "ha-1" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/other", instance_id: "ha-1" }),
      });
      expect(res.status).toBe(403);
      expect((await jsonBody(res)).error).toMatch(/does not match registered/i);
    });

    it("requires SRAT-Signature for /v1/start (401 without sig)", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "x" }),
      });
      expect(res.status).toBe(401);
    });

    it("rejects invalid srat_callback_url (non-https)", async () => {
      const { app } = await prepareRegisteredApp("https://srat.example.com/cb", "ha-1");
      // instance is registered for https, but we try http – should 400 before mismatch check? Actually 400 for invalid url
      // register another not needed – just test invalid url path with instance
      await registerInstance(app, "ha-http", "https://srat.example.com/cb");
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: "http://srat.example.com/cb", instance_id: "ha-http" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: "http://srat.example.com/cb", instance_id: "ha-http" }),
      });
      // invalid url -> 400 (exact mismatch also triggers but invalid first)
      expect([400, 403]).toContain(res.status);
    });

    it("allows loopback http for dev when registered", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      await registerInstance(app, "loop", "http://localhost:3000/callback?state=x");
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: "http://localhost:3000/callback?state=x", instance_id: "loop" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: "http://localhost:3000/callback?state=x", instance_id: "loop" }),
      });
      expect(res.status).toBe(200);
    });

    it("rejects unknown provider", async () => {
      const { app } = await prepareRegisteredApp("https://srat.example.com/cb", "ha-1");
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "nope", srat_callback_url: "https://srat.example.com/cb", instance_id: "ha-1" }))) }, body: JSON.stringify({ provider: "nope", srat_callback_url: "https://srat.example.com/cb", instance_id: "ha-1" }),
      });
      expect(res.status).toBe(400);
      const body = await jsonBody(res);
      expect(body.error).toMatch(/unknown provider/i);
    });

    it("trims trailing slash from BROKER_PUBLIC_URL", async () => {
      const env = testEnv({ BROKER_PUBLIC_URL: "https://broker.example.com/" });
      const instStore = new MemoryInstanceStore();
      const sessStore = new MemorySessionStore();
      const { app } = createTestApp(env, { store: sessStore, instanceStore: instStore });
      await app.request("/v1/instances/register", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/instances/register", JSON.stringify({ instance_id: "ha-1", redirect_url: "https://srat.example.com/cb" }))) }, body: JSON.stringify({ instance_id: "ha-1", redirect_url: "https://srat.example.com/cb" }),
      });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "ha-1" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "ha-1" }),
      });
      expect(res.status).toBe(200);
      const body = await jsonBody(res);
      expect((body.auth_url as string)).toContain("redirect_uri=https%3A%2F%2Fbroker.example.com%2Fv1%2Fcallback");
    });

    it("rejects when BROKER_PUBLIC_URL not configured", async () => {
      const env = testEnv({ BROKER_PUBLIC_URL: "" });
      const { app } = createTestApp(env, { store, instanceStore });
      await app.request("/v1/instances/register", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/instances/register", JSON.stringify({ instance_id: "ha-1", redirect_url: "https://srat.example.com/cb" }))) }, body: JSON.stringify({ instance_id: "ha-1", redirect_url: "https://srat.example.com/cb" }),
      });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "ha-1" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "ha-1" }),
      });
      expect(res.status).toBe(500);
    });
  });

  describe("GET /v1/callback", () => {
    it("400 when missing code or state (json mode)", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      const res = await app.request("/v1/callback?code=abc", { headers: { accept: "application/json" } });
      expect(res.status).toBe(400);
    });

    it("400 html when missing code (browser mode) shows i18n error", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      const res = await app.request("/v1/callback?code=abc");
      expect(res.status).toBe(400);
      const text = await res.text();
      expect(text).toContain("Invalid request");
    });

    it("410 when session unknown/expired", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      const res = await app.request("/v1/callback?code=c&state=unknown-sess", { headers: { accept: "application/json" } });
      expect(res.status).toBe(410);
    });

    it("410 html when session unknown shows localized (it)", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      const res = await app.request("/v1/callback?code=c&state=unknown", { headers: { "accept-language": "it" } });
      expect(res.status).toBe(410);
      const text = await res.text();
      expect(text).toContain("Sessione scaduta");
    });

    it("exchanges code and returns HTML page with redirect (browser mode)", async () => {
      const mockFetch = vi.fn(async () =>
        new Response(
          JSON.stringify({
            access_token: "at",
            token_type: "bearer",
            refresh_token: "rt",
            expires_in: 14400,
            account_id: "dbid:123",
          }),
          { status: 200, headers: { "content-type": "application/json" } }
        )
      ) as unknown as typeof fetch;
      const { app } = createTestApp(undefined, { store, instanceStore, fetchImpl: mockFetch });
      const cbUrl = "https://srat.example.com/cb?state=srState";
      await registerInstance(app, "ha-1", cbUrl);
      const startRes = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: cbUrl, instance_id: "ha-1" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: cbUrl, instance_id: "ha-1" }),
      });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      const cbRes = await app.request(`/v1/callback?code=thecode&state=${session_id}`, { method: "GET" });
      expect(cbRes.status).toBe(200);
      expect(cbRes.headers.get("content-type")).toContain("text/html");
      const html = await cbRes.text();
      expect(html).toContain("Authorization successful");
      expect(html).toContain("https://srat.example.com/cb?state=srState");
      expect(mockFetch).toHaveBeenCalledTimes(1);
      const stored = await store.get(session_id);
      expect(stored?.tokenJson).toBeDefined();
      const envelope = JSON.parse(stored!.tokenJson!);
      expect(envelope.access_token).toBe("at");
      expect(envelope.refresh_token).toBe("rt");
      expect(envelope.account_id).toBe("dbid:123");
      expect(envelope.expiry).toBe("2026-01-01T04:00:00.000Z");
      expect(envelope.token_type).toBe("bearer");
    });

    it("json mode still 302 when Accept: application/json", async () => {
      const mockFetch = vi.fn(async () =>
        new Response(JSON.stringify({ access_token: "at", expires_in: 3600 }), { status: 200 })
      ) as unknown as typeof fetch;
      const { app } = createTestApp(undefined, { store, instanceStore, fetchImpl: mockFetch });
      const cbUrl = "https://srat.example.com/cb";
      await registerInstance(app, "ha-json", cbUrl);
      const startRes = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: cbUrl, instance_id: "ha-json" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: cbUrl, instance_id: "ha-json" }),
      });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      const cbRes = await app.request(`/v1/callback?code=thecode&state=${session_id}`, { headers: { accept: "application/json" } });
      expect(cbRes.status).toBe(302);
      expect(cbRes.headers.get("location")).toBe(cbUrl);
    });

    it("502 html when provider token endpoint returns error", async () => {
      const mockFetch = vi.fn(async () =>
        new Response(JSON.stringify({ error: "invalid_grant", error_description: "bad code" }), { status: 400 })
      ) as unknown as typeof fetch;
      const { app } = createTestApp(undefined, { store, instanceStore, fetchImpl: mockFetch });
      const cbUrl = "https://srat.example.com/cb";
      await registerInstance(app, "ha-1", cbUrl);
      const startRes = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: cbUrl, instance_id: "ha-1" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: cbUrl, instance_id: "ha-1" }),
      });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      const cbRes = await app.request(`/v1/callback?code=bad&state=${session_id}`);
      expect(cbRes.status).toBe(502);
      const html = await cbRes.text();
      expect(html).toContain("Token exchange failed");
    });

    it("502 when fetch throws (json mode hides detail)", async () => {
      const mockFetch = vi.fn(async () => {
        throw new Error("network down");
      }) as unknown as typeof fetch;
      const { app } = createTestApp(undefined, { store, instanceStore, fetchImpl: mockFetch });
      const cbUrl = "https://srat.example.com/cb";
      await registerInstance(app, "ha-1", cbUrl);
      const startRes = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: cbUrl, instance_id: "ha-1" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: cbUrl, instance_id: "ha-1" }),
      });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      const cbRes = await app.request(`/v1/callback?code=x&state=${session_id}`, { headers: { accept: "application/json" } });
      expect(cbRes.status).toBe(502);
      const body = await jsonBody(cbRes);
      expect(body.error).toMatch(/token exchange failed/);
      expect(body.error).not.toMatch(/network down/);
    });

    it("410 when instance expired between start and callback", async () => {
      const mockFetch = vi.fn(async () => new Response(JSON.stringify({ access_token: "at" }), { status: 200 })) as unknown as typeof fetch;
      const { app } = createTestApp(undefined, { store, instanceStore, fetchImpl: mockFetch });
      const cbUrl = "https://srat.example.com/cb";
      await registerInstance(app, "ha-exp", cbUrl);
      const startRes = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: cbUrl, instance_id: "ha-exp" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: cbUrl, instance_id: "ha-exp" }),
      });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      vi.advanceTimersByTime(3601_000);
      const cbRes = await app.request(`/v1/callback?code=c&state=${session_id}`, { headers: { accept: "application/json" } });
      expect(cbRes.status).toBe(410);
    });
  });

  describe("GET /v1/session/:id", () => {
    it("404 without consuming when not yet completed (early polling)", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      await registerInstance(app, "ha-1", "https://srat.example.com/cb");
      const startRes = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", ...(await signedHeaders(await getOrCreateDefaultKeyPair(app), "POST", "/v1/start", JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "ha-1" }))) }, body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "ha-1" }),
      });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      const first = await signedSessionRequest(app, session_id);
      expect(first.status).toBe(404);
      expect(first.headers.get("cache-control")).toContain("no-store");
      const second = await signedSessionRequest(app, session_id);
      expect(second.status).toBe(404);
    });

    it("on first successful completed fetch returns token + credentials and consumes (single use)", async () => {
      const mockFetch = vi.fn(async () =>
        new Response(JSON.stringify({ access_token: "at", token_type: "bearer", refresh_token: "rt", expires_in: 3600, account_id: "acc1" }), {
          status: 200,
        })
      ) as unknown as typeof fetch;
      const { app } = createTestApp(undefined, { store, instanceStore, fetchImpl: mockFetch });
      await registerInstance(app, "ha-1", "https://srat.example.com/cb");
      const startRes = await signedStartRequest(app, { provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "ha-1" });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      await app.request(`/v1/callback?code=c&state=${session_id}`, { headers: { accept: "application/json" } });
      const got = await signedSessionRequest(app, session_id);
      expect(got.status).toBe(200);
      expect(got.headers.get("cache-control")).toContain("no-store");
      const body = (await jsonBody(got)) as { token_json: string; account_label: string; client_id: string; client_secret: string };
      expect(body.token_json).toBeDefined();
      expect(JSON.parse(body.token_json).access_token).toBe("at");
      expect(body.account_label).toBe("acc1");
      expect(body.client_id).toBe("dropbox-id");
      expect(body.client_secret).toBe("dropbox-secret");
      const second = await signedSessionRequest(app, session_id);
      expect(second.status).toBe(404);
      const secondBody = await jsonBody(second);
      expect(secondBody.error).toMatch(/expired or already used/i);
    });

    it("requires SRAT-Signature for /v1/session (401 without sig)", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      const res = await app.request("/v1/session/some-id");
      expect(res.status).toBe(401);
    });

    it("path escapes session id (valid UUID)", async () => {
      const validId = "550e8400-e29b-41d4-a716-446655440000";
      const customStore = new MemorySessionStore();
      await customStore.set(validId, { provider: "dropbox", sratCallbackUrl: "https://srat.example.com/cb", createdAt: Date.now(), tokenJson: `{"access_token":"at","expiry":"2026-01-01T00:01:40.000Z"}`, accountLabel: "acc", clientId: "id", clientSecret: "sec" }, 600);
      const { app: app2 } = createTestApp(undefined, { store: customStore, instanceStore });
      const kp = await getOrCreateDefaultKeyPair(app2);
      const encoded = encodeURIComponent(validId);
      const honoPath = `/v1/session/${encoded.replaceAll("%20", " ")}`;
      const headers = await signedHeaders(kp, "GET", honoPath, "");
      const res = await app2.request(`/v1/session/${encoded}`, { headers });
      expect(res.status).toBe(200);
    });

    it("session id rejects invalid charset (slash/space)", async () => {
      const { app: app2 } = createTestApp(undefined, { store: new MemorySessionStore(), instanceStore });
      const kp = await getOrCreateDefaultKeyPair(app2);
      const invalidId = "a/b c";
      const encoded = encodeURIComponent(invalidId);
      const honoPath = `/v1/session/${encoded.replaceAll("%20", " ")}`;
      const headers = await signedHeaders(kp, "GET", honoPath, "");
      const res = await app2.request(`/v1/session/${encoded}`, { headers });
      expect(res.status).toBe(400);
      expect((await res.json() as any).error).toMatch(/invalid session_id/);
    });
  });

  describe("auth SRAT-Signature", () => {
    it("rejects wrong signature (401)", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      await registerInstance(app, "ha-1", "https://srat.example.com/cb");
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: 'SRAT-Signature client_id="AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", t="1234567890", nonce="bad-nonce-1234567890", sig="AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"' },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "ha-1" }),
      });
      expect(res.status).toBe(401);
    });

    // Intentional: verifies dev bypass allows without sig — must keep BROKER_DISABLE_AUTH, do not migrate to signed
    it("allows when BROKER_DISABLE_AUTH (dev)", async () => {
      const env = testEnv({ BROKER_DISABLE_AUTH: "true", BROKER_PUBLIC_URL: "http://localhost:8787" });
      const { app } = createTestApp(env, { store, instanceStore });
      // register still needs auth bypass – should allow
      const reg = await app.request("/v1/instances/register", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ instance_id: "ha-dev", redirect_url: "https://srat.example.com/cb" }),
      });
      expect(reg.status).toBe(200);
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "ha-dev" }),
      });
      expect(res.status).toBe(200);
    });

    // Intentional: verifies fail-closed without sig/dev-bypass — must keep no BROKER_DISABLE_AUTH, do not migrate to signed
    it("rejects when BROKER_DISABLE_AUTH not set (fail-closed, no sig)", async () => {
      const env = testEnv();
      delete (env as Record<string, string | undefined>).BROKER_DISABLE_AUTH;
      const { app } = createTestApp(env, { store, instanceStore });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "x" }),
      });
      expect(res.status).toBe(401);
    });

    it("handles non-ASCII signature header (401 not 500)", async () => {
      const { app } = createTestApp(undefined, { store, instanceStore });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "SRAT-Signature client_id=\"é\", t=\"123\", nonce=\"bad\", sig=\"é\"" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb", instance_id: "ha-1" }),
      });
      expect(res.status).toBe(401);
    });
  });
});
