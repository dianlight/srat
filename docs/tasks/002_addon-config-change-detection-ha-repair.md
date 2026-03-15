# [FEATURE]: Addon Config Change Detection with UI Popup and HA Repair

**Target Repo:** `srat`  **Status:** 📅 Planned  **Issue Link:** _TBD_

## 🎯 Objective

Detect when the Home Assistant Supervisor changes the addon configuration (`/data/options.json`) and react in two parallel channels: show a popup/banner in the SRAT frontend web UI, and raise a HA Repair issue (or persistent notification as fallback) so the user knows a config reload is needed. Detection should prefer HA Supervisor events when available and fall back to a configurable fixed-interval file watch (fsnotify) when running outside the Supervisor context.

> _Context for Copilot: The backend already has `fsnotify` used in `UpgradeService` (`service/upgrade_service.go`) for watching binary files, `persistent_notification` create/dismiss in `HomeAssistantService` (`service/homeassistant_service.go`), and `AppConfigData.RequiresRestart` in `dto/app_config.go`. The event bus already emits `AppConfigEvent` after user-initiated config saves. The new flow covers **externally-initiated** changes (Supervisor writes `/data/options.json`) that the backend has not acted on yet._

## 🛠️ Technical Specifications

- **Inputs:**
  - `/data/options.json` — file written by HA Supervisor on addon config change
  - HA Supervisor WebSocket event `supervisor/event` of type `addon_config_changed` (when running under Supervisor)
  - A fallback polling interval (e.g., every 60 s), configurable via `AppConfig`

- **Outputs:**
  - SSE event of type `app_config_changed` emitted on the SRAT event bus → consumed by the frontend
  - Frontend popup/snackbar: "Addon configuration has changed. Reload required." with a **Reload** action button
  - HA Repair issue (`ir.async_create_issue`) via the Go backend calling the Supervisor API; falls back to `persistent_notification` if Repairs API is unavailable
  - Repair/notification auto-dismissed once the config has been reloaded (restart or live-reload)

- **Dependencies:**
  - `backend/src/service/homeassistant_service.go` — existing `persistent_notification` helpers; add Repair API calls here
  - `backend/src/service/upgrade_service.go` — reference implementation for `fsnotify` file watcher + debounce pattern
  - `backend/src/config/addon_options.go` — options file path constant
  - `backend/src/events/` — define new `AddonConfigChangedEvent` event type
  - `backend/src/dto/app_config.go` — `AppConfigData.RequiresRestart` flag
  - `frontend/src/` — new snackbar/banner component triggered by the new SSE event
  - `custom_components/srat/` — alternative path: listen for the HA push event and call `async_reload` entry

## 📝 Task List

- [ ] Task 1: Define `AddonConfigChangedEvent` in `backend/src/events/events.go` and add emit helper to event bus
- [ ] Task 2: Implement `AddonConfigWatcherService` in `backend/src/service/` — watch `/data/options.json` via `fsnotify` with debounce; on change compute a content hash to suppress spurious triggers
- [ ] Task 3: Try HA Supervisor events first — subscribe to `supervisor/event` WebSocket topic; if not available (no Supervisor token) fall back to the fsnotify watcher with configurable interval
- [ ] Task 4: On config change detected, emit `AddonConfigChangedEvent` on event bus and set `AppConfigData.RequiresRestart = true` in the config DTO
- [ ] Task 5: Add `CreateRepairIssue` / `DismissRepairIssue` helpers to `HomeAssistantService` using HA Supervisor Repairs API (`/core/api/repairs/issues`); fall back to `persistent_notification` when Repairs endpoint returns 404
- [ ] Task 6: Wire the watcher service into the fx dependency graph (`appsetup.go`)
- [ ] Task 7: Frontend — subscribe to the new `app_config_changed` SSE event in RTK Query; show a persistent `Snackbar` / `Alert` with a **Reload** button calling `POST /api/app-config/reload` (or triggering browser reload)
- [ ] Task 8: Auto-dismiss the HA Repair issue / persistent notification after a successful config reload
- [ ] Task 9: Unit testing — `AddonConfigWatcherService` with a mock fsnotify watcher; test hash-based deduplication, debounce, and fallback path
- [ ] Task 10: Integration testing — end-to-end: write to a temp options file, verify `AddonConfigChangedEvent` emitted and `RequiresRestart` flipped
- [ ] Task 11: Frontend component test — `AddonConfigChangedBanner` renders on event, Reload button triggers correct action
- [ ] Task 12: Update OpenAPI spec and regenerate frontend types (`cd frontend && bun run gen`)
- [ ] Task 13: Documentation — update `docs/SETTINGS_DOCUMENTATION.md` with the change-detection behaviour

## 🧠 Implementation Notes (Copilot Context)

### Detection strategy

```
if SUPERVISOR_TOKEN present && Supervisor WS available:
    subscribe to supervisor/event topic
    filter events where type == "addon_config_changed" and slug == self
else:
    start fsnotify watcher on /data/options.json
    debounce 500 ms (same pattern as UpgradeService.watchForDevelopUpdates)
    compute SHA-256 of file content; only emit if hash changed
    fallback: also run a time.Ticker with configurable interval (default 60 s)
        to re-read file and compare hash (handles NFS/overlay FS where inotify may not fire)
```

### HA Repair issue vs persistent notification

- Prefer `POST /core/api/repairs/issues` (HA 2022.9+):
  ```json
  {
    "domain": "srat",
    "issue_id": "addon_config_changed",
    "severity": "warning",
    "translation_key": "addon_config_changed"
  }
  ```
- Detect availability: if the Supervisor `/core/api/repairs/issues` returns `404`, fall back to `persistent_notification.create` (already implemented in `HomeAssistantService`).
- Dismiss on reload: call `DELETE /core/api/repairs/issues/srat/addon_config_changed` or `persistent_notification.dismiss`.

### Frontend popup

- Add a new `useAddonConfigChangedBanner` hook that watches the SSE stream for `app_config_changed` events.
- Render a `MUI Alert severity="warning"` inside a persistent `Snackbar` at the top of the page layout (not a blocking modal).
- Include a **Reload config** button; on click call the reload endpoint and dismiss the banner.
- Use `AppConfigData.requires_restart` from the existing config endpoint as the initial state source (for page refreshes).

### File path constant

Define `AddonOptionsFilePath = "/data/options.json"` in `config/addon_options.go` or a new `config/constants.go`.

### Fallback interval config

Add `addonConfigPollInterval` (default `60s`) to `AppConfig` schema, exposed in the settings UI under "Advanced".

## 🔗 Code References & TODOs

- [ ] `backend/src/service/upgrade_service.go:178` — reference fsnotify + debounce pattern to reuse
- [ ] `backend/src/service/homeassistant_service.go:590` — `CreatePersistentNotification` to extend with Repair API
- [ ] `backend/src/dto/app_config.go:25` — `RequiresRestart bool` field (already exists, ensure it is set on external change)
- [ ] `backend/src/events/events.go` — add `AddonConfigChangedEvent` type and `EmitAddonConfigChanged` helper
- [ ] `backend/src/internal/appsetup/appsetup.go` — register `AddonConfigWatcherService` in fx
- [ ] `frontend/src/` — `TODO: AddonConfigChangedBanner` component + `useAddonConfigChangedBanner` hook
- [ ] `custom_components/srat/__init__.py` — optionally handle `app_config_changed` WS event to trigger `async_reload` without requiring HA Repairs
