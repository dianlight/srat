# Remote Environment Test: All Supported Filesystems on Share Use

Date: 2026-11-01
Host: HAOS 2025.8.0, SRAT 2026.9.1-iss1246, amd64
Addon slug: local_sambanas2

## Environment

- Admin Samba password: overnice (from samba_users table)
- Test users created: fsuser1, fsuser2, fsuser3 (password: changeme)
  - fsuser1, fsuser2: read/write on test shares
  - fsuser3: read-only on test shares
- leave_front_door_open: true (API unauthenticated)

## Supported filesystem adapters (backend registry)

| FS | canMount | canFormat | Test partition | Result |
|----|----------|-----------|----------------|--------|
| ext4 | yes | yes | sdb1 (200MB) | PASS |
| vfat | yes | yes | Kingston part1 (32MB) | PASS |
| ntfs | yes | yes | sdc1 (61GB) | PASS |
| exfat | yes | yes | sdb2 (2TB) | PASS (device id changed to by-uuid after format) |
| btrfs | yes | yes | sdd1 (1.1GB) | PASS |
| xfs | yes | yes | sdd2 (2GB) | FAIL - mkfs.xfs 7.x cannot accept device with -f; adapter reports false success |
| f2fs | yes | yes | sdb1 (200MB, reused) | PASS |
| apfs | mount only | no | none available | SKIP - no apfs volume |
| gfs2 | no | no | n/a | SKIP - not supported on system |
| hfsplus | no | no | n/a | SKIP - not supported on system |
| reiserfs | no | no | n/a | SKIP - not supported on system |
| zfs | no | no | n/a | SKIP - not supported on system |

## Per-filesystem test sequence

For each testable FS type:

1. Find or format a spare partition to the target FS type (force format).
2. Wait for format completion (~20s for large partitions).
3. Mount via POST /api/volume/mount with default flags (empty flags/custom_flags).
4. Create share via POST /api/share with users admin/fsuser1/fsuser2 (rw) and fsuser3 (ro), guest_ok=false.
5. Wait ~10s for Samba config regeneration.
6. Verify share appears in `smbclient -L`.
7. smbclient checks:
   - guest mount -> Permission denied (expected)
   - admin: mkdir admin-<sid>, rw confirmed
   - fsuser1: mkdir user1-<sid>, rw in /admin-<sid>
   - fsuser2: rw in /admin-<sid> and /user1-<sid>
   - fsuser3: read ls ok, mkdir -> NT_STATUS_ACCESS_DENIED
8. Cleanup: remove test dirs, delete share, unmount volume.

## Results summary

- ext4: PASS
- vfat: PASS (required 10s sync wait; 3s was insufficient)
- ntfs: PASS
- exfat: PASS (device id changed from by-part to by-uuid after format; mount by new id)
- btrfs: PASS
- xfs: FAIL - mkfs.xfs 7.0.1 in addon container does not accept `-f <device>` positional; adapter reports success but filesystem unchanged (ext4). Potential bug: false success.
- f2fs: PASS
- apfs: SKIP - no apfs volume available
- gfs2/hfsplus/reiserfs/zfs: SKIP - not supported on this system (canMount=false)

## Issues found

1. XFS adapter reports successful format when mkfs.xfs fails (false success). The mkfs.xfs 7.x binary in the addon container requires a different invocation to force-overwrite an existing filesystem. Adapter should detect non-zero exit / usage output.
2. exFAT partition device id changes from `usb-...-part2` to `by-uuid-...` after format; mount by stale id fails with 404 Device Not Found. SRAT should resolve by partition or refresh enumeration.
3. Samba config regeneration needs ~10s after share creation; shorter waits (3s) caused share not to appear in smbclient list.
4. Cleanup unmount via DELETE /api/volume often returns 406 Not Acceptable; shares/volumes sometimes require manual cleanup.

## Notes

- Test users fsuser1/fsuser2/fsuser3 left in database (password: changeme) for future runs.
- Test partitions reformatted; original data on those partitions was destroyed (test volumes only).
