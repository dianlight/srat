-- D1 migration: clients (persistent) and nonces (replay protection, TTL 10m)
CREATE TABLE IF NOT EXISTS clients (
  id TEXT PRIMARY KEY,
  data TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS nonces (
  id TEXT PRIMARY KEY,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_nonces_expires_at ON nonces(expires_at);
