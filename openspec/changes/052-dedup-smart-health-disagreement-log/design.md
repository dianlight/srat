# Design

## Context

Stop the per-poll `SMART lib CheckHealth disagreed with smartctl; trusting smartctl PASSED` WARN (currently emitted on every health check for affected drives, ~1/min) by emitting it once per unique disagreement pattern, mirroring the existing `logHealthBits` once-per-pattern suppression in the exec backend.

## Goals / Non-Goals

**Goals:**

- Deliver: Dedup SMART health disagreement log

**Non-Goals:**

- Unrelated refactoring outside the task scope.

## Decisions

- Add a small per-device disagreement cache (device → last pattern/timestamp) guarded for concurrent access.
- Emit WARN on first sight / pattern change; DEBUG on repeats.
- Unit testing (state transitions: first, repeat suppressed, pattern change re-emits, expiry).
- Remote verification: healthy drive shows one WARN then silence across polls.
- Update docs/CHANGELOG as needed.
- Capture the lessons learned and update documentation.
- Ask to create a PR with the task implementation and link it here for tracking.

## Risks / Trade-offs

- Behaviour changes are covered by targeted tests before handoff.
- Legacy task content remains the reference until the change is archived.
