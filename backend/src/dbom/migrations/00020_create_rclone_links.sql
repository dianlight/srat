-- +goose up
CREATE TABLE IF NOT EXISTS rclone_links (
    target_kind TEXT NOT NULL,
    target_id TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT '',
    remote_path TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'unlinked',
    oauth_state TEXT NOT NULL DEFAULT '',
    last_sync_at DATETIME,
    last_sync_result TEXT NOT NULL DEFAULT '',
    last_sync_message TEXT NOT NULL DEFAULT '',
    auto_sync NUMERIC NOT NULL DEFAULT false,
    schedule_minutes INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    PRIMARY KEY (target_kind, target_id)
);
CREATE INDEX IF NOT EXISTS idx_rclone_links_deleted_at ON rclone_links (deleted_at);
-- +goose down
DROP INDEX IF EXISTS idx_rclone_links_deleted_at;
DROP TABLE IF EXISTS rclone_links;
