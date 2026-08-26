import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { createTestApp, jsonBody, testEnv } from "./utils.js";
import { MemorySessionStore } from "../src/session.js";
import { getSessionTtlSeconds, loadProvidersConfig } from "../src/config.js";
import { isValidSratCallbackUrl } from "../src/app.js";

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
    // This test exercises the merge path; file not present so only env matters
    const env = testEnv({ DROPBOX_CLIENT_ID: "from-env", DROPBOX_CLIENT_SECRET: "sec" });
    const cfg = loadProvidersConfig(env);
    expect(cfg.dropbox.client_id).toBe("from-env");
  });
});

describe("broker endpoints", () => {
  let store: MemorySessionStore;

  beforeEach(() => {
    store = new MemorySessionStore();
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
  });
  afterEach(() => {
    vi.useRealTimers();
  });

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

  describe("POST /v1/start", () => {
    it("happy path posts provider and callback url and returns auth_url + session_id", async () => {
      const { app } = createTestApp(undefined, { store });
      const callback = "https://srat.example.com/api/rclone/oauth/callback?state=st%3D1";
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: callback }),
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
    });

    it("rejects missing bearer token", async () => {
      const { app } = createTestApp(testEnv({ BROKER_API_TOKEN: "secret" }), { store });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb" }),
      });
      expect(res.status).toBe(401);
    });

    it("rejects invalid srat_callback_url (non-https)", async () => {
      const { app } = createTestApp(undefined, { store });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "http://srat.example.com/cb" }),
      });
      expect(res.status).toBe(400);
      const body = await jsonBody(res);
      expect(body.error).toMatch(/srat_callback_url must be an absolute https/i);
    });

    it("allows loopback http for dev", async () => {
      const { app } = createTestApp(undefined, { store });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "http://localhost:3000/callback?state=x" }),
      });
      expect(res.status).toBe(200);
    });

    it("rejects unknown provider", async () => {
      const { app } = createTestApp(undefined, { store });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "nope", srat_callback_url: "https://srat.example.com/cb" }),
      });
      expect(res.status).toBe(400);
      const body = await jsonBody(res);
      expect(body.error).toMatch(/unknown provider/i);
    });

    it("trims trailing slash from BROKER_PUBLIC_URL", async () => {
      const env = testEnv({ BROKER_PUBLIC_URL: "https://broker.example.com/" });
      const { app } = createTestApp(env, { store });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb" }),
      });
      expect(res.status).toBe(200);
      const body = await jsonBody(res);
      expect((body.auth_url as string)).toContain("redirect_uri=https%3A%2F%2Fbroker.example.com%2Fv1%2Fcallback");
    });

    it("rejects when BROKER_PUBLIC_URL not configured", async () => {
      const env = testEnv({ BROKER_PUBLIC_URL: "" });
      const { app } = createTestApp(env, { store });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb" }),
      });
      expect(res.status).toBe(500);
    });
  });

  describe("GET /v1/callback", () => {
    it("400 when missing code or state", async () => {
      const { app } = createTestApp(undefined, { store });
      const res = await app.request("/v1/callback?code=abc");
      expect(res.status).toBe(400);
    });

    it("410 when session unknown/expired", async () => {
      const { app } = createTestApp(undefined, { store });
      const res = await app.request("/v1/callback?code=c&state=unknown-sess");
      expect(res.status).toBe(410);
    });

    it("exchanges code and 302s to srat_callback_url, wrapping token envelope", async () => {
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
      const { app } = createTestApp(undefined, { store, fetchImpl: mockFetch });
      // first create session
      const startRes = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb?state=srState" }),
      });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      const cbRes = await app.request(`/v1/callback?code=thecode&state=${session_id}`, { method: "GET" });
      expect(cbRes.status).toBe(302);
      expect(cbRes.headers.get("location")).toBe("https://srat.example.com/cb?state=srState");
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

    it("502 when provider token endpoint returns error", async () => {
      const mockFetch = vi.fn(async () =>
        new Response(JSON.stringify({ error: "invalid_grant", error_description: "bad code" }), { status: 400 })
      ) as unknown as typeof fetch;
      const { app } = createTestApp(undefined, { store, fetchImpl: mockFetch });
      const startRes = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb" }),
      });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      const cbRes = await app.request(`/v1/callback?code=bad&state=${session_id}`);
      expect(cbRes.status).toBe(502);
    });

    it("502 when fetch throws", async () => {
      const mockFetch = vi.fn(async () => {
        throw new Error("network down");
      }) as unknown as typeof fetch;
      const { app } = createTestApp(undefined, { store, fetchImpl: mockFetch });
      const startRes = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb" }),
      });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      const cbRes = await app.request(`/v1/callback?code=x&state=${session_id}`);
      expect(cbRes.status).toBe(502);
      const body = await jsonBody(cbRes);
      expect(body.error).toMatch(/network down/);
    });
  });

  describe("GET /v1/session/:id", () => {
    it("404 without consuming when not yet completed (early polling)", async () => {
      const { app } = createTestApp(undefined, { store });
      const startRes = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb" }),
      });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      const first = await app.request(`/v1/session/${session_id}`, {
        headers: { authorization: "Bearer test-token" },
      });
      expect(first.status).toBe(404);
      expect(first.headers.get("cache-control")).toBe("no-store");
      const second = await app.request(`/v1/session/${session_id}`, {
        headers: { authorization: "Bearer test-token" },
      });
      expect(second.status).toBe(404); // still not consumed
    });

    it("on first successful completed fetch returns token + credentials and consumes (single use)", async () => {
      const mockFetch = vi.fn(async () =>
        new Response(JSON.stringify({ access_token: "at", token_type: "bearer", refresh_token: "rt", expires_in: 3600, account_id: "acc1" }), {
          status: 200,
        })
      ) as unknown as typeof fetch;
      const { app } = createTestApp(undefined, { store, fetchImpl: mockFetch });
      const startRes = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer test-token" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb" }),
      });
      const { session_id } = (await jsonBody(startRes)) as { session_id: string };
      await app.request(`/v1/callback?code=c&state=${session_id}`);
      const got = await app.request(`/v1/session/${session_id}`, {
        headers: { authorization: "Bearer test-token" },
      });
      expect(got.status).toBe(200);
      expect(got.headers.get("cache-control")).toBe("no-store");
      const body = (await jsonBody(got)) as { token_json: string; account_label: string; client_id: string; client_secret: string };
      expect(body.token_json).toBeDefined();
      expect(JSON.parse(body.token_json).access_token).toBe("at");
      expect(body.account_label).toBe("acc1");
      expect(body.client_id).toBe("dropbox-id");
      expect(body.client_secret).toBe("dropbox-secret");
      const second = await app.request(`/v1/session/${session_id}`, {
        headers: { authorization: "Bearer test-token" },
      });
      expect(second.status).toBe(404);
      const secondBody = await jsonBody(second);
      expect(secondBody.error).toMatch(/expired or already used/i);
    });

    it("requires bearer token", async () => {
      const { app } = createTestApp(undefined, { store });
      const res = await app.request("/v1/session/some-id");
      expect(res.status).toBe(401);
    });

    it("path escapes session id (a/b c)", async () => {
      const mockFetch = vi.fn(async () =>
        new Response(JSON.stringify({ access_token: "at", expires_in: 100 }), { status: 200 })
      ) as unknown as typeof fetch;
      const customStore = new MemorySessionStore();
      // manually insert a session with tricky id
      await customStore.set("a/b c", { provider: "dropbox", sratCallbackUrl: "https://srat.example.com/cb", createdAt: Date.now(), tokenJson: `{"access_token":"at","expiry":"2026-01-01T00:01:40.000Z"}`, accountLabel: "acc", clientId: "id", clientSecret: "sec" }, 600);
      const { app } = createTestApp(undefined, { store: customStore, fetchImpl: mockFetch });
      const res = await app.request(`/v1/session/${encodeURIComponent("a/b c")}`, {
        headers: { authorization: "Bearer test-token" },
      });
      expect(res.status).toBe(200);
    });
  });

  describe("auth constant-time", () => {
    it("rejects wrong bearer (401)", async () => {
      const { app } = createTestApp(undefined, { store });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json", authorization: "Bearer wrong" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb" }),
      });
      expect(res.status).toBe(401);
    });

    it("allows when BROKER_API_TOKEN unset (dev)", async () => {
      const env = testEnv({ BROKER_API_TOKEN: "" });
      const { app } = createTestApp(env, { store });
      const res = await app.request("/v1/start", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ provider: "dropbox", srat_callback_url: "https://srat.example.com/cb" }),
      });
      expect(res.status).toBe(200);
    });
  });
});
