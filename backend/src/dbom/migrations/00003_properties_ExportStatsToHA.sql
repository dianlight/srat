-- +goose up
INSERT INTO properties (key, value,created_at,updated_at) VALUES ('ExportStatsToHA', 'false', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

-- +goose down
DELETE FROM properties WHERE key='ExportStatsToHA';
