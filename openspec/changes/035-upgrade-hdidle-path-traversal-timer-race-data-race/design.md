# Design

## Context

Fix four related security and reliability defects in the upgrade and HDIdle subsystems: 1. **ZipSlip bypass** — `extractFile` in `upgrade_service.go` has a `dest != "."` escape that disables path traversal protection when the destination is the current directory.

## Goals / Non-Goals

**Goals:**

- Deliver: Upgrade & HDIdle Path Traversal, Timer Race, and Data Race

**Non-Goals:**

- Unrelated refactoring outside the task scope.

## Decisions

- In `upgrade_service.go extractFile`, remove the `dest != "."` guard — always verify `strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(dest)+string(os.PathSeparator))`
- Add `pattern:"[a-zA-Z0-9_-]+"` constraint to the `DiskID` path parameter struct tags in `hdidle_handler.go` (all four handler methods)
- After constructing `devicePath := "/dev/disk/by-id/" + input.DiskID`, assert `strings.HasPrefix(filepath.Clean(devicePath), "/dev/disk/by-id/")` and return HTTP 400 on violation
- Fix the debounce timer race in `watchForDevelopUpdates`: after `debounceTimer.Stop()` returns false, drain the timer channel (`<-debounceTimer.C`); protect the `selfupdate.Apply` call with a `sync.Mutex` so only one invocation runs at a time
- Fix `HDIdleService.IsRunning()` data race: add `s.mu.RLock()` / `s.mu.RUnlock()`, or replace with an `atomic.Bool running` field updated under the mutex
- Replace `http.DefaultClient.Do(req)` in `upgrade_service.go:553` with a dedicated `http.Client{Timeout: 30*time.Minute}`; remove the `// #nosec G704` suppression
- Add tests: (a) verify `extractFile` rejects `../escape` zip entries with `dest = "."`; (b) verify `DiskID = "../../sda"` returns 400; (c) run `go test -race` on `hdidle_service.go` coverage
- Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark B-SEC-07, B-SEC-08, B-SEC-09, B-REL-05, B-REL-06 resolved

## Risks / Trade-offs

- Behaviour changes are covered by targeted tests before handoff.
- Legacy task content remains the reference until the change is archived.
