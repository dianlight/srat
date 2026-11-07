# Proposal

## Why

Three low-risk, high-value code-quality improvements to the Go backend: (1) migrate four `errors.As(err, &e)` call sites in `unixsamba` to the Go 1.26 `errors.AsType[T]` pattern; (2) split the oversized `filesystem_service.go` (~944 lines) by extracting the async operation runner into a dedicated type; (3) split `hdidle_service.go` (~823 lines) by extracting disk-state tracking into a separate type.

## What Changes

- Check whether `gitlab.com/tozd/go/errors` exposes `AsType[T]`; if yes, migrate the four `errors.As` call sites in `unixsamba/unixsamba.go`; if no, import stdlib `errors` with an alias and use `stderrors.AsType[T]`
- Extract `filesystemOperationRunner` struct from `filesystem_service.go` — move `startOperation`, `finishOperation`, `wg`, `operationMu`, and operation-context tracking to `service/filesystem_operation_runner.go`
- Update `FilesystemService` to embed or hold a `*filesystemOperationRunner`; ensure all callers compile without change
- Extract `hdIdleDiskState` struct from `hdidle_service.go` — move per-disk state tracking fields and their methods to `service/hdidle_disk_state.go`
- Update `HDIdleService` to embed or hold a `*hdIdleDiskState`; ensure all callers compile without change
- Run `go build ./...` and `go vet ./...` to confirm no regressions after each split
- Run existing tests: `cd backend/src && go test ./service ./unixsamba` — all must pass
- Run `go_diagnostics` (gopls) on modified files and fix any reported issues
- Remove stale "errors.As → errors.AsType Migration" section from `docs/FUTURE_IMPROVEMENTS.md`

## Capabilities

### New Capabilities

- `backend-code-quality-errors-astype-migration-service-splits`: Backend Code Quality — errors.AsType Migration and Service Splits

### Modified Capabilities

(none)

## Impact

- **Inputs:** Existing Go source files
- **Outputs:**
- `unixsamba/unixsamba.go` — four `errors.As` sites updated to `errors.AsType[T]` (or kept as-is if tozd/go/errors does not expose the stdlib form — see notes)
- `service/filesystem_service.go` → `service/filesystem_service.go` + `service/filesystem_operation_runner.go`
- `service/hdidle_service.go` → `service/hdidle_service.go` + `service/hdidle_disk_state.go`
- **Dependencies:**
- `backend/src/unixsamba/unixsamba.go`
- `backend/src/service/filesystem_service.go`
- `backend/src/service/hdidle_service.go`
- Legacy task: `docs/tasks/007_backend-refactoring-errors-service-splits.md`
