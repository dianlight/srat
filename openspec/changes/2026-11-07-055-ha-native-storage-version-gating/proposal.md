# Proposal

## Why

Home Assistant 2026.11 (Supervisor 2026.10.1+) introduced native local-disk storage (`disk` mount type via UDisks2). SambaNAS2's compatibility layer (task 054) currently detects HA-managed mounts by path prefix and disables Srat's own mount/format/label actions, but it does not yet:

- gate the feature on the detected Supervisor/Core version, so users on older HA see (or attempt to use) a feature that cannot work;
- expose a setting to choose whether SRAT should use the HA Storage API for mount/unmount at all, forcing the current path-based heuristic even where HA support exists.

## What Changes

- Detect the running Home Assistant Supervisor / Core version at startup and on version changes; expose it through `dto.ContextState` / `SystemCapabilities` (not `Settings`).
- Gate HA-native storage handling behind a version check (Supervisor >= 2026.10.1 / Core >= 2026.11) and surface a "not supported on this HA version" state in the UI.
- Add a setting `ha_native_storage_mode` with values `off`, `on`, `auto` (default `auto`): when `on`, SRAT relies on HA Storage for mounts and hides its own mount UI; when `off`, SRAT always mounts itself; when `auto`, SRAT uses HA when the version supports it and falls back to self-mount otherwise.
- Add a backend service that reconciles the setting with the detected version and publishes a capability flag consumed by the volumes/shares UI and by the mount orchestrator.
- Persist the setting in `dto.Settings` (user-configurable) and mirror it to the frontend via OpenAPI regeneration.

## Capabilities

### New Capabilities

- `ha-native-storage`: version-gated, configurable use of Home Assistant native local-disk storage.

### Modified Capabilities

- `volumes`: mount capability flags now depend on the HA-native storage mode and detected version.
- `settings`: new `ha_native_storage_mode` setting.

## Impact

- `backend/src/dto/context.go` / `dto/settings.go` / `dto/system_capabilities.go` (if present) — version state and new setting.
- `backend/src/service/supervisor_service.go` — version detection and mount gating.
- `backend/src/service/volume/mount_orchestrator.go` — respect the mode when deciding whether to mount.
- `backend/src/api/system.go` (capabilities endpoint) and `backend/src/api/volumes.go`.
- `frontend/src/pages/volumes/` and `frontend/src/pages/settings/` — mode selector and capability badge.
- OpenAPI regeneration required.