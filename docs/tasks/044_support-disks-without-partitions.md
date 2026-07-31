<!-- DOCTOC SKIP -->

# [FIX]: Support Disks Without Partitions

**Target Repo:** `srat`
**Status:** 🔄 In Progress
**Issue Link:** [dianlight/srat#849](https://github.com/dianlight/srat/issues/849), [dianlight/hassio-addons#716](https://github.com/dianlight/hassio-addons/issues/716)

## 🎯 Objective

Make disks that have **no partition table** (raw "superfloppy" whole-disk filesystem) or whose filesystems are **not reported by the HA Supervisor** visible and mountable in SRAT. Detection must be **automatic**—the owner explicitly rejected the "manual insert of disk path" approach in #849. Scope is **visibility + mount** of the detected whole-disk filesystem; no format or create-partition-table actions are part of this task.

> _Context for Copilot: A USB disk formatted directly (like an old floppy, no MBR/GPT) or with a filesystem UDisks2 cannot probe is currently dropped at three points in the hardware and volume services and never reaches the UI. The fix keeps these drives in the disk map, probes the raw block device for a filesystem magic signature, and synthesizes a whole-disk partition entry so it can be mounted._

## 🛠️ Technical Specifications

- **Inputs:**
  - HA Supervisor `/hardware/info` response where `drives[].filesystems` is empty or `null`
  - Raw block device of the whole disk (e.g. `/dev/sda`) for fallback probing
- **Outputs:**
  - `dto.DiskMap` containing partition-less drives with a synthesized whole-disk `dto.Partition` (no partition number, `DevicePath` = disk device path) when a filesystem is detected
  - `volumes` WebSocket event / `GET /api/volumes` payloads including these disks so the UI can list and mount them
  - Frontend "Raw disk / no partition table" state with a mount action (gated by read-only mode and non-system disk)
- **Dependencies:**
  - `backend/src/service/hardware_service.go` (drop points at lines 123-124, 133-134)
  - `backend/src/service/volume_service.go` (nil-partition guards at lines 395, 554, 653, 902)
  - `backend/src/converter/ha_hardware_to_dto.go` (goverter `DriveToDisk`, `filesystemsToPartitionsMap`)
  - `backend/src/internal/darwinstubs/mount` → `mount.FSFromBlock` (u-root magic scan, `vendor/github.com/u-root/u-root/pkg/mount/magic.go:196`; **Linux-only**, returns `errUnsupported` on darwin)
  - `backend/src/internal/darwinstubs/mount/loop` for loop-device test attachment (Linux-only)
  - `backend/test/data/image.dmg` (existing ext4 superfloppy fixture, volume label `NO_PARTITION`)
  - `backend/test/data/rawfs_no_parttable.dmg` (**new** FAT32 superfloppy fixture to create)
  - Frontend: `Volumes.tsx`, `VolumesTreeView.tsx`, `VolumeDetailsPanel.tsx` (already tolerate `disk.partitions || {}`)

## 📝 Task List

- [x] Task 1: Remove/adjust drop point 1—`hardware_service.go:123-124` (`if drive.Filesystems == nil || len(*drive.Filesystems) == 0 { continue }`): keep partition-less drives in the pipeline instead of skipping
- [x] Task 2: Remove/adjust drop point 2—`hardware_service.go:133-134` (`if diskDto.Partitions == nil || len(*diskDto.Partitions) == 0 { continue }`): carry drives with empty partition maps in `dto.DiskMap`
- [x] Task 3: Add whole-disk filesystem probe—bounded `mount.FSFromBlock` call on the disk device path, Linux-only, never on `System`/protected drives, results cached with the hardware info cache (30-min `hwCacheKey`)
- [x] Task 4: Synthesize whole-disk partition entry when a raw filesystem magic is found (no partition number, `DevicePath`/`LegacyDeviceName` = disk dev name, `Name`/`Label` from probe where available)
- [x] Task 5: Fix `volume_service.go` nil-partition guards (lines 395 `findPartitionByDevName`, 554 `getVolumesData`, 653 `findDiskForDevicePath`, 902) so partition-less disks flow through mount/status logic
- [x] Task 6: Update `converter/ha_hardware_to_dto.go` so empty `Filesystems` no longer causes the drive to be treated as absent; keep `filesystemsToPartitionsMap` empty-map behavior
- [x] Task 7: Wire events—udev add/remove and format-refresh paths (`handleFilesystemTaskEvent` and related) handle disks with nil/empty partitions; emit `volumes` updates when a partition-less disk appears/disappears
- [x] Task 8: Create FAT32 superfloppy fixture `backend/test/data/rawfs_no_parttable.dmg` (see `docs/replicate-partitionless-disk-macos.md` Scenario C) and commit it
- [x] Task 9: Unit tests—hardware service keeps empty-partition drives (mockio `GetHardwareInfo`, testify/suite + fxtest, per `backend_test.instructions.md`)
- [x] Task 10: Unit tests—probe + synthesized partition using `loop.FindDevice()` + `loop.SetFile()` with `image.dmg` (ext4) **and** `rawfs_no_parttable.dmg` (FAT32); `suite.T().Skip("No loop device available")` when loop is unavailable (darwin CI)
- [x] Task 11: Frontend—disk-level "Raw disk / no partition table" presentation in `VolumesTreeView`/`VolumeDetailsPanel` for disks with zero partitions
- [x] Task 12: Frontend—mount action for the detected whole-disk filesystem, shown inactive with `readOnlyActionTooltip` in read-only mode and hidden for `System` disks
- [x] Task 13: Frontend tests (Vitest + RTL + `user-event`, MSW handlers in `frontend/src/mocks/customHandlers.ts`) for the empty-disk state and mount action
- [x] Task 14: Update related documentation—volumes/troubleshooting docs that currently state unpartitioned disks are unsupported; link `docs/replicate-partitionless-disk-macos.md`
- [x] Task 15: Update `CHANGELOG.md` under `## [ 🚧 Unreleased ]` (per `/update-changelog` skill)
- [ ] Task 16: Manual validation on HAOS with a physical USB prepared per `docs/replicate-partitionless-disk-macos.md` (Scenario A superfloppy; Scenario B #716 replica) — deferred, see [PR #867](https://github.com/dianlight/srat/pull/867)
- [x] Task 17: Ask to create a PR with the task implementation and link it here for tracking — [PR #867](https://github.com/dianlight/srat/pull/867)

## 🧠 Implementation Notes (Copilot Context)

### Root-cause chain (verified)

1. SRAT learns about disks **only** from the HA Supervisor `/hardware/info` API (`hardwareService.GetHardwareInfo()`, 30-min go-cache, gated by `state.HACoreReady`).
2. Supervisor (`supervisor/api/hardware.py` → `drive_struct`) builds `drives[].filesystems[]` **only** from UDisks2 block devices exposing the `.Filesystem` D-Bus interface:

   ```python
   [filesystem_struct(block) for block in udisks2.block_devices
    if block.filesystem and block.drive == drive.object_path]
   ```

   If UDisks2 cannot probe a recognized filesystem (raw whole-disk filesystem, zeroed/unreadable filesystem, missing driver) the list is **empty**—even when the disk has a perfectly valid filesystem.
3. SRAT then drops the drive at three points:
   - `hardware_service.go:123`: `if drive.Filesystems == nil || len(*drive.Filesystems) == 0 { continue }` ("Skipping drive with no filesystems")
   - `hardware_service.go:133`: `if diskDto.Partitions == nil || len(*diskDto.Partitions) == 0 { continue }`
   - `volume_service.go:554`: `if disk.Partitions == nil { continue }` (plus the same guard at 395, 653, 902)
4. The frontend never receives the disk → "no actions are possible on raw disk" (#849). The UI itself already tolerates empty partition maps (`disk.partitions || {}` in `Volumes.tsx` lines 55/174/278/328, `VolumesTreeView.tsx` lines 82/498, `VolumeDetailsPanel.tsx:435` shows `0 Partition(s)`).

### Issue #716 data point (MBR decode)

The reporter's "unreadable" TOSHIBA disk actually **has** an MBR: signature `55AA` at `0x1FE`; partition entry at `0x1BE` with boot flag `0x00`, type `0x07` (NTFS/exFAT), LBA start `0x00032800`. UDisks2 still reports no filesystems (likely unreadable/zero filesystem signature or missing ntfs support on HAOS), so Supervisor returns `filesystems: []` and SRAT drops it. This scenario must also be covered: the fix makes the drive **visible** even when the probe finds nothing mountable (listed as raw/unreadable, no mount action).

### Fix design

- **Keep, don't skip**: remove the three `continue` guards so partition-less drives stay in `dto.DiskMap`. Log at debug level as today.
- **Probe fallback**: for a kept drive with zero partitions, attempt `mount.FSFromBlock(<disk device path>)` once per cache window. u-root scans filesystem magics (ext2/3/4, vfat, ntfs, hfs+, apfs, ...). On darwin it returns `errUnsupported`—guard with a runtime/OS check and skip the probe.
- **Synthesize partition**: on successful probe, add one `dto.Partition` keyed by the disk device name (e.g. `sda`) with no partition number, `DevicePath` = disk device path, so all existing mount logic (`mountPartition`, `findPartitionByDevName`, udev matching via `extractDevice` regex `p?\d+$` which already strips partition digits) works unchanged.
- **Safety**:
  - Never probe or synthesize on `System` drives (check the `System` flag from Supervisor `Filesystem`/`Drive` data where available, plus existing protected-device logic).
  - Bounded reads only (magic scan reads a few KB at fixed offsets); no full-device scans.
  - Probe failures are non-fatal: the disk stays visible as raw/unreadable.
- **No manual path entry**: detection is fully automatic per the owner decision in #849.
- **Out of scope**: format / create-partition-table actions on empty disks (deferred); patching Supervisor/UDisks2 upstream.

### Test plan

- Fixtures: reuse `backend/test/data/image.dmg` (ext4 superfloppy, `partition-scheme: none`, verified via `hdiutil imageinfo`); add `rawfs_no_parttable.dmg` (FAT32 superfloppy, generation steps in Scenario C of the repro doc).
- Mock-level tests (no root needed): mock `HardwareServiceInterface.GetHardwareInfo()` returning a disk with `Partitions: nil`/empty; assert the disk survives `getVolumesData()` and appears in the `volumes` payload.
- Loop-level tests (Linux only): `loop.FindDevice()` → `loop.SetFile(device, "../../test/data/image.dmg")` → `mount.FSFromBlock(device)` returns `ext4`; same with the FAT32 fixture returning `vfat`. Skip via `suite.T().Skip("No loop device available")` on darwin.
- Follow `backend_test.instructions.md`: external `_test` package, testify/suite, fxtest DI, mockio v2, cancel context **before** waiting on the WaitGroup in `TearDownTest`.

### Frontend notes

- Disks with zero partitions: render the disk node with a "Raw disk (no partition table)" hint instead of an empty partition list.
- Mount action targets the disk-level synthesized partition (whole-disk device path); disabled with the existing `readOnlyActionTooltip` (`VolumeDetailsPanel.tsx:213`) in read-only mode; hidden/disabled for system disks.
- Do not add format/partition-table UI in this task.

## 🔗 Code References & TODOs

- [ ] `backend/src/service/hardware_service.go:123-124`—drop point 1 (drive with no filesystems)
- [ ] `backend/src/service/hardware_service.go:133-134`—drop point 2 (DTO with no partitions)
- [ ] `backend/src/service/volume_service.go:395`—`findPartitionByDevName` nil guard
- [ ] `backend/src/service/volume_service.go:554`—`getVolumesData` nil guard
- [ ] `backend/src/service/volume_service.go:653`—`findDiskForDevicePath` nil guard
- [ ] `backend/src/service/volume_service.go:902`—additional nil-partition skip
- [ ] `backend/src/converter/ha_hardware_to_dto.go`—`DriveToDisk` / `filesystemsToPartitionsMap` empty-drive handling
- [ ] `backend/src/internal/darwinstubs/mount/mount_linux.go:26`—`FSFromBlock` probe entry point
- [ ] `backend/test/data/image.dmg`—existing ext4 superfloppy fixture (reuse)
- [ ] `backend/test/data/rawfs_no_parttable.dmg`—new FAT32 superfloppy fixture (create)
- [ ] `frontend/src/pages/volumes/`—`Volumes.tsx`, `VolumesTreeView.tsx`, `VolumeDetailsPanel.tsx` empty-disk UI + mount action
- [ ] `docs/replicate-partitionless-disk-macos.md`—USB replication guide (Scenarios A/B/C)
