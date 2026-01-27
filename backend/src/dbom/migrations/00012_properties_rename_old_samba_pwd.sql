-- +goose up
UPDATE properties set key="HASmbPassword" where key="_ha_mount_user_password_"; 
-- +goose down
UPDATE properties set key="_ha_mount_user_password_" where key="HASmbPassword";