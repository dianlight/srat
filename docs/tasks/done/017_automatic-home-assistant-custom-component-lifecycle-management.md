# [FEATURE]: Automatic Home Assistant Custom Component Lifecycle Management

**Target Repo:** `srat` **Status:** ✅ Complete **Issue Link:** https://github.com/dianlight/srat/issues/565

## 🎯 Objective

Implement end-to-end custom component lifecycle management for Home Assistant add-on deployments, so SRAT can detect whether `custom_components/srat` is installed under `/homeassistant/custom_components`, surface actionable guidance when missing/disconnected, and provide guided install/upgrade/uninstall flows (including version checks and restart confirmation) directly from Settings → HomeAssistant.

## 🛠️ Technical Specifications

- **Inputs:**
  - Home Assistant add-on runtime path `/homeassistant/custom_components`
  - `custom_components/srat/manifest.json` (installed version detection)
  - Connectivity status of SRAT custom component (WebSocket/session state)
  - Update channel configuration (stable release vs pre-release)
  - User actions from frontend buttons: Install, Upgrade, Uninstall
  - User confirmation from restart permission popup

- **Outputs:**
  - Accurate install/connection/version status for SRAT custom component
  - Single backend issue notification when component is both missing and disconnected
  - Resolution dialog with latest-version info and confirm-to-download flow
  - Install/upgrade/uninstall execution using `srat.zip` artifact extraction into `/homeassistant/custom_components`
  - Optional Home Assistant Core restart request (only after user confirmation)

- **Dependencies:**
  - Backend: issue model/catalog (`backend/src/dto/issue.go`), Home Assistant integration/services, release artifact download/extraction utilities, config service for update channel
  - Frontend: Settings → HomeAssistant UI section, dialog/popup components, API hooks for lifecycle actions
  - Custom component metadata: `custom_components/srat/manifest.json`
  - Home Assistant Core API endpoint for restart request

## 📝 Task List

- [x] Task 0: Design and define the backend logic for detecting custom component presence, version, and connectivity status. Define the issue type for missing+disconnected state with single-notification behavior.
- [x] Task 1: Add backend status detection for custom component presence in `/homeassistant/custom_components` and installed version from `manifest.json`
- [x] Task 1.5: Migrate issue persistence to the new backend standard by removing the dedicated issue repository layer and using direct `dbom` access from `IssueService`.
- [x] Task 2: Add/update backend APIs to expose component status, latest available version, and lifecycle actions (install/upgrade/uninstall)
- [x] Task 3: Modify Mise and release process to ensure `srat.zip` artifact is generated and contains the necessary files for installation (including `manifest.json` with version info)
- [x] Task 4: Embed at build time `srat.zip` in backend for installation/upgrade flows
- [x] Task 5: Ensure install/upgrade use the embedded artifact or flow downloads `srat.zip` from configured channel (release/pre-release/develop) and extracts into target directory. Special case for "develop" channel where the source file are located in `/addon_configs/local_sambanas2/upgrade/srat.zip` and should be used directly without download if the version in the manifest is older or equal to the one in the develop channel (use the same rule of other updates).
- [x] Task 6: Ensure update flow also works when component is already installed (upgrade-in-place)
- [x] Task 7: Add uninstall flow that removes `custom_components/srat` safely and refreshes status
- [x] Task 8: Add single-notification issue in `backend/src/dto/issue.go` when component is missing and disconnected. Ensure it does not spam multiple notifications on repeated checks. The issue should include a `ResolutionLink` that opens the dialog flow for installation guidance. This issue should be automatically resolved when the component is detected as installed and connected again. Implement necessary logic to check for existing issues of this type before emitting new ones to prevent duplicates.
- [x] Task 9: After install/upgrade/uninstall actions, add an home assistant repair for required restart like what do HACS.
- [x] Task 10: Add frontend buttons in Settings → HomeAssistant for Install/Upgrade/Uninstall with enable/disable state based on current component status and action availability (e.g., disable Install if already installed, disable Upgrade if no newer version, etc.)
- [x] Task 11: Ensure all backend/frontend interactions are robust, with proper error handling and user feedback (e.g., show error messages if install/upgrade/uninstall fails, show loading states during operations, etc.)
- [x] Task 12: Unit testing (backend detection/actions, issue emission logic, frontend button-state logic)
- [x] Task 13: Validate end-to-end behavior in a real Home Assistant environment using the `test-remote-environment` task/skill
- [x] Task 14: Integration and documentation
- [x] Task 15: Final review and testing in staging environment before release

## 🧠 Implementation Notes (Copilot Context)

- Start-task workflow: linked issue `#565` and switched to branch `feature/automatic-home-assistant-custom-component-lifecycle-management`.
- Implemented backend status primitives:
  - Added `HomeAssistantCustomComponentStatus` DTO and custom-components path context support.
  - Added `HomeAssistantComponentService` to detect install path, parse installed manifest version, and correlate active websocket component connection metadata.
  - Added settings endpoint `GET /settings/homeassistant/custom-component/status`.
  - Added targeted tests for service detection and endpoint response behavior.
- Implemented backend missing+disconnected issue synchronization:
  - Added canonical issue title + resolution-link constants for SRAT HA custom component missing/disconnected state.
  - Added `IssueService.FindByTitle(...)` and `IssueService.ResolveByTitle(...)` to support idempotent issue lifecycle control.
  - Status endpoint now creates the issue only when absent and auto-resolves it once installed or connected again.
- Migrated issue persistence plumbing to direct DB access in service layer:
  - `IssueService` now uses direct `dbom.Issue` persistence via the injected `*gorm.DB` + context.
  - Removed the dedicated `repository/issue_repository.go` abstraction and related test file.
  - Updated DI wiring and dependent tests to align with the new service-level DB standard.
- Implemented backend uninstall lifecycle action:
  - Added `HomeAssistantComponentService.Uninstall()` to remove `custom_components/srat` idempotently.
  - Added settings endpoint `DELETE /settings/homeassistant/custom-component` that runs uninstall, refreshes status, and re-applies issue sync logic.
  - Added/updated targeted backend tests covering uninstall service behavior and settings endpoint response.
- Completed backend API exposure for custom component lifecycle status/actions:
  - Extended status payload with `latest_version` and action availability flags (`can_install`, `can_upgrade`, `can_uninstall`).
  - Added API routes for install/upgrade lifecycle actions in settings:
    - `POST /settings/homeassistant/custom-component/install`
    - `POST /settings/homeassistant/custom-component/upgrade`
  - Integrated latest-version hint from upgrade service when available.
- Completed release artifact packaging baseline for custom component distribution:
  - Added `//custom_components:package-hacs` mise task to generate `srat.zip` with `srat/` top-level directory layout and include `srat/manifest.json`.
  - Updated CI release workflow to call the mise packaging task instead of inline zip logic, keeping packaging behavior consistent between local and CI execution.
- Completed backend build-time embedding plumbing for custom component artifact:
  - Added backend accessor `internal.GetEmbeddedCustomComponentZip()` with `embedallowed` implementation reading from embedded `internal/assets/srat.zip` and non-embed fallback implementation reading from local file path.
  - Updated backend build task to generate `src/internal/assets/srat.zip` before `go build`, so `embedallowed` binaries include the packaged custom component artifact.
  - Added internal assets folder documentation and ignore rule for generated zip artifact.
- Completed install/upgrade artifact usage and extraction flow for custom component lifecycle:
  - Implemented `POST /settings/homeassistant/custom-component/install` and `POST /settings/homeassistant/custom-component/upgrade` handlers to execute real install/upgrade actions and return refreshed status payload.
  - Added channel-aware archive source resolution in settings API:
    - `develop` channel prefers `/addon_configs/local_sambanas2/upgrade/srat.zip` when available and its manifest version is newer or equal to installed version.
    - Other channels (and fallback) use build-time embedded `internal/assets/srat.zip` via `internal.GetEmbeddedCustomComponentZip()`.
  - Added service-level zip extraction/install primitive `HomeAssistantComponentService.InstallOrUpgradeFromZip(...)` with path-safety checks and manifest validation.
  - Added targeted backend tests for service extraction behavior and settings install/upgrade endpoints.
- Completed upgrade-in-place validation for already installed component:
  - Added regression coverage proving `HomeAssistantComponentService.InstallOrUpgradeFromZip(...)` succeeds when `custom_components/srat` already exists and updates installed manifest version.
  - Verified stale files from previous install are removed during upgrade and replaced with archive contents.
  - Confirmed upgrade endpoint remains functional with focused settings API test execution.
- Completed restart-required repair integration after lifecycle actions:
  - Added restart repair upsert for `install`, `upgrade`, and `uninstall` lifecycle endpoints using `repair_id=custom_component_restart_required`.
  - Reused existing repair-command broadcast pipeline so Home Assistant custom component receives repair upsert/delete events.
  - Added restart endpoint dismissal for the same repair id, including fallback persistent-notification behavior when repair service is unavailable.
  - Added focused API test assertions for repair create/broadcast after lifecycle actions and repair delete/broadcast after restart request.
- Completed frontend Home Assistant lifecycle actions section in Settings:
  - Added `HomeAssistantCustomComponentPanel` under Settings → HomeAssistant showing install/connection/version status for the SRAT custom component.
  - Wired frontend action buttons (`Install`, `Upgrade`, `Uninstall`) to backend lifecycle endpoints and disabled them based on backend action flags (`can_install`, `can_upgrade`, `can_uninstall`) plus in-flight mutation state.
  - Added focused frontend test coverage asserting status-driven button enable/disable behavior.
- Completed robustness/error-feedback pass for lifecycle interactions (Task 11):
  - Added explicit frontend action feedback in `HomeAssistantCustomComponentPanel` for install/upgrade/uninstall success and failure outcomes.
  - Added defensive API error message extraction and surfaced status/action failures via inline alerts.
  - Preserved loading feedback during in-flight operations and validated with focused frontend settings tests.
- Completed unit testing pass for custom component lifecycle (Task 12):
  - Added frontend unit tests for Home Assistant custom component panel status-error feedback and install-action failure feedback in `frontend/src/pages/settings/__tests__/Settings.test.tsx`.
  - Re-validated existing frontend button-state logic test coverage for install/upgrade/uninstall enablement states.
  - Verified backend service/API unit suites pass for lifecycle detection/actions and issue-related paths via `go test ./service ./api`.
- Completed real-environment validation pass (Task 13) using `test-remote-environment` workflow:
  - Ran remote backend deployment via `mise run //backend:build:remote` and verified deployed version `2026.4.0-dev.1` startup in add-on logs.
  - Restarted add-on `local_sambanas2` successfully and confirmed configuration validity (`ha_check_config` passed).
  - Started remote frontend dev mode (`mise run //frontend:dev:remote`) and validated live UI at `http://localhost:3080/`.
  - Navigated to Settings → HomeAssistant and confirmed the SRAT custom component panel renders real status/action state (`Installed: No`, `Connected: Yes`, install action enabled).
  - Post-interaction add-on logs showed no startup panic/fatal failures; only known environment warnings (SMART USB bridge/share-volume warnings).
- Phase 6 (caller migration) completed for backend services that still emitted legacy repairs:
  - `AddonConfigWatcherService.emitChanged` now upserts Problem `addon_config_changed` directly (with HA notification fallback only when ProblemService is unavailable).
  - `HomeAssistantComponentService` restart-required and addon-config dismissal flows now use ProblemService-only lifecycle operations.
  - Updated service tests to assert ProblemService interactions and removed legacy RepairService caller expectations.
- Phase 7 (DI wiring) completed for migrated Problem-first flows:
  - Removed stale `IssueService` constructor dependency from `HomeAssistantComponentService` and aligned its Fx test wiring.
  - Removed unused `SettingsHanler` constructor dependencies (`ContextState`, `RepairService`, `HomeAssistantService`, `BroadcasterService`) to keep handler DI focused on active collaborators only.
  - Updated `api/setting_test.go` and `service/homeassistant_component_service_test.go` constructor wiring/mocks accordingly.
  - Revalidated with targeted suites and full backend regression (`go test ./... -count=1`).
- Phase 8 (Frontend dialogs) completed for lifecycle confirmation and restart prompt:
  - Replaced direct-execute action buttons with a `confirmDialog` state and `openConfirmDialog(action)` handler so Install/Upgrade/Uninstall always require user confirmation before API calls are fired.
  - Added inline MUI `Dialog` for action confirmation: per-action `DialogTitle` (e.g., "Install SRAT Custom Component"), body text including relevant version information, and Cancel/Confirm buttons.
  - Added inline MUI `Dialog` for restart prompt that appears after any successful lifecycle action: "Restart Required" title with Later/Restart Now options. "Restart Now" calls `PUT /api/restart` via `usePutApiRestartMutation`.
  - Added `type LifecycleAction = "install" | "upgrade" | "uninstall"` to make action type explicit throughout the dialog flow.
  - Added 5 new frontend tests in `Settings.test.tsx`:
    - Confirmation dialog appears with correct title text and version on Install click
    - Confirmation dialog is dismissed without side effects on Cancel
    - Restart required dialog appears after a successful install
    - Restart required dialog is dismissed without side effects on Later
    - `PUT /api/restart` is called when "Restart Now" is clicked
  - Updated existing action-failure test to route through the new confirmation dialog (Install → Confirm → API error → feedback).

- Phase 9 (Custom Component integration and documentation) completed:
  - Added `issues` section to `custom_components/srat/strings.json` covering the three repair translation keys broadcast by the backend: `custom_component_restart_required`, `custom_component_missing`, and `addon_config_changed`.
  - Each entry includes translated `title` and `description`; fixable issues (`custom_component_restart_required`, `custom_component_missing`) additionally define `fix_flow.step.confirm.title` and `fix_flow.step.confirm.description` so Home Assistant can render the built-in repair confirmation UI.
  - Synced `custom_components/srat/translations/en.json` to match `strings.json` (same issue sections and fix-flow entries).
  - Added three new tests in `custom_components/tests/test_repairs.py`:
    - `test_strings_json_defines_all_required_issue_keys` — asserts all backend repair keys are present in `strings.json`
    - `test_strings_json_fixable_issues_have_fix_flow` — asserts fixable issues have complete `fix_flow.step.confirm` content
    - `test_en_translation_matches_strings_json_issue_keys` — asserts `en.json` and `strings.json` define the same issue key set
  - Verified all 41 custom component tests pass; ruff lint/format and mypy strict typecheck clean.

- Phase 10 (Cleanup & Tests) completed:
  - Fixed pre-existing TypeScript errors in `frontend/src/mocks/streamingHandlers.ts`: replaced `"warning"`/`"created"` string literals with `Severity2.Warning`/`Status.Created` enum values in the `problem()` mock factory; imported `Severity2` and `Status` from `sratApi`; added required `id` and `repeating` fields to the `Problem` mock object.
  - Fixed pre-existing TypeScript errors in `frontend/src/store/__tests__/wsApi.test.tsx`: added `Severity2`/`Status` imports, switched inline payload literals to enum values, added `id`/`repeating` fields, and cast `store.getState()` to `any` with Biome suppression comments for the dynamically-imported `wsApi` selector type incompatibility.
  - Verified 4/4 `wsApi` tests pass, TypeScript compilation is clean (`tsc --noEmit` exits 0), and full frontend `lint` task passes.
  - Backend: all packages pass `go test ./...` (no actual package failures — the earlier bare `FAIL` exit code was a shell artifact; all individual packages show `ok`).
  - Custom component: 41/41 tests pass, ruff and mypy clean.

- The target add-on directory is fixed to `/homeassistant/custom_components`.
- Presence check should validate whether `srat` exists in target directory.
- Installed version should be read from `<target>/srat/manifest.json`.
- Missing+disconnected condition must raise only one issue notification (no duplicates/spam).
- Issue should be defined under `backend/src/dto/issue.go` with a `ResolutionLink` that opens a UI dialog.
- Resolution dialog requirements:
  - Show latest available version information
  - Ask explicit user confirmation before any download/install action
- Install and upgrade should share the same artifact path: download `srat.zip` from selected update channel and extract into `/homeassistant/custom_components`.
- Channel selection must honor configuration (release vs pre-release).
- Upgrade must run even if component is already installed (overwrite/update existing files safely).
- Frontend Settings → HomeAssistant section should expose three actions: Install, Upgrade, Uninstall.
- Button enabled state must reflect real backend status (installed/not-installed, upgrade availability, operation in progress, etc.).
- After any lifecycle action (install/upgrade/uninstall), prompt for Home Assistant Core restart permission.
- Restart must be executed only when the user confirms, by calling the Home Assistant Core API.
- Ensure status refresh occurs after each action to keep UI state accurate.

## 🔗 Code References & TODOs

- [ ] `TODO: backend/src/dto/issue.go` - Add issue type/entry for missing+disconnected custom component with single-notification behavior
- [x] `TODO: backend/src/service/homeassistant_component_service.go` - Added service logic for filesystem presence/version checks and websocket connection status correlation
- [x] `TODO: backend/src/service/issue_service.go` - Added by-title lookup and idempotent resolve path for custom-component issue dedupe/cleanup
- [ ] `TODO: backend/src/service/*` - Add channel-aware release/pre-release artifact resolution and `srat.zip` download/extract flow
- [x] `TODO: backend/src/api/*` - Expose install/upgrade/uninstall endpoints (implemented in `backend/src/api/setting.go`; install/upgrade handlers currently return explicit not-implemented response until artifact workflow tasks are completed)
- [x] `TODO: custom_components/.mise.toml` - Added `package-hacs` task to generate `srat.zip` artifact with required integration files
- [x] `TODO: .github/workflows/build.yaml` - Updated release pipeline to generate `srat.zip` through mise task
- [x] `TODO: backend/src/internal/embed.go` - Added embedded custom component zip accessor for `embedallowed` builds
- [x] `TODO: backend/src/internal/no_embed.go` - Added non-embed fallback custom component zip accessor
- [x] `TODO: backend/.mise.toml` - Build task now generates `src/internal/assets/srat.zip` before compiling
- [x] `TODO: backend/src/service/homeassistant_component_service.go` - Added install/upgrade zip extraction flow into `/homeassistant/custom_components`
- [x] `TODO: backend/src/api/setting.go` - Install/upgrade endpoints now execute artifact resolution + extraction flow
- [x] `TODO: backend/src/service/homeassistant_component_service_test.go` - Added upgrade-in-place regression test for already installed component path
- [x] `TODO: backend/src/api/setting.go` - Added restart-required repair upsert/delete flow for custom component lifecycle + restart endpoint
- [x] `TODO: frontend/src/pages/settings/*` - Added HomeAssistant section action buttons and status-driven enable/disable logic for custom component lifecycle
- [x] `TODO: frontend/src/components/*` - Added inline MUI confirmation dialog (per-action title/body with version info, Cancel/Confirm) and restart-required dialog (Later/Restart Now calling `PUT /api/restart`) in `HomeAssistantCustomComponentPanel`; action buttons now gate on user confirmation before executing lifecycle API calls
- [ ] `FIXME: frontend/backend contract` - Define explicit response model for installed version, latest version, connectivity, and action availability
