<!-- DOCTOC SKIP -->

# [FIX]: commandexec Snapshot Memory Leak and Busy-Wait Elimination

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** _None — discovered in performance review 2026-04-28_

## 🎯 Objective

Fix two related performance defects in `backend/src/internal/commandexec/runner.go`:

1. **Memory leak** — completed execution snapshots are stored in `Service.snapshots` indefinitely and never evicted. Each entry can hold up to 500 `CommandOutputLineSnapshot` structs. Long-running servers accumulate this data from filesystem checks, SMART tests, samba restarts, and upgrade flows.
2. **Busy-wait poll** — `executeWithInput` polls the snapshots map every 10 ms with a `time.Ticker`, spinning a goroutine and contending on `sync.RWMutex` for the full duration of every synchronous command.

> *These two fixes are bundled because they both touch the same `runner.go` internals and share the same test suite.*

## 🛠️ Technical Specifications

- **Inputs:** `commandexec.Service` internal state; `ExecutionSnapshot.Running` flag
- **Outputs:** Completed snapshots evicted after a configurable TTL; synchronous callers block on a channel signal instead of polling
- **Dependencies:** `backend/src/internal/commandexec/runner.go`, `backend/src/internal/commandexec/runner_test.go`

## 📝 Task List

- [ ] Task 1: Add `doneCh chan struct{}` to `executionState` struct alongside `snapshot` and `quiet`
- [ ] Task 2: Close `doneCh` in the `cmd.Wait()` goroutine immediately after writing the final snapshot state, before emitting the STOP/ERROR event
- [ ] Task 3: Replace the `time.NewTicker(10ms)` poll in `executeWithInput` with a `select { case <-doneCh; case <-ctx.Done() }` wait
- [ ] Task 4: Add a TTL eviction goroutine (or integrate with `GetSnapshot`) that removes entries where `!snapshot.Running && time.Since(finishedAt) > evictionTTL`; default `evictionTTL = 5 * time.Minute`, configurable via `NewCommandExecutor` option
- [ ] Task 5: Add a `maxSnapshots` cap (default 500) to prevent unbounded growth when TTL eviction can't keep up; evict oldest finished entries first
- [ ] Task 6: Update `runner_test.go` to verify: (a) finished snapshots are evicted after TTL; (b) `Execute` returns without polling; (c) concurrent `Start` + `GetSnapshot` is race-free (`go test -race`)
- [ ] Task 7: Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark B-PERF-01 and B-PERF-02 resolved

## 🧠 Implementation Notes

```go
type executionState struct {
    snapshot dto.CommandExecutionSnapshot
    quiet    bool
    doneCh   chan struct{} // closed when Running transitions to false
}

// In executeWithInput — replace ticker poll:
state, _ := s.snapshots[executionID]
select {
case <-state.doneCh:
case <-ctx.Done():
    return dto.CommandExecutionSnapshot{}, ctx.Err()
}
snapshot, _ := s.GetSnapshot(executionID)
```

For TTL eviction, a simple background ticker at `evictionTTL/2` interval is sufficient:

```go
go func() {
    ticker := time.NewTicker(evictionTTL / 2)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            s.evictExpired()
        case <-ctx.Done():
            return
        }
    }
}()
```

`evictExpired` acquires a write lock, scans for entries where `!snapshot.Running` and `time.Since(finishedAt) > TTL`, and deletes them.

## 🔗 Code References & TODOs

- [ ] `TODO: backend/src/internal/commandexec/runner.go:41` — add `doneCh` to `executionState`
- [ ] `TODO: backend/src/internal/commandexec/runner.go:232-254` — replace polling with channel wait
- [ ] `TODO: backend/src/internal/commandexec/runner.go:49-54` — add eviction goroutine to constructor
