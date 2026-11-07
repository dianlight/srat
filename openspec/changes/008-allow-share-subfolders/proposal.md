# Proposal

## Why

Samba shares currently map directly to the mount-point root (`/mnt/<device>`). Users who organize data into subfolders (for example a `movies` or `photos/2024` directory) cannot expose just that subdirectory through a share without re-arranging their mounts.

## What Changes

- Add an optional `subfolder` field to share create/update requests.
- Resolve the Samba share path as `<mount_root>/<subfolder>` when a subfolder is set; default to the mount root.
- Create the subfolder on the filesystem when it does not exist.
- Reject absolute paths and parent-traversal (`..`) segments in the subfolder.
- Return the resolved share path in listings for UI display.

## Capabilities

### New Capabilities

- `shares`: Samba share behavior with optional mount-relative subfolder configuration.

### Modified Capabilities

(none)

## Impact

- `backend/src/dto/` — add `Subfolder` to share DTOs.
- `backend/src/dbom/` — add column + migration.
- `backend/src/service/share_service.go` — path resolution and directory creation.
- `backend/src/templates/smb.gtpl` — emit resolved path.
- `frontend/src/pages/shares/` — share form input.
- API surface change: OpenAPI regeneration required.
- Tracker: [srat#184](https://github.com/dianlight/srat/issues/184)
