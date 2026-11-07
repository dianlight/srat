# Tasks

## 1. Implementation

- [ ] 1.1 In `upgrade_service.go extractFile`, remove the `dest != "."` guard — always verify `strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(dest)+string(os.PathSeparator))` — verified by targeted tests
- [ ] 1.2 Add `pattern:"[a-zA-Z0-9_-]+"` constraint to the `DiskID` path parameter struct tags in `hdidle_handler.go` (all four handler methods) — verified by targeted tests
- [ ] 1.3 After constructing `devicePath := "/dev/disk/by-id/" + input.DiskID`, assert `strings.HasPrefix(filepath.Clean(devicePath), "/dev/disk/by-id/")` and return HTTP 400 on violation — verified by targeted tests
- [ ] 1.4 Fix the debounce timer race in `watchForDevelopUpdates`: after `debounceTimer.Stop()` returns false, drain the timer channel (`<-debounceTimer.C`); protect the `selfupdate.Apply` call with a `sync.Mutex` so only one invocation runs at a time — verified by targeted tests
- [ ] 1.5 Fix `HDIdleService.IsRunning()` data race: add `s.mu.RLock()` / `s.mu.RUnlock()`, or replace with an `atomic.Bool running` field updated under the mutex — verified by targeted tests
- [ ] 1.6 Replace `http.DefaultClient.Do(req)` in `upgrade_service.go:553` with a dedicated `http.Client{Timeout: 30*time.Minute}`; remove the `// #nosec G704` suppression — verified by targeted tests
- [ ] 1.7 Add tests: (a) verify `extractFile` rejects `../escape` zip entries with `dest = "."`; (b) verify `DiskID = "../../sda"` returns 400; (c) run `go test -race` on `hdidle_service.go` coverage — verified by targeted tests
- [ ] 1.8 Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark B-SEC-07, B-SEC-08, B-SEC-09, B-REL-05, B-REL-06 resolved — verified by targeted tests

## 2. Verification

- [ ] 2.1 Run the relevant test suites and confirm the change passes — verified by a green test run
