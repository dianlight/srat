<!-- DOCTOC SKIP -->

# [FIX]: Addon Restart Device-Not-Found Race

**Target Repo:** `srat`
**Status:** 🔄 In Progress
**Issue Link:** [srat#1266](https://github.com/dianlight/srat/issues/1266), [hassio-addons#767](https://github.com/dianlight/hassio-addons/issues/767)

## 🎯 Objective

Fix transient `Device 'ata-...' not found during startup` on soft add-on Restart. The by-id symlink (or Supervisor hardware snapshot) is not yet visible when the first automount runs, the failure is terminal (no retry), and only a full host reboot recovers.

> _Context for Copilot: Startup retry + recheck direction. Invalidate HW cache and schedule a bounded delayed refresh on ErrorDeviceNotFound so the mount retries once the device appears. Keep the automount guard bounding the chain._

## 🛠️ Technical Specifications

- **Inputs:** `MountPointEvent ADD/UPDATE` with `IsToMountAtStartup=true`, `MountVolume` returning `ErrorDeviceNotFound`
- **Outputs:** HW cache invalidated, one delayed `refresh()` scheduled, existing guard still bounds total attempts
- **Dependencies:** `backend/src/service/volume/udev_handler.go`, `backend/src/service/volume/mount_orchestrator.go`, `backend/src/service/volume_service.go` wiring (no wiring change needed)

## 📝 Task List

- [x] Task 1: Create task doc with hassio-addons#767 + srat#1266 links
- [x] Task 2: Failing test — DeviceNotFound schedules HW invalidate + delayed refresh
- [x] Task 3: Fix HandleMountPointEvent DeviceNotFound path + ScheduleDeviceNotFoundRetry
- [x] Task 4: Run backend tests + coverage gate, hk check

## 🧠 Implementation Notes (Copilot Context)

- `MountOrchestrator.MountVolume` (`mount_orchestrator.go:257`) returns `ErrorDeviceNotFound` on missing `os.Stat(DevicePath)`.
- `UdevHandler.HandleMountPointEvent` (`udev_handler.go:650`) records failure + notification but never refreshes; with no further udev events the mount never retries.
- Fix: on `errors.Is(err, dto.ErrorDeviceNotFound)`, invalidate hardware, schedule `time.AfterFunc` delayed `refresh()` (default 5s, test-settable), then keep existing Record + notification so the chain stays bounded by `maxAutomountAttempts` (5).
- Follow existing `SchedulePartitionAddRetry` pattern (ctx-cancel guard, nil guards).

## 🔗 Code References & TODOs

- [ ] `hassio-addons#767` — Device not found during add-on restart, `/mnt/Ext_HD`, `stat /dev/disk/by-id/...: no such file`
- [ ] `srat#1266` — mirror closes once tracked here
