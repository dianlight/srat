# Proposal

## Why

Home Assistant 2026.11 (Supervisor 2026.10.1+) introduced native local-disk storage (`disk` mount type via UDisks2). SambaNAS2's compatibility layer (change 054) detects HA-managed mounts by path prefix and disables Srat's own mount/format/label/unmount actions, but it does not yet:

- gate the feature on the detected Supervisor/Core version, so users on older HA see (or attempt to use) a feature that cannot work;
- expose a setting to choose whether SRAT should use the HA Storage API for mount/unmount at all, forcing the path-based heuristic even where HA support exists;
- let SRAT provision disks through Home Assistant instead of mounting them privately under `/mnt`, which is the only way a disk mounted by SRAT becomes visible to HA Core, the media library and other addons without the `/media` symlink workaround.

## What Changes

- Detect the running Home Assistant Supervisor / Core version at startup and on version changes; expose it through `dto.ContextState` / `SystemCapabilities` (not `Settings`).
- Gate HA-native storage handling behind a version check (Supervisor >= 2026.10.1 / Core >= 2026.11) and surface a "not supported on this HA version" state in the UI.
- Add a setting `ha_native_storage_mode` with values `off`, `on`, `auto` (default `auto`), where `on` makes HA Storage the only backend, `auto` (default) offers both backends with HA preselected when supported, and `off` keeps the legacy local-only behavior.
- Add a backend service that reconciles the setting with the detected version and publishes a capability flag consumed by the volumes/shares UI and by the mount orchestrator.
- Persist the setting in `dto.Settings` (user-configurable) and mirror it to the frontend via OpenAPI regeneration.
- Allow SRAT to create and remove `disk` mounts through the Supervisor mount API (`POST`/`DELETE /mounts`), with a per-disk backend choice in the mount dialog, while keeping the existing path-prefix guard as the fail-safe for `off` mode.
- Track SRAT-provisioned HA mounts with a persisted ownership marker so unmount is allowed only for mounts SRAT created, while format/check/label stay blocked on any HA-backed row.

## Capabilities

### New Capabilities

- `ha-native-storage`: version-gated, configurable provisioning and use of Home Assistant native local-disk storage. Depends on change 054's `ha-native-storage-compat`.

### Modified Capabilities

- `volumes`: mount affordances and capability flags now depend on the HA-native storage mode and the detected version; the mount dialog gains a backend selector.
- `settings`: new `ha_native_storage_mode` setting.

## Impact

- `backend/src/dto/context.go`, `dto/settings.go`, `dto/system_capabilities.go` — version state, new setting, capability flag.
- `backend/src/dto/mount_point_data.go` — persisted HA mount ownership marker.
- `backend/src/service/ha_root_service.go` — version source and cache invalidation.
- `backend/src/service/supervisor_service.go` — version-gated mount gating; create/remove disk mounts.
- `backend/src/service/volume/mount_orchestrator.go`, `mount_repository.go`, `udev_handler.go` — guard and ownership behavior.
- `backend/src/service/share_service.go`, `mount_intelligence.go` — HA ownership propagation.
- `backend/src/service/server_process_service.go` — media symlink handling for HA-backed mounts.
- `backend/src/api/filesystems.go`, `api/volumes.go`, `api/system.go` — capability endpoint and format guards.
- `frontend/src/pages/volumes/`, `frontend/src/pages/settings/` — mode selector, capability badge, backend selector in the mount dialog.
- OpenAPI regeneration and frontend types required.
