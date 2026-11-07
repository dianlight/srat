# Proposal

## Why

Introduce proactive disk health degradation detection (SMART, capacity pressure, mount-related instability) and route high-confidence warnings through the Repairs pipeline so users can intervene before data loss or service interruption occurs.

## What Changes

- Define degradation taxonomy and severity thresholds
- Implement disk-health aggregation logic from SMART + filesystem signals
- Add confidence gating to reduce false positives on transient errors
- Emit repair create/update/delete events with stable identifiers
- Add user guidance for mitigation and urgency levels
- Unit testing for threshold, aggregation, and dedupe logic
- Integration tests covering degraded → recovered transitions
- Integration and documentation updates
- Code review, cleanup, and final validation
- Ask to create a PR with the task implementation and link it here for tracking

## Capabilities

### New Capabilities

- `disk-health-degradation-alerts`: Disk Health Degradation Alerts

### Modified Capabilities

(none)

## Impact

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
- Legacy task: `docs/tasks/022_disk-health-degradation-alerts.md`
