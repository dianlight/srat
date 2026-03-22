# [FEATURE]: HA SRAT Connectivity Loss Detection

**Target Repo:** `srat` **Status:** 📅 Planned **Issue Link:** _TBD_

## 🎯 Objective

Detect and surface connectivity loss between Home Assistant and SRAT (especially WebSocket disruption and prolonged reconnection failure), then trigger clear Repairs guidance so users can quickly restore control-plane communication.

## 🛠️ Technical Specifications

- **Inputs:**
  - WebSocket connection lifecycle signals (`connected`, `disconnected`, `reconnecting`)
  - Heartbeat/timestamp state from backend and custom component
  - Existing event propagation and repair proxy mechanisms

- **Outputs:**
  - Connectivity watchdog that classifies outage severity/duration
  - Repair events for prolonged or repeated disconnection scenarios
  - Auto-resolution of connectivity repairs on stable reconnection
  - Diagnostics context (last seen time, retry attempts, likely causes)

- **Dependencies:**
  - `backend/src/api/ws.go`
  - `backend/src/service/broadcaster_service.go`
  - `backend/src/service/repair_service.go`
  - `custom_components/srat/websocket_client.py`
  - `custom_components/srat/__init__.py`

## 📝 Task List

- [ ] Task 1: Define connectivity states and outage thresholds
- [ ] Task 2: Implement backend + custom component watchdog inputs
- [ ] Task 3: Add transition-based outage detection and dedupe rules
- [ ] Task 4: Emit repair lifecycle events for prolonged connectivity loss
- [ ] Task 5: Auto-resolve repairs after stable reconnection window
- [ ] Task 6: Add user-facing diagnostics and recommended recovery actions
- [ ] Task 7: Unit and integration tests for disconnect/reconnect edge cases
- [ ] Task 8: Integration and documentation updates
- [ ] Task 9: Code review, cleanup, and final validation
- [ ] Task 10: Ask to create a PR with the task implementation and link it here for tracking

## 🧠 Implementation Notes (Copilot Context)

- Keep short transient reconnects quiet to avoid false alarms.
- Use stability windows before declaring both outage and recovery.
- Prefer one canonical connectivity status source to avoid split-brain between backend and component views.
- Use stable repair IDs so reconnect flapping does not create duplicate issues.
- Include enough diagnostics in events to aid support and remote troubleshooting.

## 🔗 Code References & TODOs

- [ ] `TODO: backend/src/api/ws.go` - Expose reliable connection lifecycle hooks
- [ ] `TODO: backend/src/service/repair_service.go` - Integrate connectivity outage repair lifecycle
- [ ] `TODO: custom_components/srat/websocket_client.py` - Publish reconnection and heartbeat metadata
- [ ] `TODO: custom_components/srat/__init__.py` - Handle repair-related connectivity callbacks
- [ ] `TODO: docs/HOME_ASSISTANT_INTEGRATION.md` - Document outage detection and recovery workflow
