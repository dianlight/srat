# Proposal

## Why

Samba shares currently map directly to the mount-point root (`/mnt/<device>`). Users who organize data into subfolders (for example a `movies` or `photos/2024` directory) cannot expose just that subdirectory through a share without re-arranging their mounts.

The most common case is a home NAS with one large data disk shared by a family: every person needs a private folder that the others cannot open. Samba enforces `valid users` per share, so per-person privacy needs one share per folder, all on the same volume. Re-partitioning the disk per person is the only workaround today, and it wastes space and is risky on a disk that already holds data. This is also what users ask for in [hassio-addons#768](https://github.com/dianlight/hassio-addons/issues/768) ("mounting specific subdirectories and granular user permissions").

## What Changes

- Add an optional `subfolder` field to share create/update requests.
- Resolve the Samba share path as `<mount_root>/<subfolder>` when a subfolder is set; default to the mount root.
- Create the subfolder on the filesystem when it does not exist, with the mode and owner of the volume root, and only on a mounted volume.
- Reject absolute paths, parent-traversal (`..`), empty segments, backslashes and control characters in the subfolder, and any subfolder that resolves outside the volume through a symlink.
- Allow several shares on one volume, one per folder; reject two active shares exposing the same folder (the volume root counts as a folder).
- Gate setting or changing a subfolder behind the `share_subfolders` beta lab feature (Lab Mode). Existing subfolder shares keep their subfolder when Lab Mode is turned off.
- Apply the subfolder to NFS exports and to Home Assistant NFS mounts.

## Capabilities

### New Capabilities

- `shares`: Samba share behavior with optional mount-relative subfolder configuration.

### Modified Capabilities

(none)

## Impact

- `backend/src/dto/` — `Subfolder` on `SharedResource` (pointer: absent keeps the stored value), validation helpers, `ErrorShareFolderInUse`.
- `backend/src/dbom/` — `subfolder` column + goose migration 00020.
- `backend/src/service/share_service.go`, `share_directory.go` — validation, folder resolution and creation, all shares of a volume follow its mount events.
- `backend/src/service/server_process_service.go`, `supervisor_service.go` — resolved path in `smb.conf`, NFS exports and HA NFS mounts.
- `backend/src/api/shares.go` — Lab gating (403), folder conflicts (409), readable validation errors (422).
- `backend/src/service/lab_features.go` — `share_subfolders` beta feature.
- `frontend/src/pages/shares/` — lab-gated Subfolder input, shared volumes selectable with subfolders.
- API surface change: OpenAPI and frontend types gain `subfolder`.
- Tracker: [srat#184](https://github.com/dianlight/srat/issues/184)
