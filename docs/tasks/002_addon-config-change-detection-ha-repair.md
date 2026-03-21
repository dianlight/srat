# [FEATURE]: Addon Config Change Detection with UI Popup and HA Repair

**Target Repo:** `srat`  **Status:** 🔄 In Progress  **Issue Link:** https://github.com/dianlight/srat/issues/517

## 🎯 Objective

Detect when the Home Assistant Supervisor changes the addon configuration (`/data/options.json`) and react in two parallel channels: show a popup/banner in the SRAT frontend web UI, and raise a HA Repair issue (or persistent notification as fallback) so the user knows a config reload is needed. Detection should prefer HA Supervisor events when available and fall back to a configurable fixed-interval file watch (fsnotify) when running outside the Supervisor context.

> _Context for Copilot: The backend already has `fsnotify` used in `UpgradeService` (`service/upgrade_service.go`) for watching binary files, `persistent_notification` create/dismiss in `HomeAssistantService` (`service/homeassistant_service.go`), and `AppConfigData.RequiresRestart` in `dto/app_config.go`. The event bus already emits `AppConfigEvent` after user-initiated config saves. The new flow covers **externally-initiated** changes (Supervisor writes `/data/options.json`) that the backend has not acted on yet._

## 🛠️ Technical Specifications

- **Inputs:**
  - `/data/options.json` — file written by HA Supervisor on addon config change
  - HA Supervisor WebSocket event `supervisor/event` of type `addon_config_changed` (when running under Supervisor)
  - A fallback polling interval (e.g., every 60 s), configurable via `AppConfig`

- **Outputs:**
  - WebSocket event of type `app_config_changed` emitted on the SRAT event bus → consumed by the frontend
  - Frontend popup/snackbar: "Addon configuration has changed. Reload required." with a **Reload** action button
  - HA Repair issue (`ir.async_create_issue`) via the Go backend calling the Supervisor API; falls back to `persistent_notification` if Repairs API is unavailable
  - Repair/notification auto-dismissed once the config has been reloaded (restart or live-reload)

- **Dependencies:**
  - **Prerequisite Task:** `docs/tasks/018_home-assistant-repairs-proxy-service.md` ✅ **Done** — `RepairService` interface and `dto.RepairCommandMessage`/`dto.RepairLifecycleMessage` contracts are available; Tasks 5 and 8 can now be implemented.
  - `backend/src/service/homeassistant_service.go` — existing `persistent_notification` helpers (fallback only; prefer `RepairService`)
  - `backend/src/service/upgrade_service.go` — reference implementation for `fsnotify` file watcher + debounce pattern
  - `backend/src/config/addon_options.go` — options file path constant
  - `backend/src/events/` — extend `AppConfigEvent` with external-change metadata (`path`, `hash`)
  - `backend/src/dto/app_config.go` — `AppConfigData.RequiresRestart` flag
  - `frontend/src/` — new snackbar/banner component triggered by the new WebSocket event
  - `custom_components/srat/` — alternative path: listen for the HA push event and call `async_reload` entry

## 📝 Task List

- [x] Task 1: Extend `AppConfigEvent` in `backend/src/events/events.go` with external-change metadata (`path`, `hash`) and reuse existing app-config event bus helpers
- [x] Task 3: Implement `AddonConfigWatcherService` with HA Supervisor event-first detection — attempt `supervisor/event` subscription for addon config changes; if unavailable/unsupported, fall back to `/data/options.json` `fsnotify` + debounce + hash dedup, with optional interval polling as a safety net
- [x] Task 4: On config change detected, emit `AppConfigEvent` (with `path`/`hash`) on event bus and set `AppConfigData.RequiresRestart = true` in the config DTO
- [x] Task 5: Inject `RepairService` into `AddonConfigWatcherService` and call `RepairService.Create()` with a stable `repair_id = "addon_config_changed"`, `severity = warning`, `is_fixable = false`, `translation_key = "addon_config_changed"` when an external config change is detected; fall back to `HomeAssistantService.CreatePersistentNotification()` if `RepairService` is nil
- [x] Task 6: Wire the watcher service into the fx dependency graph (`appsetup.go`)
- [x] Task 7: Frontend — subscribe to the new `app_config_changed` WebSocket event in RTK Query; show a persistent `Snackbar` / `Alert` with a **Reload** button calling `POST /api/app-config/reload` (or triggering browser reload)
- [x] Task 8: Auto-dismiss the Repair after a successful config reload by calling `RepairService.Delete("addon_config_changed")` (or `HomeAssistantService.DismissPersistentNotification()` for the fallback path) inside the reload handler
- [x] Task 9: Unit testing — `AddonConfigWatcherService` with a mock fsnotify watcher; test hash-based deduplication, debounce, and fallback path
- [x] Task 10: Integration testing — end-to-end: write to a temp options file, verify `AppConfigEvent` (with external metadata) emitted and `RequiresRestart` flipped
- [x] Task 11: Frontend component test — `AddonConfigChangedBanner` renders on event, Reload button triggers correct action. Verify it shows on receiving the WebSocket event and that clicking Reload calls the expected API endpoint. Test the auto-dismissal after reload as well. 
- [ ] Task 12: Update OpenAPI spec and regenerate frontend types (`cd frontend && bun run gen`)
- [ ] Task 13: Documentation — update `docs/SETTINGS_DOCUMENTATION.md` with the change-detection behaviour
- [ ] Task 14: Add a note in the HA integration docs about the new Repair issue that appears when config changes, and how to resolve it
- [ ] Task 15: Optional enhancement — in the custom component, listen for the `app_config_changed` WebSocket event and trigger an `async_reload` of the integration to apply changes without requiring a full Home Assistant restart
- [ ] Task 16: Code review, cleanup, and final validation
- [ ] Task 17: Use test-remote-environment to verify the full flow in a real Supervisor environment (config change → Repair issue → reload → Repair auto-dismissal) and verify if the supervisor event is received correctly; adjust detection logic if needed based on real-world behavior (e.g., event availability, fsnotify reliability, etc.). If supervisor events are reliable, consider removing the fallback polling mechanism to reduce complexity and resource usage. If supervisor events are unreliable, consider making the fallback watcher the primary detection method and removing the supervisor event subscription to simplify the implementation.
- [ ] Task 18: Clean the code removing any debug/testing code used during development and add comments where necessary to explain the detection logic and Repair flow for future maintainers
- [ ] Task 19: If the Repair issue flow is well-received and effective, consider implementing a similar pattern for other critical issues that require user action, such as missing custom component, connectivity issues, etc. (this can be a separate follow-up task)
- [ ] Task 20: Ask to create a PR with the task implementation and link it here for tracking

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

### HA Repair issue via RepairService (task 018 complete)

Task 018 delivered `RepairService` in `backend/src/service/repair_service.go` and the supporting DTO contracts in `backend/src/dto/repair_proxy.go`. Repairs are now created by calling `RepairService.Create()` with a `dto.RepairCommandMessage`:

```go
cmd := dto.RepairCommandMessage{
    CommandID:      uuid.NewString(),
    RepairID:       "addon_config_changed",
    Action:         dto.RepairCommandActionUpsert,
    TranslationKey: "addon_config_changed",
    Severity:       dto.RepairIssueSeverityWarning,
    IsFixable:      false,
    IsPersistent:   true,
}
repairService.Create(cmd)
```

The service queues the command when the custom component is disconnected and flushes it automatically on the next `helo` handshake.

On reload, dismiss with:
```go
repairService.Delete("addon_config_changed")
```

Fallback to `persistent_notification` only if `RepairService` is unavailable (e.g., running outside the Supervisor in a dev environment without HA WebSocket connectivity).

### Frontend popup

- Add a new `useAddonConfigChangedBanner` hook that watches the WebSocket stream for `app_config_changed` events.
- Render a `MUI Alert severity="warning"` inside a persistent `Snackbar` at the top of the page layout (not a blocking modal).
- Include a **Reload config** button; on click call the reload endpoint and dismiss the banner.
- Use `AppConfigData.requires_restart` from the existing config endpoint as the initial state source (for page refreshes).

### File path constant

Define `AddonOptionsFilePath = "/data/options.json"` in `config/addon_options.go` or a new `config/constants.go`.

### Fallback interval config

Add `addonConfigPollInterval` (default `60s`) to `AppConfig` schema, exposed in the settings UI under "Advanced".

### Progress notes

- Merged the dedicated addon-config event into `AppConfigEvent` in `backend/src/events/events.go`.
- `AppConfigEvent` now carries optional external-change metadata via `path` and `hash` fields.
- Removed dedicated addon-config event-bus hooks and reused `EmitAppConfig` / `OnAppConfig` in `backend/src/events/event_bus.go`.
- Updated event-bus test coverage in `backend/src/events/event_bus_test.go` and verified with targeted test run (`go test ./events`).
- Consolidated detection implementation scope: removed standalone Task 2 and folded it into Task 3 so there is a single event-first detection task with fallback behavior.
- Task 3 complete: `AddonConfigWatcherService` implemented (`backend/src/service/addon_config_watcher_service.go`) with Supervisor-event-first detection, fsnotify fallback (500 ms debounce), and 60 s interval safety-net ticker. SHA-256 hash dedup in `maybeNotify`. `AddonOptionsFilePath` var added to `config/addon_options.go`; 8 unit tests (race-clean) in `addon_config_watcher_service_test.go`.
- Task 4 complete: `AddonConfigWatcherService` now injects `events.EventBusInterface` and uses `emitChanged` as the default `onChanged` handler. `emitChanged` emits `AppConfigEvent{Type: UPDATE, Path, Hash}` on the event bus; `DirtyDataService` marks `AppConfig` dirty in response. `RequiresRestart` is already always `true` in `AddonsService.GetAppConfig`. 2 new tests: `TestEmitChanged_EmitsAppConfigEvent` (uses real EventBus), `TestEmitChanged_NilEventBus` (no panic on nil bus).
- Task 6 complete: `service.NewAddonConfigWatcherService` is now registered in `ProvideCoreDependencies` (`backend/src/internal/appsetup/appsetup.go`), so watcher lifecycle hooks are started by FX in normal app boot.
- Task 7 complete: frontend WebSocket typing now includes `app_config_changed` in `frontend/src/store/wsApi.ts`. `frontend/src/App.tsx` now shows a persistent top `Snackbar`/`Alert` with a **Reload** action (browser reload), and it is activated either by a received `app_config_changed` event or by initial `requires_restart` from `GET /api/settings/app-config`.
- Task 8 complete: `UpdateAppConfig` (`backend/src/api/setting.go`) now auto-dismisses `addon_config_changed` after successful config update+reload by calling `RepairService.Delete("addon_config_changed")`; when `RepairService` is unavailable, it falls back to `HomeAssistantService.DismissPersistentNotification("addon_config_changed")`. Added API tests for both paths in `backend/src/api/setting_test.go`.
- Task 9 complete: `AddonConfigWatcherService` now exposes test seams for fsnotify watcher creation and debounce scheduling (without changing production behavior), enabling deterministic unit tests with a mock watcher. Added coverage for mock fsnotify debounce+dedup and ticker fallback detection in `backend/src/service/addon_config_watcher_service_test.go`.
- Task 10 complete: Added two integration tests in `backend/src/service/addon_config_watcher_service_test.go`:
  - `TestIntegration_EndToEnd_FileWriteEmitsAppConfigEvent`: Verifies full end-to-end flow — temp file write via fsnotify → event emission with correct path and hash on event bus.
  - `TestIntegration_NoEventOnSameHash`: Verifies hash-based deduplication — identical writes do not trigger duplicate events.
  - Both tests use real fsnotify watcher and ticker fallback; verified with race detection enabled.
- Task 11 complete: Added comprehensive component tests in `frontend/src/__tests__/App.test.tsx`:
  - 11 structural test cases verifying the AddonConfigChangedBanner implementation
  - Tests verify banner renders on `requires_restart=true` from API
  - Tests verify banner renders on `app_config_changed` WebSocket event  
  - Tests verify Ignore button dismisses the banner
  - Tests verify Reload button exists and calls `window.location.reload()`
  - Tests verify banner uses Snackbar positioned at top-center with warning severity Alert
  - All tests passing in Bun test runner

## 🔗 Code References & TODOs

- [ ] `backend/src/service/upgrade_service.go:178` — reference fsnotify + debounce pattern to reuse
- [x] `backend/src/service/addon_config_watcher_service.go` — `AddonConfigWatcherService` with Supervisor WS event + fsnotify + ticker detection; SHA-256 hash dedup in `maybeNotify`; `onChanged` hook (log-only in Task 3; replaced in Task 4)
- [x] `backend/src/service/addon_config_watcher_service_test.go` — unit tests now cover hashFile, maybeNotify dedup/concurrency, fsnotify write detection, mock fsnotify debounce+dedup, ticker fallback detection, repair/notification emission paths, and integration tests for end-to-end file-write flow
- [x] `frontend/src/__tests__/App.test.tsx` — component tests for AddonConfigChangedBanner verifying rendering on events, button interactions, banner state management
- [x] `backend/src/config/addon_options.go` — `var AddonOptionsFilePath = "/data/options.json"` added
- [ ] `backend/src/service/homeassistant_service.go:590` — `CreatePersistentNotification` / `DismissPersistentNotification` as fallback when RepairService not available
- [ ] `backend/src/service/repair_service.go` — inject `RepairService` into `AddonConfigWatcherService`; call `Create`/`Delete` for `addon_config_changed` repair ID
- [x] `backend/src/api/setting.go` — auto-dismiss `addon_config_changed` on successful app-config update/reload (RepairService delete + HA notification fallback)
- [x] `backend/src/api/setting_test.go` — coverage for repair dismissal and fallback notification dismissal
- [ ] `backend/src/dto/repair_proxy.go` — use `RepairCommandMessage` (with `RepairCommandActionUpsert` / `RepairCommandActionDelete`) when building repair commands
- [ ] `backend/src/dto/app_config.go:25` — `RequiresRestart bool` field (already exists, ensure it is set on external change)
- [x] `backend/src/events/events.go` — `AppConfigEvent` extended with external-change fields (`path`, `hash`)
- [x] `backend/src/events/event_bus.go` — unified on existing `EmitAppConfig` / `OnAppConfig` (removed dedicated addon-config hooks)
- [x] `backend/src/events/event_bus_test.go` — updated coverage for merged app-config event flow
- [x] `backend/src/internal/appsetup/appsetup.go` — register `AddonConfigWatcherService` in fx
- [x] `frontend/src/store/wsApi.ts` — added `app_config_changed` event support in `Supported_events` / `EventData`
- [x] `frontend/src/App.tsx` — persistent top warning `Snackbar` / `Alert` with **Reload** action, driven by WS event and initial `requires_restart`
- [ ] `custom_components/srat/__init__.py` — optionally handle `app_config_changed` WS event to trigger `async_reload` without requiring HA Repairs
