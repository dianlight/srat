# [FEATURE]: Disk Health Degradation Alerts

**Target Repo:** `srat` 
**Status:** 📅 Planned 
**Issue Link:** _TBD_

## 🎯 Objective

Introduce proactive disk health degradation detection (SMART, capacity pressure, mount-related instability) and route high-confidence warnings through the Repairs pipeline so users can intervene before data loss or service interruption occurs.

## 🛠️ Technical Specifications

- **Inputs:**
  - Existing SMART and disk status data available in backend services
  - Filesystem/mount state and capacity telemetry
  - Existing event bus and repair proxy channel

- **Outputs:**
  - Unified disk-risk evaluator producing normalized severity levels
  - Deduplicated repair notifications for actionable degradations
  - Clear recovery/resolution behavior when conditions improve
  - Documentation for thresholds and operator guidance

- **Dependencies:**
  - `backend/src/service/smart_service.go`
  - `backend/src/service/filesystem/`
  - `backend/src/service/repair_service.go`
  - `backend/src/events/`
  - `custom_components/srat/repairs.py`

## 📝 Task List

- [ ] Task 1: Define degradation taxonomy and severity thresholds
- [ ] Task 2: Implement disk-health aggregation logic from SMART + filesystem signals
- [ ] Task 3: Add confidence gating to reduce false positives on transient errors
- [ ] Task 4: Emit repair create/update/delete events with stable identifiers
- [ ] Task 5: Add user guidance for mitigation and urgency levels
- [ ] Task 6: Unit testing for threshold, aggregation, and dedupe logic
- [ ] Task 7: Integration tests covering degraded → recovered transitions
- [ ] Task 8: Integration and documentation updates
- [ ] Task 9: Code review, cleanup, and final validation
- [ ] Task 10: Ask to create a PR with the task implementation and link it here for tracking

## 🧠 Implementation Notes (Copilot Context)

- Favor high-signal indicators first (hard SMART failures, repeated read/write error patterns).
- Keep thresholds configurable enough for future tuning, but ship conservative defaults.
- Avoid alert storms by binding notifications to state transitions and significant severity changes.
- Reconcile/resolve prior repairs automatically once the disk state is healthy for a stability window.
- Preserve backend platform-agnostic design while using HA repairs for user-facing workflow.

## 🔗 Code References & TODOs

- [ ] `TODO: backend/src/service/smart_service.go` - Expose normalized health indicators
- [ ] `TODO: backend/src/service/filesystem` - Contribute mount/capacity risk signals
- [ ] `TODO: backend/src/service/repair_service.go` - Wire disk degradation lifecycle to repairs
- [ ] `TODO: custom_components/srat/repairs.py` - Define disk-health issue payloads and resolution handling
- [ ] `TODO: docs/SMART_FRONTEND_ARCHITECTURE.md` - Document alert behavior and operator expectations
