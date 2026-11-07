# Tasks

## 1. Version Detection and Capability

- [ ] 1.1 Add `SupervisorVersion` and `HACoreVersion` (or reuse `HACoreReady`) fields to `dto.ContextState`; add `HaNativeStorageSupported` to `SystemCapabilities` (create the DTO if absent, served by `GET /api/capabilities`).
- [ ] 1.2 Implement version parsing in `SupervisorService` using the Supervisor `/info` endpoint and `Masterminds/semver/v3`; set the capability flag; re-check on HA started events and on a periodic ticker.
- [ ] 1.3 Handle missing Supervisor/Core version gracefully (default capability `false`); fall back to the custom-component handshake version when available.
- [ ] 1.4 Add backend unit tests for version parsing and capability computation.

## 2. Settings and Effective Mode

- [ ] 2.1 Add `HaNativeStorageMode string` (enum `off|on|auto`, default `auto`) to `dto.Settings`, with DB migration defaulting to `auto`.
- [ ] 2.2 Add an effective-mode resolver (setting + capability) in a small service or helper; expose it via the capabilities endpoint as `ha_native_storage_mode_effective`.
- [ ] 2.3 Add a settings UI control (lab-gated initially, following `HomeAssistantPanel.tsx`) for the mode; persist via the existing settings API.
- [ ] 2.4 Add backend unit tests for the resolver (auto→on/off, explicit on/off).

## 3. Mount Orchestrator and Volumes UI

- [ ] 3.1 Make `MountVolume` and `ShouldSkipAutomount` honor the effective mode: `on` blocks all self-mounts; `off` disables HA-managed detection.
- [ ] 3.2 Update the volumes page to hide mount/automount controls when the effective mode is `on`, and to show a capability badge / "not supported on this HA version" state.
- [ ] 3.3 Update the HA-managed mount detection to be a no-op when the effective mode is `off`.
- [ ] 3.4 Add backend and frontend tests for the mode-driven behavior.

## 4. API Surface and Documentation

- [ ] 4.1 Update OpenAPI spec and regenerate frontend types (`mise run //backend:generate:openapi`, `mise run //frontend:gen`).
- [ ] 4.2 Document the new setting and capability in `docs/SETTINGS_DOCUMENTATION.md` and the volumes test docs.
- [ ] 4.3 Update `docs/tasks/054_ha-2026-11-native-usb-mounting.md` to reference this change.