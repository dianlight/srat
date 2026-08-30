import type { SessionRecord } from "./types.js";

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
