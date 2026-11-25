# Share Subfolders

## Overview

A share can expose a folder inside its volume instead of the whole volume
([#184](https://github.com/dianlight/srat/issues/184)). Several shares can then
live on one volume, each with its own users, for example one private folder per
person on a single data disk:

| Share  | Volume       | Subfolder | Users     |
| ------ | ------------ | --------- | --------- |
| FAMILY | `/mnt/DATA1` | `Family`  | all users |
| ALICE  | `/mnt/DATA1` | `alice`   | `alice`   |
| BOB    | `/mnt/DATA1` | `bob`     | `bob`     |

Share subfolders are a **beta Lab feature** (`share_subfolders`): they are
available when `experimental_lab_mode` is enabled.

## Using Subfolders

1. Enable Lab Mode in **Settings**.
2. Create a share and pick a volume. With subfolders available, volumes that
   already carry a share can be selected too.
3. Pick the **Subfolder**: the folder button opens a browser of the volume's
   folders, where you click through folders, filter them by name, or name a
   new folder. You can also type the path, for example `alice` or
   `photos/2024`. Leave it empty to share the whole volume.
4. Assign the users who may read or write the share.

A missing folder is created with the mode and owner of the volume root, so it
follows the same permissions as the rest of the volume.

## Rules

- The subfolder is relative to the volume: `/` separates folders; absolute
  paths, `.` and `..` segments, empty segments, backslashes, control
  characters and folder names that start or end with a space are refused.
  Names are never trimmed or otherwise changed, and each name can have 255
  bytes at most.
- A subfolder can never point outside its volume, also through a symlink.
  Symlinks that stay inside the volume are followed. Folders are looked up and
  created relative to the open volume, so a symlink swapped in meanwhile
  cannot redirect them.
- Two shares cannot expose the same folder, also under two names through a
  symlink. The volume root counts as a folder too: once a share exposes the
  whole volume, other shares on it need a subfolder. A disabled share keeps its
  folder, so enabling it again never creates a duplicate.
- A folder is only created on a mounted volume, never in the empty mount point
  directory.
- When the folder of a share disappears, the share is reported invalid and left
  out of `smb.conf`. It never falls back to exposing the whole volume.
- Internal (system) shares and shares with a standard share name (such as
  `addons`) cannot use a subfolder.

## Lab Mode and Existing Shares

Setting or changing a subfolder requires Lab Mode. A share that already uses a
subfolder keeps it, and stays editable, when Lab Mode is turned off: removing
the subfolder would widen the share to the whole volume.

## Samba, NFS and Home Assistant

The subfolder is applied wherever the share is published:

- `smb.conf`: `path = <mount point>/<subfolder>`.
- NFS exports: only the folder is exported, with `subtree_check` so file
  handles cannot reach the rest of the volume. Spaces and other special
  characters in the path are escaped as `exports(5)` requires.
- Home Assistant network mounts over NFS (`ha_use_nfs`) mount the folder; when
  the subfolder changes, the mount is updated to the new folder.
- Media links for media shares point at the folder.

On a volume with several shares, the volume reports the share of the volume
root when there is one.

## API

`SharedResource` has an optional `subfolder` string:

- On create, empty or absent means the volume root.
- On update, an absent `subfolder` keeps the stored value, and `""` moves the
  share to the volume root. Partial updates therefore never widen a share.

```json
{
  "name": "ALICE",
  "subfolder": "alice",
  "mount_point_data": { "path": "/mnt/DATA1" }
}
```

`GET /api/shares/folders?mount_path=/mnt/DATA1&path=photos` lists the folders
inside a folder of a mounted volume (`path` empty is the volume root), for the
folder browser. Hidden folders, the volume root's `lost+found` and symlinks are
left out, at most 1000 folders are returned (`truncated` tells when there were
more), and the same containment rules apply.

Errors:

- `403` — a new or changed subfolder, or a folder listing, while Lab Mode is
  off.
- `404` — the folder listing names a volume that is not mounted.
- `409` — another share already exposes the folder.
- `422` — the subfolder is invalid, points outside the volume, or cannot be
  created; the message explains why.
