-- D1 migration: instances table for HA instance registration (TTL 1h)
CREATE TABLE IF NOT EXISTS instances (
  id TEXT PRIMARY KEY,
  data TEXT NOT NULL,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_instances_expires_at ON instances(expires_at);
