<!-- DOCTOC SKIP -->

# [FIX]: Upgrade & HDIdle Path Traversal, Timer Race, and Data Race

**Target Repo:** `srat`
**Status:** 📅 Planned
**Issue Link:** _None — discovered in security/reliability review 2026-04-28_

## 🎯 Objective

Fix four related security and reliability defects in the upgrade and HDIdle subsystems:

1. **ZipSlip bypass** — `extractFile` in `upgrade_service.go` has a `dest != "."` escape that disables path traversal protection when the destination is the current directory.
2. **`disk_id` path traversal** — the HDIdle handler concatenates an unvalidated path component into `/dev/disk/by-id/`, allowing `../../sda` traversal.
3. **Debounce timer race** — `watchForDevelopUpdates` can launch two concurrent `selfupdate.Apply` calls when a timer fires while its previous callback is still running.
4. **`HDIdleService.IsRunning()` data race** — reads `s.stopChan` without the mutex, racing with `Start`/`Stop` writes.
5. **`http.DefaultClient` with no timeout** in upgrade download.

## 🛠️ Technical Specifications

- **Inputs:** `backend/src/service/upgrade_service.go`, `backend/src/api/hdidle_handler.go`, `backend/src/service/hdidle_service.go`
- **Outputs:** No path traversal; single concurrent install; race-free running check
- **Dependencies:** `path/filepath`, `net/http`, `sync`, `os`

## 📝 Task List

- [ ] Task 1: In `upgrade_service.go extractFile`, remove the `dest != "."` guard — always verify `strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(dest)+string(os.PathSeparator))`
- [ ] Task 2: Add `pattern:"[a-zA-Z0-9_-]+"` constraint to the `DiskID` path parameter struct tags in `hdidle_handler.go` (all four handler methods)
- [ ] Task 3: After constructing `devicePath := "/dev/disk/by-id/" + input.DiskID`, assert `strings.HasPrefix(filepath.Clean(devicePath), "/dev/disk/by-id/")` and return HTTP 400 on violation
- [ ] Task 4: Fix the debounce timer race in `watchForDevelopUpdates`: after `debounceTimer.Stop()` returns false, drain the timer channel (`<-debounceTimer.C`); protect the `selfupdate.Apply` call with a `sync.Mutex` so only one invocation runs at a time
- [ ] Task 5: Fix `HDIdleService.IsRunning()` data race: add `s.mu.RLock()` / `s.mu.RUnlock()`, or replace with an `atomic.Bool running` field updated under the mutex
- [ ] Task 6: Replace `http.DefaultClient.Do(req)` in `upgrade_service.go:553` with a dedicated `http.Client{Timeout: 30*time.Minute}`; remove the `// #nosec G704` suppression
- [ ] Task 7: Add tests: (a) verify `extractFile` rejects `../escape` zip entries with `dest = "."`; (b) verify `DiskID = "../../sda"` returns 400; (c) run `go test -race` on `hdidle_service.go` coverage
- [ ] Task 8: Update `docs/SECURITY_OPTIMIZATION_REVIEW.md` to mark B-SEC-07, B-SEC-08, B-SEC-09, B-REL-05, B-REL-06 resolved

## 🧠 Implementation Notes

```go
// upgrade_service.go — ZipSlip fix
func extractFile(dest string, f *zip.File) (os.FileInfo, error) {
    path := filepath.Join(dest, f.Name)
    // Remove the `dest != "."` escape:
    if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
        return nil, errors.Errorf("illegal file path: %s", path)
    }
    ...
}
```

```go
// hdidle_handler.go — disk_id validation struct tag
DiskID string `path:"disk_id" pattern:"[a-zA-Z0-9_-]+" maxLength:"255" doc:"Disk identifier"`
```

```go
// upgrade_service.go — dedicated HTTP client
var upgradeHTTPClient = &http.Client{
    Timeout: 30 * time.Minute,
}
// Replace: http.DefaultClient.Do(req)
// With:    upgradeHTTPClient.Do(req)
```

```go
// hdidle_service.go — atomic running flag
type HDIdleService struct {
    ...
    running atomic.Bool
}
func (s *HDIdleService) IsRunning() bool { return s.running.Load() }
// In Start: s.running.Store(true)
// In Stop:  s.running.Store(false)
```

## 🔗 Code References & TODOs

- [ ] `TODO: backend/src/service/upgrade_service.go:467-470` — remove dest != "." escape
- [ ] `TODO: backend/src/api/hdidle_handler.go:60,80,116,220` — add pattern constraint + path assertion
- [ ] `TODO: backend/src/service/upgrade_service.go:251-256` — fix debounce timer race
- [ ] `TODO: backend/src/service/hdidle_service.go` — fix IsRunning data race
- [ ] `TODO: backend/src/service/upgrade_service.go:553` — replace DefaultClient
