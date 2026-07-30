<!-- DOCTOC SKIP -->

# Replicating a Partitionless Disk on macOS

Step-by-step guide to create USB drives and disk images that reproduce the
"disk without partitions" bug tracked in [dianlight/srat#849](https://github.com/dianlight/srat/issues/849)
and [dianlight/hassio-addons#716](https://github.com/dianlight/hassio-addons/issues/716).

Implementation work for the fix is tracked in [Task 044—Support Disks Without Partitions](tasks/044_support-disks-without-partitions.md).

## Background

Two distinct on-disk layouts trigger the bug:

| Scenario | Layout | Reproduces |
| -------- | ------ | ---------- |
| A | No partition table at all; a valid filesystem starts at sector 0 ("superfloppy") | srat#849 |
| B | Valid MBR with a partition entry, but the partition content has no readable filesystem signature | hassio-addons#716 |

In both cases the Linux kernel can see the device (and often the filesystem),
but UDisks2 does not expose a `.Filesystem` D-Bus interface for any block
device of the drive. The Home Assistant Supervisor then reports the drive with
an empty `filesystems` list, and SRAT drops it—the disk never
appears in the UI and no action is possible.

## Safety Warnings

- Every `dd` and `diskutil partitionDisk` command below **destroys all data** on the target disk.
- Triple-check the disk identifier with `diskutil list external physical` before each write. Writing to the wrong device can erase your system disk.
- Use a spare USB stick (any size ≥ 1 GB) dedicated to testing.
- When macOS shows "The disk you inserted was not readable by this computer" after a write, select **Ignore**—never **Initialize**.

## Scenario A—Drive With No Partition Table (srat#849)

A disk with no partition table whose first sector is a valid FAT32 filesystem.

### A.1 Create the raw filesystem image

```bash
hdiutil create -size 1g -fs "MS-DOS FAT32" -layout NONE -volname NO_PART rawfs.dmg
```

Verify the image has no partition scheme:

```bash
hdiutil imageinfo rawfs.dmg | grep -iE "partition"
# Expected: partition-scheme: none   /   Partitioned: false

file rawfs.dmg
# Expected: DOS/MBR boot sector ... FAT32 ... (no partition entries)
```

> **Note (developer sandbox):** on sandboxed shells the attach/newfs step of
> `hdiutil create -fs ...` can fail with `hdiutil: create failed - Operation not permitted`.
> Run the command in a normal (non-sandboxed) macOS terminal session.

### A.2 Write the image to the USB drive

```bash
# Identify the USB drive (example: /dev/disk4)—check size and name carefully
diskutil list external physical

# Unmount every volume on it
diskutil unmountDisk /dev/diskN

# Write the superfloppy image (use rdisk for raw, faster access)
sudo dd if=rawfs.dmg of=/dev/rdiskN bs=1m

# Eject
diskutil eject /dev/diskN
```

Verify there are no partitions:

```bash
diskutil list external physical
# /dev/diskN shows a single entry with no FDisk_partition_scheme / GUID_partition_scheme children
```

## Scenario B—MBR Drive With Unreadable Filesystem (hassio-addons#716)

A disk that has a valid MBR partition entry (like the reporter's TOSHIBA drive:
`55AA` signature at `0x1FE`, partition entry at `0x1BE` with type `0x07`) but
whose partition content cannot be probed by UDisks2, so the Supervisor reports
`filesystems: []`.

### B.1 Partition the USB drive as MBR

```bash
diskutil unmountDisk /dev/diskN
diskutil partitionDisk /dev/diskN 1 MBR "MS-DOS FAT32" DATA 100%
```

### B.2 Destroy the filesystem signature (keep the MBR)

```bash
diskutil unmountDisk /dev/diskN

# Zero the first MiB of the partition—wipes the FAT boot sector and backup,
# leaving the MBR partition entry at 0x1BE intact
sudo dd if=/dev/zero of=/dev/rdiskNs1 bs=512 count=2048

diskutil eject /dev/diskN
```

Optional—for an even closer replica of the reporter's disk, change the
partition type to `0x07` (NTFS/exFAT) after step B.2:

```bash
sudo fdisk -e /dev/rdiskN
# fdisk: 1> edit 1
# Partition id ('0' to disable): 07
# fdisk:*1> write
# fdisk: 1> quit
```

Verify the MBR survived and the filesystem is gone:

```bash
sudo xxd -l 512 /dev/rdiskN | tail -4
# 0x1BE: partition entry present; 0x1FE: 55 aa signature

sudo xxd -l 64 /dev/rdiskNs1
# All zeros—no filesystem magic
```

## Scenario C—Generating the Test Fixtures

Unit tests for Task 044 use superfloppy images stored in `backend/test/data/`.
They are loop-mounted on Linux CI (tests skip automatically on macOS where no
loop device exists).

### C.1 Existing fixture: `image.dmg` (ext4, label `NO_PARTITION`)

Already committed. Structure: `partition-scheme: none`, `Partitioned: false`,
ext2/4 filesystem starting at sector 0. To regenerate it on macOS via Docker
(no native `mkfs.ext4` on macOS):

```bash
docker run --rm -v "$PWD":/data alpine sh -c \
  'apk add --no-cache e2fsprogs && \
   dd if=/dev/zero of=/data/image.dmg bs=1m count=16 && \
   mkfs.ext4 -L NO_PARTITION /data/image.dmg'
```

On Linux the same image is produced without Docker:

```bash
dd if=/dev/zero of=image.dmg bs=1m count=16
mkfs.ext4 -L NO_PARTITION image.dmg
```

### C.2 New fixture: `rawfs_no_parttable.dmg` (FAT32)

Created as part of Task 044 (task list item 8). On a normal macOS session:

```bash
hdiutil create -size 16m -fs "MS-DOS FAT32" -layout NONE \
  -volname NO_PART_FAT backend/test/data/rawfs_no_parttable.dmg
```

Verify before committing:

```bash
hdiutil imageinfo backend/test/data/rawfs_no_parttable.dmg | grep -iE "partition"
# Expected: partition-scheme: none   /   Partitioned: false

file backend/test/data/rawfs_no_parttable.dmg
# Expected: DOS/MBR boot sector ... FAT32

xxd -s 446 -l 64 backend/test/data/rawfs_no_parttable.dmg
# Expected: all zeros—the four MBR partition entries must be empty
```

Keep the image small (16 MB)—it is committed to the repository.

## Verifying the Bug on HAOS

1. Plug the prepared USB drive into the Home Assistant OS host.
2. Compare what the kernel sees with what the Supervisor reports:

   ```bash
   # On the host (SSH / Terminal & SSH add-on with protection mode off)
   lsblk -f
   # Scenario A: /dev/sdX shows FSTYPE=vfat directly on the disk (no sdX1)
   # Scenario B: /dev/sdX1 exists but FSTYPE is empty

   ha hardware info
   # The drive appears under drives, but its filesystems list is empty
   ```

3. Observe SRAT behavior:
   - **Before the fix:** the disk is missing from the Volumes page; no mount action exists.
   - **After the fix (Task 044):** the disk is listed as a raw disk without a partition table, the detected whole-disk filesystem is shown, and a mount action is available (read-only mode off, non-system disk).

## Restoring the USB Drive

Return the stick to normal use when testing is done:

```bash
diskutil eraseDisk "MS-DOS FAT32" USB MBR /dev/diskN
```

## Related

- [Task 044—Support Disks Without Partitions](tasks/044_support-disks-without-partitions.md)
- [dianlight/srat#849—Support disk without partitions](https://github.com/dianlight/srat/issues/849)
- [dianlight/hassio-addons#716—No way to manually mount disk](https://github.com/dianlight/hassio-addons/issues/716)
- Upstream root cause: [supervisor/api/hardware.py](https://github.com/home-assistant/supervisor/blob/main/supervisor/api/hardware.py) (`drive_struct` builds `filesystems` only from UDisks2 `.Filesystem` block devices)
