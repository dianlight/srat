# Proposal

## Why

Fix transient `Device 'ata-...' not found during startup` on soft add-on Restart. The by-id symlink (or Supervisor hardware snapshot) is not yet visible when the first automount runs, the failure is terminal (no retry), and only a full host reboot recovers.

## What Changes

- Create task doc with hassio-addons#767 + srat#1266 links
- Failing test — DeviceNotFound schedules HW invalidate + delayed refresh
- Fix HandleMountPointEvent DeviceNotFound path + ScheduleDeviceNotFoundRetry
- Run backend tests + coverage gate, hk check

## Capabilities

### New Capabilities

- `addon-restart-device-not-found-race`: Addon Restart Device-Not-Found Race

### Modified Capabilities

(none)

## Impact

- **Inputs:** `MountPointEvent ADD/UPDATE` with `IsToMountAtStartup=true`, `MountVolume` returning `ErrorDeviceNotFound`
- **Outputs:** HW cache invalidated, one delayed `refresh()` scheduled, existing guard still bounds total attempts
- **Dependencies:** `backend/src/service/volume/udev_handler.go`, `backend/src/service/volume/mount_orchestrator.go`, `backend/src/service/volume_service.go` wiring (no wiring change needed)
- Legacy task: `docs/tasks/053_addon-restart-device-race.md`
