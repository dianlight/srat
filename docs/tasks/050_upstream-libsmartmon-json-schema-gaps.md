<!-- DOCTOC SKIP -->

# [FIX]: Upstream libsmartmon JSON schema gaps (self-test status, smart-status)

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** https://github.com/dianlight/srat/issues/1196

## 🎯 Objective

Resolve the root cause found in #1196 at its source: the bundled `libsmartmon_go.so` emits an `ata_smart_data.self_test` object containing only `polling_minutes` (verified via strings on the shipped `.so`: `"self_test":{"polling_minutes":{"short":`) and no usable `smart_status`, while its `checkHealth` reports failure on drives that `smartctl -H` passes (Kingston SV300S37A240G, `smartctl -a` exit 0). Either fix upstream (`dianlight/smartmontools-sdk`) or carry a documented vendored patch, then remove the service-layer workarounds (tracker fallback, exec verify client) if they become redundant.

> _Context for Copilot: SRAT currently works around these gaps in `backend/src/service/smart_service.go` (self-test tracker + exec verify client). Those workarounds stay until the schema is fixed; this task tracks the durable fix._

## 🛠️ Technical Specifications

- **Inputs:** Current `libsmartmon_go.so` JSON output for ATA devices; upstream SDK repo access.
- **Outputs:** `self_test.status` (`value`, `string`, `remaining_percent`) and `smart_status.passed` present in lib JSON; `checkHealth` agreeing with `smartctl -H` on reference drives.
- **Dependencies:** `dianlight/smartmontools-sdk` (C wrapper + Go bindings), `backend/patches/` workflow (`mise run //backend:patch`), Kingston SV300S37A240G test box for verification.

## 📝 Task List

- [ ] Task 1: Confirm exact gaps against upstream C source (self_test.status, smart_status.passed, checkHealth logic).
- [ ] Task 2: File upstream issue/PR in `dianlight/smartmontools-sdk` with evidence from #1196.
- [ ] Task 3: If upstream is slow, add a vendored patch under `backend/patches/` following the patch workflow.
- [ ] Task 4: Reassess SRAT workarounds (tracker, verify client); remove or narrow them if redundant.
- [ ] Task 5: Unit testing (service + converter coverage ≥70% on touched code).
- [ ] Task 6: Remote verification on test box (`docs/test/backend/002.003_smart-test-lifecycle.md`).
- [ ] Task 7: Update docs/CHANGELOG as needed.
- [ ] Task 8: Capture the lessons learned and update documentation.
- [ ] Task 9: Ask to create a PR with the task implementation and link it here for tracking.

## 🧠 Implementation Notes (Copilot Context)

- Evidence: `.so` format string shows self_test marshalled with polling_minutes only; API showed `is_test_passed:false` + `unknown/unknown` status via lib while exec `smartctl` reported PASSED and live progress.
- If patching vendored Go bindings only (e.g., populating `SmartStatus` via `checkSmartStatus`-equivalent in lib `GetSMARTInfo`), note it cannot conjure `self_test.status` if the C JSON lacks it — the C layer must emit it.
- Keep the exec verify client until lib ground truth is proven on the Kingston drive.
- Never run raw `go test` in `backend/src`; use `mise run //backend:test`.

## 🔗 Code References & TODOs

- [ ] `backend/src/service/smart_service.go` — `newVerifyClient`, `testTracker`, `GetTestStatus`, `GetHealthStatus` (workarounds to revisit).
- [ ] `backend/src/vendor/github.com/dianlight/smartmontools-sdk/bindings/go/v8/backends/lib/lib.go` — `getSmartJSON`, `CheckHealth`.
- [ ] `backend/src/vendor/github.com/dianlight/smartmontools-sdk/bindings/go/v8/backends/exec/helpers.go` — `checkSmartStatus` (reference semantics).
