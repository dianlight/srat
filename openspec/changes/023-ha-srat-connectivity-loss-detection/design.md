# Design

## Context

Detect and surface connectivity loss between Home Assistant and SRAT (especially WebSocket disruption and prolonged reconnection failure), then trigger clear Repairs guidance so users can quickly restore control-plane communication.

## Goals / Non-Goals

**Goals:**

- Deliver: HA SRAT Connectivity Loss Detection

**Non-Goals:**

- Unrelated refactoring outside the task scope.

## Decisions

- Define connectivity states and outage thresholds
- Implement backend + custom component watchdog inputs
- Add transition-based outage detection and dedupe rules
- Emit repair lifecycle events for prolonged connectivity loss
- Auto-resolve repairs after stable reconnection window
- Add user-facing diagnostics and recommended recovery actions
- Unit and integration tests for disconnect/reconnect edge cases
- Integration and documentation updates

## Risks / Trade-offs

- Behaviour changes are covered by targeted tests before handoff.
- Legacy task content remains the reference until the change is archived.
