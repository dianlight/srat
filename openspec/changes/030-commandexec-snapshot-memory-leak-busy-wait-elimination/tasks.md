# Tasks

## 1. Implementation

- [ ] 1.1 Add `doneCh chan struct{}` to `executionState` struct alongside `snapshot` and `quiet` — verified by targeted tests
- [ ] 1.2 Close `doneCh` in the `cmd.Wait()` goroutine immediately after writing the final snapshot state, before emitting the STOP/ERROR event — verified by targeted tests
- [ ] 1.3 Replace the `time.NewTicker(10ms)` poll in `executeWithInput` with a `select { case <-doneCh; case <-ctx.Done() }` wait — verified by targeted tests
- [ ] 1.4 Add a TTL eviction goroutine (or integrate with `GetSnapshot`) that removes entries where `!snapshot.Running && time.Since(finishedAt) > evictionTTL`; default `evictionTTL = 5 * time.Minute`, configurable via `NewCommandExecutor` option — verified by targeted tests
- [ ] 1.5 Add a `maxSnapshots` cap (default 500) to prevent unbounded growth when TTL eviction can't keep up; evict oldest finished entries first — verified by targeted tests
- [ ] 1.6 Update `runner_test.go` to verify: (a) finished snapshots are evicted after TTL; (b) `Execute` returns without polling; (c) concurrent `Start` + `GetSnapshot` is race-free (`go test -race`) — verified by targeted tests
- [ ] 1.7 Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark B-PERF-01 and B-PERF-02 resolved — verified by targeted tests

## 2. Verification

- [ ] 2.1 Run the relevant test suites and confirm the change passes — verified by a green test run
