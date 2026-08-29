import type { SessionRecord } from "./types.js";

export interface SessionStore {
  get(id: string): Promise<SessionRecord | null>;
  set(id: string, data: SessionRecord, ttlSeconds: number): Promise<void>;
  delete(id: string): Promise<void>;
  /** Atomically get and delete — returns the record if existed, null otherwise. Only one concurrent caller should receive the completed session. */
  consume(id: string): Promise<SessionRecord | null>;
}

/** In-memory store for Node/Render single-instance and tests. Supports TTL expiry. */
export class MemorySessionStore implements SessionStore {
  private map = new Map<string, { data: SessionRecord; expiresAt: number }>();

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
    // isolates, KV conditional writes would be needed (not yet available in KV API).
    await this.kv.delete(id);
    return parsed;
  }
}
