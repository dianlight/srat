# [FEATURE]: Samba Service Health Monitoring

**Target Repo:** `srat` 
**Status:** 📅 Planned 
**Issue Link:** _TBD_

## 🎯 Objective

Add proactive monitoring for Samba service health so SRAT can detect when core Samba daemons are degraded or stopped, notify users through the existing Repairs flow, and provide actionable recovery guidance before shares become silently unavailable.

## 🛠️ Technical Specifications

- **Inputs:**
  - Samba process/service state from backend runtime checks (`smbd`, optional companion daemons)
  - Existing event infrastructure and scheduler/ticker capabilities
  - Current repair proxy pipeline through backend ↔ custom component

- **Outputs:**
  - Reliable backend service-health detector with state transitions (`healthy`, `degraded`, `down`)
  - Deduplicated repair events when service degradation is detected
  - Auto-resolution/cleanup of repairs after healthy recovery
  - User-visible diagnostics and remediation hints

- **Dependencies:**
  - `backend/src/service/system_service.go`
  - `backend/src/service/server_process_service.go`
  - `backend/src/service/repair_service.go`
  - `backend/src/api/ws.go`
  - `custom_components/srat/repairs.py`

## 📝 Task List

- [ ] Task 1: Define Samba service health states and transition rules
- [ ] Task 2: Implement backend health probes (startup + periodic)
- [ ] Task 3: Add debounce/deduplication to avoid noisy repeated alerts
- [ ] Task 4: Emit repair create/update/delete based on state transitions
- [ ] Task 5: Add user-facing remediation messaging for service recovery
- [ ] Task 6: Unit testing for probe logic and transition handling
- [ ] Task 7: Integration testing against Home Assistant add-on runtime
- [ ] Task 8: Integration and documentation updates
- [ ] Task 9: Code review, cleanup, and final validation
- [ ] Task 10: Ask to create a PR with the task implementation and link it here for tracking

## 🧠 Implementation Notes (Copilot Context)

- Use transition-driven alerts instead of constant polling notifications.
- Keep detection resilient to short service restarts during normal operations.
- Prefer existing backend abstractions for command/process checks to keep tests deterministic.
- Reuse established repair identifiers and idempotency strategy from existing repair tasks.
- Ensure recovered state removes stale repair issues automatically.

## 🔗 Code References & TODOs

- [ ] `TODO: backend/src/service` - Add or extend Samba health checker service
- [ ] `TODO: backend/src/service/repair_service.go` - Integrate Samba health repair lifecycle
- [ ] `TODO: backend/src/api/ws.go` - Publish health-related events where needed
- [ ] `TODO: custom_components/srat/repairs.py` - Add Samba service health repair definitions
- [ ] `TODO: docs/SETTINGS_DOCUMENTATION.md` - Document monitoring behavior and user actions
