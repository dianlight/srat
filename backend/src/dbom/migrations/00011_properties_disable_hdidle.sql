-- +goose up
INSERT OR REPLACE INTO properties (key, value,created_at,updated_at) VALUES ('HDIdleEnabled', 'false', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
-- +goose down
DELETE FROM properties WHERE key='HDIdleEnabled';
