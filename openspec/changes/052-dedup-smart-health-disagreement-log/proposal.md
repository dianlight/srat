# Proposal

## Why

Stop the per-poll `SMART lib CheckHealth disagreed with smartctl; trusting smartctl PASSED` WARN (currently emitted on every health check for affected drives, ~1/min) by emitting it once per unique disagreement pattern, mirroring the existing `logHealthBits` once-per-pattern suppression in the exec backend.

## What Changes

- Add a small per-device disagreement cache (device → last pattern/timestamp) guarded for concurrent access.
- Emit WARN on first sight / pattern change; DEBUG on repeats.
- Unit testing (state transitions: first, repeat suppressed, pattern change re-emits, expiry).
- Remote verification: healthy drive shows one WARN then silence across polls.
- Update docs/CHANGELOG as needed.
- Capture the lessons learned and update documentation.
- Ask to create a PR with the task implementation and link it here for tracking.

## Capabilities

### New Capabilities

- `dedup-smart-health-disagreement-log`: Dedup SMART health disagreement log

### Modified Capabilities

(none)

## Impact

- **Inputs:** Health-check calls on drives where lib `CheckHealth=false` but exec verify passes.
- **Outputs:** First occurrence logged at WARN with device + versions; repeats suppressed until the pattern changes (or a cooldown expires).
- **Dependencies:** `backend/src/service/smart_service.go` (`GetHealthStatus`); precedent: `logHealthBits` in `backends/exec/exec_backend.go` (per-device once-per-pattern cache with mutex).
- Legacy task: `docs/tasks/052_dedup-smart-health-disagreement-log.md`
