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
3. Enter the **Subfolder**, for example `alice` or `photos/2024`. Leave it
   empty to share the whole volume.
4. Assign the users who may read or write the share.

A missing folder is created with the mode and owner of the volume root, so it
follows the same permissions as the rest of the volume.

## Rules

- The subfolder is relative to the volume: `/` separates folders; absolute
  paths, `.` and `..` segments, empty segments, backslashes and control
  characters are refused.
- A subfolder can never point outside its volume, also through a symlink.
  Symlinks that stay inside the volume are followed.
- Two shares cannot expose the same folder. The volume root counts as a folder
  too: once a share exposes the whole volume, other shares on it need a
  subfolder.
- A folder is only created on a mounted volume, never in the empty mount point
  directory.
- When the folder of a share disappears, the share is reported invalid and left
  out of `smb.conf`. It never falls back to exposing the whole volume.
- Internal (system) shares cannot use a subfolder.

## Lab Mode and Existing Shares

Setting or changing a subfolder requires Lab Mode. A share that already uses a
subfolder keeps it, and stays editable, when Lab Mode is turned off: removing
the subfolder would widen the share to the whole volume.

## Samba, NFS and Home Assistant

The subfolder is applied wherever the share is published:

- `smb.conf`: `path = <mount point>/<subfolder>`.
- NFS exports: only the folder is exported.
- Home Assistant network mounts over NFS (`ha_use_nfs`) mount the folder.

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

Errors:

- `403` — a new or changed subfolder while Lab Mode is off.
- `409` — another share already exposes the folder.
- `422` — the subfolder is invalid, points outside the volume, or cannot be
  created; the message explains why.
