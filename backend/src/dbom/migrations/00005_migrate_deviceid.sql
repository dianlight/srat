-- +goose up
--ALTER TABLE `mount_point_paths` DROP COLUMN IF EXISTS `device`;
CREATE INDEX IF NOT EXISTS `idx_mount_point_paths_device_id` ON `mount_point_paths`(`device_id`);


-- +goose down
DROP INDEX `idx_mount_point_paths_device_id` ON `mount_point_paths`;
ALTER TABLE `mount_point_paths` ADD COLUMN `device` VARCHAR(255) NULL;
