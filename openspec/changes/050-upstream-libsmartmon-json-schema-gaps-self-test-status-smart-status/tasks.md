# Tasks

## 1. Implementation

- [ ] 1.1 Confirm exact gaps against upstream C source (self_test.status, smart_status.passed, checkHealth logic) — verified by targeted tests
- [ ] 1.2 File upstream issue/PR in `dianlight/smartmontools-sdk` with evidence from #1196 — verified by targeted tests
- [ ] 1.3 If upstream is slow, add a vendored patch under `backend/patches/` following the patch workflow — verified by targeted tests
- [ ] 1.4 Reassess SRAT workarounds (tracker, verify client); remove or narrow them if redundant — verified by targeted tests
- [ ] 1.5 Unit testing (service + converter coverage ≥70% on touched code) — verified by targeted tests
- [ ] 1.6 Remote verification on test box (`docs/test/backend/002.003_smart-test-lifecycle.md`) — verified by targeted tests
- [ ] 1.7 Update docs/CHANGELOG as needed — verified by targeted tests
- [ ] 1.8 Capture the lessons learned and update documentation — verified by targeted tests
- [ ] 1.9 Ask to create a PR with the task implementation and link it here for tracking — verified by targeted tests

## 2. Verification

- [ ] 2.1 Run the relevant test suites and confirm the change passes — verified by a green test run
