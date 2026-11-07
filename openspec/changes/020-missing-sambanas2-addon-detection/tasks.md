# Tasks

## 1. Implementation

- [ ] 1.1 Define the missing/unavailable add-on state model and detection rules — verified by targeted tests
- [ ] 1.2 Add backend detection logic (startup + periodic recheck + event-driven where available) — verified by targeted tests
- [ ] 1.3 Add deduplication/idempotency so repeated unchanged states do not spam repairs — verified by targeted tests
- [ ] 1.4 Emit repair create/update/delete actions via existing repair proxy contracts — verified by targeted tests
- [ ] 1.5 Add frontend/custom component messaging for clear remediation guidance — verified by targeted tests
- [ ] 1.6 Unit testing for detection transitions and dedupe behavior — verified by targeted tests
- [ ] 1.7 Integration testing in remote Home Assistant environment — verified by targeted tests
- [ ] 1.8 Integration and documentation updates — verified by targeted tests
- [ ] 1.9 Code review, cleanup, and final validation — verified by targeted tests
- [ ] 1.10 Ask to create a PR with the task implementation and link it here for tracking — verified by targeted tests

## 2. Verification

- [ ] 2.1 Run the relevant test suites and confirm the change passes — verified by a green test run
