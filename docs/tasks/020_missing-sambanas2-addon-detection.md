# [FEATURE]: Missing SambaNAS2 Addon Detection

**Target Repo:** `srat` **Status:** 📅 Planned **Issue Link:** _TBD_

## 🎯 Objective

Detect when the SambaNAS2 add-on is missing, disabled, or unavailable from the Home Assistant Supervisor, and surface a clear, actionable Repair flow so users can recover quickly without manual log inspection.

## 🛠️ Technical Specifications

- **Inputs:**
  - Home Assistant Supervisor add-on metadata/state (Apps API and/or event stream)
  - Existing SRAT startup checks and periodic health checks
  - Existing WebSocket channel between backend and custom component

- **Outputs:**
  - A backend detection path that identifies missing/unavailable SambaNAS2 add-on states
  - A Repair command/event emitted through the existing repair proxy pipeline
  - Frontend and custom component user-facing guidance for remediation
  - Clear state transitions to avoid repeated duplicate alerts for unchanged conditions

- **Dependencies:**
  - `backend/src/service/addon_config_watcher_service.go` (reference pattern for detection + dedupe)
  - `backend/src/service/repair_service.go`
  - `backend/src/api/ws.go`
  - `custom_components/srat/repairs.py`
  - `custom_components/srat/websocket_client.py`
  - Home Assistant Supervisor add-on APIs/events

## 📝 Task List

- [ ] Task 1: Define the missing/unavailable add-on state model and detection rules
- [ ] Task 2: Add backend detection logic (startup + periodic recheck + event-driven where available)
- [ ] Task 3: Add deduplication/idempotency so repeated unchanged states do not spam repairs
- [ ] Task 4: Emit repair create/update/delete actions via existing repair proxy contracts
- [ ] Task 5: Add frontend/custom component messaging for clear remediation guidance
- [ ] Task 6: Unit testing for detection transitions and dedupe behavior
- [ ] Task 7: Integration testing in remote Home Assistant environment
- [ ] Task 8: Integration and documentation updates
- [ ] Task 9: Code review, cleanup, and final validation
- [ ] Task 10: Ask to create a PR with the task implementation and link it here for tracking

## 🧠 Implementation Notes (Copilot Context)

- Reuse the same design principles used in Task 002 (multi-path detection, immediate repair emission, dedupe by stable state/hash) where applicable.
- Prefer state-transition-driven notifications (`healthy -> missing`, `missing -> restored`) rather than timer-driven repeated notifications.
- Ensure recovery path auto-resolves or deletes stale repairs after state restoration.
- Keep repair identifiers stable to preserve idempotency across reconnects/restarts.
- Treat temporary Supervisor API failures as `unknown` and avoid false-positive “missing addon” repairs until confidence threshold is met.
- Keep backend platform-agnostic; Home Assistant-specific repair wiring should stay in custom component proxy boundaries.

## 🔗 Code References & TODOs

- [ ] `TODO: backend/src/service` - Introduce or extend add-on presence detector service
- [ ] `TODO: backend/src/service/repair_service.go` - Add missing-addon repair state integration
- [ ] `TODO: backend/src/api/ws.go` - Ensure detection events are propagated through established channels
- [ ] `TODO: custom_components/srat/repairs.py` - Add/adjust issue definitions and resolution behavior
- [ ] `TODO: docs/HOME_ASSISTANT_INTEGRATION.md` - Document missing-addon detection and recovery flow
