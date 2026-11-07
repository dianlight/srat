# Design

## Context

Resolve the root cause found in #1196 at its source: the bundled `libsmartmon_go.so` emits an `ata_smart_data.self_test` object containing only `polling_minutes` (verified via strings on the shipped `.so`: `"self_test":{"polling_minutes":{"short":`) and no usable `smart_status`, while its `checkHealth` reports failure on drives that `smartctl -H` passes (Kingston SV300S37A240G, `smartctl -a` exit 0). Either fix upstream (`dianlight/smartmontools-sdk`) or carry a documented vendored patch, then remove the service-layer workarounds (tracker fallback, exec verify client) if they become redundant.

## Goals / Non-Goals

**Goals:**

- Deliver: Upstream libsmartmon JSON schema gaps (self-test status, smart-status)

**Non-Goals:**

- Unrelated refactoring outside the task scope.

## Decisions

- Confirm exact gaps against upstream C source (self_test.status, smart_status.passed, checkHealth logic).
- File upstream issue/PR in `dianlight/smartmontools-sdk` with evidence from #1196.
- If upstream is slow, add a vendored patch under `backend/patches/` following the patch workflow.
- Reassess SRAT workarounds (tracker, verify client); remove or narrow them if redundant.
- Unit testing (service + converter coverage ≥70% on touched code).
- Remote verification on test box (`docs/test/backend/002.003_smart-test-lifecycle.md`).
- Update docs/CHANGELOG as needed.
- Capture the lessons learned and update documentation.

## Risks / Trade-offs

- Behaviour changes are covered by targeted tests before handoff.
- Legacy task content remains the reference until the change is archived.
