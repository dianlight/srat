-- +goose up
DELETE FROM issues;
CREATE UNIQUE INDEX IF NOT EXISTS `idx_issues_title` ON `issues` (`title`);
