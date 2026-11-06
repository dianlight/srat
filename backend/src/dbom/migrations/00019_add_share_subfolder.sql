-- +goose up
ALTER TABLE exported_shares ADD COLUMN subfolder TEXT DEFAULT '';

-- +goose down
ALTER TABLE exported_shares DROP COLUMN subfolder;
