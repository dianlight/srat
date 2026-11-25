# Design

## Context

Share creation currently resolves the Samba share path directly from the mounted volume root. The share model carries only the mount binding; the Samba template renders the share path from the mount data. Mount points have a single `Share` (GORM has-one), and the UI only offers volumes without a share.

## Goals / Non-Goals

**Goals:**

- Allow a share to point at a subdirectory of its mount.
- Allow several shares on one volume, one per folder, so each can have its own users.
- Keep the empty-subfolder case backward compatible (mount root).
- Prevent path traversal, including through symlinks, in user-supplied subfolder values.
- Never widen a share to the whole volume by accident.

**Non-Goals:**

- Preventing overlapping shares on purpose: a share on a folder and another on its parent are both allowed (as Samba allows); only the same folder twice is refused.
- Permission/ACL mapping strategies beyond copying the volume root's mode and owner on creation.
- Turning `MountPointData.Share` into a list: the volume keeps reporting one share.

## Decisions

- Store the subfolder as a relative path; empty means mount root (backward compatible). In the DTO it is a pointer: an update that omits it keeps the stored value, so a partial update can never widen a share to the whole volume.
- Validate the subfolder syntactically (`dto.CleanShareSubfolder`): reject absolute paths, `.`/`..` segments, empty segments, backslashes, control characters and names starting or ending with whitespace. Names are never trimmed, so a value always names the same folder.
- Resolve it on the filesystem segment by segment (`resolveShareDirectory`) through an `os.Root` opened on the volume: symlinks are followed only while they stay inside the volume, and lookups and creations happen relative to the open directory, so a symlink swapped in between the check and the creation cannot redirect it outside the volume.
- Create missing folders only when the volume is actually mounted (`/proc/self/mountinfo`), copying the volume root's mode and owner.
- `(mount path, subfolder)` is unique among active shares, disabled ones included, and two subfolders resolving to the same directory (an in-volume symlink alias) count as the same folder; conflicts return `ErrorShareFolderInUse` (409). Internal and standard-named shares cannot use a subfolder, and internal shares are not checked against other shares, so system provisioning is never blocked. Updates only re-check a changed folder, so editing other settings works while the volume is unmounted. On create an absent subfolder means the volume root, which also re-checks the folder of a restored soft-deleted share.
- `VerifyShare` marks a share whose folder is missing or escapes the volume invalid, which drops it from `smb.conf` instead of exposing the volume root.
- Mount events update every share on the volume. The volume reports the root share when there is one (deterministic preload order and disk-cache rule).
- Render the resolved path in `smb.conf`, NFS exports, HA NFS mounts and media links; the config converter keeps `Path` = mount point and `Subfolder` separate so legacy config import stays symmetric. NFS export paths are octal-escaped as `exports(5)` requires, subfolder exports use `subtree_check`, and an active HA NFS mount whose path no longer matches the share's folder is updated.
- The UI picks the subfolder with a folder browser backed by `GET /api/shares/folders`: it lists one folder level of a known, mounted volume through the same resolver, skips hidden folders, symlinks and the root `lost+found`, and caps a listing at 1000 entries.
- Lab gating lives in the API handler (like HDIdle): a new or changed subfolder needs Lab Mode (403 otherwise); keeping an existing one never does.

## Risks / Trade-offs

- Migration risk is low because the new column defaults to empty (mount root).
- Filesystem creation failures surface as validation errors before the share is stored.
- The goose migration tolerates the column already existing (created by GORM AutoMigrate).
- Existing databases that already hold two shares on one volume root keep working; only new or changed folders are checked.
