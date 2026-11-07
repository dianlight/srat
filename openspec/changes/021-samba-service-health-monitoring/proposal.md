# Proposal

## Why

Add proactive monitoring for Samba service health so SRAT can detect when core Samba daemons are degraded or stopped, notify users through the existing Repairs flow, and provide actionable recovery guidance before shares become silently unavailable.

## What Changes

- Define Samba service health states and transition rules
- Implement backend health probes (startup + periodic)
- Add debounce/deduplication to avoid noisy repeated alerts
- Emit repair create/update/delete based on state transitions
- Add user-facing remediation messaging for service recovery
- Unit testing for probe logic and transition handling
- Integration testing against Home Assistant add-on runtime
- Integration and documentation updates
- Code review, cleanup, and final validation
- Ask to create a PR with the task implementation and link it here for tracking

## Capabilities

### New Capabilities

- `samba-service-health-monitoring`: Samba Service Health Monitoring

### Modified Capabilities

(none)

## Impact

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
- Legacy task: `docs/tasks/021_samba-service-health-monitoring.md`
