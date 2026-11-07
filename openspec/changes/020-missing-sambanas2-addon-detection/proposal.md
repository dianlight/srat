# Proposal

## Why

Detect when the SambaNAS2 add-on is missing, disabled, or unavailable from the Home Assistant Supervisor, and surface a clear, actionable Repair flow so users can recover quickly without manual log inspection.

## What Changes

- Define the missing/unavailable add-on state model and detection rules
- Add backend detection logic (startup + periodic recheck + event-driven where available)
- Add deduplication/idempotency so repeated unchanged states do not spam repairs
- Emit repair create/update/delete actions via existing repair proxy contracts
- Add frontend/custom component messaging for clear remediation guidance
- Unit testing for detection transitions and dedupe behavior
- Integration testing in remote Home Assistant environment
- Integration and documentation updates
- Code review, cleanup, and final validation
- Ask to create a PR with the task implementation and link it here for tracking

## Capabilities

### New Capabilities

- `missing-sambanas2-addon-detection`: Missing SambaNAS2 Addon Detection

### Modified Capabilities

(none)

## Impact

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
- Legacy task: `docs/tasks/020_missing-sambanas2-addon-detection.md`
