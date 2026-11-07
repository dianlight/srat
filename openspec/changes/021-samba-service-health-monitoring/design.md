# Design

## Context

Add proactive monitoring for Samba service health so SRAT can detect when core Samba daemons are degraded or stopped, notify users through the existing Repairs flow, and provide actionable recovery guidance before shares become silently unavailable.

## Goals / Non-Goals

**Goals:**

- Deliver: Samba Service Health Monitoring

**Non-Goals:**

- Unrelated refactoring outside the task scope.

## Decisions

- Define Samba service health states and transition rules
- Implement backend health probes (startup + periodic)
- Add debounce/deduplication to avoid noisy repeated alerts
- Emit repair create/update/delete based on state transitions
- Add user-facing remediation messaging for service recovery
- Unit testing for probe logic and transition handling
- Integration testing against Home Assistant add-on runtime
- Integration and documentation updates

## Risks / Trade-offs

- Behaviour changes are covered by targeted tests before handoff.
- Legacy task content remains the reference until the change is archived.
