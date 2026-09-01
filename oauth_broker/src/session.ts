import type { ClientRecord, InstanceRecord, SessionRecord } from "./types.js";

export interface SessionStore {
  get(id: string): Promise<SessionRecord | null>;
  set(id: string, data: SessionRecord, ttlSeconds: number): Promise<void>;
  delete(id: string): Promise<void>;
  /**
   * Best-effort single-use get-and-delete — returns the record if existed, null otherwise.
   * Atomic in MemorySessionStore (single isolate); best-effort in KVSessionStore
   * (eventually consistent across Workers isolates — two concurrent GET /v1/session/:id
   * may both read before delete propagates; KV has no conditional writes). Only one
   * caller SHOULD receive the token; callers must tolerate at-most-once delivery.
   * For strong single-use use D1SessionStore (DELETE ... RETURNING is atomic).
   */
  consume(id: string): Promise<SessionRecord | null>;
}

/** In-memory store for Node/Render single-instance and tests. Supports TTL expiry. */
export class MemorySessionStore implements SessionStore {
  private map = new Map<string, { data: SessionRecord; expiresAt: number }>();
  /** Max entries before evicting expired and rejecting new inserts (DoS cap). */
  static readonly MAX_ENTRIES = 10_000;

  private evictExpired(): void {
    const now = Date.now();
    for (const [k, v] of this.map) {
      if (now > v.expiresAt) this.map.delete(k);
    }
  }

  async get(id: string): Promise<SessionRecord | null> {
    const entry = this.map.get(id);
    if (!entry) return null;
    if (Date.now() > entry.expiresAt) {
      this.map.delete(id);
      return null;
    }
    return entry.data;
  }

  async set(id: string, data: SessionRecord, ttlSeconds: number): Promise<void> {
    if (this.map.size >= MemorySessionStore.MAX_ENTRIES) {
      this.evictExpired();
      if (this.map.size >= MemorySessionStore.MAX_ENTRIES) {
        throw new Error("session store full – too many active sessions, retry after expiry");
      }
    }
    const expiresAt = Date.now() + ttlSeconds * 1000;
    this.map.set(id, { data, expiresAt });
  }

  async delete(id: string): Promise<void> {
    this.map.delete(id);
  }

  async consume(id: string): Promise<SessionRecord | null> {
    const entry = this.map.get(id);
    if (!entry) return null;
    if (Date.now() > entry.expiresAt) {
      this.map.delete(id);
      return null;
    }
    this.map.delete(id);
    return entry.data;
  }

  /** For tests: expose size */
  size(): number {
    return this.map.size;
  }

  clear(): void {
    this.map.clear();
  }
}

/** KV-backed store for Cloudflare Workers. Expects a Workers KVNamespace binding. */
export type KVNamespaceLike = {
  get(key: string, opts?: { type: string }): Promise<string | null>;
  put(key: string, value: string, opts?: { expirationTtl?: number }): Promise<void>;
  delete(key: string): Promise<void>;
};

export class KVSessionStore implements SessionStore {
  constructor(private kv: KVNamespaceLike) {}

  async get(id: string): Promise<SessionRecord | null> {
    const raw = await this.kv.get(id, { type: "text" });
    if (!raw) return null;
    try {
      return JSON.parse(raw) as SessionRecord;
    } catch {
      return null;
    }
  }

  async set(id: string, data: SessionRecord, ttlSeconds: number): Promise<void> {
    await this.kv.put(id, JSON.stringify(data), { expirationTtl: ttlSeconds });
  }

  async delete(id: string): Promise<void> {
    await this.kv.delete(id);
  }

  async consume(id: string): Promise<SessionRecord | null> {
    const raw = await this.kv.get(id, { type: "text" });
    if (!raw) return null;
    let parsed: SessionRecord;
    try {
      parsed = JSON.parse(raw) as SessionRecord;
    } catch {
      return null;
    }
    // Best-effort atomic: delete immediately after read; KV is eventually consistent
    // but this ensures single-use within a single isolate. For true atomicity across
    // isolates, use D1SessionStore (D1 DELETE ... RETURNING is atomic).
    await this.kv.delete(id);
    return parsed;
  }
}

/** D1-backed store for Cloudflare Workers — atomic single-use via DELETE ... RETURNING. */
export type D1DatabaseLike = {
  prepare(query: string): {
    bind(...values: unknown[]): {
      first<T = unknown>(col?: string): Promise<T | null>;
      run(): Promise<{ success: boolean; meta?: unknown; results?: unknown }>;
      all<T = unknown>(): Promise<{ results: T[] }>;
    };
  };
  batch?(statements: unknown[]): Promise<unknown[]>;
};

export interface InstanceStore {
  get(id: string): Promise<InstanceRecord | null>;
  set(id: string, data: InstanceRecord, ttlSeconds: number): Promise<void>;
  delete(id: string): Promise<void>;
  deleteByClientId?(clientId: string): Promise<number>;
}

export class MemoryInstanceStore implements InstanceStore {
  private map = new Map<string, { data: InstanceRecord; expiresAt: number }>();
  static readonly MAX_ENTRIES = 10_000;

  private evictExpired(): void {
    const now = Date.now();
    for (const [k, v] of this.map) {
      if (now > v.expiresAt) this.map.delete(k);
    }
  }

  async get(id: string): Promise<InstanceRecord | null> {
    const entry = this.map.get(id);
    if (!entry) return null;
    if (Date.now() > entry.expiresAt) {
      this.map.delete(id);
      return null;
    }
    return entry.data;
  }

  async set(id: string, data: InstanceRecord, ttlSeconds: number): Promise<void> {
    if (this.map.size >= MemoryInstanceStore.MAX_ENTRIES) {
      this.evictExpired();
      if (this.map.size >= MemoryInstanceStore.MAX_ENTRIES) {
        throw new Error("instance store full – too many active instances, retry after expiry");
      }
    }
    const expiresAt = Date.now() + ttlSeconds * 1000;
    this.map.set(id, { data, expiresAt });
  }

  async delete(id: string): Promise<void> {
    this.map.delete(id);
  }

  async deleteByClientId(clientId: string): Promise<number> {
    let deleted = 0;
    for (const [k, v] of this.map) {
      if (v.data.clientId === clientId) {
        this.map.delete(k);
        deleted++;
      }
    }
    return deleted;
  }

  size(): number {
    return this.map.size;
  }

  clear(): void {
    this.map.clear();
  }
}

export class KVInstanceStore implements InstanceStore {
  constructor(
    private kv: KVNamespaceLike,
    private prefix = "inst:",
  ) {}

  private key(id: string): string {
    return `${this.prefix}${id}`;
  }

  async get(id: string): Promise<InstanceRecord | null> {
    const raw = await this.kv.get(this.key(id), { type: "text" });
    if (!raw) return null;
    try {
      return JSON.parse(raw) as InstanceRecord;
    } catch {
      return null;
    }
  }

  async set(id: string, data: InstanceRecord, ttlSeconds: number): Promise<void> {
    await this.kv.put(this.key(id), JSON.stringify(data), { expirationTtl: ttlSeconds });
  }

  async delete(id: string): Promise<void> {
    await this.kv.delete(this.key(id));
  }

  async deleteByClientId(_clientId: string): Promise<number> {
    // KV has no list operation; best-effort: instances will expire via TTL.
    // In production D1 is preferred, so this is rarely used.
    return 0;
  }
}

export class D1InstanceStore implements InstanceStore {
  constructor(private db: D1DatabaseLike) {}

  private nowSec(): number {
    return Math.floor(Date.now() / 1000);
  }

  async get(id: string): Promise<InstanceRecord | null> {
    const row = await this.db
      .prepare("SELECT data, expires_at FROM instances WHERE id = ?")
      .bind(id)
      .first<{ data: string; expires_at: number }>();
    if (!row) return null;
    if (row.expires_at <= this.nowSec()) {
      await this.db.prepare("DELETE FROM instances WHERE id = ?").bind(id).run();
      return null;
    }
    try {
      return JSON.parse(row.data) as InstanceRecord;
    } catch {
      return null;
    }
  }

  async set(id: string, data: InstanceRecord, ttlSeconds: number): Promise<void> {
    const expiresAt = this.nowSec() + ttlSeconds;
    const json = JSON.stringify(data);
    await this.db.prepare("INSERT OR REPLACE INTO instances (id, data, expires_at) VALUES (?, ?, ?)").bind(id, json, expiresAt).run();
  }

  async delete(id: string): Promise<void> {
    await this.db.prepare("DELETE FROM instances WHERE id = ?").bind(id).run();
  }

  async deleteByClientId(clientId: string): Promise<number> {
    // D1 stores data as JSON, so filter by json_extract
    try {
      const res = await this.db.prepare("DELETE FROM instances WHERE json_extract(data, '$.clientId') = ?").bind(clientId).run() as unknown as { meta?: { changes?: number } };
      return res?.meta?.changes ?? 0;
    } catch {
      // Fallback: best-effort, instances will expire
      return 0;
    }
  }
}

export class D1SessionStore implements SessionStore {
  constructor(private db: D1DatabaseLike) {}

  private nowSec(): number {
    return Math.floor(Date.now() / 1000);
  }

  async get(id: string): Promise<SessionRecord | null> {
    const row = await this.db
      .prepare("SELECT data, expires_at FROM sessions WHERE id = ?")
      .bind(id)
      .first<{ data: string; expires_at: number }>();
    if (!row) return null;
    if (row.expires_at <= this.nowSec()) {
      // Lazily prune expired row (best-effort).
      await this.db.prepare("DELETE FROM sessions WHERE id = ?").bind(id).run();
      return null;
    }
    try {
      return JSON.parse(row.data) as SessionRecord;
    } catch {
      return null;
    }
  }

  async set(id: string, data: SessionRecord, ttlSeconds: number): Promise<void> {
    const expiresAt = this.nowSec() + ttlSeconds;
    const json = JSON.stringify(data);
    await this.db
      .prepare("INSERT OR REPLACE INTO sessions (id, data, expires_at) VALUES (?, ?, ?)")
      .bind(id, json, expiresAt)
      .run();
  }

  async delete(id: string): Promise<void> {
    await this.db.prepare("DELETE FROM sessions WHERE id = ?").bind(id).run();
  }

  async consume(id: string): Promise<SessionRecord | null> {
    // Atomic: only one isolate can DELETE ... RETURNING the row when expires_at > now.
    const now = this.nowSec();
    const row = await this.db
      .prepare("DELETE FROM sessions WHERE id = ? AND expires_at > ? RETURNING data as data")
      .bind(id, now)
      .first<{ data: string }>();
    if (!row?.data) return null;
    try {
      return JSON.parse(row.data) as SessionRecord;
    } catch {
      return null;
    }
  }
}

// ---- Client store (persistent, no TTL) ----
export interface ClientStore {
  get(id: string): Promise<ClientRecord | null>;
  set(id: string, data: ClientRecord): Promise<void>;
  delete(id: string): Promise<void>;
}

export class MemoryClientStore implements ClientStore {
  private map = new Map<string, ClientRecord>();
  async get(id: string): Promise<ClientRecord | null> {
    return this.map.get(id) ?? null;
  }
  async set(id: string, data: ClientRecord): Promise<void> {
    this.map.set(id, data);
  }
  async delete(id: string): Promise<void> {
    this.map.delete(id);
  }
  clear(): void {
    this.map.clear();
  }
  size(): number {
    return this.map.size;
  }
}

export class KVClientStore implements ClientStore {
  constructor(
    private kv: KVNamespaceLike,
    private prefix = "client:",
  ) {}
  private key(id: string): string {
    return `${this.prefix}${id}`;
  }
  async get(id: string): Promise<ClientRecord | null> {
    const raw = await this.kv.get(this.key(id), { type: "text" });
    if (!raw) return null;
    try {
      return JSON.parse(raw) as ClientRecord;
    } catch {
      return null;
    }
  }
  async set(id: string, data: ClientRecord): Promise<void> {
    await this.kv.put(this.key(id), JSON.stringify(data));
  }
  async delete(id: string): Promise<void> {
    await this.kv.delete(this.key(id));
  }
}

export class D1ClientStore implements ClientStore {
  constructor(private db: D1DatabaseLike) {}
  async get(id: string): Promise<ClientRecord | null> {
    const row = await this.db.prepare("SELECT data FROM clients WHERE id = ?").bind(id).first<{ data: string }>();
    if (!row) return null;
    try {
      return JSON.parse(row.data) as ClientRecord;
    } catch {
      return null;
    }
  }
  async set(id: string, data: ClientRecord): Promise<void> {
    await this.db.prepare("INSERT OR REPLACE INTO clients (id, data) VALUES (?, ?)").bind(id, JSON.stringify(data)).run();
  }
  async delete(id: string): Promise<void> {
    await this.db.prepare("DELETE FROM clients WHERE id = ?").bind(id).run();
  }
}

// ---- Nonce store (replay protection, TTL 10m) ----
export interface NonceStore {
  has(nonce: string): Promise<boolean>;
  set(nonce: string, ttlSeconds: number): Promise<boolean>; // true if inserted, false if already existed
}

export class MemoryNonceStore implements NonceStore {
  private map = new Map<string, number>();
  private evictExpired(except?: string): void {
    const now = Date.now();
    for (const [k, exp] of this.map) {
      if (except !== undefined && k === except) continue;
      if (now > exp) this.map.delete(k);
    }
  }
  async has(nonce: string): Promise<boolean> {
    const exp = this.map.get(nonce);
    if (exp === undefined) return false;
    if (Date.now() > exp) {
      this.map.delete(nonce);
      return false;
    }
    return true;
  }
  async set(nonce: string, ttlSeconds: number): Promise<boolean> {
    // Atomic in single isolate (JS single thread): check-then-set no interleaving
    // Do not await between check and set for this id.
    // Evict others but keep the current nonce so expired-reuse can be distinguished
    // from a fresh insert (covers the `map.delete(nonce)` branch).
    this.evictExpired(nonce);
    const existing = this.map.get(nonce);
    if (existing !== undefined) {
      if (Date.now() <= existing) return false;
      // expired – allow reuse (now reachable: evictExpired skipped this key)
      this.map.delete(nonce);
    }
    this.map.set(nonce, Date.now() + ttlSeconds * 1000);
    return true;
  }
  clear(): void {
    this.map.clear();
  }
}

export class KVNonceStore implements NonceStore {
  constructor(
    private kv: KVNamespaceLike,
    private prefix = "nonce:",
  ) {}
  private key(nonce: string): string {
    return `${this.prefix}${nonce}`;
  }
  async has(nonce: string): Promise<boolean> {
    const v = await this.kv.get(this.key(nonce), { type: "text" });
    return v !== null;
  }
  async set(nonce: string, ttlSeconds: number): Promise<boolean> {
    // Best-effort: KV has no CAS/conditional writes across isolates.
    // has-then-put is non-atomic; two concurrent requests with same nonce may both pass has()
    // before either put propagates. For strict single-use, use D1NonceStore (D1 PK is atomic).
    // We still do has() check to avoid overwriting TTL on replay; atomicity delegated to D1.
    if (await this.has(nonce)) return false;
    await this.kv.put(this.key(nonce), "1", { expirationTtl: ttlSeconds });
    // No read-after-write verification: insert is best-effort. Caller treats true as success.
    return true;
  }
}

export class D1NonceStore implements NonceStore {
  constructor(private db: D1DatabaseLike) {}
  private nowSec(): number {
    return Math.floor(Date.now() / 1000);
  }
  async has(nonce: string): Promise<boolean> {
    const row = await this.db.prepare("SELECT expires_at FROM nonces WHERE id = ?").bind(nonce).first<{ expires_at: number }>();
    if (!row) return false;
    if (row.expires_at <= this.nowSec()) {
      await this.db.prepare("DELETE FROM nonces WHERE id = ?").bind(nonce).run();
      return false;
    }
    return true;
  }
  async set(nonce: string, ttlSeconds: number): Promise<boolean> {
    const now = this.nowSec();
    const exp = now + ttlSeconds;
    // Allow reuse after TTL: prune expired row for same id before atomic insert
    try {
      await this.db.prepare("DELETE FROM nonces WHERE id = ? AND expires_at <= ?").bind(nonce, now).run();
    } catch {
      // best-effort prune
    }
    try {
      // Atomic insert via PK uniqueness: exactly one concurrent set succeeds
      await this.db.prepare("INSERT INTO nonces (id, expires_at) VALUES (?, ?)").bind(nonce, exp).run();
      return true;
    } catch {
      return false;
    }
  }
}
