# Design

## Context

Change 054 introduced a compatibility layer that detects HA-managed local disks by path prefix and disables Srat's own mount/format/label/unmount. The two open gaps were version gating and user control; this change additionally lets SRAT provision disks through Home Assistant so they become visible to HA Core, the media library and other addons without the `/media` symlink workaround.

## Goals / Non-Goals

- Goals: detect HA version; expose a capability flag; add a user-facing mode setting (`off`/`on`/`auto`); make the mount orchestrator and volumes UI honor it; let SRAT create and remove `disk` mounts through the Supervisor API with a per-disk backend choice; keep the path-prefix guard as the fail-safe in `off` mode.
- Non-Goals: changing the addon privilege profile (tracked separately in `dianlight/hassio-addons`); supporting HA versions older than 2026.11 for native mounts (those keep the self-mount path); converting existing `/mnt` mounts to HA mounts automatically; switching the backend of an already-mounted disk in place.

## Decisions

- **Version detection**: reuse `HaRootService.GetSystemInfo()`, which already calls the Supervisor `/info` endpoint and returns `Supervisor` and `Homeassistant` versions. Parse both with `Masterminds/semver/v3`. Store results in `dto.ContextState` (`SupervisorVersion`, `HACoreVersion`) and in `SystemCapabilities` (`HaNativeStorageSupported`). Re-check on `HomeAssistantEvent` STARTED and invalidate the 24-hour `haRootSystemInfoCache` on that event so an upgrade to 2026.11 flips the capability within seconds rather than a day. Fall back to the custom-component handshake version (`HAWsComponent.HAVersion`) when `/info` is unavailable; default the capability to `false` until a version is known.
- **Capability flag**: `SystemCapabilities.HaNativeStorageSupported bool`, true only when Supervisor >= 2026.10.1 and Core >= 2026.11. Runtime state, never a `Settings` field (per the DTO separation rule).
- **Setting**: `Settings.HaNativeStorageMode string` enum `off|on|auto`, default `auto`. Follow the `StandardShareNamesMode` pattern (custom type, `MarshalJSON` normalizing the zero value, `default:"auto"` tag). No DB migration is required: `defaults.Set` applies the default when the row is absent.
- **Mode matrix**:
  - `off`: only the local backend is offered; HA-path detection is disabled; SRAT mounts under `/mnt` as today.
  - `auto` (default): both backends are offered when the capability is true, HA preselected; the guard stays on regardless of the resolved version; when the capability is false the HA option is disabled with a tooltip.
  - `on`: only the HA backend is offered and requires support; when the capability is false nothing is mountable and the volumes page shows "not supported on this HA version" with the way out being `auto` or `off`.
- **Per-disk choice**: the mount dialog (`VolumeMountDialog`) gains a backend selector shown only when the capability is true and the mode is not `off`. The local backend keeps the existing path/fstype/flags/automount fields; the HA backend takes name, usage (`backup`/`media`/`share`) and read-only. The HA mount's `user_path` comes from the Supervisor response and becomes the mount point path.
- **Ownership marker**: `MountPointData.HaMountName *string` (persisted) records mounts SRAT provisioned through the Supervisor API. Unmount is allowed only when this marker is set (SRAT calls `DELETE /mounts/{name}` and removes the local row); format/check/label stay blocked on every HA-backed row, even SRAT-provisioned ones, because the filesystem is still owned by HA.
- **Guard scope**: the 054 path-prefix guard stays on for `auto` and `on` unconditionally; it is disabled only in explicit `off` mode. This keeps the fail-safe intact when the version cannot be verified (dev, docker, container installs).
- **Automount**: `is_to_mount_at_startup` is hidden/derived true for HA-backed rows, since Supervisor mounts persist and auto-mount at boot.
- **Media symlinks**: `recoverMediaUsageSymlinks` must skip HA-backed mount paths (already visible under `/media`) and remove stale links left by a previous local mount of the same disk.
- **UI**: a settings control (lab-gated initially, following `HomeAssistantPanel.tsx`) for the mode; a capability badge and "not supported on this HA version" state on the volumes page; the backend selector in the mount dialog; disabled controls carry a tooltip explaining the mode/version reason.
- **API surface**: `GET /api/capabilities` gains `ha_native_storage_supported`; `GET /api/volumes` carries the persisted `ha_mount_name` and the existing `is_ha_mounted`. OpenAPI regeneration and frontend type regeneration required.

## Risks / Trade-offs

- Version parsing depends on Supervisor being ready; until then the capability defaults to `false` (safe: local path stays available in `auto`).
- `off` mode on a HA 2026.11 system loses the visibility benefit but preserves backward compatibility.
- Adding a settings field requires the `default:"auto"` tag and the `MarshalJSON` normalizer; no DB migration is needed, but a migration is required if a semantic backfill is ever wanted for existing installs.
- The Supervisor mount API's filesystem support and mount-name validation are not fully documented; create failures must surface as a problem entry (reuse the `handleSupervisorMountTargetNotEmpty` pattern) rather than a silent error.
- HA-backed mounts are shared with HA Core and other addons, so removing one affects more than SRAT; the ownership marker is what makes that removal safe and attributable.
- `user_path` availability for `share` and `backup` usages, and nested mount propagation inside the addon namespace, still need live verification (carried over from change 054).

## Migration Plan

1. Add the setting, the capability flag and the persisted ownership marker.
2. Wire version detection into `HaRootService` with cache invalidation on HA STARTED.
3. Implement create/remove of `disk` mounts in `SupervisorService` and the backend selector in the mount dialog.
4. Make the mount orchestrator, volumes UI and filesystem handlers honor the mode and ownership marker.
5. Regenerate OpenAPI and frontend types; update docs and the 054 task document.

## Open Questions

- Should format/check/label on an HA-backed row ever be allowed, or always require removing the HA mount first? (Current plan: blocked; require removal.)
- Does the switch-backend flow (local mount converted to an HA mount) belong in this change, or stay out of scope with a block? (Current plan: out of scope; block mounting a disk that already has a local mount.)
- Does the Supervisor `/info` endpoint return Core version in all installation types (Container, Supervised)? Need to verify; fall back to the custom-component handshake version when missing.
