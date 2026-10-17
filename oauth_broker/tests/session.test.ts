import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { D1SessionStore, KVSessionStore, MemorySessionStore } from "../src/session.js";
import type { SessionRecord } from "../src/types.js";

describe("MemorySessionStore TTL", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("expires after TTL", async () => {
    const store = new MemorySessionStore();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    await store.set("s1", { provider: "dropbox", sratCallbackUrl: "https://x/cb", createdAt: Date.now() }, 600);
    expect(await store.get("s1")).not.toBeNull();
    vi.advanceTimersByTime(601_000);
    expect(await store.get("s1")).toBeNull();
  });

  it("delete removes", async () => {
    const store = new MemorySessionStore();
    await store.set("s1", { provider: "dropbox", sratCallbackUrl: "https://x/cb", createdAt: Date.now() }, 600);
    await store.delete("s1");
    expect(await store.get("s1")).toBeNull();
  });
});

describe("KVSessionStore JSON roundtrip", () => {
  it("stores via KV put with expirationTtl and parses on get", async () => {
    const kvData = new Map<string, string>();
    const kv = {
      get: async (k: string) => kvData.get(k) || null,
      put: async (k: string, v: string, opts?: { expirationTtl?: number }) => {
        expect(opts?.expirationTtl).toBe(600);
        kvData.set(k, v);
      },
      delete: async (k: string) => {
        kvData.delete(k);
      },
    };
    const store = new KVSessionStore(kv);
    await store.set("sess", { provider: "dropbox", sratCallbackUrl: "https://x/cb", createdAt: 123 }, 600);
    const got = await store.get("sess");
    expect(got?.provider).toBe("dropbox");
    await store.delete("sess");
    expect(await store.get("sess")).toBeNull();
  });

  it("returns null on malformed JSON", async () => {
    const kv = {
      get: async () => "not-json",
      put: async () => {},
      delete: async () => {},
    };
    const store = new KVSessionStore(kv);
    expect(await store.get("bad")).toBeNull();
  });
});

function createMockD1() {
  const rows = new Map<string, { data: string; expires_at: number }>();
  return {
    rows,
    db: {
      prepare(query: string) {
        const q = query.trim().toLowerCase();
        return {
          bind(...values: unknown[]) {
            if (q.startsWith("insert or replace")) {
              const [id, data, expires_at] = values as [string, string, number];
              return {
                async run() {
                  rows.set(id, { data, expires_at });
                  return { success: true };
                },
                async first<T>() {
                  return null as T | null;
                },
                async all<T>() {
                  return { results: [] as T[] };
                },
              };
            }
            if (q.startsWith("select data, expires_at from sessions where id =")) {
              const [id] = values as [string];
              return {
                async run() {
                  return { success: true };
                },
                async first<T>() {
                  const r = rows.get(id);
                  if (!r) return null;
                  return r as unknown as T;
                },
                async all<T>() {
                  return { results: [] as T[] };
                },
              };
            }
            if (q.startsWith("delete from sessions where id = ? and expires_at >")) {
              const [id, now] = values as [string, number];
              return {
                async run() {
                  return { success: true };
                },
                async first<T>() {
                  const r = rows.get(id);
                  if (!r) return null;
                  if (r.expires_at <= (now as number)) return null;
                  rows.delete(id);
                  return { data: r.data } as unknown as T;
                },
                async all<T>() {
                  return { results: [] as T[] };
                },
              };
            }
            if (q.startsWith("delete from sessions where id =")) {
              const [id] = values as [string];
              return {
                async run() {
                  rows.delete(id);
                  return { success: true };
                },
                async first<T>() {
                  return null as T | null;
                },
                async all<T>() {
                  return { results: [] as T[] };
                },
              };
            }
            throw new Error(`unhandled mock query: ${query}`);
          },
        };
      },
    } as unknown as import("../src/session.js").D1DatabaseLike,
  };
}

describe("D1SessionStore — atomic via DELETE ... RETURNING", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("stores and retrieves before expiry", async () => {
    const { db } = createMockD1();
    const store = new D1SessionStore(db);
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    const rec: SessionRecord = { provider: "dropbox", sratCallbackUrl: "https://x/cb", createdAt: Date.now() };
    await store.set("s1", rec, 600);
    const got = await store.get("s1");
    expect(got?.provider).toBe("dropbox");
  });

  it("expires after TTL and prunes on get", async () => {
    const { db, rows } = createMockD1();
    const store = new D1SessionStore(db);
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    await store.set("s1", { provider: "dropbox", sratCallbackUrl: "https://x/cb", createdAt: Date.now() }, 600);
    vi.advanceTimersByTime(601_000);
    expect(await store.get("s1")).toBeNull();
    // Row was pruned.
    expect(rows.has("s1")).toBe(false);
  });

  it("consume is atomic — only one succeeds", async () => {
    const { db } = createMockD1();
    const store = new D1SessionStore(db);
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    const rec: SessionRecord = {
      provider: "dropbox",
      sratCallbackUrl: "https://x/cb",
      createdAt: Date.now(),
      tokenJson: '{"access_token":"at"}',
      accountLabel: "acc",
    };
    await store.set("s1", rec, 600);
    const a = await store.consume("s1");
    const b = await store.consume("s1");
    expect(a?.tokenJson).toBe('{"access_token":"at"}');
    expect(b).toBeNull();
  });

  it("consume returns null on expired row and does not leak", async () => {
    const { db, rows } = createMockD1();
    const store = new D1SessionStore(db);
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    await store.set("s1", { provider: "dropbox", sratCallbackUrl: "https://x/cb", createdAt: Date.now(), tokenJson: "t" }, 1);
    vi.advanceTimersByTime(2_000);
    expect(await store.consume("s1")).toBeNull();
    expect(rows.has("s1")).toBe(true); // expired row stays until get/consume with RETURNING fails; second consume verifies not leaked
    expect(await store.get("s1")).toBeNull(); // get prunes it
  });

  it("delete removes", async () => {
    const { db } = createMockD1();
    const store = new D1SessionStore(db);
    await store.set("s1", { provider: "dropbox", sratCallbackUrl: "https://x/cb", createdAt: Date.now() }, 600);
    await store.delete("s1");
    expect(await store.get("s1")).toBeNull();
  });
});
