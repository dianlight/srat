<!-- DOCTOC SKIP -->

# [FEATURE]: Support HA 2026.11 native USB drive mounting

**Target Repo:** `srat`
**Status:** 🔄 In Progress
**Issue Link:** [srat#1530](https://github.com/dianlight/srat/issues/1530), [hassio-addons#807](https://github.com/dianlight/hassio-addons/issues/807)

## 🎯 Objective

Let Home Assistant own local-disk mounting (HA 2026.11 / Supervisor 2026.10.1+) while SambaNAS2 only manages shares over the HA-mounted paths. This tightens the addon permission model: a user can run in protected mode with HA Storage mounting the disks, and Srat exposes `/share/<name>`, `/media/<name>` or `/backup/<name>` without performing its own mounts.

The chosen direction is a **compatibility layer first**: detect HA-mounted local disks, expose them as pre-mounted `HOST` volumes flagged `is_ha_mounted`, and disable mount/format/label/unmount for them. Srat keeps its own mount path for every other disk. No addon privilege change in this phase; a reduced-privilege addon profile is a follow-up in `dianlight/hassio-addons`.

> _Context for Copilot: Upstream request hassio-addons#807 by @jondycz; maintainer replied that a compatibility layer is planned after some architectural changes. HA 2026.11 adds a `disk` mount type to the Supervisor mount API (UDisks2-backed, mounted by UUID, `.automount` + `.mount`, usage backup/media/share, read-only support). The Mount model exposes `uuid`, `device`, `filesystem` and `user_path` (the in-app path, e.g. `/media/nas`), which solves the addon path-namespace problem._

## 🛠️ Technical Specifications

- **Inputs:** Supervisor `GET /mounts` entries with `type: disk`; addon already maps `share`, `media`, `backup` read/write.
- **Outputs:** `MountPointData{Type:"HOST", IsMounted:true, IsHAMounted:true}` in the `DiskMap`, share creation allowed, mount/format/label/unmount disabled in API + UI.
- **HA-supported filesystems:** ext2/3/4, vfat, exfat, ntfs, btrfs, f2fs.
- **Dependencies:** `backend/src/homeassistant/mount/*`, `backend/src/service/supervisor_service.go`, `backend/src/service/volume/*`, `backend/src/service/volume_service.go`, `backend/src/service/share_service.go`, `backend/src/converter/ha_hardware_to_dto.go`, `frontend/src/pages/volumes/*`, `frontend/src/pages/shares/*`.

## 📝 Task List

- [x] Task 1: Create task doc with hassio-addons#807 + srat#1530 links
- [x] Task 2: Extend Supervisor mount client (`disk` type, `uuid`, `device`, `filesystem`, `user_path`)
- [x] Task 3: Add `MountPointData.IsHAMounted` marker and detect HA disk mounts in `SyncPartitionMountData`
- [x] Task 4: Guard Srat mount/unmount/automount/format/label/check and the HA self-mount bridge against HA-managed volumes
- [x] Task 5: Mark shares on HA-managed volumes `is_ha_mounted`; verify existence/mounted/read-only correctly
- [x] Task 6: Hide mount/format/label/unmount actions in the UI; show the HA badge; keep Create Share
- [x] Task 7: Tests (backend + frontend)
- [ ] Task 8: Follow-up in `dianlight/hassio-addons`: reduced-privilege / protected-mode profile for native-only use

## 🧠 Implementation Notes (Copilot Context)

### Discovery and path mapping

- Detection happens in `volume.SyncPartitionMountData`: a procfs mount whose source is the partition device and whose mount point is under `/share/`, `/media/` or `/backup/` is HA-managed (`volume.IsHAManagedMountPath`). SRAT always mounts under `/mnt`, so the prefix is unambiguous; a network mount created by the Srat→HA bridge has a cifs/nfs source and never matches a partition device.
- This relies on the HA mount being visible inside the addon namespace (`map: share/media/backup`). Supervisor uses slave bind propagation for the data volume; verify on a live HA instance.
- The Supervisor mount client now models `disk` mounts (`uuid`, `device`, `filesystem`, `user_path`). A `/mounts`-based confirmation layer that uses `user_path` is a possible follow-up if the prefix heuristic ever proves insufficient; it is not needed for the current flow.

### Data model

- `MountPointData.IsHAMounted` is read-only and annotated (never persisted from config). Reuse `SharedResource.Status.IsHAMounted` for shares.
- `UnmountVolume` already special-cases `IsHAMounted`: it emits a Share REMOVE event instead of an OS unmount. Keep that path and never call `umount` for HA-managed volumes.

### Guardrails

- `NetworkMountAllShares` / `networkUnmountLostShares`: skip shares whose volume is HA-managed to avoid mounting a share back into HA over an HA-mounted disk.
- `haosSystemMountPaths` / automount: do not automount a partition whose mount point is HA-managed.
- `ha_hardware_to_dto.mountPointsToMountPointDatas`: normalize host paths and tag HA-managed mountpoints.
- Format/label actions must be rejected server-side for HA-managed partitions, not only hidden in the UI.

### Refresh and events

- Reconcile on startup, after hardware refreshes, and on a bounded poll (HA exposes no mount event stream). Update the `DiskMap` cache before emitting events (#971).
- On HA unmount: mark the mount point invalid/unmounted, invalidate dependent shares and emit the volume/share events.

### Open questions

- Does `GET /mounts` report `user_path` for `share` and `backup` usage as well as `media`? (Documented example is `/media/nas`.)
- Does nested mount propagation make a disk mounted under the host `/share` visible inside the addon's `map: share` after container start? Network storage is documented to work; verify with a live HA instance.
- Does `/hardware/info` list a filesystem whose partition is mounted by HA (needed to correlate the raw device for SMART)? If not, correlation must fall back to `/host/disks`.

## 🔗 Code References & TODOs

- [ ] `hassio-addons#807` — Mounting drives in HA 2026.11
- [ ] `srat#1530` — mirror closes once tracked here
- [ ] `backend/src/homeassistant/mount/mount.yaml` — missing `disk` type and disk fields
- [ ] `backend/src/service/volume/mount_orchestrator.go` — `UnmountVolume` HA-mounted branch, automount guards
- [ ] `backend/src/service/supervisor_service.go` — `NetworkMountAllShares` self-mount loop guard
- [ ] `backend/src/service/share_service.go` — `VerifyShare` and `standardShareDir`
