-- +goose up
INSERT OR REPLACE INTO properties (key, value,created_at,updated_at) VALUES ('HAUseNFS', 'false', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
-- +goose down
DELETE FROM properties WHERE key='HAUseNFS';