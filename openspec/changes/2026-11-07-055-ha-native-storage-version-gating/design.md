# Design

## Context

Task 054 introduced a compatibility layer that detects HA-managed local disks by path prefix and disables Srat's own mount/format/label/unmount. The two open gaps are:

1. Version gating — the feature only works on Supervisor >= 2026.10.1 / Core >= 2026.11. Srat has no version detection today beyond `dto.ContextState.HACoreReady`.
2. User control — there is no setting to force Srat to use (or avoid) HA Storage for mounts; the path heuristic is always on.

## Goals / Non-Goals

- Goals: detect HA version, expose a capability flag, add a user-facing mode setting (`off`/`on`/`auto`), and make the mount orchestrator and volumes UI honor it.
- Non-Goals: changing the addon privilege profile (tracked separately in `dianlight/hassio-addons`); supporting HA versions older than 2026.11 for native mounts (those keep the self-mount path).

## Decisions

- **Version detection**: use the Supervisor `/info` (or `/core/info`) endpoint already reachable via `supervisor_service`; parse `supervisor_version` and `homeassistant` (Core) version with `Masterminds/semver/v3`. Store results in `dto.ContextState` (runtime) and in `SystemCapabilities` (served by `GET /api/capabilities`). Re-check on `HomeAssistantEvent` STARTED and on a periodic ticker.
- **Capability flag**: `SystemCapabilities.HaNativeStorageSupported bool` (true when Supervisor >= 2026.10.1 and Core >= 2026.11). This is runtime state, not a `Settings` field (per the DTO separation rule).
- **Setting**: `Settings.HaNativeStorageMode string` enum `off|on|auto`, default `auto`. Persisted via the existing settings repository and exposed through OpenAPI.
- **Effective mode resolution**: `auto` resolves to `on` when `HaNativeStorageSupported`, else `off`. The resolved value drives both backend behavior and the UI.
- **Mount orchestrator**: `MountVolume` and `ShouldSkipAutomount` already reject HA-managed paths; additionally, when the effective mode is `on`, SRAT must not attempt any self-mount even for non-HA paths (protected-mode-like behavior for mounts), and the volumes UI must hide mount/automount controls entirely. When `off`, the HA-managed detection is disabled and Srat mounts everything itself (legacy behavior).
- **UI**: a settings control (lab-gated initially, following `HomeAssistantPanel.tsx`) plus a capability badge on the volumes page; disabled controls show a tooltip explaining the mode/version reason.

## Risks / Trade-offs

- Version parsing depends on Supervisor being ready; until then the capability defaults to `false` (safe: self-mount path).
- `off` mode on a HA 2026.11 system loses the security benefit but preserves backward compatibility for users who prefer Srat-managed mounts.
- Adding a settings field requires OpenAPI regeneration and a DB migration (default value).

## Migration Plan

1. Add the setting with default `auto` and the capability flag.
2. Wire version detection into `SupervisorService`.
3. Make the mount orchestrator and volumes UI consume the effective mode.
4. Regenerate OpenAPI and frontend types; update docs.

## Open Questions

- Should `on` mode also disable SMART/format for HA-managed disks, or keep read-only SMART? (Current plan: keep SMART, disable format/label.)
- Does the Supervisor `/info` endpoint return Core version in all installation types (Container, Supervised)? Need to verify; fall back to `homeassistant` version from the custom component handshake if missing.