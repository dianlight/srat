<!-- DOCTOC SKIP -->

# [REFACTOR]: Dedup SMART health disagreement log

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** https://github.com/dianlight/srat/issues/1196

## 🎯 Objective

Stop the per-poll `SMART lib CheckHealth disagreed with smartctl; trusting smartctl PASSED` WARN (currently emitted on every health check for affected drives, ~1/min) by emitting it once per unique disagreement pattern, mirroring the existing `logHealthBits` once-per-pattern suppression in the exec backend.

> _Context for Copilot: The #1196 fix replaced the scary per-minute `SMART pre-failure detected` spam (empty attributes) with an honest disagreement notice, but it still fires every poll. Suppress repeats so logs stay actionable._

## 🛠️ Technical Specifications

- **Inputs:** Health-check calls on drives where lib `CheckHealth=false` but exec verify passes.
- **Outputs:** First occurrence logged at WARN with device + versions; repeats suppressed until the pattern changes (or a cooldown expires).
- **Dependencies:** `backend/src/service/smart_service.go` (`GetHealthStatus`); precedent: `logHealthBits` in `backends/exec/exec_backend.go` (per-device once-per-pattern cache with mutex).

## 📝 Task List

- [ ] Task 1: Add a small per-device disagreement cache (device → last pattern/timestamp) guarded for concurrent access.
- [ ] Task 2: Emit WARN on first sight / pattern change; DEBUG on repeats.
- [ ] Task 3: Unit testing (state transitions: first, repeat suppressed, pattern change re-emits, expiry).
- [ ] Task 4: Remote verification: healthy drive shows one WARN then silence across polls.
- [ ] Task 5: Update docs/CHANGELOG as needed.
- [ ] Task 6: Capture the lessons learned and update documentation.
- [ ] Task 7: Ask to create a PR with the task implementation and link it here for tracking.

## 🧠 Implementation Notes (Copilot Context)

- Pattern key suggestion: `deviceId + CheckHealth result + verify result + OverallStatus`.
- Keep it in the service layer (the exec `healthBitsCache` lives in the vendored backend and is out of reach without a patch).
- Mind the existing mutexes (`s.mutex`, `testMu`); use a dedicated mutex or atomics, never nest under `s.mutex` in callbacks.
- Never run raw `go test` in `backend/src`; use `mise run //backend:test`. New/changed code needs ≥70% statement coverage.

## 🔗 Code References & TODOs

- [ ] `backend/src/service/smart_service.go` — `GetHealthStatus` disagreement WARN (~line 541).
- [ ] `backend/src/vendor/github.com/dianlight/smartmontools-sdk/bindings/go/v8/backends/exec/exec_backend.go` — `logHealthBits` (suppression precedent).
