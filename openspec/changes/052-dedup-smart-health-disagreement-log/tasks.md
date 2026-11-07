# Tasks

## 1. Implementation

- [ ] 1.1 Add a small per-device disagreement cache (device → last pattern/timestamp) guarded for concurrent access — verified by targeted tests
- [ ] 1.2 Emit WARN on first sight / pattern change; DEBUG on repeats — verified by targeted tests
- [ ] 1.3 Unit testing (state transitions: first, repeat suppressed, pattern change re-emits, expiry) — verified by targeted tests
- [ ] 1.4 Remote verification: healthy drive shows one WARN then silence across polls — verified by targeted tests
- [ ] 1.5 Update docs/CHANGELOG as needed — verified by targeted tests
- [ ] 1.6 Capture the lessons learned and update documentation — verified by targeted tests
- [ ] 1.7 Ask to create a PR with the task implementation and link it here for tracking — verified by targeted tests

## 2. Verification

- [ ] 2.1 Run the relevant test suites and confirm the change passes — verified by a green test run
