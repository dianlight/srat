# [FEATURE]: Automatic Home Assistant Custom Component Lifecycle Management

**Target Repo:** `srat` **Status:** 📅 Planned **Issue Link:** _TBD_

## 🎯 Objective

Implement end-to-end custom component lifecycle management for Home Assistant add-on deployments, so SRAT can detect whether `custom_components/srat` is installed under `/config/custom_components`, surface actionable guidance when missing/disconnected, and provide guided install/upgrade/uninstall flows (including version checks and restart confirmation) directly from Settings → HomeAssistant.

## 🛠️ Technical Specifications

- **Inputs:**
  - Home Assistant add-on runtime path `/config/custom_components`
  - `custom_components/srat/manifest.json` (installed version detection)
  - Connectivity status of SRAT custom component (WebSocket/session state)
  - Update channel configuration (stable release vs pre-release)
  - User actions from frontend buttons: Install, Upgrade, Uninstall
  - User confirmation from restart permission popup

- **Outputs:**
  - Accurate install/connection/version status for SRAT custom component
  - Single backend issue notification when component is both missing and disconnected
  - Resolution dialog with latest-version info and confirm-to-download flow
  - Install/upgrade/uninstall execution using `srat.zip` artifact extraction into `/config/custom_components`
  - Optional Home Assistant Core restart request (only after user confirmation)

- **Dependencies:**
  - Backend: issue model/catalog (`backend/src/dto/issue.go`), Home Assistant integration/services, release artifact download/extraction utilities, config service for update channel
  - Frontend: Settings → HomeAssistant UI section, dialog/popup components, API hooks for lifecycle actions
  - Custom component metadata: `custom_components/srat/manifest.json`
  - Home Assistant Core API endpoint for restart request

## 📝 Task List

- [ ] Task 1: Add backend status detection for custom component presence in `/config/custom_components` and installed version from `manifest.json`
- [ ] Task 2: Add/update backend APIs to expose component status, latest available version, and lifecycle actions (install/upgrade/uninstall)
- [ ] Task 3: Modify Makefile and release process to ensure `srat.zip` artifact is generated and contains the necessary files for installation (including `manifest.json` with version info)
- [ ] Task 4: Embed at build time `srat.zip` for installation/upgrade flows
- [ ] Task 5: Ensure install/upgrade use the embedded artifact or flow downloads `srat.zip` from configured channel (release/pre-release/develop) and extracts into target directory. Special case for "develop" channel where the source file are located in `/addon_configs/local_sambanas2/upgrade/srat.zip` and should be used directly without download if the version in the manifest is older or equal to the one in the develop channel (use the same rule of other updates).
- [ ] Task 6: Ensure update flow also works when component is already installed (upgrade-in-place)
- [ ] Task 7: Add uninstall flow that removes `custom_components/srat` safely and refreshes status
- [ ] Task 8: Add single-notification issue in `backend/src/dto/issue.go` when component is missing and disconnected. Ensure it does not spam multiple notifications on repeated checks. The issue should include a `ResolutionLink` that opens the dialog flow for installation guidance. This issue should be automatically resolved when the component is detected as installed and connected again. Implement necessary logic to check for existing issues of this type before emitting new ones to prevent duplicates.
- [ ] Task 9: After install/upgrade/uninstall actions, trigger home assistant core restart confirmation popup in frontend, and only call the restart API if user confirms. Ensure that the status is refreshed after restart to reflect any changes.
- [ ] Task 10: Add frontend buttons in Settings → HomeAssistant for Install/Upgrade/Uninstall with enable/disable state based on current component status and action availability (e.g., disable Install if already installed, disable Upgrade if no newer version, etc.)
- [ ] Task 11: Ensure all backend/frontend interactions are robust, with proper error handling and user feedback (e.g., show error messages if install/upgrade/uninstall fails, show loading states during operations, etc.)
- [ ] Task 12: Unit testing (backend detection/actions, issue emission logic, frontend button-state logic)
- [ ] Task 13: Integration and documentation

## 🧠 Implementation Notes (Copilot Context)

- The target add-on directory is fixed to `/config/custom_components`.
- Presence check should validate whether `srat` exists in target directory.
- Installed version should be read from `<target>/srat/manifest.json`.
- Missing+disconnected condition must raise only one issue notification (no duplicates/spam).
- Issue should be defined under `backend/src/dto/issue.go` with a `ResolutionLink` that opens a UI dialog.
- Resolution dialog requirements:
  - Show latest available version information
  - Ask explicit user confirmation before any download/install action
- Install and upgrade should share the same artifact path: download `srat.zip` from selected update channel and extract into `/config/custom_components`.
- Channel selection must honor configuration (release vs pre-release).
- Upgrade must run even if component is already installed (overwrite/update existing files safely).
- Frontend Settings → HomeAssistant section should expose three actions: Install, Upgrade, Uninstall.
- Button enabled state must reflect real backend status (installed/not-installed, upgrade availability, operation in progress, etc.).
- After any lifecycle action (install/upgrade/uninstall), prompt for Home Assistant Core restart permission.
- Restart must be executed only when the user confirms, by calling the Home Assistant Core API.
- Ensure status refresh occurs after each action to keep UI state accurate.

## 🔗 Code References & TODOs

- [ ] `TODO: backend/src/dto/issue.go` - Add issue type/entry for missing+disconnected custom component with single-notification behavior
- [ ] `TODO: backend/src/homeassistant/core_service.go` - Create and add service logic for filesystem presence/version checks and lifecycle operations
- [ ] `TODO: backend/src/service/*` - Add channel-aware release/pre-release artifact resolution and `srat.zip` download/extract flow
- [ ] `TODO: backend/src/api/*` - Expose status and install/upgrade/uninstall endpoints (or WebSocket commands) used by frontend
- [ ] `TODO: frontend/src/pages/settings/*` - Add HomeAssistant section action buttons and state-driven enable/disable logic
- [ ] `TODO: frontend/src/components/*` - Add resolution dialog and restart confirmation popup integration
- [ ] `FIXME: frontend/backend contract` - Define explicit response model for installed version, latest version, connectivity, and action availability
