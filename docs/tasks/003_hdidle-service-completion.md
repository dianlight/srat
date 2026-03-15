# [FEATURE]: HDIdle Service Completion

**Target Repo:** `srat`  **Status:** 📅 Planned  **Issue Link:** _TBD_

## 🎯 Objective

Complete the HDIdle integration end-to-end: restore the missing backend API endpoints (`/hdidle/status`, `/hdidle/effective-config`, `DELETE /disk/{disk_id}/hdidle/config`), implement the required service methods, and un-hide the HDIdle UI sections in the Settings page and the Volume partition action menu that are currently gated behind `// TODO: Enable when HDIdle feature is ready` comments.

> _Context for Copilot: Handler implementations already exist in `api/hdidle_handler.go` but are not registered. `HDIdleServiceInterface` is missing `GetStatus` and `GetEffectiveConfig`. The frontend disables HDIdle UI with TODO guards in `Settings.tsx:112` and `PartitionActionItems.ts:143`._

## 🛠️ Technical Specifications

- **Inputs:**
  - Per-disk HDIdle config (existing, stored in DB)
  - System hd-idle process status (via `hdidle_service.go`)

- **Outputs:**
  - `GET /hdidle/status` → `HDIdleStatus` struct
  - `GET /hdidle/effective-config` → `HDIdleEffectiveConfig` struct (merged global + per-disk overrides)
  - `DELETE /disk/{disk_id}/hdidle/config` → removes per-disk override, reverts to global config
  - Frontend: HDIdle settings panel enabled; per-partition action menu item enabled

- **Dependencies:**
  - `backend/src/api/hdidle_handler.go` — handler methods exist, need registration
  - `backend/src/service/hdidle_service.go` — `HDIdleServiceInterface` needs two new methods
  - `frontend/src/pages/settings/Settings.tsx` — remove TODO guard (line 112)
  - `frontend/src/pages/volumes/components/PartitionActionItems.ts` — remove TODO guard (line 143)

## 📝 Task List

- [ ] Task 1: Add `GetStatus() (*HDIdleStatus, errors.E)` and `GetEffectiveConfig() HDIdleEffectiveConfig` to `HDIdleServiceInterface`
- [ ] Task 2: Implement `GetStatus` in `HDIdleService` (query running hd-idle process state)
- [ ] Task 3: Implement `GetEffectiveConfig` in `HDIdleService` (merge global + per-disk DB config)
- [ ] Task 4: Register the three route handlers (`getServiceStatus`, `getEffectiveConfig`, `deleteConfig`) in `RegisterHDIdleHandler`
- [ ] Task 5: Remove `// TODO: Enable when HDIdle feature is ready` guard in `Settings.tsx:112`
- [ ] Task 6: Remove `// TODO: not ready to be enabled` guard in `PartitionActionItems.ts:143`
- [ ] Task 7: Unit tests — `GetStatus` and `GetEffectiveConfig` service methods incl. edge cases (service not running, no per-disk overrides)
- [ ] Task 8: API handler tests — `GET /hdidle/status`, `GET /hdidle/effective-config`, `DELETE /disk/{disk_id}/hdidle/config`
- [ ] Task 9: Frontend component test — HDIdle settings panel renders and partition action item is visible
- [ ] Task 10: Update OpenAPI spec and regenerate frontend types (`cd frontend && bun run gen`)
- [ ] Task 11: Documentation — update `docs/HDIDLE_SERVICE.md` with the new endpoints

## 🧠 Implementation Notes (Copilot Context)

### Interface additions

```go
// Add to HDIdleServiceInterface in service/hdidle_service.go
GetStatus() (*HDIdleStatus, errors.E)
GetEffectiveConfig() HDIdleEffectiveConfig
```

### GetStatus implementation hints

- Check if the `hd-idle` process is running using `os.FindProcess` or parsing `/proc`
- Return idle counters per disk if the hd-idle daemon exposes them (via log or procfs)
- If hd-idle is not running, return a status struct with `Running: false`

### GetEffectiveConfig implementation hints

- Load global config from `AppConfig`
- Load per-disk overrides from the DB (existing `getConfig` logic)
- Merge: per-disk value wins; fall back to global; fall back to compiled-in default
- Return as `map[DiskID]HDIdleEffectiveConfig` or a flat struct, matching the existing DTO pattern

### Route registration

```go
// In RegisterHDIdleHandler
huma.Register(api, huma.Operation{...}, handler.getServiceStatus)
huma.Register(api, huma.Operation{...}, handler.getEffectiveConfig)
huma.Register(api, huma.Operation{...}, handler.deleteConfig)
```

### Frontend un-gating

Both guards are simple boolean conditions; set them to `true` (or remove the condition) once the backend endpoints are confirmed working.

## 🔗 Code References & TODOs

- [ ] `backend/src/api/hdidle_handler.go` — `getServiceStatus`, `getEffectiveConfig`, `deleteConfig` exist but unregistered
- [ ] `backend/src/service/hdidle_service.go` — `HDIdleServiceInterface` missing `GetStatus`, `GetEffectiveConfig`
- [ ] `frontend/src/pages/settings/Settings.tsx:112` — `// TODO: Enable when HDIdle feature is ready`
- [ ] `frontend/src/pages/volumes/components/PartitionActionItems.ts:143` — `// TODO: not ready to be enabled`
- [ ] `docs/FUTURE_IMPROVEMENTS.md` — "HDIdle Service: Missing Global-Status and Delete-Config Endpoints" section (remove once done)
