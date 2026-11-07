# Proposal

## Why

Detect and surface connectivity loss between Home Assistant and SRAT (especially WebSocket disruption and prolonged reconnection failure), then trigger clear Repairs guidance so users can quickly restore control-plane communication.

## What Changes

- Define connectivity states and outage thresholds
- Implement backend + custom component watchdog inputs
- Add transition-based outage detection and dedupe rules
- Emit repair lifecycle events for prolonged connectivity loss
- Auto-resolve repairs after stable reconnection window
- Add user-facing diagnostics and recommended recovery actions
- Unit and integration tests for disconnect/reconnect edge cases
- Integration and documentation updates
- Code review, cleanup, and final validation
- Ask to create a PR with the task implementation and link it here for tracking

## Capabilities

### New Capabilities

- `ha-srat-connectivity-loss-detection`: HA SRAT Connectivity Loss Detection

### Modified Capabilities

(none)

## Impact

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
- Legacy task: `docs/tasks/023_ha-srat-connectivity-loss-detection.md`
