import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { MemorySessionStore, KVSessionStore } from "../src/session.js";

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
