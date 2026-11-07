# Design

## Context

Three low-risk, high-value code-quality improvements to the Go backend: (1) migrate four `errors.As(err, &e)` call sites in `unixsamba` to the Go 1.26 `errors.AsType[T]` pattern; (2) split the oversized `filesystem_service.go` (~944 lines) by extracting the async operation runner into a dedicated type; (3) split `hdidle_service.go` (~823 lines) by extracting disk-state tracking into a separate type.

## Goals / Non-Goals

**Goals:**

- Deliver: Backend Code Quality — errors.AsType Migration and Service Splits

**Non-Goals:**

- Unrelated refactoring outside the task scope.

## Decisions

- Check whether `gitlab.com/tozd/go/errors` exposes `AsType[T]`; if yes, migrate the four `errors.As` call sites in `unixsamba/unixsamba.go`; if no, import stdlib `errors` with an alias and use `stderrors.AsType[T]`
- Extract `filesystemOperationRunner` struct from `filesystem_service.go` — move `startOperation`, `finishOperation`, `wg`, `operationMu`, and operation-context tracking to `service/filesystem_operation_runner.go`
- Update `FilesystemService` to embed or hold a `*filesystemOperationRunner`; ensure all callers compile without change
- Extract `hdIdleDiskState` struct from `hdidle_service.go` — move per-disk state tracking fields and their methods to `service/hdidle_disk_state.go`
- Update `HDIdleService` to embed or hold a `*hdIdleDiskState`; ensure all callers compile without change
- Run `go build ./...` and `go vet ./...` to confirm no regressions after each split
- Run existing tests: `cd backend/src && go test ./service ./unixsamba` — all must pass
- Run `go_diagnostics` (gopls) on modified files and fix any reported issues

## Risks / Trade-offs

- Behaviour changes are covered by targeted tests before handoff.
- Legacy task content remains the reference until the change is archived.
