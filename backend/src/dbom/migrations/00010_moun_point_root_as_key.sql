-- +goose up
UPDATE mount_point_paths SET root = path WHERE fs_type = 'native';

-- +goose down
UPDATE mount_point_paths SET root = '/' WHERE fs_type = 'native';