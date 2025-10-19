-- +goose up
INSERT INTO properties (key, value,created_at,updated_at) VALUES ('HDIdleDefaultIdleTime', '60', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO properties (key, value,created_at,updated_at) VALUES ('HDIdleDefaultCommandType', 'scsi', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO properties (key, value,created_at,updated_at) VALUES ('HDIdleDefaultPowerCondition', '0', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO properties (key, value,created_at,updated_at) VALUES ('HDIdleIgnoreSpinDownDetection', 'false', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO properties (key, value,created_at,updated_at) VALUES ('HDIdleEnabled', 'false', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
-- +goose down
DELETE FROM properties WHERE key='HDIdleDefaultIdleTime';
DELETE FROM properties WHERE key='HDIdleDefaultCommandType';
DELETE FROM properties WHERE key='HDIdleDefaultPowerCondition';
DELETE FROM properties WHERE key='HDIdleIgnoreSpinDownDetection';
DELETE FROM properties WHERE key='HDIdleEnabled';
