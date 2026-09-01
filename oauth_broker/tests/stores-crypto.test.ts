import { describe, it, expect, vi } from "vitest";
import {
  MemoryClientStore,
  MemoryNonceStore,
  KVClientStore,
  KVNonceStore,
  MemoryInstanceStore,
  D1NonceStore,
  D1ClientStore,
  D1InstanceStore,
  D1SessionStore,
  KVInstanceStore,
  KVSessionStore,
  MemorySessionStore,
} from "../src/session.js";
import { bodyHashBase64Url, buildStringToSign, computeClientId, generateEd25519KeyPair, isValidClientId, isValidNonce, isValidPublicKeyB64Url, parseSratSignature, verifyEd25519 } from "../src/crypto.js";

describe("crypto helpers", () => {
  it("computeClientId and isValid", async () => {
    const kp = await generateEd25519KeyPair();
    expect(isValidClientId(kp.clientId)).toBe(true);
    expect(isValidPublicKeyB64Url(kp.publicKeyB64Url)).toBe(true);
    expect(computeClientId(kp.publicKeyB64Url)).toBe(kp.clientId);
    expect(isValidClientId("bad")).toBe(false);
    expect(isValidPublicKeyB64Url("bad")).toBe(false);
    expect(isValidNonce("n-" + "a".repeat(20))).toBe(true);
    expect(isValidNonce("short")).toBe(false);
    expect(parseSratSignature(`SRAT-Signature client_id="${kp.clientId}", t="123", nonce="n-1234567890123456", sig="abcd"`)).not.toBeNull();
    expect(parseSratSignature("Bearer test")).toBeNull();
    expect(bodyHashBase64Url("")).toBe("47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU");
    expect(buildStringToSign(kp.clientId, "POST", "/v1/start", "123", "nonce-1234567890", "hash")).toContain(kp.clientId);
    const bad = await verifyEd25519(kp.publicKeyB64Url, "msg", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA");
    expect(bad).toBe(false);
  });
});

describe("MemoryClientStore", () => {
  it("set/get/delete", async () => {
    const s = new MemoryClientStore();
    await s.set("cid1", { clientId: "cid1", publicKey: "pub", createdAt: Date.now() });
    expect((await s.get("cid1"))?.clientId).toBe("cid1");
    await s.delete("cid1");
    expect(await s.get("cid1")).toBeNull();
  });

  it("clear and size", async () => {
    const s = new MemoryClientStore();
    expect(s.size()).toBe(0);
    await s.set("cid1", { clientId: "cid1", publicKey: "pub1", createdAt: Date.now() });
    await s.set("cid2", { clientId: "cid2", publicKey: "pub2", createdAt: Date.now() });
    expect(s.size()).toBe(2);
    s.clear();
    expect(s.size()).toBe(0);
    expect(await s.get("cid1")).toBeNull();
    expect(await s.get("cid2")).toBeNull();
  });
});

describe("MemoryNonceStore", () => {
  it("has/set dedup", async () => {
    const s = new MemoryNonceStore();
    expect(await s.set("nonce-abc-1234567890", 600)).toBe(true);
    expect(await s.has("nonce-abc-1234567890")).toBe(true);
    expect(await s.set("nonce-abc-1234567890", 600)).toBe(false);
  });

  it("clear removes all entries", async () => {
    const s = new MemoryNonceStore();
    await s.set("nonce-clear-1-1234567890", 600);
    await s.set("nonce-clear-2-1234567890", 600);
    expect(await s.has("nonce-clear-1-1234567890")).toBe(true);
    s.clear();
    expect(await s.has("nonce-clear-1-1234567890")).toBe(false);
    expect(await s.has("nonce-clear-2-1234567890")).toBe(false);
  });
});

describe("KVClientStore", () => {
  it("roundtrip", async () => {
    const map = new Map<string, string>();
    const kv = { get: async (k: string) => map.get(k) || null, put: async (k: string, v: string) => { map.set(k, v); }, delete: async (k: string) => { map.delete(k); } };
    const s = new KVClientStore(kv);
    await s.set("cid", { clientId: "cid", publicKey: "pub", createdAt: 123 });
    expect((await s.get("cid"))?.publicKey).toBe("pub");
    await s.delete("cid");
    expect(await s.get("cid")).toBeNull();
  });
});

describe("KVNonceStore", () => {
  it("roundtrip", async () => {
    const map = new Map<string, string>();
    const kv = { get: async (k: string) => map.get(k) || null, put: async (k: string, v: string) => { map.set(k, v); }, delete: async (k: string) => { map.delete(k); } };
    const s = new KVNonceStore(kv);
    expect(await s.set("nonce-xyz-1234567890", 600)).toBe(true);
    expect(await s.has("nonce-xyz-1234567890")).toBe(true);
    expect(await s.set("nonce-xyz-1234567890", 600)).toBe(false);
  });
});

describe("MemoryInstanceStore with clientId", () => {
  it("stores clientId", async () => {
    const s = new MemoryInstanceStore();
    await s.set("id1", { instanceId: "id1", redirectUrl: "https://x/cb", createdAt: Date.now(), clientId: "cid1" }, 600);
    expect((await s.get("id1"))?.clientId).toBe("cid1");
  });
});

// ---- Helpers for D1/KV nonce single-use tests ----
function createFakeD1ForNonces() {
  const nonces = new Map<string, number>();
  const db = {
    prepare: (sql: string) => ({
      bind: (...vals: unknown[]) => ({
        first: async <T>() => {
          if (sql.includes("SELECT expires_at FROM nonces")) {
            const id = vals[0] as string;
            const exp = nonces.get(id);
            if (exp === undefined) return null as unknown as T;
            return { expires_at: exp } as unknown as T;
          }
          if (sql.includes("SELECT data FROM clients")) {
            return null as unknown as T;
          }
          if (sql.includes("SELECT data, expires_at FROM instances")) {
            return null as unknown as T;
          }
          if (sql.includes("SELECT data, expires_at FROM sessions")) {
            return null as unknown as T;
          }
          return null as unknown as T;
        },
        run: async () => {
          if (sql.includes("DELETE FROM nonces WHERE id = ? AND expires_at <=")) {
            const [id, now] = vals as [string, number];
            const exp = nonces.get(id);
            if (exp !== undefined && exp <= now) nonces.delete(id);
            return { success: true };
          }
          if (sql.includes("DELETE FROM nonces WHERE id = ?")) {
            nonces.delete(vals[0] as string);
            return { success: true };
          }
          if (sql.includes("INSERT INTO nonces")) {
            const [id, exp] = vals as [string, number];
            if (nonces.has(id)) throw new Error("UNIQUE constraint failed: nonces.id");
            nonces.set(id, exp as number);
            return { success: true };
          }
          return { success: true };
        },
        all: async () => ({ results: [] }),
      }),
    }),
    _nonces: nonces,
  };
  return db;
}

function createFakeD1Full() {
  const nonces = new Map<string, number>();
  const clients = new Map<string, string>();
  const instances = new Map<string, { data: string; expires_at: number }>();
  const sessions = new Map<string, { data: string; expires_at: number }>();
  return {
    prepare: (sql: string) => ({
      bind: (...vals: unknown[]) => ({
        first: async <T>() => {
          if (sql.includes("SELECT expires_at FROM nonces")) {
            const id = vals[0] as string;
            const exp = nonces.get(id);
            if (exp === undefined) return null as unknown as T;
            return { expires_at: exp } as unknown as T;
          }
          if (sql.includes("SELECT data FROM clients")) {
            const id = vals[0] as string;
            const data = clients.get(id);
            if (data === undefined) return null as unknown as T;
            return { data } as unknown as T;
          }
          if (sql.includes("SELECT data, expires_at FROM instances")) {
            const id = vals[0] as string;
            const row = instances.get(id);
            if (!row) return null as unknown as T;
            return row as unknown as T;
          }
          if (sql.includes("SELECT data, expires_at FROM sessions")) {
            const id = vals[0] as string;
            const row = sessions.get(id);
            if (!row) return null as unknown as T;
            return row as unknown as T;
          }
          if (sql.includes("DELETE FROM sessions WHERE id = ? AND expires_at > ? RETURNING")) {
            const [id, now] = vals as [string, number];
            const row = sessions.get(id);
            if (!row || row.expires_at <= now) return null as unknown as T;
            sessions.delete(id);
            return { data: row.data } as unknown as T;
          }
          return null as unknown as T;
        },
        run: async () => {
          if (sql.includes("DELETE FROM nonces WHERE id = ? AND expires_at <=")) {
            const [id, now] = vals as [string, number];
            const exp = nonces.get(id);
            if (exp !== undefined && exp <= now) nonces.delete(id);
            return { success: true };
          }
          if (sql.startsWith("DELETE FROM nonces WHERE id = ?")) {
            nonces.delete(vals[0] as string);
            return { success: true };
          }
          if (sql.includes("INSERT INTO nonces")) {
            const [id, exp] = vals as [string, number];
            if (nonces.has(id)) throw new Error("UNIQUE constraint failed: nonces.id");
            nonces.set(id, exp as number);
            return { success: true };
          }
          if (sql.includes("INSERT OR REPLACE INTO clients")) {
            const [id, data] = vals as [string, string];
            clients.set(id, data);
            return { success: true };
          }
          if (sql.includes("DELETE FROM clients")) {
            clients.delete(vals[0] as string);
            return { success: true };
          }
          if (sql.includes("INSERT OR REPLACE INTO instances")) {
            const [id, data, exp] = vals as [string, string, number];
            instances.set(id, { data, expires_at: exp });
            return { success: true };
          }
          if (sql.includes("DELETE FROM instances WHERE id = ?") && !sql.includes("AND")) {
            instances.delete(vals[0] as string);
            return { success: true };
          }
          if (sql.includes("INSERT OR REPLACE INTO sessions")) {
            const [id, data, exp] = vals as [string, string, number];
            sessions.set(id, { data, expires_at: exp });
            return { success: true };
          }
          if (sql.includes("DELETE FROM sessions WHERE id = ?") && !sql.includes("AND")) {
            sessions.delete(vals[0] as string);
            return { success: true };
          }
          if (sql.includes("DELETE FROM sessions WHERE id = ? AND expires_at > ?")) {
            // handled in first() for RETURNING variant; run variant no-op
            return { success: true };
          }
          return { success: true };
        },
        all: async () => ({ results: [] }),
      }),
    }),
    _nonces: nonces,
    _clients: clients,
    _instances: instances,
    _sessions: sessions,
  };
}

describe("MemoryNonceStore single-use + TTL", () => {
  it("second sequential set is rejected, has() true within TTL", async () => {
    const s = new MemoryNonceStore();
    expect(await s.set("nonce-mem-123456789012", 600)).toBe(true);
    expect(await s.has("nonce-mem-123456789012")).toBe(true);
    expect(await s.set("nonce-mem-123456789012", 600)).toBe(false);
    expect(await s.has("nonce-mem-123456789012")).toBe(true);
  });

  it("concurrent sets: only one wins", async () => {
    const s = new MemoryNonceStore();
    const [a, b] = await Promise.all([s.set("nonce-mem-conc-1234567890", 600), s.set("nonce-mem-conc-1234567890", 600)]);
    expect([a, b].filter(Boolean).length).toBe(1);
    expect(await s.has("nonce-mem-conc-1234567890")).toBe(true);
  });

  it("expired nonce can be reused after TTL (fake timers)", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    const s = new MemoryNonceStore();
    expect(await s.set("nonce-mem-exp-1234567890", 10)).toBe(true);
    expect(await s.has("nonce-mem-exp-1234567890")).toBe(true);
    vi.advanceTimersByTime(11_000);
    expect(await s.has("nonce-mem-exp-1234567890")).toBe(false);
    expect(await s.set("nonce-mem-exp-1234567890", 10)).toBe(true);
    expect(await s.has("nonce-mem-exp-1234567890")).toBe(true);
    vi.useRealTimers();
  });

  it("expired-reuse branch: evictExpired spares current nonce for explicit delete", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    const s = new MemoryNonceStore();
    await s.set("nonce-a-123456789012", 5);
    await s.set("nonce-b-123456789012", 5);
    vi.advanceTimersByTime(6_000);
    // Both nonces are now expired. Next set for nonce-a should take the
    // `if (existing !== undefined) { if (Date.now() > existing) map.delete }` path:
    // evictExpired('nonce-a-...') keeps nonce-a while evicting nonce-b.
    expect(await s.set("nonce-a-123456789012", 5)).toBe(true);
    expect(await s.has("nonce-a-123456789012")).toBe(true);
    expect(await s.has("nonce-b-123456789012")).toBe(false);
    vi.useRealTimers();
  });

  it("boundary: expiration == Date.now() at set time is still valid (not expired)", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    const s = new MemoryNonceStore();
    await s.set("nonce-boundary-12345678", 10); // expires at T0 + 10_000
    vi.advanceTimersByTime(10_000); // now == expiration exactly
    // has() uses `Date.now() > exp` so equality is still valid, not evicted
    expect(await s.has("nonce-boundary-12345678")).toBe(true);
    // set() uses `Date.now() <= existing` so equality is duplicate, not reuse
    expect(await s.set("nonce-boundary-12345678", 10)).toBe(false);
    // evictExpired with except keeps it, other keys with same expiry would not
    expect(await s.has("nonce-boundary-12345678")).toBe(true);
    vi.advanceTimersByTime(1); // now = expiration + 1 -> expired
    expect(await s.has("nonce-boundary-12345678")).toBe(false);
    // now reuse is allowed via expired-reuse delete branch
    expect(await s.set("nonce-boundary-12345678", 10)).toBe(true);
    expect(await s.has("nonce-boundary-12345678")).toBe(true);
    vi.useRealTimers();
  });

  it("boundary: TTL 0 expiration equals Date.now() at set time is still valid", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    const s = new MemoryNonceStore();
    expect(await s.set("nonce-ttl0-123456789012", 0)).toBe(true);
    // expires_at = Date.now() + 0*1000 == now, so has() should be true (now > exp is false)
    expect(await s.has("nonce-ttl0-123456789012")).toBe(true);
    // duplicate set with same ttl 0 while still valid should be rejected (now <= existing is true)
    expect(await s.set("nonce-ttl0-123456789012", 0)).toBe(false);
    expect(await s.has("nonce-ttl0-123456789012")).toBe(true);
    vi.advanceTimersByTime(1); // now = expiration + 1 -> expired
    expect(await s.has("nonce-ttl0-123456789012")).toBe(false);
    // now reuse allowed via expired-reuse branch
    expect(await s.set("nonce-ttl0-123456789012", 0)).toBe(true);
    expect(await s.has("nonce-ttl0-123456789012")).toBe(true);
    vi.useRealTimers();
  });
});

describe("KVNonceStore single-use semantics", () => {
  it("sequential second set is rejected", async () => {
    const map = new Map<string, string>();
    const kv = { get: async (k: string) => map.get(k) || null, put: async (k: string, v: string) => { map.set(k, v); }, delete: async (k: string) => { map.delete(k); } };
    const s = new KVNonceStore(kv);
    expect(await s.set("nonce-kv-seq-1234567890", 600)).toBe(true);
    expect(await s.has("nonce-kv-seq-1234567890")).toBe(true);
    expect(await s.set("nonce-kv-seq-1234567890", 600)).toBe(false);
  });

  it("concurrent sets may both succeed due to KV lack of CAS (documents limitation)", async () => {
    // Simulate KV with delayed put to expose race: two concurrent has() both see false before either put
    const map = new Map<string, string>();
    let putDelay = 5;
    const kv = {
      get: async (k: string) => map.get(k) || null,
      put: async (k: string, v: string) => {
        await new Promise((r) => setTimeout(r, putDelay));
        map.set(k, v);
      },
      delete: async (k: string) => { map.delete(k); },
    };
    const s = new KVNonceStore(kv);
    const [a, b] = await Promise.all([s.set("nonce-kv-race-1234567890", 600), s.set("nonce-kv-race-1234567890", 600)]);
    // With no CAS, both may succeed – we assert at least one succeeds and document best-effort
    expect(a || b).toBe(true);
    // Sequential after race must be rejected
    expect(await s.set("nonce-kv-race-1234567890", 600)).toBe(false);
  });

  it("has() reflects put, delete via expiry TTL not simulated here", async () => {
    const map = new Map<string, string>();
    const kv = { get: async (k: string) => map.get(k) || null, put: async (k: string, v: string) => { map.set(k, v); }, delete: async (k: string) => { map.delete(k); } };
    const s = new KVNonceStore(kv, "test-nonce:");
    expect(await s.has("nonce-kv-has-1234567890")).toBe(false);
    await s.set("nonce-kv-has-1234567890", 600);
    expect(await s.has("nonce-kv-has-1234567890")).toBe(true);
    expect(map.has("test-nonce:nonce-kv-has-1234567890")).toBe(true);
  });
});

describe("D1NonceStore single-use atomic semantics", () => {
  it("first set true, second false, has true within TTL", async () => {
    const db = createFakeD1ForNonces();
    const s = new D1NonceStore(db as any);
    expect(await s.set("nonce-d1-12345678901234", 600)).toBe(true);
    expect(await s.has("nonce-d1-12345678901234")).toBe(true);
    expect(await s.set("nonce-d1-12345678901234", 600)).toBe(false);
    expect(await s.has("nonce-d1-12345678901234")).toBe(true);
  });

  it("concurrent sets: only one wins via PK uniqueness", async () => {
    const db = createFakeD1ForNonces();
    const s = new D1NonceStore(db as any);
    const [a, b] = await Promise.all([s.set("nonce-d1-conc-1234567890", 600), s.set("nonce-d1-conc-1234567890", 600)]);
    expect([a, b].filter(Boolean).length).toBe(1);
    expect(await s.has("nonce-d1-conc-1234567890")).toBe(true);
  });

  it("expired nonce can be reused after TTL (fake timers + D1 expiry)", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    const db = createFakeD1ForNonces();
    const s = new D1NonceStore(db as any);
    expect(await s.set("nonce-d1-exp-1234567890", 10)).toBe(true);
    expect(await s.has("nonce-d1-exp-1234567890")).toBe(true);
    vi.advanceTimersByTime(11_000);
    // has() should lazily prune expired and return false
    expect(await s.has("nonce-d1-exp-1234567890")).toBe(false);
    // reuse allowed after expiry (prune + insert)
    expect(await s.set("nonce-d1-exp-1234567890", 10)).toBe(true);
    expect(await s.has("nonce-d1-exp-1234567890")).toBe(true);
    vi.useRealTimers();
  });

  it("has() prunes expired without explicit set", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    const db = createFakeD1ForNonces();
    const s = new D1NonceStore(db as any);
    await s.set("nonce-d1-prune-1234567890", 5);
    vi.advanceTimersByTime(6_000);
    expect(await s.has("nonce-d1-prune-1234567890")).toBe(false);
    // map should be empty after prune
    expect((db as any)._nonces.size).toBe(0);
    vi.useRealTimers();
  });
});

describe("D1ClientStore / D1InstanceStore / D1SessionStore / KV stores coverage", () => {
  it("D1ClientStore set/get/delete", async () => {
    const db = createFakeD1Full();
    const s = new D1ClientStore(db as any);
    await s.set("cid-d1", { clientId: "cid-d1", publicKey: "pub-d1", createdAt: Date.now() });
    expect((await s.get("cid-d1"))?.publicKey).toBe("pub-d1");
    await s.delete("cid-d1");
    expect(await s.get("cid-d1")).toBeNull();
  });

  it("D1InstanceStore set/get expiry and delete", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    const db = createFakeD1Full();
    const s = new D1InstanceStore(db as any);
    await s.set("inst-d1", { instanceId: "inst-d1", redirectUrl: "https://x/cb", createdAt: Date.now(), clientId: "cid" }, 10);
    expect((await s.get("inst-d1"))?.clientId).toBe("cid");
    vi.advanceTimersByTime(11_000);
    expect(await s.get("inst-d1")).toBeNull();
    await s.set("inst-d1", { instanceId: "inst-d1", redirectUrl: "https://x/cb", createdAt: Date.now(), clientId: "cid2" }, 10);
    expect((await s.get("inst-d1"))?.clientId).toBe("cid2");
    await s.delete("inst-d1");
    expect(await s.get("inst-d1")).toBeNull();
    vi.useRealTimers();
  });

  it("D1SessionStore set/get/consume atomic and expiry", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    const db = createFakeD1Full();
    const s = new D1SessionStore(db as any);
    await s.set("sess-d1", { provider: "dropbox", sratCallbackUrl: "https://x/cb", createdAt: Date.now(), tokenJson: `{"a":1}` } as any, 10);
    expect((await s.get("sess-d1"))?.provider).toBe("dropbox");
    const consumed = await s.consume("sess-d1");
    expect(consumed?.provider).toBe("dropbox");
    expect(await s.get("sess-d1")).toBeNull();
    expect(await s.consume("sess-d1")).toBeNull();
    // expiry path
    await s.set("sess-exp", { provider: "dropbox", sratCallbackUrl: "https://x/cb", createdAt: Date.now() } as any, 5);
    vi.advanceTimersByTime(6_000);
    expect(await s.get("sess-exp")).toBeNull();
    expect(await s.consume("sess-exp")).toBeNull();
    vi.useRealTimers();
  });

  it("KVInstanceStore and KVSessionStore roundtrip", async () => {
    const map = new Map<string, string>();
    const kv = { get: async (k: string) => map.get(k) || null, put: async (k: string, v: string) => { map.set(k, v); }, delete: async (k: string) => { map.delete(k); } };
    const inst = new KVInstanceStore(kv);
    await inst.set("inst-kv", { instanceId: "inst-kv", redirectUrl: "https://x/cb", createdAt: Date.now(), clientId: "cid" }, 600);
    expect((await inst.get("inst-kv"))?.clientId).toBe("cid");
    await inst.delete("inst-kv");
    expect(await inst.get("inst-kv")).toBeNull();

    const sess = new KVSessionStore(kv);
    await sess.set("sess-kv", { provider: "dropbox", sratCallbackUrl: "https://x/cb", createdAt: Date.now() } as any, 600);
    expect((await sess.get("sess-kv"))?.provider).toBe("dropbox");
    const c = await sess.consume("sess-kv");
    expect(c?.provider).toBe("dropbox");
    expect(await sess.get("sess-kv")).toBeNull();
  });

  it("MemorySessionStore consume and expiry", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    const s = new MemorySessionStore();
    await s.set("mem-sess", { provider: "dropbox", sratCallbackUrl: "https://x/cb", createdAt: Date.now() } as any, 5);
    expect((await s.get("mem-sess"))?.provider).toBe("dropbox");
    expect((await s.consume("mem-sess"))?.provider).toBe("dropbox");
    expect(await s.consume("mem-sess")).toBeNull();
    await s.set("mem-exp", { provider: "dropbox", sratCallbackUrl: "https://x/cb", createdAt: Date.now() } as any, 5);
    vi.advanceTimersByTime(6_000);
    expect(await s.get("mem-exp")).toBeNull();
    expect(await s.consume("mem-exp")).toBeNull();
    vi.useRealTimers();
  });
});

describe("session stores JSON parse error branches", () => {
  it("KVSessionStore get/consume returns null on corrupt JSON", async () => {
    const map = new Map<string, string>();
    const kv = { get: async (k: string) => map.get(k) || null, put: async (k: string, v: string) => { map.set(k, v); }, delete: async (k: string) => { map.delete(k); } };
    const s = new KVSessionStore(kv);
    map.set("bad-json", "{not:");
    expect(await s.get("bad-json")).toBeNull();
    map.set("bad-json2", "<<<corrupt>>>");
    expect(await s.consume("bad-json2")).toBeNull();
  });

  it("KVInstanceStore get returns null on corrupt JSON", async () => {
    const map = new Map<string, string>();
    const kv = { get: async (k: string) => map.get(k) || null, put: async (k: string, v: string) => { map.set(k, v); }, delete: async (k: string) => { map.delete(k); } };
    const s = new KVInstanceStore(kv);
    map.set("inst:bad", "not-json{");
    expect(await s.get("bad")).toBeNull();
  });

  it("KVClientStore get returns null on corrupt JSON", async () => {
    const map = new Map<string, string>();
    const kv = { get: async (k: string) => map.get(k) || null, put: async (k: string, v: string) => { map.set(k, v); }, delete: async (k: string) => { map.delete(k); } };
    const s = new KVClientStore(kv);
    map.set("client:bad", "{bad json");
    expect(await s.get("bad")).toBeNull();
  });

  it("D1ClientStore get returns null on corrupt JSON", async () => {
    const db = createFakeD1Full();
    const s = new D1ClientStore(db as any);
    // directly corrupt underlying map
    (db as any)._clients.set("bad-cid", "not-json{{{");
    expect(await s.get("bad-cid")).toBeNull();
  });

  it("D1InstanceStore get returns null on corrupt JSON", async () => {
    const db = createFakeD1Full();
    const s = new D1InstanceStore(db as any);
    (db as any)._instances.set("bad-inst", { data: "<<corrupt>>", expires_at: Math.floor(Date.now() / 1000) + 600 });
    expect(await s.get("bad-inst")).toBeNull();
  });

  it("D1SessionStore get and consume return null on corrupt JSON", async () => {
    const db = createFakeD1Full();
    const s = new D1SessionStore(db as any);
    const future = Math.floor(Date.now() / 1000) + 600;
    (db as any)._sessions.set("bad-sess", { data: "{not-json", expires_at: future });
    expect(await s.get("bad-sess")).toBeNull();
    // corrupt consume: put again with bad data, consume should catch JSON.parse
    (db as any)._sessions.set("bad-sess2", { data: "[[[", expires_at: future });
    expect(await s.consume("bad-sess2")).toBeNull();
    // ensure pruned after consume? still null on second consume
    expect(await s.consume("bad-sess2")).toBeNull();
  });

  it("D1SessionStore consume returns null when row.data missing (expired or not found)", async () => {
    const db = createFakeD1Full();
    const s = new D1SessionStore(db as any);
    // no row
    expect(await s.consume("nope")).toBeNull();
    // row with empty data
    const future = Math.floor(Date.now() / 1000) + 600;
    (db as any)._sessions.set("empty-data", { data: "", expires_at: future });
    // empty string is falsy -> row?.data is missing -> null path
    expect(await s.consume("empty-data")).toBeNull();
  });
});
