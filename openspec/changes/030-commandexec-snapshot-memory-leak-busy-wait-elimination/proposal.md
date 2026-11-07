# Proposal

## Why

Fix two related performance defects in `backend/src/internal/commandexec/runner.go`: 1. **Memory leak** — completed execution snapshots are stored in `Service.snapshots` indefinitely and never evicted. Each entry can hold up to 500 `CommandOutputLineSnapshot` structs. Long-running servers accumulate this data from filesystem checks, SMART tests, samba restarts, and upgrade flows.

## What Changes

- Add `doneCh chan struct{}` to `executionState` struct alongside `snapshot` and `quiet`
- Close `doneCh` in the `cmd.Wait()` goroutine immediately after writing the final snapshot state, before emitting the STOP/ERROR event
- Replace the `time.NewTicker(10ms)` poll in `executeWithInput` with a `select { case <-doneCh; case <-ctx.Done() }` wait
- Add a TTL eviction goroutine (or integrate with `GetSnapshot`) that removes entries where `!snapshot.Running && time.Since(finishedAt) > evictionTTL`; default `evictionTTL = 5 * time.Minute`, configurable via `NewCommandExecutor` option
- Add a `maxSnapshots` cap (default 500) to prevent unbounded growth when TTL eviction can't keep up; evict oldest finished entries first
- Update `runner_test.go` to verify: (a) finished snapshots are evicted after TTL; (b) `Execute` returns without polling; (c) concurrent `Start` + `GetSnapshot` is race-free (`go test -race`)
- Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark B-PERF-01 and B-PERF-02 resolved

## Capabilities

### New Capabilities

- `commandexec-snapshot-memory-leak-busy-wait-elimination`: commandexec Snapshot Memory Leak and Busy-Wait Elimination

### Modified Capabilities

(none)

## Impact

- **Inputs:** `commandexec.Service` internal state; `ExecutionSnapshot.Running` flag
- **Outputs:** Completed snapshots evicted after a configurable TTL; synchronous callers block on a channel signal instead of polling
- **Dependencies:** `backend/src/internal/commandexec/runner.go`, `backend/src/internal/commandexec/runner_test.go`
- Legacy task: `docs/tasks/030_commandexec-memory-leak-snapshot-eviction.md`
