# [REFACTOR]: Backend Code Quality — errors.AsType Migration and Service Splits

**Target Repo:** `srat`  
**Status:** 📅 Planned  
**Issue Link:** _TBD_

## 🎯 Objective

Three low-risk, high-value code-quality improvements to the Go backend: (1) migrate four `errors.As(err, &e)` call sites in `unixsamba` to the Go 1.26 `errors.AsType[T]` pattern; (2) split the oversized `filesystem_service.go` (~944 lines) by extracting the async operation runner into a dedicated type; (3) split `hdidle_service.go` (~823 lines) by extracting disk-state tracking into a separate type.

> _Context for Copilot: Go version is 1.26 (see `go.mod`). The `unixsamba` package imports `gitlab.com/tozd/go/errors` (not stdlib). `filesystem_service.go` and `hdidle_service.go` are the two largest service files; splitting them requires no interface changes since public `*Interface` types are preserved._

## 🛠️ Technical Specifications

- **Inputs:** Existing Go source files
- **Outputs:**
  - `unixsamba/unixsamba.go` — four `errors.As` sites updated to `errors.AsType[T]` (or kept as-is if tozd/go/errors does not expose the stdlib form — see notes)
  - `service/filesystem_service.go` → `service/filesystem_service.go` + `service/filesystem_operation_runner.go`
  - `service/hdidle_service.go` → `service/hdidle_service.go` + `service/hdidle_disk_state.go`
- **Dependencies:**
  - `backend/src/unixsamba/unixsamba.go`
  - `backend/src/service/filesystem_service.go`
  - `backend/src/service/hdidle_service.go`

## 📝 Task List

- [ ] Task 1: Check whether `gitlab.com/tozd/go/errors` exposes `AsType[T]`; if yes, migrate the four `errors.As` call sites in `unixsamba/unixsamba.go`; if no, import stdlib `errors` with an alias and use `stderrors.AsType[T]`
- [ ] Task 2: Extract `filesystemOperationRunner` struct from `filesystem_service.go` — move `startOperation`, `finishOperation`, `wg`, `operationMu`, and operation-context tracking to `service/filesystem_operation_runner.go`
- [ ] Task 3: Update `FilesystemService` to embed or hold a `*filesystemOperationRunner`; ensure all callers compile without change
- [ ] Task 4: Extract `hdIdleDiskState` struct from `hdidle_service.go` — move per-disk state tracking fields and their methods to `service/hdidle_disk_state.go`
- [ ] Task 5: Update `HDIdleService` to embed or hold a `*hdIdleDiskState`; ensure all callers compile without change
- [ ] Task 6: Run `go build ./...` and `go vet ./...` to confirm no regressions after each split
- [ ] Task 7: Run existing tests: `cd backend/src && go test ./service ./unixsamba` — all must pass
- [ ] Task 8: Run `go_diagnostics` (gopls) on modified files and fix any reported issues

## 🧠 Implementation Notes (Copilot Context)

### errors.AsType migration

```go
// Current pattern (4 sites in unixsamba/unixsamba.go)
var e errors.E
if errors.As(err, &e) {
    // use e
}

// Option A — if tozd/go/errors exposes AsType
if e, ok := errors.AsType[errors.E](err); ok {
    // use e
}

// Option B — use stdlib with alias
import stderrors "errors"
// ...
if e, ok := stderrors.AsType[tozderrors.E](err); ok { ... }
```
Check by grepping the vendor source: `grep -r "AsType" backend/src/vendor/gitlab.com/tozd/`.

### filesystem_service.go split

Extract the following to `filesystem_operation_runner.go`:
```go
type filesystemOperationRunner struct {
    mu         sync.Mutex
    operations map[string]operationContext // devicePath → ctx + cancel
    wg         sync.WaitGroup
    ctx        context.Context
}

func (r *filesystemOperationRunner) startOperation(devicePath, opType string) (context.Context, context.CancelFunc, errors.E)
func (r *filesystemOperationRunner) finishOperation(devicePath string)
func (r *filesystemOperationRunner) abortOperation(devicePath string) errors.E
```
`FilesystemService` holds `runner *filesystemOperationRunner` and delegates these calls.

### hdidle_service.go split

Extract per-disk state (current idle timers, spindown history) to `hdidle_disk_state.go`:
```go
type hdIdleDiskState struct {
    mu    sync.RWMutex
    state map[string]*DiskIdleInfo // devicePath → info
}
```
`HDIdleService` holds `diskState *hdIdleDiskState`.

### No interface changes required

Both splits move internal helper types and functions only. Public `FilesystemServiceInterface` and `HDIdleServiceInterface` remain unchanged. This means zero impact on API handlers, tests, or FX registration.

## 🔗 Code References & TODOs

- [ ] `backend/src/unixsamba/unixsamba.go` — 4× `var e errors.E; errors.As(err, &e)` pattern
- [ ] `backend/src/service/filesystem_service.go` — ~944 lines; extract operation runner
- [ ] `backend/src/service/hdidle_service.go` — ~823 lines; extract disk-state tracker
- [ ] `docs/FUTURE_IMPROVEMENTS.md` — "Large Service Files — Splitting Candidates" and "errors.As → errors.AsType Migration" sections (remove once done)
