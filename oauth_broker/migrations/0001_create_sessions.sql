-- D1 migration: sessions table for OAuth broker (atomic single-use via DELETE ... RETURNING)
-- Apply with: wrangler d1 create srat-oauth-broker --env staging
--             wrangler d1 migrations apply srat-oauth-broker --env staging
CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  data TEXT NOT NULL,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
