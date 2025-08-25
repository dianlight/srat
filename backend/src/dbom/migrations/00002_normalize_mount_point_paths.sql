-- +goose up
DELETE from mount_point_paths where path in ('/lib/modules', '/etc/hosts');