# [FIX]: Disable SMART Integration Setting

**Target Repo:** `srat`  **Status:** 📅 Planned  **Issue Link:** [hassio-addons#596](https://github.com/dianlight/hassio-addons/issues/596)

## 🎯 Objective

Mitigate the excessive disk wake-up behavior reported in `hassio-addons#596` by adding an SRAT setting that disables SMART integration entirely when enabled. The new setting must default to `false` so existing installations keep current behavior unless the user explicitly opts out. When the setting is turned on, SRAT should stop performing background SMART-driven polling and avoid SMART integration paths that can wake sleeping disks, providing a pragmatic mitigation that can be implemented independently from the larger HDIdle work tracked in task `003`.

> _Context for Copilot: `DiskStatsService` currently calls SMART operations during periodic health collection, and task `003` already tracks the broader investigation into excessive SMART polling. This task extracts a smaller, user-controlled mitigation: a boolean field added to the `Settings` DTO (alongside existing bool settings such as `HDIdleEnabled`) that gates SMART integration at runtime._

## 🛠️ Technical Specifications

- **Inputs:**
  - `backend/src/dto/settings.go` — `Settings` DTO where the new field must be added (alongside existing booleans such as `HDIdleEnabled`, `SMBoverQUIC`)
  - Existing settings API at `/api/settings` and frontend form in `Settings.tsx`
  - Existing SMART polling in `backend/src/service/disk_stats_service.go`
  - Existing SMART API/UI surfaces in backend and frontend
- **Outputs:**
  - New boolean `Settings` field to disable SMART integration, defaulting to `false`
  - Backend behavior that skips SMART polling/integration work when the setting is enabled
  - Settings UI renders the new toggle (field case in `Settings.tsx` or schema-driven fallback, following the HDIdle pattern)
  - Tests covering default value handling, persistence, and SMART polling bypass behavior
- **Dependencies:**
  - `docs/tasks/003_hdidle-service-completion.md` — related mitigation context; this task can proceed independently as a focused fix
  - `backend/src/dto/settings.go` — `Settings` DTO; add the new bool field here with `default:"false"`
  - `backend/src/service/setting_service.go` — settings persistence layer
  - `backend/src/service/setting_service_test.go` — test helper for save/load field assertion
  - `backend/src/service/disk_stats_service.go` — current SMART polling and cache lookup path
  - `backend/src/service/smart_service.go` — core SMART operations that may need runtime gating or explicit no-op behavior
  - `frontend/src/pages/settings/Settings.tsx` — settings form that renders individual fields; handles `hdidle_enabled` as a model to follow
  - `frontend/src/pages/settings/__tests__/Settings.test.tsx` — settings panel regression coverage
  - `frontend/src/mocks/handlers.ts` — MSW mock responses; must include the new field
  - `docs/SMART_SERVICE.md` — SMART behavior documentation

## 📝 Task List

- [ ] Task 1: Add a new boolean field `DisableSmart` (JSON: `disable_smart`, tag `default:"false"`) to `backend/src/dto/settings.go` `Settings` struct, following the same pattern as `HDIdleEnabled`
- [ ] Task 2: Add a `case 'disable_smart':` block in `frontend/src/pages/settings/Settings.tsx` and update the MSW mock in `frontend/src/mocks/handlers.ts` to include the new field
- [ ] Task 3: Update `DiskStatsService` to bypass SMART polling, SMART cache refresh, and related background SMART collection when the disable setting is enabled
- [ ] Task 4: Review direct SMART API/UI behavior when SMART integration is disabled and define the minimal safe behavior (for example hide controls, skip queries, or return a clear disabled state) without breaking unrelated disk data
- [ ] Task 5: Add backend tests for default config behavior, schema exposure, and SMART polling bypass when the setting is enabled
- [ ] Task 6: Add frontend/settings regression coverage in `Settings.test.tsx` proving the new toggle renders and preserves the default `false` behavior
- [ ] Task 7: Documentation — update task `003` references and `docs/SMART_SERVICE.md` to describe the new mitigation setting and its intended use for sleeping-disk scenarios

## 🧠 Implementation Notes (Copilot Context)

### Why this task exists separately from task `003`

Task `003` contains a broad follow-up item about investigating SMART polling intervals and idle thresholds. This task is narrower and lower risk: introduce a user-controlled kill switch for SMART integration so users affected by `hassio-addons#596` can stop SMART-related wake-ups immediately, even before a more sophisticated polling strategy is implemented.

### Settings pipeline observations

The new field belongs in the `Settings` DTO, not the addon app-config pipeline. `dto.Settings` is the single source of truth for user-configurable SRAT settings persisted via `SettingService` and exposed at `/api/settings`.

Relevant flow:

- DTO field declaration in `backend/src/dto/settings.go`
- persistence and retrieval in `backend/src/service/setting_service.go`
- settings form rendering in `frontend/src/pages/settings/Settings.tsx` (see the `hdidle_enabled` case for the switch/toggle pattern to follow)

### Behavioral requirement

The new option must default to `false`.

That means:

- existing installs keep SMART integration enabled unless the user changes the setting
- enabling the new option should disable SRAT-side SMART integration work
- the mitigation should focus first on stopping background SMART polling that can wake disks

### Polling hotspot

`DiskStatsService` currently performs SMART checks in the periodic stats loop and in per-disk enrichment methods:

- `updateDiskStats()`
- `populatePerDiskInfo()`
- `isSmartEnabled()` cache-miss path

Those are the first places to gate on the new setting.

### Scope caution

The new field is added exclusively to `dto.Settings` (the user-facing settings DTO), not to `addon_json_config.go` or the addon supervisor config pipeline. Do not confuse it with the existing `enable_smart` field that lives in the addon config model — that field controls a different configuration layer.

If the exact field name is unclear during implementation, keep the task focused on the desired behavior: **a boolean field in `dto.Settings` that defaults to `false` and disables SRAT-side SMART integration work when `true`**.

## 🔗 Code References & TODOs

- [ ] `docs/tasks/003_hdidle-service-completion.md:42` — existing broad SMART mitigation note tied to `hassio-addons#596`
- [ ] `backend/src/dto/settings.go` — add `DisableSmart bool json:"disable_smart,omitempty" default:"false"` to `Settings` struct
- [ ] `backend/src/service/setting_service.go` — verify field is persisted/loaded correctly (should require no changes if DTO struct tags are correct)
- [ ] `backend/src/service/setting_service_test.go` — add save/load test for `DisableSmart` following the `testFieldUpdateAndLoad` helper pattern
- [ ] `backend/src/service/disk_stats_service.go:169` — periodic disk health update path
- [ ] `backend/src/service/disk_stats_service.go:334` — per-disk SMART enrichment path
- [ ] `backend/src/service/disk_stats_service.go:465` — cache-miss SMART query path
- [ ] `backend/src/service/smart_service.go` — direct SMART operations to review for disabled-mode behavior
- [ ] `frontend/src/pages/settings/Settings.tsx` — add `case 'disable_smart':` block following the `hdidle_enabled` switch/toggle pattern
- [ ] `frontend/src/mocks/handlers.ts` — add `disable_smart: false` to MSW mock GET `/api/settings` responses
- [ ] `frontend/src/pages/settings/__tests__/Settings.test.tsx` — add coverage for the new toggle field
- [ ] `docs/SMART_SERVICE.md` — document how and when to use the mitigation setting
- [ ] [hassio-addons#596](https://github.com/dianlight/hassio-addons/issues/596) — SMART activity keeps disks awake; disabling SMART is the current workaround
