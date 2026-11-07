# Design

## Context

Detect when the SambaNAS2 add-on is missing, disabled, or unavailable from the Home Assistant Supervisor, and surface a clear, actionable Repair flow so users can recover quickly without manual log inspection.

## Goals / Non-Goals

**Goals:**

- Deliver: Missing SambaNAS2 Addon Detection

**Non-Goals:**

- Unrelated refactoring outside the task scope.

## Decisions

- Define the missing/unavailable add-on state model and detection rules
- Add backend detection logic (startup + periodic recheck + event-driven where available)
- Add deduplication/idempotency so repeated unchanged states do not spam repairs
- Emit repair create/update/delete actions via existing repair proxy contracts
- Add frontend/custom component messaging for clear remediation guidance
- Unit testing for detection transitions and dedupe behavior
- Integration testing in remote Home Assistant environment
- Integration and documentation updates

## Risks / Trade-offs

- Behaviour changes are covered by targeted tests before handoff.
- Legacy task content remains the reference until the change is archived.
